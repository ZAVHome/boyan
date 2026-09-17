package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"boyan/internal/auth"
	"boyan/internal/models"
)

// JWTAuthMiddleware извлекает JWT токен из заголовка Authorization, cookie или query, валидирует его и помещает в контекст.
func JWTAuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := extractToken(r)
			if tokenString != "" {
				claims, err := auth.ValidateToken(tokenString, jwtSecret)
				if err == nil && claims != nil {
					ctx := auth.WithUser(r.Context(), claims)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth требует наличия валидного пользователя в контексте.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.UserFromContext(r.Context())
		if !ok || claims == nil {
			writeJSONError(w, http.StatusUnauthorized, "Unauthorized: authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin требует роли администратора.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.UserFromContext(r.Context())
		if !ok || claims == nil {
			writeJSONError(w, http.StatusUnauthorized, "Unauthorized: authentication required")
			return
		}
		if claims.Role != models.RoleAdmin {
			writeJSONError(w, http.StatusForbidden, "Forbidden: administrator privileges required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractToken(r *http.Request) string {
	// 1. Authorization: Bearer <token>
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	// 2. Cookie boyan_token
	if cookie, err := r.Cookie("boyan_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// 3. URL query parameter ?token=...
	if qToken := r.URL.Query().Get("token"); qToken != "" {
		return qToken
	}

	return ""
}

func writeJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
