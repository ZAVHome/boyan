package middleware

import (
	"net/http"

	"boyan/internal/storage"
)

// OPDSAuth обеспечивает аутентификацию для E-Ink читалок через Basic Auth или URL token.
func OPDSAuth(userRepo *storage.UserRepository, allowAnonymous bool, staticToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if allowAnonymous {
				next.ServeHTTP(w, r)
				return
			}

			// 1. Проверка через токен в URL (?token=...)
			token := r.URL.Query().Get("token")
			if token != "" && staticToken != "" && token == staticToken {
				next.ServeHTTP(w, r)
				return
			}

			// 2. Проверка через HTTP Basic Auth
			username, password, ok := r.BasicAuth()
			if ok {
				user, err := userRepo.GetByUsername(r.Context(), username)
				if err == nil && user != nil && user.IsActive {
					if userRepo.VerifyPassword(user, password) {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			// Требуем аутентификацию
			w.Header().Set("WWW-Authenticate", `Basic realm="Next-Gen OPDS Suite (Boyan)"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		})
	}
}
