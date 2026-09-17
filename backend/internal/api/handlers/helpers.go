package handlers

import (
	"encoding/json"
	"net/http"

	"boyan/internal/i18n"
)

// APIErrorResponse представляет стандартизированный ответ об ошибке REST API.
type APIErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]string{"error": message})
}

// writeAPIError возвращает локализованную ошибку со стандартизированным машинным кодом.
func writeAPIError(w http.ResponseWriter, r *http.Request, status int, code string, args ...any) {
	tr := i18n.FromContext(r.Context())
	msg := tr.T(code, args...)
	writeJSON(w, status, APIErrorResponse{
		Error: msg,
		Code:  code,
	})
}
