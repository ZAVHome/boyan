package storage

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite" // Чистый Go драйвер SQLite с поддержкой WAL и FTS5
)

// DBPool объединяет раздельные пулы соединений для записи и чтения SQLite.
type DBPool struct {
	Writer *sqlx.DB
	Reader *sqlx.DB
}

// NewSQLitePool инициализирует пул соединений с базой данных SQLite.
func NewSQLitePool(ctx context.Context, dbPath string, busyTimeoutMS int, cacheSizeKB int) (*DBPool, error) {
	// Создаем директорию базы данных, если она отсутствует
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory %s: %w", dir, err)
	}

	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(%d)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=synchronous(NORMAL)&_pragma=cache_size(-%d)",
		dbPath, busyTimeoutMS, cacheSizeKB)

	// 1. Пул писателя (строго 1 открытое соединение во избежание SQLITE_BUSY)
	writerDB, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open writer db: %w", err)
	}
	writerDB.SetMaxOpenConns(1)
	writerDB.SetMaxIdleConns(1)
	writerDB.SetConnMaxLifetime(time.Hour)

	// Проверяем соединение
	if err := writerDB.PingContext(ctx); err != nil {
		writerDB.Close()
		return nil, fmt.Errorf("ping writer db: %w", err)
	}

	// 2. Пул читателей (до 10 параллельных соединений)
	readerDB, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		writerDB.Close()
		return nil, fmt.Errorf("open reader db: %w", err)
	}
	readerDB.SetMaxOpenConns(10)
	readerDB.SetMaxIdleConns(5)
	readerDB.SetConnMaxLifetime(time.Hour)

	if err := readerDB.PingContext(ctx); err != nil {
		writerDB.Close()
		readerDB.Close()
		return nil, fmt.Errorf("ping reader db: %w", err)
	}

	slog.Info("SQLite database initialized in WAL mode",
		"path", dbPath,
		"busy_timeout_ms", busyTimeoutMS,
		"cache_size_kb", cacheSizeKB,
	)

	return &DBPool{
		Writer: writerDB,
		Reader: readerDB,
	}, nil
}

// Close закрывает оба пула соединений.
func (p *DBPool) Close() error {
	var errW, errR error
	if p.Writer != nil {
		errW = p.Writer.Close()
	}
	if p.Reader != nil {
		errR = p.Reader.Close()
	}
	if errW != nil {
		return errW
	}
	return errR
}
