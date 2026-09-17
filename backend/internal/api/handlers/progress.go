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
		writeAPIError(w, r, http.StatusUnauthorized, "AUTH_REQUIRED")
		return
	}

	bookID := chi.URLParam(r, "id")
	if bookID == "" {
		writeAPIError(w, r, http.StatusBadRequest, "BOOK_ID_REQUIRED")
		return
	}

	p, err := h.progressRepo.GetProgress(r.Context(), claims.UserID, bookID)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "DB_ERROR")
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
		writeAPIError(w, r, http.StatusUnauthorized, "AUTH_REQUIRED")
		return
	}

	bookID := chi.URLParam(r, "id")
	if bookID == "" {
		writeAPIError(w, r, http.StatusBadRequest, "BOOK_ID_REQUIRED")
		return
	}

	var req SaveProgressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "INVALID_JSON")
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
		writeAPIError(w, r, http.StatusInternalServerError, "PROGRESS_SAVE_FAILED")
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
		writeAPIError(w, r, http.StatusUnauthorized, "AUTH_REQUIRED")
		return
	}

	bookID := chi.URLParam(r, "id")
	if bookID == "" {
		writeAPIError(w, r, http.StatusBadRequest, "BOOK_ID_REQUIRED")
		return
	}

	var req AddToShelfRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "INVALID_JSON")
		return
	}

	if req.ShelfType != "reading" && req.ShelfType != "finished" && req.ShelfType != "favorite" {
		writeAPIError(w, r, http.StatusBadRequest, "SHELF_INVALID_TYPE")
		return
	}

	if err := h.progressRepo.AddToShelf(r.Context(), claims.UserID, bookID, req.ShelfType); err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "SHELF_ADD_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Added to shelf"})
}

// RemoveFromShelf удаляет книгу с полки пользователя.
func (h *ProgressHandler) RemoveFromShelf(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.UserFromContext(r.Context())
	if !ok || claims == nil {
		writeAPIError(w, r, http.StatusUnauthorized, "AUTH_REQUIRED")
		return
	}

	bookID := chi.URLParam(r, "id")
	shelfType := chi.URLParam(r, "type")
	if bookID == "" || shelfType == "" {
		writeAPIError(w, r, http.StatusBadRequest, "BOOK_OR_SHELF_REQUIRED")
		return
	}

	if err := h.progressRepo.RemoveFromShelf(r.Context(), claims.UserID, bookID, shelfType); err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "SHELF_REMOVE_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Removed from shelf"})
}

// GetShelfBooks возвращает список книг на заданной полке пользователя.
func (h *ProgressHandler) GetShelfBooks(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.UserFromContext(r.Context())
	if !ok || claims == nil {
		writeAPIError(w, r, http.StatusUnauthorized, "AUTH_REQUIRED")
		return
	}

	shelfType := chi.URLParam(r, "type")
	if shelfType == "" {
		writeAPIError(w, r, http.StatusBadRequest, "SHELF_REQUIRED")
		return
	}

	books, err := h.progressRepo.GetUserShelfBooks(r.Context(), claims.UserID, shelfType)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "SHELF_GET_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"shelf_type": shelfType,
		"items":      books,
		"total":      len(books),
	})
}
