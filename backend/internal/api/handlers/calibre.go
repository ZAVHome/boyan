package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"boyan/internal/importer/calibre"
)

type CalibreHandler struct {
	importer *calibre.Importer
}

func NewCalibreHandler(importer *calibre.Importer) *CalibreHandler {
	return &CalibreHandler{importer: importer}
}

type ImportCalibreRequest struct {
	Path      string `json:"path"`
	CopyFiles bool   `json:"copy_files"`
}

// ImportCalibre выполняет импорт библиотеки Calibre из переданного пути.
// @Summary Импорт библиотеки Calibre
// @Description Сканирует и импортирует каталог Calibre по указанному пути
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ImportCalibreRequest true "Параметры импорта библиотеки Calibre"
// @Success 200 {object} calibre.ImportStats
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/admin/import/calibre [post]
func (h *CalibreHandler) ImportCalibre(w http.ResponseWriter, r *http.Request) {
	var req ImportCalibreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Path = strings.TrimSpace(req.Path)
	if req.Path == "" {
		writeJSONError(w, http.StatusBadRequest, "Path to Calibre library is required")
		return
	}

	stats, err := h.importer.ImportLibrary(r.Context(), req.Path, calibre.ImportOptions{
		CopyFiles: req.CopyFiles,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stats)
}
