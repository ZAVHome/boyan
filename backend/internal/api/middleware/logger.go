package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// SlogLogger логирует HTTP-запросы через стандартный slog.
func SlogLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		t1 := time.Now()

		defer func() {
			reqID := middleware.GetReqID(r.Context())
			slog.Info("HTTP Request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(t1).Milliseconds(),
				"remote_addr", r.RemoteAddr,
				"req_id", reqID,
			)
		}()

		next.ServeHTTP(ww, r)
	})
}
