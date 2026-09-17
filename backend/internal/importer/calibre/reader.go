package calibre

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// Reader обеспечивает чтение метаданных Calibre из metadata.db в режиме Read-Only.
type Reader struct {
	db         *sqlx.DB
	calibreDir string
}

// Open открывает базу metadata.db внутри указанной директории библиотеки Calibre.
func Open(calibreDir string) (*Reader, error) {
	dbPath := filepath.Join(calibreDir, "metadata.db")
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("calibre database not found at %s: %w", dbPath, err)
	}

	dsn := fmt.Sprintf("file:%s?mode=ro&_pragma=busy_timeout(5000)", filepath.ToSlash(dbPath))
	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open calibre sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping calibre sqlite: %w", err)
	}

	return &Reader{
		db:         db,
		calibreDir: calibreDir,
	}, nil
}

// Close закрывает соединение с базой Calibre.
func (r *Reader) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// ListBooks возвращает список всех книг из таблицы `books`.
func (r *Reader) ListBooks(ctx context.Context) ([]CalibreBook, error) {
	query := `SELECT id, title, sort, timestamp, pubdate, series_index, path, has_cover FROM books ORDER BY id ASC`
	var books []CalibreBook
	err := r.db.SelectContext(ctx, &books, query)
	if err != nil {
		return nil, fmt.Errorf("query calibre books: %w", err)
	}
	return books, nil
}

// GetAuthorsForBook возвращает авторов для конкретной книги.
func (r *Reader) GetAuthorsForBook(ctx context.Context, bookID int64) ([]CalibreAuthor, error) {
	query := `
		SELECT a.id, a.name, a.sort
		FROM authors a
		JOIN books_authors_link l ON l.author = a.id
		WHERE l.book = ?
		ORDER BY l.id ASC
	`
	var authors []CalibreAuthor
	err := r.db.SelectContext(ctx, &authors, query, bookID)
	if err != nil {
		return nil, fmt.Errorf("query book authors: %w", err)
	}
	return authors, nil
}

// GetSeriesForBook возвращает серии для книги (обычно одна).
func (r *Reader) GetSeriesForBook(ctx context.Context, bookID int64) ([]CalibreSeries, error) {
	query := `
		SELECT s.id, s.name, s.sort
		FROM series s
		JOIN books_series_link l ON l.series = s.id
		WHERE l.book = ?
	`
	var series []CalibreSeries
	err := r.db.SelectContext(ctx, &series, query, bookID)
	if err != nil {
		return nil, fmt.Errorf("query book series: %w", err)
	}
	return series, nil
}

// GetTagsForBook возвращает теги (жанры/категории) книги.
func (r *Reader) GetTagsForBook(ctx context.Context, bookID int64) ([]CalibreTag, error) {
	query := `
		SELECT t.id, t.name
		FROM tags t
		JOIN books_tags_link l ON l.tag = t.id
		WHERE l.book = ?
	`
	var tags []CalibreTag
	err := r.db.SelectContext(ctx, &tags, query, bookID)
	if err != nil {
		return nil, fmt.Errorf("query book tags: %w", err)
	}
	return tags, nil
}

// GetDataForBook возвращает форматы и файлы, привязанные к книге.
func (r *Reader) GetDataForBook(ctx context.Context, bookID int64) ([]CalibreData, error) {
	query := `
		SELECT id, book, format, uncompressed_size, name
		FROM data
		WHERE book = ?
	`
	var dataList []CalibreData
	err := r.db.SelectContext(ctx, &dataList, query, bookID)
	if err != nil {
		return nil, fmt.Errorf("query book data formats: %w", err)
	}
	return dataList, nil
}

// GetCommentForBook возвращает аннотацию к книге.
func (r *Reader) GetCommentForBook(ctx context.Context, bookID int64) (string, error) {
	query := `SELECT text FROM comments WHERE book = ? LIMIT 1`
	var text string
	err := r.db.GetContext(ctx, &text, query, bookID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("query book comment: %w", err)
	}
	return text, nil
}

// GetIdentifiersForBook возвращает идентификаторы (ISBN, Goodreads и др.).
func (r *Reader) GetIdentifiersForBook(ctx context.Context, bookID int64) (map[string]string, error) {
	query := `SELECT type, val FROM identifiers WHERE book = ?`
	var list []CalibreIdentifier
	err := r.db.SelectContext(ctx, &list, query, bookID)
	if err != nil {
		return nil, fmt.Errorf("query book identifiers: %w", err)
	}
	res := make(map[string]string, len(list))
	for _, item := range list {
		res[item.Type] = item.Val
	}
	return res, nil
}
