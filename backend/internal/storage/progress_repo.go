package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"boyan/internal/models"
)

// ReadProgress фиксирует прогресс и позицию чтения книги пользователем.
type ReadProgress struct {
	UserID          string    `db:"user_id" json:"user_id"`
	BookID          string    `db:"book_id" json:"book_id"`
	Format          string    `db:"format" json:"format"`
	ProgressPercent float64   `db:"progress_percent" json:"progress_percent"` // 0.0 - 100.0
	Position        string    `db:"position" json:"position"`                 // CFI или параграф
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

type ProgressRepository struct {
	pool *DBPool
}

func NewProgressRepository(pool *DBPool) *ProgressRepository {
	return &ProgressRepository{pool: pool}
}

// SaveProgress сохраняет или обновляет прогресс чтения книги.
func (r *ProgressRepository) SaveProgress(ctx context.Context, p *ReadProgress) error {
	p.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Writer.ExecContext(ctx, `
		INSERT INTO read_progress (user_id, book_id, format, progress_percent, position, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, book_id) DO UPDATE SET
			format = excluded.format,
			progress_percent = excluded.progress_percent,
			position = excluded.position,
			updated_at = excluded.updated_at
	`, p.UserID, p.BookID, p.Format, p.ProgressPercent, p.Position, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert read_progress: %w", err)
	}
	return nil
}

// GetProgress возвращает текущий прогресс чтения книги пользователем.
func (r *ProgressRepository) GetProgress(ctx context.Context, userID, bookID string) (*ReadProgress, error) {
	var p ReadProgress
	err := r.pool.Reader.GetContext(ctx, &p, `
		SELECT * FROM read_progress WHERE user_id = ? AND book_id = ?
	`, userID, bookID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// AddToShelf добавляет книгу на полку пользователя ("reading", "finished", "favorite").
func (r *ProgressRepository) AddToShelf(ctx context.Context, userID, bookID, shelfType string) error {
	_, err := r.pool.Writer.ExecContext(ctx, `
		INSERT INTO user_shelves (user_id, book_id, shelf_type, added_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id, book_id, shelf_type) DO NOTHING
	`, userID, bookID, shelfType, time.Now().UTC())
	return err
}

// RemoveFromShelf удаляет книгу с полки.
func (r *ProgressRepository) RemoveFromShelf(ctx context.Context, userID, bookID, shelfType string) error {
	_, err := r.pool.Writer.ExecContext(ctx, `
		DELETE FROM user_shelves WHERE user_id = ? AND book_id = ? AND shelf_type = ?
	`, userID, bookID, shelfType)
	return err
}

// GetUserShelfBooks возвращает книги с заданной полки пользователя.
func (r *ProgressRepository) GetUserShelfBooks(ctx context.Context, userID, shelfType string) ([]models.Book, error) {
	var books []models.Book
	err := r.pool.Reader.SelectContext(ctx, &books, `
		SELECT b.*
		FROM books b
		JOIN user_shelves us ON b.id = us.book_id
		WHERE us.user_id = ? AND us.shelf_type = ?
		ORDER BY us.added_at DESC
	`, userID, shelfType)
	return books, err
}
