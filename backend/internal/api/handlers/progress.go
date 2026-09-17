package handlers

import (
	"encoding/json"
	"net/http"

	"boyan/internal/auth"
	"boyan/internal/storage"

	"github.com/go-chi/chi/v5"
)

type ProgressHandler struct {
	progressRepo *storage.ProgressRepository
	bookRepo     *storage.BookRepository
}

func NewProgressHandler(progressRepo *storage.ProgressRepository, bookRepo *storage.BookRepository) *ProgressHandler {
	return &ProgressHandler{
		progressRepo: progressRepo,
		bookRepo:     bookRepo,
	}
}

type SaveProgressRequest struct {
	Format          string  `json:"format"`
	ProgressPercent float64 `json:"progress_percent"`
	Position        string  `json:"position"`
}

// GetProgress возвращает сохраненный прогресс чтения текущего пользователя для книги.
func (h *ProgressHandler) GetProgress(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.UserFromContext(r.Context())
	if !ok || claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	bookID := chi.URLParam(r, "id")
	if bookID == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing book ID")
		return
	}

	p, err := h.progressRepo.GetProgress(r.Context(), claims.UserID, bookID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if p == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":          claims.UserID,
			"book_id":          bookID,
			"format":           "",
			"progress_percent": 0.0,
			"position":         "",
		})
		return
	}

	writeJSON(w, http.StatusOK, p)
}

// SaveProgress сохраняет прогресс чтения и при необходимости обновляет полки ("reading" / "finished").
func (h *ProgressHandler) SaveProgress(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.UserFromContext(r.Context())
	if !ok || claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	bookID := chi.URLParam(r, "id")
	if bookID == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing book ID")
		return
	}

	var req SaveProgressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if req.ProgressPercent < 0 {
		req.ProgressPercent = 0
	}
	if req.ProgressPercent > 100 {
		req.ProgressPercent = 100
	}

	record := &storage.ReadProgress{
		UserID:          claims.UserID,
		BookID:          bookID,
		Format:          req.Format,
		ProgressPercent: req.ProgressPercent,
		Position:        req.Position,
	}

	if err := h.progressRepo.SaveProgress(r.Context(), record); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save reading progress")
		return
	}

	// Автоматическое обновление полок
	if req.ProgressPercent >= 99.5 {
		_ = h.progressRepo.RemoveFromShelf(r.Context(), claims.UserID, bookID, "reading")
		_ = h.progressRepo.AddToShelf(r.Context(), claims.UserID, bookID, "finished")
	} else if req.ProgressPercent > 0 {
		_ = h.progressRepo.AddToShelf(r.Context(), claims.UserID, bookID, "reading")
	}

	writeJSON(w, http.StatusOK, record)
}

type AddToShelfRequest struct {
	ShelfType string `json:"shelf_type"` // "reading", "finished", "favorite"
}

// AddToShelf добавляет книгу на полку пользователя.
func (h *ProgressHandler) AddToShelf(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.UserFromContext(r.Context())
	if !ok || claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	bookID := chi.URLParam(r, "id")
	if bookID == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing book ID")
		return
	}

	var req AddToShelfRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if req.ShelfType != "reading" && req.ShelfType != "finished" && req.ShelfType != "favorite" {
		writeJSONError(w, http.StatusBadRequest, "Invalid shelf type. Must be: 'reading', 'finished', or 'favorite'")
		return
	}

	if err := h.progressRepo.AddToShelf(r.Context(), claims.UserID, bookID, req.ShelfType); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to add book to shelf")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Added to shelf"})
}

// RemoveFromShelf удаляет книгу с полки пользователя.
func (h *ProgressHandler) RemoveFromShelf(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.UserFromContext(r.Context())
	if !ok || claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	bookID := chi.URLParam(r, "id")
	shelfType := chi.URLParam(r, "type")
	if bookID == "" || shelfType == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing book ID or shelf type")
		return
	}

	if err := h.progressRepo.RemoveFromShelf(r.Context(), claims.UserID, bookID, shelfType); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to remove book from shelf")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Removed from shelf"})
}

// GetShelfBooks возвращает список книг на заданной полке пользователя.
func (h *ProgressHandler) GetShelfBooks(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.UserFromContext(r.Context())
	if !ok || claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	shelfType := chi.URLParam(r, "type")
	if shelfType == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing shelf type")
		return
	}

	books, err := h.progressRepo.GetUserShelfBooks(r.Context(), claims.UserID, shelfType)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to get shelf books")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"shelf_type": shelfType,
		"items":      books,
		"total":      len(books),
	})
}
