package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// QuarantineItem представляет файл, ожидающий модерации из-за конфликта дубликатов.
type QuarantineItem struct {
	ID             string    `db:"id" json:"id"`
	FilePath       string    `db:"file_path" json:"file_path"`
	FileSize       int64     `db:"file_size" json:"file_size"`
	SHA256         string    `db:"sha256" json:"sha256"`
	Format         string    `db:"format" json:"format"`
	ParsedTitle    string    `db:"parsed_title" json:"parsed_title"`
	ParsedAuthors  string    `db:"parsed_authors" json:"parsed_authors"`
	ExistingBookID *string   `db:"existing_book_id" json:"existing_book_id,omitempty"`
	ConflictType   string    `db:"conflict_type" json:"conflict_type"` // "exact_hash", "same_format", "fuzzy_match"
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

type QuarantineRepository struct {
	pool *DBPool
}

func NewQuarantineRepository(pool *DBPool) *QuarantineRepository {
	return &QuarantineRepository{pool: pool}
}

// AddToQuarantine сохраняет запись о дубликате в очередь карантина.
func (r *QuarantineRepository) AddToQuarantine(ctx context.Context, item *QuarantineItem) error {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}

	_, err := r.pool.Writer.ExecContext(ctx, `
		INSERT INTO quarantine (
			id, file_path, file_size, sha256, format, parsed_title, parsed_authors, existing_book_id, conflict_type, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.FilePath, item.FileSize, item.SHA256, item.Format,
		item.ParsedTitle, item.ParsedAuthors, item.ExistingBookID, item.ConflictType, item.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert quarantine item: %w", err)
	}
	return nil
}

// ListQuarantine возвращает список элементов в карантине.
func (r *QuarantineRepository) ListQuarantine(ctx context.Context, offset, limit int) ([]QuarantineItem, int, error) {
	var total int
	err := r.pool.Reader.GetContext(ctx, &total, `SELECT COUNT(*) FROM quarantine`)
	if err != nil {
		return nil, 0, err
	}

	var items []QuarantineItem
	err = r.pool.Reader.SelectContext(ctx, &items, `
		SELECT * FROM quarantine ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, limit, offset)
	return items, total, err
}

// GetQuarantineItem возвращает элемент карантина по ID.
func (r *QuarantineRepository) GetQuarantineItem(ctx context.Context, id string) (*QuarantineItem, error) {
	var item QuarantineItem
	err := r.pool.Reader.GetContext(ctx, &item, `SELECT * FROM quarantine WHERE id = ?`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

// DeleteQuarantineItem удаляет элемент из очереди карантина.
func (r *QuarantineRepository) DeleteQuarantineItem(ctx context.Context, id string) error {
	_, err := r.pool.Writer.ExecContext(ctx, `DELETE FROM quarantine WHERE id = ?`, id)
	return err
}
