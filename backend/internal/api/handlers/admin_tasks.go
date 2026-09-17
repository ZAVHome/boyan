package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"boyan/internal/config"
	"boyan/internal/services"
	"boyan/internal/watcher"

	"github.com/go-chi/chi/v5"
)

type AdminTasksHandler struct {
	taskManager *services.TaskManager
	watcher     *watcher.Watcher
	cfg         *config.Config
}

func NewAdminTasksHandler(
	taskManager *services.TaskManager,
	watcher *watcher.Watcher,
	cfg *config.Config,
) *AdminTasksHandler {
	return &AdminTasksHandler{
		taskManager: taskManager,
		watcher:     watcher,
		cfg:         cfg,
	}
}

// ListTasks возвращает список недавних фоновых задач.
func (h *AdminTasksHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}

	tasks := h.taskManager.ListTasks(limit)
	writeJSON(w, http.StatusOK, map[string]any{
		"tasks": tasks,
		"total": len(tasks),
	})
}

// GetTask возвращает текущее состояние и прогресс фоновой задачи.
func (h *AdminTasksHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	task, err := h.taskManager.GetTask(id)
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, "NOT_FOUND")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// CancelTask отменяет выполнение активной задачи.
func (h *AdminTasksHandler) CancelTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	success := h.taskManager.CancelTask(id)
	if !success {
		writeAPIError(w, r, http.StatusBadRequest, "TASK_CANNOT_CANCEL")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Task cancellation requested",
	})
}

type RunScanRequest struct {
	Target string `json:"target"` // "watch_dir" или "library_dir"
}

// RunScan запускает асинхронное сканирование папки в TaskManager.
func (h *AdminTasksHandler) RunScan(w http.ResponseWriter, r *http.Request) {
	var req RunScanRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	scanDir := h.cfg.Storage.WatchDir
	if req.Target == "library_dir" {
		scanDir = h.cfg.Storage.LibraryDir
	}

	task, taskCtx := h.taskManager.CreateTask(context.Background(), "scan_library", 0)
	h.taskManager.StartTask(task.ID)

	go func(taskID, dir string, ctx context.Context) {
		err := h.watcher.ScanDirectoryWithProgress(ctx, dir, func(processed, total int, currentItem string, err error) {
			h.taskManager.UpdateProgress(taskID, processed, total, currentItem)
			if err != nil {
				h.taskManager.AddError(taskID, fmt.Sprintf("%s: %v", currentItem, err))
			}
		})

		if err != nil {
			if ctx.Err() != nil {
				// Задача была отменена пользователем
				return
			}
			h.taskManager.FailTask(taskID, err)
		} else {
			h.taskManager.CompleteTask(taskID, fmt.Sprintf("Scanning of directory %s completed", dir))
		}
	}(task.ID, scanDir, taskCtx)

	writeJSON(w, http.StatusAccepted, task)
}
