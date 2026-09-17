package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"path/filepath"

	"boyan/internal/config"
	"boyan/internal/models"
	"boyan/internal/parsers/cover"
	"boyan/internal/storage"

	"github.com/go-chi/chi/v5"
)

type AdminBooksHandler struct {
	cfg        *config.Config
	bookRepo   *storage.BookRepository
	coverCache *cover.CoverCache
}

func NewAdminBooksHandler(cfg *config.Config, bookRepo *storage.BookRepository, coverCache *cover.CoverCache) *AdminBooksHandler {
	return &AdminBooksHandler{
		cfg:        cfg,
		bookRepo:   bookRepo,
		coverCache: coverCache,
	}
}

// ListBooks возвращает список книг для админ-панели с фильтрацией и пагинацией.
func (h *AdminBooksHandler) ListBooks(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	limit := 20
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	offset := 0
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}

	var books []models.Book
	var total int
	var err error

	if q != "" {
		books, total, err = h.bookRepo.SearchBooksFTS(r.Context(), q, offset, limit)
	} else {
		books, total, err = h.bookRepo.ListBooks(r.Context(), offset, limit)
	}

	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"books":  books,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}


// UpdateBookMetadata обновляет метаданные книги.
func (h *AdminBooksHandler) UpdateBookMetadata(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req models.UpdateBookMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "INVALID_JSON")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeAPIError(w, r, http.StatusBadRequest, "BOOK_TITLE_REQUIRED")
		return
	}

	err := h.bookRepo.UpdateBookMetadata(r.Context(), id, req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	updated, err := h.bookRepo.GetBookByID(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// DeleteBook удаляет книгу из базы данных и опционально с диска.
func (h *AdminBooksHandler) DeleteBook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	deleteFiles := r.URL.Query().Get("delete_files") == "true"

	var files []models.BookFile
	if deleteFiles {
		var err error
		files, err = h.bookRepo.GetBookFiles(r.Context(), id)
		if err != nil {
			slog.Warn("Failed to get book files before delete", "book_id", id, "err", err)
		}
	}

	err := h.bookRepo.DeleteBook(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	deletedFromDiskCount := 0
	if deleteFiles && len(files) > 0 {
		for _, f := range files {
			if f.FilePath != "" {
				if err := os.Remove(f.FilePath); err == nil {
					deletedFromDiskCount++
					slog.Info("Deleted book file from disk", "path", f.FilePath)
				} else {
					slog.Warn("Failed to delete book file from disk", "path", f.FilePath, "err", err)
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":               "Book deleted successfully",
		"deleted_from_disk_count": deletedFromDiskCount,
	})
}

// BatchAction выполняет групповые действия над выбранными книгами.
func (h *AdminBooksHandler) BatchAction(w http.ResponseWriter, r *http.Request) {
	var req models.BatchBookActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "INVALID_JSON")
		return
	}

	if len(req.BookIDs) == 0 {
		writeAPIError(w, r, http.StatusBadRequest, "NO_BOOKS_SELECTED")
		return
	}

	res := models.BatchActionResult{
		Errors: make([]string, 0),
	}

	switch req.Action {
	case "delete":
		filePaths, err := h.bookRepo.BatchDeleteBooks(r.Context(), req.BookIDs)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		res.SuccessCount = len(req.BookIDs)

		if req.DeleteFiles && len(filePaths) > 0 {
			for _, path := range filePaths {
				if path != "" {
					_ = os.Remove(path)
				}
			}
		}

	case "set_genre":
		if strings.TrimSpace(req.GenreCode) == "" {
			writeAPIError(w, r, http.StatusBadRequest, "GENRE_CODE_REQUIRED")
			return
		}
		err := h.bookRepo.BatchUpdateGenres(r.Context(), req.BookIDs, req.GenreCode)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		res.SuccessCount = len(req.BookIDs)

	case "set_series":
		if strings.TrimSpace(req.SeriesName) == "" {
			writeAPIError(w, r, http.StatusBadRequest, "SERIES_NAME_REQUIRED")
			return
		}
		err := h.bookRepo.BatchUpdateSeries(r.Context(), req.BookIDs, req.SeriesName)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		res.SuccessCount = len(req.BookIDs)

	case "regenerate_cover":
		for _, bookID := range req.BookIDs {
			files, err := h.bookRepo.GetBookFiles(r.Context(), bookID)
			if err != nil || len(files) == 0 {
				res.ErrorCount++
				res.Errors = append(res.Errors, fmt.Sprintf("book %s: no files found", bookID))
				continue
			}

			// Пробуем сгенерировать обложку для первого подходящего файла
			file := files[0]
			err = h.extractAndSaveCover(bookID, file)
			if err != nil {
				res.ErrorCount++
				res.Errors = append(res.Errors, fmt.Sprintf("book %s: %v", bookID, err))
			} else {
				res.SuccessCount++
			}
		}

	default:
		writeAPIError(w, r, http.StatusBadRequest, "UNKNOWN_BATCH_ACTION")
		return
	}

	writeJSON(w, http.StatusOK, res)
}

// RegenerateCover выполняет принудительную перегенерацию обложки конкретной книги.
func (h *AdminBooksHandler) RegenerateCover(w http.ResponseWriter, r *http.Request) {
	bookID := chi.URLParam(r, "id")
	files, err := h.bookRepo.GetBookFiles(r.Context(), bookID)
	if err != nil || len(files) == 0 {
		writeAPIError(w, r, http.StatusNotFound, "NOT_FOUND")
		return
	}

	file := files[0]
	err = h.extractAndSaveCover(bookID, file)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to regenerate cover: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Cover regenerated successfully",
	})
}

func (h *AdminBooksHandler) extractAndSaveCover(bookID string, file models.BookFile) error {
	fullPath := file.FilePath
	if !filepath.IsAbs(fullPath) && h.cfg != nil && h.cfg.Storage.LibraryDir != "" {
		fullPath = filepath.Join(h.cfg.Storage.LibraryDir, fullPath)
	}

	rawCover, err := cover.ExtractRawCoverFromFile(fullPath, file.Format)
	if err != nil {
		return err
	}

	thumbSize := 400
	if h.cfg != nil && h.cfg.Metadata.CoverThumbnailSize > 0 {
		thumbSize = h.cfg.Metadata.CoverThumbnailSize
	}

	processed, err := cover.ProcessCover(rawCover, thumbSize)
	if err != nil {
		return err
	}

	_, err = h.coverCache.SaveCover(bookID, processed)
	if err == nil {
		_ = h.bookRepo.SetCoverCached(context.Background(), bookID, true)
	}
	return err
}

