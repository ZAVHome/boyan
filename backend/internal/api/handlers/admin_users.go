package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"boyan/internal/auth"
	"boyan/internal/models"
	"boyan/internal/storage"

	"github.com/go-chi/chi/v5"
)

type AdminUsersHandler struct {
	userRepo *storage.UserRepository
}

func NewAdminUsersHandler(userRepo *storage.UserRepository) *AdminUsersHandler {
	return &AdminUsersHandler{userRepo: userRepo}
}

// ListUsers возвращает постраничный список пользователей с фильтрами.
func (h *AdminUsersHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	role := models.Role(strings.TrimSpace(r.URL.Query().Get("role")))

	limit := 20
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	offset := 0
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}

	var isActive *bool
	if activeStr := r.URL.Query().Get("is_active"); activeStr != "" {
		val := activeStr == "true" || activeStr == "1"
		isActive = &val
	}

	filter := models.UserFilter{
		Query:    q,
		Role:     role,
		IsActive: isActive,
		Limit:    limit,
		Offset:   offset,
	}

	resp, err := h.userRepo.ListUsers(r.Context(), filter)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// CreateUser создает нового пользователя администратором.
func (h *AdminUsersHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "INVALID_JSON")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeAPIError(w, r, http.StatusBadRequest, "AUTH_FIELDS_REQUIRED")
		return
	}

	if req.Role == "" {
		req.Role = models.RoleUser
	}

	existing, _ := h.userRepo.GetByUsername(r.Context(), req.Username)
	if existing != nil {
		writeAPIError(w, r, http.StatusConflict, "USER_ALREADY_EXISTS")
		return
	}

	user, err := h.userRepo.CreateUser(r.Context(), req.Username, req.Password, req.Role)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !req.IsActive {
		_ = h.userRepo.UpdateUser(r.Context(), user.ID, user.Role, false)
		user.IsActive = false
	}

	writeJSON(w, http.StatusCreated, user)
}

// GetUser возвращает информацию о пользователе по его ID.
func (h *AdminUsersHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := h.userRepo.GetByID(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if user == nil {
		writeAPIError(w, r, http.StatusNotFound, "NOT_FOUND")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// UpdateUser изменяет роль и статус активности пользователя.
func (h *AdminUsersHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "INVALID_JSON")
		return
	}

	if req.Role == "" {
		req.Role = models.RoleUser
	}

	// Защита: нельзя отключить самого себя или снять с себя роль admin
	if claims, ok := auth.UserFromContext(r.Context()); ok && claims != nil {
		if claims.UserID == id {
			if !req.IsActive || req.Role != models.RoleAdmin {
				writeAPIError(w, r, http.StatusBadRequest, "CANNOT_DEMOTE_SELF")
				return
			}
		}
	}

	err := h.userRepo.UpdateUser(r.Context(), id, req.Role, req.IsActive)
	if err != nil {
		if err == sql.ErrNoRows {
			writeAPIError(w, r, http.StatusNotFound, "NOT_FOUND")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "User updated successfully"})
}

// UpdatePassword сбрасывает/устанавливает новый пароль пользователя.
func (h *AdminUsersHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req models.UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "INVALID_JSON")
		return
	}

	if strings.TrimSpace(req.Password) == "" {
		writeAPIError(w, r, http.StatusBadRequest, "PASSWORD_EMPTY")
		return
	}

	err := h.userRepo.UpdatePassword(r.Context(), id, req.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			writeAPIError(w, r, http.StatusNotFound, "NOT_FOUND")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Password changed successfully"})
}

// DeleteUser удаляет пользователя по ID.
func (h *AdminUsersHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Защита: нельзя удалить самого себя
	if claims, ok := auth.UserFromContext(r.Context()); ok && claims != nil {
		if claims.UserID == id {
			writeAPIError(w, r, http.StatusBadRequest, "CANNOT_DELETE_SELF")
			return
		}
	}

	err := h.userRepo.DeleteUser(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeAPIError(w, r, http.StatusNotFound, "NOT_FOUND")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "User deleted successfully"})
}
