package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

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
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse multipart form: %v", err))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Missing 'file' in form-data")
		return
	}
	defer file.Close()

	// Сохраняем во временный файл
	tempDir := filepath.Join(".", "data", "tmp")
	_ = os.MkdirAll(tempDir, 0755)

	tempFile, err := os.CreateTemp(tempDir, "upload-*"+filepath.Ext(header.Filename))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to create temp file")
		return
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if _, err := io.Copy(tempFile, file); err != nil {
		tempFile.Close()
		writeJSONError(w, http.StatusInternalServerError, "Failed to save uploaded file")
		return
	}
	tempFile.Close()

	// Запускаем пайплайн обработки через Watcher
	res, err := h.watcher.ProcessFile(r.Context(), tempPath, header.Filename)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Failed to process book: %v", err))
		return
	}

	statusCode := http.StatusOK
	if res.Status == "imported" {
		statusCode = http.StatusCreated
	}

	writeJSON(w, statusCode, res)
}
