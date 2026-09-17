package logger

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// LogEntry представляет структурированную запись лога для админ-панели.
type LogEntry struct {
	Timestamp time.Time      `json:"timestamp"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Attrs     map[string]any `json:"attrs,omitempty"`
}

// LogBuffer хранит последние N записей логов в кольцевом буфере в памяти.
type LogBuffer struct {
	mu       sync.RWMutex
	capacity int
	entries  []LogEntry
}

var globalBuffer = NewLogBuffer(500)

// GlobalBuffer возвращает синглтон буфера логов.
func GlobalBuffer() *LogBuffer {
	return globalBuffer
}

func NewLogBuffer(capacity int) *LogBuffer {
	if capacity <= 0 {
		capacity = 500
	}
	return &LogBuffer{
		capacity: capacity,
		entries:  make([]LogEntry, 0, capacity),
	}
}

// Add добавляет запись в буфер.
func (b *LogBuffer) Add(entry LogEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.entries) >= b.capacity {
		b.entries = b.entries[1:]
	}
	b.entries = append(b.entries, entry)
}

// GetEntries возвращает отфильтрованный список последних записей.
func (b *LogBuffer) GetEntries(limit int, levelFilter string) []LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var filtered []LogEntry
	for i := len(b.entries) - 1; i >= 0; i-- {
		e := b.entries[i]
		if levelFilter != "" && e.Level != levelFilter {
			continue
		}
		filtered = append(filtered, e)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	return filtered
}

// BufferedHandler оборачивает базовый slog.Handler и дублирует записи в LogBuffer.
type BufferedHandler struct {
	inner  slog.Handler
	buffer *LogBuffer
}

func NewBufferedHandler(inner slog.Handler, buffer *LogBuffer) *BufferedHandler {
	return &BufferedHandler{
		inner:  inner,
		buffer: buffer,
	}
}

func (h *BufferedHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *BufferedHandler) Handle(ctx context.Context, record slog.Record) error {
	attrs := make(map[string]any)
	record.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	h.buffer.Add(LogEntry{
		Timestamp: record.Time.UTC(),
		Level:     record.Level.String(),
		Message:   record.Message,
		Attrs:     attrs,
	})

	return h.inner.Handle(ctx, record)
}

func (h *BufferedHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &BufferedHandler{
		inner:  h.inner.WithAttrs(attrs),
		buffer: h.buffer,
	}
}

func (h *BufferedHandler) WithGroup(name string) slog.Handler {
	return &BufferedHandler{
		inner:  h.inner.WithGroup(name),
		buffer: h.buffer,
	}
}
