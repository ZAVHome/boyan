package handlers

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"boyan/internal/config"
	"boyan/internal/logger"
	"boyan/internal/services"
	"boyan/internal/storage"
)

type AdminDashboardHandler struct {
	cfg        *config.Config
	pool       *storage.DBPool
	bookRepo   *storage.BookRepository
}

func NewAdminDashboardHandler(
	cfg *config.Config,
	pool *storage.DBPool,
	bookRepo *storage.BookRepository,
) *AdminDashboardHandler {
	return &AdminDashboardHandler{
		cfg:      cfg,
		pool:     pool,
		bookRepo: bookRepo,
	}
}

// GetStats возвращает общую статистику библиотеки и распределение форматов.
func (h *AdminDashboardHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.bookRepo.GetStorageStats(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// GetHostMetrics возвращает показатели RAM, CPU, аптайма и диска.
func (h *AdminDashboardHandler) GetHostMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := services.CollectHostMetrics(h.cfg.Storage.LibraryDir)
	writeJSON(w, http.StatusOK, metrics)
}

// DatabaseMetrics описывает состояние файлов базы данных SQLite.
type DatabaseMetrics struct {
	DBSizeBytes  int64  `json:"db_size_bytes"`
	WALSizeBytes int64  `json:"wal_size_bytes"`
	DBPath       string `json:"db_path"`
	FTSCount     int    `json:"fts_count"`
}

// GetDatabaseMetrics возвращает размеры файлов SQLite и статус FTS5.
func (h *AdminDashboardHandler) GetDatabaseMetrics(w http.ResponseWriter, r *http.Request) {
	dbPath := h.cfg.Database.SQLite.Path
	var dbSize, walSize int64

	if fi, err := os.Stat(dbPath); err == nil {
		dbSize = fi.Size()
	}
	walPath := dbPath + "-wal"
	if fi, err := os.Stat(walPath); err == nil {
		walSize = fi.Size()
	}

	var ftsCount int
	_ = h.pool.Reader.GetContext(r.Context(), &ftsCount, `SELECT COUNT(*) FROM books_fts`)

	writeJSON(w, http.StatusOK, DatabaseMetrics{
		DBSizeBytes:  dbSize,
		WALSizeBytes: walSize,
		DBPath:       dbPath,
		FTSCount:     ftsCount,
	})
}

// CheckpointDatabase выполняет ручной сброс WAL и VACUUM.
func (h *AdminDashboardHandler) CheckpointDatabase(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, err := h.pool.Writer.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "WAL checkpoint failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "WAL checkpoint completed successfully",
	})
}

// CacheMetrics описывает состояние кэша миниатюр обложек.
type CacheMetrics struct {
	CacheDir   string `json:"cache_dir"`
	TotalFiles int    `json:"total_files"`
	SizeBytes  int64  `json:"size_bytes"`
}

// GetCacheMetrics возвращает объем и число файлов в локальном кэше обложек.
func (h *AdminDashboardHandler) GetCacheMetrics(w http.ResponseWriter, r *http.Request) {
	cacheDir := h.cfg.Metadata.CoverCacheDir
	if cacheDir == "" {
		cacheDir = filepath.Join("data", "covers")
	}

	var totalFiles int
	var totalSize int64

	_ = filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalFiles++
			totalSize += info.Size()
		}
		return nil
	})

	writeJSON(w, http.StatusOK, CacheMetrics{
		CacheDir:   cacheDir,
		TotalFiles: totalFiles,
		SizeBytes:  totalSize,
	})
}

// PurgeCache очищает все сгенерированные файлы из кэша обложек.
func (h *AdminDashboardHandler) PurgeCache(w http.ResponseWriter, r *http.Request) {
	cacheDir := h.cfg.Metadata.CoverCacheDir
	if cacheDir == "" {
		cacheDir = filepath.Join("data", "covers")
	}

	deletedCount := 0
	_ = filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			_ = os.Remove(path)
			deletedCount++
		}
		return nil
	})

	// Сбрасываем флаги cover_cached в таблице books
	_, _ = h.pool.Writer.ExecContext(context.Background(), `UPDATE books SET cover_cached = FALSE`)

	writeJSON(w, http.StatusOK, map[string]any{
		"message":       "Cover cache purged successfully",
		"deleted_files": deletedCount,
	})
}

// GetLogs возвращает записи журнала из буфера.
func (h *AdminDashboardHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	levelFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("level")))

	entries := logger.GlobalBuffer().GetEntries(limit, levelFilter)
	writeJSON(w, http.StatusOK, map[string]any{
		"entries": entries,
		"total":   len(entries),
	})
}

// GetTelegramStatus возвращает текущий статус Telegram-бота.
func (h *AdminDashboardHandler) GetTelegramStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":         h.cfg.Telegram.Enabled,
		"has_token":       h.cfg.Telegram.BotToken != "",
		"allowed_users":   len(h.cfg.Telegram.AllowedUserIDs),
	})
}
