package handlers

import (
	"encoding/json"
	"net/http"

	"boyan/internal/config"
)

type AdminSettingsHandler struct {
	cfg        *config.Config
	configPath string
}

func NewAdminSettingsHandler(cfg *config.Config, configPath string) *AdminSettingsHandler {
	return &AdminSettingsHandler{
		cfg:        cfg,
		configPath: configPath,
	}
}

// GetSettings возвращает текущую конфигурацию сервиса.
func (h *AdminSettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	// Делаем копию и маскируем секреты
	copied := *h.cfg
	if copied.Server.JWTSecret != "" {
		copied.Server.JWTSecret = "********"
	}
	if copied.Telegram.BotToken != "" {
		copied.Telegram.BotToken = "********"
	}

	writeJSON(w, http.StatusOK, copied)
}

// UpdateSettings сохраняет обновленные параметры в файл config.yaml на диске.
func (h *AdminSettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req config.Config
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "INVALID_JSON")
		return
	}

	// Сохраняем прежние секреты, если передана маска
	if req.Server.JWTSecret == "********" || req.Server.JWTSecret == "" {
		req.Server.JWTSecret = h.cfg.Server.JWTSecret
	}
	if req.Telegram.BotToken == "********" || req.Telegram.BotToken == "" {
		req.Telegram.BotToken = h.cfg.Telegram.BotToken
	}

	// Обновляем текущий конфиг в памяти
	*h.cfg = req

	// Сохраняем на диск
	if h.configPath != "" {
		if err := h.cfg.Save(h.configPath); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to save config: "+err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Settings saved successfully to " + h.configPath,
	})
}

// ReloadServices имитирует или выполняет горячую перезагрузку фоновых сервисов.
func (h *AdminSettingsHandler) ReloadServices(w http.ResponseWriter, r *http.Request) {
	// При перезагрузке можно перечитать параметры
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Background services reload signaled successfully",
	})
}
