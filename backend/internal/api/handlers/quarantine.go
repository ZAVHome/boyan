package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"boyan/internal/config"
	"boyan/internal/i18n"
	"boyan/internal/models"
	"boyan/internal/parsers/cover"
	"boyan/internal/parsers/fb2"
	"boyan/internal/parsers/zip"
	"boyan/internal/storage"
	"boyan/internal/watcher"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type QuarantineHandler struct {
	cfg            *config.Config
	quarantineRepo *storage.QuarantineRepository
	bookRepo       *storage.BookRepository
	coverCache     *cover.CoverCache
}

func NewQuarantineHandler(
	cfg *config.Config,
	quarantineRepo *storage.QuarantineRepository,
	bookRepo *storage.BookRepository,
	coverCache *cover.CoverCache,
) *QuarantineHandler {
	return &QuarantineHandler{
		cfg:            cfg,
		quarantineRepo: quarantineRepo,
		bookRepo:       bookRepo,
		coverCache:     coverCache,
	}
}

// ListQuarantine возвращает элементы очереди карантина с пагинацией.
func (h *QuarantineHandler) ListQuarantine(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	offset := (page - 1) * perPage

	items, total, err := h.quarantineRepo.ListQuarantine(r.Context(), offset, perPage)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "QUARANTINE_LIST_FAILED")
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))

	writeJSON(w, http.StatusOK, map[string]any{
		"items":       items,
		"total":       total,
		"page":        page,
		"per_page":    perPage,
		"total_pages": totalPages,
	})
}

// GetQuarantineItem возвращает детальную информацию об одном элементе карантина.
func (h *QuarantineHandler) GetQuarantineItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeAPIError(w, r, http.StatusBadRequest, "QUARANTINE_ID_REQUIRED")
		return
	}

	item, err := h.quarantineRepo.GetQuarantineItem(r.Context(), id)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "DB_ERROR")
		return
	}
	if item == nil {
		writeAPIError(w, r, http.StatusNotFound, "QUARANTINE_NOT_FOUND")
		return
	}

	var existingBook *models.Book
	if item.ExistingBookID != nil && *item.ExistingBookID != "" {
		existingBook, _ = h.bookRepo.GetBookByID(r.Context(), *item.ExistingBookID)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"item":          item,
		"existing_book": existingBook,
	})
}

type ResolveQuarantineRequest struct {
	Action string `json:"action"` // "discard", "replace", "attach_format", "keep_both"
}

// ResolveQuarantine разрешает конфликт дубликата в карантине.
func (h *QuarantineHandler) ResolveQuarantine(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeAPIError(w, r, http.StatusBadRequest, "QUARANTINE_ID_REQUIRED")
		return
	}

	var req ResolveQuarantineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "INVALID_JSON")
		return
	}

	item, err := h.quarantineRepo.GetQuarantineItem(r.Context(), id)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "DB_ERROR")
		return
	}
	if item == nil {
		writeAPIError(w, r, http.StatusNotFound, "QUARANTINE_NOT_FOUND")
		return
	}

	tr := i18n.FromContext(r.Context())

	switch req.Action {
	case "discard":
		_ = os.Remove(item.FilePath)
		_ = h.quarantineRepo.DeleteQuarantineItem(r.Context(), id)
		writeJSON(w, http.StatusOK, map[string]string{"message": tr.T("QUARANTINE_DISCARDED")})
		return

	case "replace":
		if item.ExistingBookID == nil || *item.ExistingBookID == "" {
			writeAPIError(w, r, http.StatusBadRequest, "QUARANTINE_NO_EXISTING")
			return
		}

		existingBook, err := h.bookRepo.GetBookByID(r.Context(), *item.ExistingBookID)
		if err != nil || existingBook == nil {
			writeAPIError(w, r, http.StatusNotFound, "BOOK_NOT_FOUND")
			return
		}

		ext := "." + item.Format
		relPath := watcher.FormatPath(h.cfg.Storage.PathTemplate, existingBook, existingBook.Authors, existingBook.Series, ext)
		destPath := filepath.Join(h.cfg.Storage.LibraryDir, relPath)

		if err := copyFile(item.FilePath, destPath); err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "QUARANTINE_COPY_FAILED", err)
			return
		}

		// Ищем существующий файл с этим форматом
		var targetFile *models.BookFile
		for _, f := range existingBook.Files {
			if f.Format == item.Format {
				targetFile = &f
				break
			}
		}

		if targetFile != nil {
			targetFile.FilePath = relPath
			targetFile.FileSize = item.FileSize
			targetFile.SHA256 = item.SHA256
			_ = h.bookRepo.UpdateBookFile(r.Context(), targetFile)
		} else {
			_ = h.bookRepo.AddBookFile(r.Context(), &models.BookFile{
				ID:        uuid.NewString(),
				BookID:    existingBook.ID,
				Format:    item.Format,
				FilePath:  relPath,
				FileSize:  item.FileSize,
				SHA256:    item.SHA256,
				CreatedAt: time.Now().UTC(),
			})
		}

		_ = os.Remove(item.FilePath)
		_ = h.quarantineRepo.DeleteQuarantineItem(r.Context(), id)
		writeJSON(w, http.StatusOK, map[string]string{"message": tr.T("QUARANTINE_REPLACED")})
		return

	case "attach_format":
		if item.ExistingBookID == nil || *item.ExistingBookID == "" {
			writeAPIError(w, r, http.StatusBadRequest, "QUARANTINE_NO_EXISTING")
			return
		}

		existingBook, err := h.bookRepo.GetBookByID(r.Context(), *item.ExistingBookID)
		if err != nil || existingBook == nil {
			writeAPIError(w, r, http.StatusNotFound, "BOOK_NOT_FOUND")
			return
		}

		ext := "." + item.Format
		relPath := watcher.FormatPath(h.cfg.Storage.PathTemplate, existingBook, existingBook.Authors, existingBook.Series, ext)
		destPath := filepath.Join(h.cfg.Storage.LibraryDir, relPath)

		if err := copyFile(item.FilePath, destPath); err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "QUARANTINE_COPY_FAILED", err)
			return
		}

		newFile := &models.BookFile{
			ID:        uuid.NewString(),
			BookID:    existingBook.ID,
			Format:    item.Format,
			FilePath:  relPath,
			FileSize:  item.FileSize,
			SHA256:    item.SHA256,
			CreatedAt: time.Now().UTC(),
		}
		if err := h.bookRepo.AddBookFile(r.Context(), newFile); err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "DB_ERROR")
			return
		}

		_ = os.Remove(item.FilePath)
		_ = h.quarantineRepo.DeleteQuarantineItem(r.Context(), id)
		writeJSON(w, http.StatusOK, map[string]string{"message": tr.T("QUARANTINE_ATTACHED")})
		return

	case "keep_both":
		// Импортируем файл как совершенно новую книгу
		newBookID := uuid.NewString()
		info, innerPath, err := parseQuarantineFile(item.FilePath, item.Format)
		if err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "QUARANTINE_PARSE_FAILED", err)
			return
		}

		info.Book.ID = newBookID
		ext := "." + item.Format
		relPath := watcher.FormatPath(h.cfg.Storage.PathTemplate, &info.Book, info.Authors, info.Series, ext)
		destPath := filepath.Join(h.cfg.Storage.LibraryDir, relPath)

		if err := copyFile(item.FilePath, destPath); err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "QUARANTINE_COPY_FAILED", err)
			return
		}

		if len(info.CoverBytes) > 0 && h.coverCache != nil {
			if processed, err := cover.ProcessCover(info.CoverBytes, h.cfg.Metadata.CoverThumbnailSize); err == nil {
				if _, err := h.coverCache.SaveCover(newBookID, processed); err == nil {
					info.Book.CoverCached = true
				}
			}
		}

		bookFile := &models.BookFile{
			ID:               uuid.NewString(),
			BookID:           newBookID,
			Format:           item.Format,
			FilePath:         relPath,
			ArchiveInnerPath: innerPath,
			FileSize:         item.FileSize,
			SHA256:           item.SHA256,
			CreatedAt:        time.Now().UTC(),
		}

		if err := h.bookRepo.SaveBook(r.Context(), &info.Book, info.Authors, info.Series, info.Genres, bookFile); err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "QUARANTINE_SAVE_FAILED", err)
			return
		}

		_ = os.Remove(item.FilePath)
		_ = h.quarantineRepo.DeleteQuarantineItem(r.Context(), id)
		writeJSON(w, http.StatusOK, map[string]any{"message": tr.T("QUARANTINE_KEPT_BOTH"), "book_id": newBookID})
		return

	default:
		writeAPIError(w, r, http.StatusBadRequest, "QUARANTINE_INVALID_ACTION")
		return
	}
}

func parseQuarantineFile(filePath, format string) (*fb2.FB2BookInfo, string, error) {
	if format == "fb2.zip" || strings.HasSuffix(filePath, ".zip") {
		stream, err := zip.ExtractFB2Stream(filePath)
		if err != nil {
			return nil, "", err
		}
		defer stream.Close()

		info, err := fb2.ParseFB2(stream)
		if err != nil {
			return nil, "", err
		}

		innerPath := ""
		if zr, file, err := zip.OpenFB2FromZipFile(filePath); err == nil {
			innerPath = file.Name
			zr.Close()
		}
		return info, innerPath, nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	info, err := fb2.ParseFB2(f)
	return info, "", err
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = out.ReadFrom(in)
	return err
}
