package handlers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"boyan/internal/i18n"
	"boyan/internal/parsers/fb2"
	"boyan/internal/watcher"
)

type UploadHandler struct {
	watcher *watcher.Watcher
}

func NewUploadHandler(w *watcher.Watcher) *UploadHandler {
	return &UploadHandler{watcher: w}
}

// UploadBook обрабатывает загрузку файла книги через multipart-форму.
func (h *UploadHandler) UploadBook(w http.ResponseWriter, r *http.Request) {
	// Ограничиваем размер загружаемого файла (например, 100 МБ)
	if err := r.ParseMultipartForm(100 * 1024 * 1024); err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "UPLOAD_PARSE_FAILED", err)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "UPLOAD_MISSING_FILE")
		return
	}
	defer file.Close()

	// Сохраняем во временный файл
	tempDir := filepath.Join(".", "data", "tmp")
	_ = os.MkdirAll(tempDir, 0755)

	tempFile, err := os.CreateTemp(tempDir, "upload-*"+filepath.Ext(header.Filename))
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "UPLOAD_TEMP_FAILED")
		return
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if _, err := io.Copy(tempFile, file); err != nil {
		tempFile.Close()
		writeAPIError(w, r, http.StatusInternalServerError, "UPLOAD_SAVE_FAILED")
		return
	}
	tempFile.Close()

	// Автоматическая санитизация входящих FB2 файлов перед инжестом
	if strings.HasSuffix(strings.ToLower(header.Filename), ".fb2") {
		_, _ = fb2.SanitizeFB2File(tempPath)
	}

	// Запускаем пайплайн обработки через Watcher
	res, err := h.watcher.ProcessFile(r.Context(), tempPath, header.Filename)
	if err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "UPLOAD_FAILED", err)
		return
	}

	// Локализуем статусное сообщение для клиента
	tr := i18n.FromContext(r.Context())
	switch res.Status {
	case "imported":
		res.Message = tr.T("WATCHER_IMPORTED")
	case "quarantined":
		if res.ConflictType == "exact_hash" {
			res.Message = tr.T("WATCHER_EXACT_DUPLICATE")
		} else {
			res.Message = tr.T("WATCHER_SAME_FORMAT")
		}
	case "skipped":
		res.Message = tr.T("WATCHER_EXACT_SKIPPED")
	case "format_attached":
		res.Message = tr.T("WATCHER_FORMAT_ATTACHED", filepath.Ext(header.Filename), res.BookID)
	}

	statusCode := http.StatusOK
	if res.Status == "imported" {
		statusCode = http.StatusCreated
	}

	writeJSON(w, statusCode, res)
}
