package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"boyan/internal/i18n"
	"boyan/internal/parsers/zip"
	"boyan/internal/storage"
)

// Streamer отвечает за безопасную доставку и потоковую отдачу файлов книг клиентам.
type Streamer struct {
	repo          *storage.BookRepository
	libraryDir    string
	streamFromZIP bool
}

func NewStreamer(repo *storage.BookRepository, libraryDir string, streamFromZIP bool) *Streamer {
	return &Streamer{
		repo:          repo,
		libraryDir:    libraryDir,
		streamFromZIP: streamFromZIP,
	}
}

func writeStreamerError(w http.ResponseWriter, r *http.Request, status int, code string, args ...any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	tr := i18n.FromContext(r.Context())
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": tr.T(code, args...),
		"code":  code,
	})
}

// ServeBookFile отдает файл книги в HTTP-ответ. Если запрошен fb2, а на диске fb2.zip — стримит из zip без распаковки.
func (s *Streamer) ServeBookFile(w http.ResponseWriter, r *http.Request, bookID, format string) {
	file, err := s.repo.GetBookFileByFormat(r.Context(), bookID, format)
	if err != nil {
		writeStreamerError(w, r, http.StatusInternalServerError, "DB_ERROR")
		return
	}
	if file == nil {
		writeStreamerError(w, r, http.StatusNotFound, "FORMAT_NOT_FOUND")
		return
	}

	book, _ := s.repo.GetBookByID(r.Context(), bookID)
	downloadName := s.buildDownloadFilename(book, format)

	fullPath := file.FilePath
	if !filepath.IsAbs(fullPath) && s.libraryDir != "" {
		fullPath = filepath.Join(s.libraryDir, fullPath)
	}

	// Проверяем, нужно ли стримить FB2 напрямую из архива .fb2.zip
	reqFormat := strings.ToLower(format)
	fileFormat := strings.ToLower(file.Format)

	if s.streamFromZIP && reqFormat == "fb2" && (fileFormat == "fb2.zip" || strings.HasSuffix(strings.ToLower(file.FilePath), ".zip")) {
		s.streamFB2FromZip(w, r, fullPath, downloadName)
		return
	}

	// Прямая отдача с диска через http.ServeFile (с поддержкой Range и кеширования)
	if _, err := os.Stat(fullPath); err != nil {
		writeStreamerError(w, r, http.StatusNotFound, "FILE_NOT_FOUND")
		return
	}

	w.Header().Set("Content-Type", MIMETypeForFormat(file.Format))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`,
		sanitizeHeaderAscii(downloadName), url.PathEscape(downloadName)))

	http.ServeFile(w, r, fullPath)
}

func (s *Streamer) streamFB2FromZip(w http.ResponseWriter, r *http.Request, zipPath, downloadName string) {
	rc, err := zip.ExtractFB2Stream(zipPath)
	if err != nil {
		writeStreamerError(w, r, http.StatusInternalServerError, "ZIP_STREAM_FAILED", err)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", "application/x-fictionbook+xml")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`,
		sanitizeHeaderAscii(downloadName), url.PathEscape(downloadName)))

	// Потоковая передача напрямую в сеть без буферизации в RAM
	_, _ = io.Copy(w, rc)
}

func (s *Streamer) buildDownloadFilename(book any, format string) string {
	ext := format
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	// Fallback имя файла
	return "book" + ext
}

func sanitizeHeaderAscii(name string) string {
	var sb strings.Builder
	for _, r := range name {
		if r >= 32 && r <= 126 && r != '"' && r != '\\' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}
	return sb.String()
}

// MIMETypeForFormat сопоставляет формат книги с официальным MIME-типом OPDS.
func MIMETypeForFormat(format string) string {
	switch strings.ToLower(format) {
	case "fb2":
		return "application/x-fictionbook+xml"
	case "fb2.zip":
		return "application/x-zip-compressed-fb2"
	case "epub":
		return "application/epub+zip"
	case "mobi":
		return "application/x-mobipocket-ebook"
	case "pdf":
		return "application/pdf"
	case "djvu":
		return "image/vnd.djvu"
	default:
		return "application/octet-stream"
	}
}
