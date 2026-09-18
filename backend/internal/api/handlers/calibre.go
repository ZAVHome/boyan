package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"boyan/internal/importer/calibre"
	"boyan/internal/services"
)

type CalibreHandler struct {
	importer    *calibre.Importer
	taskManager *services.TaskManager
}

func NewCalibreHandler(importer *calibre.Importer, taskManager *services.TaskManager) *CalibreHandler {
	return &CalibreHandler{
		importer:    importer,
		taskManager: taskManager,
	}
}

type ImportCalibreRequest struct {
	Path      string `json:"path"`
	CopyFiles bool   `json:"copy_files"`
}

// ImportCalibre выполняет импорт библиотеки Calibre из переданного пути.
// @Summary Импорт библиотеки Calibre
// @Description Запускает фоновую задачу импорта каталога Calibre по указанному пути
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ImportCalibreRequest true "Параметры импорта библиотеки Calibre"
// @Success 202 {object} services.Task
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/admin/import/calibre [post]
func (h *CalibreHandler) ImportCalibre(w http.ResponseWriter, r *http.Request) {
	var req ImportCalibreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "INVALID_JSON")
		return
	}

	req.Path = strings.TrimSpace(req.Path)
	if req.Path == "" {
		writeAPIError(w, r, http.StatusBadRequest, "CALIBRE_PATH_REQUIRED")
		return
	}

	// Предварительная проверка доступности базы metadata.db перед запуском фоновой задачи
	reader, err := calibre.Open(req.Path)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = reader.Close()

	// Регистрация и запуск задачи в TaskManager
	task, taskCtx := h.taskManager.CreateTask(context.Background(), "import_calibre", 0)
	h.taskManager.StartTask(task.ID)
	snapshot, err := h.taskManager.GetTask(task.ID)
	if err != nil {
		snapshot = task
	}

	go func(taskID, p string, opts calibre.ImportOptions, ctx context.Context) {
		stats, err := h.importer.ImportLibraryWithProgress(ctx, p, opts, func(processed, total int, currentItem string, err error) {
			h.taskManager.UpdateProgress(taskID, processed, total, currentItem)
			if err != nil {
				h.taskManager.AddError(taskID, fmt.Sprintf("%s: %v", currentItem, err))
			}
		})

		if err != nil {
			if ctx.Err() != nil {
				// Задача отменена пользователем
				return
			}
			h.taskManager.FailTask(taskID, err)
		} else {
			h.taskManager.CompleteTask(taskID, fmt.Sprintf("Imported %d of %d books (%d skipped, %d format files attached)",
				stats.ImportedBooks, stats.TotalCalibreBooks, stats.Skipped, stats.FormatsAttached))
		}
	}(task.ID, req.Path, calibre.ImportOptions{CopyFiles: req.CopyFiles}, taskCtx)

	writeJSON(w, http.StatusAccepted, snapshot)
}
