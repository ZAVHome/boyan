package cover

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// CoverCache управляет локальным дисковым кешем обложек с автоматической очисткой по алгоритму LRU.
type CoverCache struct {
	dir      string
	maxBytes int64
	mu       sync.RWMutex
}

// NewCoverCache создает новый экземпляр кеша обложек.
func NewCoverCache(dir string, maxMB int) (*CoverCache, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create cover cache dir %s: %w", dir, err)
	}

	if maxMB <= 0 {
		maxMB = 500 // 500 МБ по умолчанию
	}

	return &CoverCache{
		dir:      dir,
		maxBytes: int64(maxMB) * 1024 * 1024,
	}, nil
}

// GetCoverPath проверяет наличие обложки в кеше и обновляет время доступа (LRU touch).
func (c *CoverCache) GetCoverPath(bookID string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	filePath := filepath.Join(c.dir, fmt.Sprintf("%s.jpg", bookID))
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return "", false
	}

	// Обновляем время доступа/модификации для LRU (в фоновом режиме)
	go func(p string) {
		now := time.Now()
		_ = os.Chtimes(p, now, now)
	}(filePath)

	return filePath, true
}

// SaveCover сохраняет миниатюру обложки в кеш и при необходимости запускает LRU-очистку.
func (c *CoverCache) SaveCover(bookID string, data []byte) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	filePath := filepath.Join(c.dir, fmt.Sprintf("%s.jpg", bookID))
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("write cover file: %w", err)
	}

	// Проверяем лимит квоты размера кеша
	go c.PruneLRU()

	return filePath, nil
}

type fileModTime struct {
	path    string
	size    int64
	modTime time.Time
}

// PruneLRU проверяет суммарный объем кеша и при превышении лимита удаляет наименее используемые файлы.
func (c *CoverCache) PruneLRU() {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries, err := os.ReadDir(c.dir)
	if err != nil {
		slog.Error("Failed to read cover cache dir for pruning", "err", err)
		return
	}

	var files []fileModTime
	var totalSize int64

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		fullPath := filepath.Join(c.dir, entry.Name())
		files = append(files, fileModTime{
			path:    fullPath,
			size:    info.Size(),
			modTime: info.ModTime(),
		})
		totalSize += info.Size()
	}

	if totalSize <= c.maxBytes {
		return // Объем в пределах нормы
	}

	slog.Warn("Cover cache exceeded quota, initiating LRU pruning",
		"current_bytes", totalSize,
		"max_bytes", c.maxBytes,
	)

	// Сортируем от самых старых к самым новым (LRU)
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	// Очищаем до 80% от максимальной квоты
	targetSize := int64(float64(c.maxBytes) * 0.8)
	var freedBytes int64

	for _, file := range files {
		if totalSize-freedBytes <= targetSize {
			break
		}
		if err := os.Remove(file.path); err == nil {
			freedBytes += file.size
		}
	}

	slog.Info("Cover cache LRU pruning completed",
		"freed_bytes", freedBytes,
		"remaining_bytes", totalSize-freedBytes,
	)
}

// ClearAll полностью очищает весь дисковый кеш обложек.
func (c *CoverCache) ClearAll() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			_ = os.Remove(filepath.Join(c.dir, entry.Name()))
		}
	}
	return nil
}
