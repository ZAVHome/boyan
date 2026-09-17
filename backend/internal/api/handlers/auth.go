package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"boyan/internal/auth"
	"boyan/internal/config"
	"boyan/internal/models"
	"boyan/internal/storage"
)

type AuthHandler struct {
	cfg      *config.Config
	userRepo *storage.UserRepository
}

func NewAuthHandler(cfg *config.Config, userRepo *storage.UserRepository) *AuthHandler {
	return &AuthHandler{
		cfg:      cfg,
		userRepo: userRepo,
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserDTO struct {
	ID       string      `json:"id"`
	Username string      `json:"username"`
	Role     models.Role `json:"role"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserDTO   `json:"user"`
}

// Login обрабатывает вход пользователя, выдает JWT токен и выставляет cookie.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	user, err := h.userRepo.GetByUsername(r.Context(), req.Username)
	if err != nil || user == nil {
		writeJSONError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	if !user.IsActive {
		writeJSONError(w, http.StatusForbidden, "User account is inactive")
		return
	}

	if !h.userRepo.VerifyPassword(user, req.Password) {
		writeJSONError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	token, expiresAt, err := auth.GenerateToken(user, h.cfg.Server.JWTSecret, h.cfg.Server.JWTExpirationHours)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Устанавливаем защищенную cookie
	maxAge := int(time.Until(expiresAt).Seconds())
	http.SetCookie(w, &http.Cookie{
		Name:     "boyan_token",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	resp := LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User: UserDTO{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		},
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// Me возвращает информацию о текущем авторизованном пользователе.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.UserFromContext(r.Context())
	if !ok || claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	resp := struct {
		User UserDTO `json:"user"`
	}{
		User: UserDTO{
			ID:       claims.UserID,
			Username: claims.Username,
			Role:     claims.Role,
		},
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// Logout удаляет cookie авторизации.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "boyan_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"})
}
