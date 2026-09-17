package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"boyan/internal/storage"
	"boyan/internal/version"
)

// HealthHandler обрабатывает запросы проверки работоспособности сервера.
type HealthHandler struct {
	pool *storage.DBPool
}

func NewHealthHandler(pool *storage.DBPool) *HealthHandler {
	return &HealthHandler{pool: pool}
}

func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	dbStatus := "connected"
	if err := h.pool.Reader.PingContext(r.Context()); err != nil {
		dbStatus = "disconnected: " + err.Error()
	}

	res := map[string]any{
		"status":    "ok",
		"db":        dbStatus,
		"timestamp": time.Now().UTC(),
		"service":   "Boyan",
		"version":   version.Full(),
	}

	w.Header().Set("Content-Type", "application/json")
	if dbStatus != "connected" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	_ = json.NewEncoder(w).Encode(res)
}

func (h *HealthHandler) Ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"pong":      true,
		"timestamp": time.Now().UTC(),
	})
}
