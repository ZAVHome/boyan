package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"boyan/internal/parsers/cover"
	"boyan/internal/storage"

	"github.com/go-chi/chi/v5"
)

// BooksHandler обрабатывает запросы к книгам и обложкам.
type BooksHandler struct {
	repo       *storage.BookRepository
	coverCache *cover.CoverCache
}

func NewBooksHandler(repo *storage.BookRepository, coverCache *cover.CoverCache) *BooksHandler {
	return &BooksHandler{
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
		http.Error(w, `{"error":"search books failed"}`, http.StatusInternalServerError)
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
		http.Error(w, `{"error":"missing book id"}`, http.StatusBadRequest)
		return
	}

	book, err := h.repo.GetBookByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	if book == nil {
		http.Error(w, `{"error":"book not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(book)
}

// GetCover отдает кешированное изображение обложки с поддержкой ETag и 304 Not Modified.
func (h *BooksHandler) GetCover(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing book id", http.StatusBadRequest)
		return
	}

	filePath, ok := h.coverCache.GetCoverPath(id)
	if !ok {
		http.Error(w, "cover not found", http.StatusNotFound)
		return
	}

	// http.ServeFile автоматически устанавливает ETag, Last-Modified и отвечает 304 Not Modified
	http.ServeFile(w, r, filePath)
}
