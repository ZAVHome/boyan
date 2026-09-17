package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"path/filepath"
	"strconv"

	"boyan/internal/config"
	"boyan/internal/parsers/cover"
	"boyan/internal/storage"

	"github.com/go-chi/chi/v5"
)

// BooksHandler обрабатывает запросы к книгам и обложкам.
type BooksHandler struct {
	cfg        *config.Config
	repo       *storage.BookRepository
	coverCache *cover.CoverCache
}

func NewBooksHandler(cfg *config.Config, repo *storage.BookRepository, coverCache *cover.CoverCache) *BooksHandler {
	return &BooksHandler{
		cfg:        cfg,
		repo:       repo,
		coverCache: coverCache,
	}
}

// ListBooks возвращает список книг с поддержкой поиска и пагинации.
func (h *BooksHandler) ListBooks(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	offset := (page - 1) * perPage
	query := r.URL.Query().Get("q")

	books, total, err := h.repo.SearchBooksFTS(r.Context(), query, offset, perPage)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "SEARCH_FAILED")
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))

	response := map[string]any{
		"items":       books,
		"total":       total,
		"page":        page,
		"per_page":    perPage,
		"total_pages": totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// GetBook возвращает подробную информацию о книге по её ID.
func (h *BooksHandler) GetBook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeAPIError(w, r, http.StatusBadRequest, "BOOK_ID_REQUIRED")
		return
	}

	book, err := h.repo.GetBookByID(r.Context(), id)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "DB_ERROR")
		return
	}
	if book == nil {
		writeAPIError(w, r, http.StatusNotFound, "BOOK_NOT_FOUND")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(book)
}

// GetCover отдает кешированное изображение обложки с поддержкой ETag и 304 Not Modified.
func (h *BooksHandler) GetCover(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeAPIError(w, r, http.StatusBadRequest, "BOOK_ID_REQUIRED")
		return
	}

	var filePath string
	if h.coverCache != nil {
		if path, ok := h.coverCache.GetCoverPath(id); ok {
			filePath = path
		}
	}

	if filePath == "" {
		// Обложки нет в кеше — пробуем извлечь на лету из файла книги
		extractedPath, err := h.tryExtractCover(r.Context(), id)
		if err != nil || extractedPath == "" {
			writeAPIError(w, r, http.StatusNotFound, "COVER_NOT_FOUND")
			return
		}
		filePath = extractedPath
	}

	// http.ServeFile автоматически устанавливает ETag, Last-Modified и отвечает 304 Not Modified
	http.ServeFile(w, r, filePath)
}

func (h *BooksHandler) tryExtractCover(ctx context.Context, bookID string) (string, error) {
	if h.coverCache == nil || h.repo == nil {
		return "", fmt.Errorf("dependencies not initialized")
	}

	files, err := h.repo.GetBookFiles(ctx, bookID)
	if err != nil || len(files) == 0 {
		return "", fmt.Errorf("book files not found")
	}

	for _, file := range files {
		fullPath := file.FilePath
		if !filepath.IsAbs(fullPath) && h.cfg != nil && h.cfg.Storage.LibraryDir != "" {
			fullPath = filepath.Join(h.cfg.Storage.LibraryDir, fullPath)
		}

		rawBytes, err := cover.ExtractRawCoverFromFile(fullPath, file.Format)
		if err != nil || len(rawBytes) == 0 {
			continue
		}

		thumbSize := 400
		if h.cfg != nil && h.cfg.Metadata.CoverThumbnailSize > 0 {
			thumbSize = h.cfg.Metadata.CoverThumbnailSize
		}

		processed, err := cover.ProcessCover(rawBytes, thumbSize)
		if err != nil {
			continue
		}

		savedPath, err := h.coverCache.SaveCover(bookID, processed)
		if err != nil {
			return "", err
		}

		_ = h.repo.SetCoverCached(ctx, bookID, true)
		return savedPath, nil
	}

	return "", fmt.Errorf("no cover extracted from book files")
}
