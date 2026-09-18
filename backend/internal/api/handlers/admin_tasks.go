package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"boyan/internal/config"
	"boyan/internal/models"
	"boyan/internal/parsers/fb2"
	"boyan/internal/services"
	"boyan/internal/storage"
	"boyan/internal/watcher"

	"github.com/go-chi/chi/v5"
)

type AdminTasksHandler struct {
	taskManager *services.TaskManager
	watcher     *watcher.Watcher
	cfg         *config.Config
	bookRepo    *storage.BookRepository
}

func NewAdminTasksHandler(
	taskManager *services.TaskManager,
	watcher *watcher.Watcher,
	cfg *config.Config,
	bookRepo *storage.BookRepository,
) *AdminTasksHandler {
	return &AdminTasksHandler{
		taskManager: taskManager,
		watcher:     watcher,
		cfg:         cfg,
		bookRepo:    bookRepo,
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
	snapshot, err := h.taskManager.GetTask(task.ID)
	if err != nil {
		snapshot = task
	}

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

	writeJSON(w, http.StatusAccepted, snapshot)
}

// RunRepairFB2 запускает фоновую задачу инспекции и санитизации всех FB2-файлов в библиотеке.
func (h *AdminTasksHandler) RunRepairFB2(w http.ResponseWriter, r *http.Request) {
	if h.bookRepo == nil {
		writeAPIError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR")
		return
	}

	ctx := r.Context()
	fb2Files, err := h.bookRepo.GetAllFB2Files(ctx)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "DB_ERROR", err)
		return
	}

	task, taskCtx := h.taskManager.CreateTask(context.Background(), "repair_fb2", len(fb2Files))
	h.taskManager.StartTask(task.ID)
	snapshot, err := h.taskManager.GetTask(task.ID)
	if err != nil {
		snapshot = task
	}

	go func(taskID string, files []models.BookFile, bgCtx context.Context) {
		total := len(files)
		repairedCount := 0
		errorCount := 0

		for idx, file := range files {
			select {
			case <-bgCtx.Done():
				h.taskManager.CancelTask(taskID)
				return
			default:
			}

			fullPath := file.FilePath
			if !filepath.IsAbs(fullPath) && h.cfg != nil && h.cfg.Storage.LibraryDir != "" {
				fullPath = filepath.Join(h.cfg.Storage.LibraryDir, fullPath)
			}

			_, err := os.Stat(fullPath)
			if err != nil {
				h.taskManager.AddError(taskID, fmt.Sprintf("Файл не найден: %s", file.FilePath))
				errorCount++
				h.taskManager.UpdateProgress(taskID, idx+1, total, filepath.Base(file.FilePath))
				continue
			}

			modified, err := fb2.SanitizeFB2File(fullPath)
			if err != nil {
				h.taskManager.AddError(taskID, fmt.Sprintf("Ошибка санитизации %s: %v", file.FilePath, err))
				errorCount++
			} else if modified {
				repairedCount++
				if newInfo, statErr := os.Stat(fullPath); statErr == nil {
					file.FileSize = newInfo.Size()
				}
				if newHash, hashErr := calculateFileHash(fullPath); hashErr == nil {
					file.SHA256 = newHash
				}
				_ = h.bookRepo.UpdateBookFile(bgCtx, &file)
			}

			h.taskManager.UpdateProgress(taskID, idx+1, total, filepath.Base(file.FilePath))
		}

		resultMsg := fmt.Sprintf("Проверено: %d, исправлено: %d, ошибок: %d", total, repairedCount, errorCount)
		h.taskManager.CompleteTask(taskID, resultMsg)
	}(task.ID, fb2Files, taskCtx)

	writeJSON(w, http.StatusAccepted, snapshot)
}

func calculateFileHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
