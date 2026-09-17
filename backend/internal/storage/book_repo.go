package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"boyan/internal/models"

	"github.com/google/uuid"
)

// BookRepository предоставляет методы для работы с книгами и их метаданными.
type BookRepository struct {
	pool *DBPool
}

func NewBookRepository(pool *DBPool) *BookRepository {
	return &BookRepository{pool: pool}
}

// SaveBook сохраняет книгу со всеми связанными авторами, сериями, жанрами и файлами в единой транзакции.
func (r *BookRepository) SaveBook(
	ctx context.Context,
	book *models.Book,
	authors []models.AuthorDetail,
	seriesList []models.SeriesDetail,
	genres []models.Genre,
	file *models.BookFile,
) error {
	tx, err := r.pool.Writer.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if book.ID == "" {
		book.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if book.CreatedAt.IsZero() {
		book.CreatedAt = now
	}
	book.UpdatedAt = now

	// 1. Вставка / обновление книги
	_, err = tx.ExecContext(ctx, `
		INSERT INTO books (
			id, title, original_title, annotation, language, published_date, publisher, isbn, cover_cached, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			original_title = excluded.original_title,
			annotation = excluded.annotation,
			language = excluded.language,
			published_date = excluded.published_date,
			publisher = excluded.publisher,
			isbn = excluded.isbn,
			cover_cached = excluded.cover_cached,
			updated_at = excluded.updated_at
	`, book.ID, book.Title, book.OriginalTitle, book.Annotation, book.Language,
		book.PublishedDate, book.Publisher, book.ISBN, book.CoverCached, book.CreatedAt, book.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert book: %w", err)
	}

	// 2. Обработка авторов
	var authorNames []string
	for i, a := range authors {
		if a.ID == "" {
			a.ID = uuid.NewString()
		}
		if a.SortName == "" {
			a.SortName = a.Name
		}
		authorNames = append(authorNames, a.Name)

		// Сохраняем автора
		_, err = tx.ExecContext(ctx, `
			INSERT INTO authors (id, name, sort_name) VALUES (?, ?, ?)
			ON CONFLICT(name) DO UPDATE SET sort_name = excluded.sort_name
		`, a.ID, a.Name, a.SortName)
		if err != nil {
			return fmt.Errorf("upsert author %s: %w", a.Name, err)
		}

		// Получаем реальный ID автора при конфликте
		var actualAuthorID string
		err = tx.GetContext(ctx, &actualAuthorID, `SELECT id FROM authors WHERE name = ?`, a.Name)
		if err != nil {
			return fmt.Errorf("get author id for %s: %w", a.Name, err)
		}

		role := a.Role
		if role == "" {
			role = "author"
		}
		order := a.Order
		if order == 0 {
			order = i + 1
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO book_authors (book_id, author_id, role, author_order)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(book_id, author_id, role) DO UPDATE SET author_order = excluded.author_order
		`, book.ID, actualAuthorID, role, order)
		if err != nil {
			return fmt.Errorf("link book author: %w", err)
		}
	}

	// 3. Обработка серий
	var seriesNames []string
	for _, s := range seriesList {
		if s.ID == "" {
			s.ID = uuid.NewString()
		}
		if s.SortName == "" {
			s.SortName = s.Name
		}
		seriesNames = append(seriesNames, s.Name)

		_, err = tx.ExecContext(ctx, `
			INSERT INTO series (id, name, sort_name) VALUES (?, ?, ?)
			ON CONFLICT(name) DO UPDATE SET sort_name = excluded.sort_name
		`, s.ID, s.Name, s.SortName)
		if err != nil {
			return fmt.Errorf("upsert series %s: %w", s.Name, err)
		}

		var actualSeriesID string
		err = tx.GetContext(ctx, &actualSeriesID, `SELECT id FROM series WHERE name = ?`, s.Name)
		if err != nil {
			return fmt.Errorf("get series id for %s: %w", s.Name, err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO book_series (book_id, series_id, series_index)
			VALUES (?, ?, ?)
			ON CONFLICT(book_id, series_id) DO UPDATE SET series_index = excluded.series_index
		`, book.ID, actualSeriesID, s.Index)
		if err != nil {
			return fmt.Errorf("link book series: %w", err)
		}
	}

	// 4. Обработка жанров
	for _, g := range genres {
		if g.Code == "" {
			continue
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO genres (code, name_ru, name_en, category_ru, category_en)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(code) DO NOTHING
		`, g.Code, g.NameRU, g.NameEN, g.CategoryRU, g.CategoryEN)
		if err != nil {
			return fmt.Errorf("insert genre %s: %w", g.Code, err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO book_genres (book_id, genre_code)
			VALUES (?, ?)
			ON CONFLICT(book_id, genre_code) DO NOTHING
		`, book.ID, g.Code)
		if err != nil {
			return fmt.Errorf("link book genre: %w", err)
		}
	}

	// 5. Обработка файла
	if file != nil {
		if file.ID == "" {
			file.ID = uuid.NewString()
		}
		file.BookID = book.ID
		if file.CreatedAt.IsZero() {
			file.CreatedAt = now
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO book_files (id, book_id, format, file_path, archive_inner_path, file_size, sha256, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, file.ID, file.BookID, file.Format, file.FilePath, file.ArchiveInnerPath, file.FileSize, file.SHA256, file.CreatedAt)
		if err != nil {
			return fmt.Errorf("insert book file: %w", err)
		}
	}

	// 6. Обновление FTS5 индекса
	_, err = tx.ExecContext(ctx, `DELETE FROM books_fts WHERE book_id = ?`, book.ID)
	if err != nil {
		return fmt.Errorf("clear fts: %w", err)
	}

	authorsStr := strings.Join(authorNames, " ")
	seriesStr := strings.Join(seriesNames, " ")

	_, err = tx.ExecContext(ctx, `
		INSERT INTO books_fts (book_id, title, original_title, annotation, author_names, series_names)
		VALUES (?, ?, ?, ?, ?, ?)
	`, book.ID, book.Title, book.OriginalTitle, book.Annotation, authorsStr, seriesStr)
	if err != nil {
		return fmt.Errorf("insert fts: %w", err)
	}

	return tx.Commit()
}

// GetBookByID возвращает книгу с ее авторами, сериями, жанрами и файлами.
func (r *BookRepository) GetBookByID(ctx context.Context, id string) (*models.Book, error) {
	var book models.Book
	err := r.pool.Reader.GetContext(ctx, &book, `SELECT * FROM books WHERE id = ?`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get book by id: %w", err)
	}

	// Загрузка авторов
	err = r.pool.Reader.SelectContext(ctx, &book.Authors, `
		SELECT a.id, a.name, a.sort_name, ba.role, ba.author_order
		FROM authors a
		JOIN book_authors ba ON a.id = ba.author_id
		WHERE ba.book_id = ?
		ORDER BY ba.author_order ASC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("get book authors: %w", err)
	}

	// Загрузка серий
	err = r.pool.Reader.SelectContext(ctx, &book.Series, `
		SELECT s.id, s.name, s.sort_name, bs.series_index
		FROM series s
		JOIN book_series bs ON s.id = bs.series_id
		WHERE bs.book_id = ?
		ORDER BY s.name ASC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("get book series: %w", err)
	}

	// Загрузка жанров
	err = r.pool.Reader.SelectContext(ctx, &book.Genres, `
		SELECT g.code, g.name_ru, g.name_en, g.category_ru, g.category_en
		FROM genres g
		JOIN book_genres bg ON g.code = bg.genre_code
		WHERE bg.book_id = ?
	`, id)
	if err != nil {
		return nil, fmt.Errorf("get book genres: %w", err)
	}

	// Загрузка файлов
	err = r.pool.Reader.SelectContext(ctx, &book.Files, `
		SELECT * FROM book_files WHERE book_id = ?
	`, id)
	if err != nil {
		return nil, fmt.Errorf("get book files: %w", err)
	}

	return &book, nil
}

// ListBooks возвращает список книг с пагинацией и общее количество.
func (r *BookRepository) ListBooks(ctx context.Context, offset, limit int) ([]models.Book, int, error) {
	var total int
	err := r.pool.Reader.GetContext(ctx, &total, `SELECT COUNT(*) FROM books`)
	if err != nil {
		return nil, 0, fmt.Errorf("count books: %w", err)
	}

	var books []models.Book
	err = r.pool.Reader.SelectContext(ctx, &books, `
		SELECT * FROM books ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list books: %w", err)
	}

	// Заполняем авторов для каждой книги
	for i := range books {
		_ = r.pool.Reader.SelectContext(ctx, &books[i].Authors, `
			SELECT a.id, a.name, a.sort_name, ba.role, ba.author_order
			FROM authors a
			JOIN book_authors ba ON a.id = ba.author_id
			WHERE ba.book_id = ?
			ORDER BY ba.author_order ASC
		`, books[i].ID)

		_ = r.pool.Reader.SelectContext(ctx, &books[i].Series, `
			SELECT s.id, s.name, s.sort_name, bs.series_index
			FROM series s
			JOIN book_series bs ON s.id = bs.series_id
			WHERE bs.book_id = ?
		`, books[i].ID)

		_ = r.pool.Reader.SelectContext(ctx, &books[i].Files, `
			SELECT * FROM book_files WHERE book_id = ?
		`, books[i].ID)
	}

	return books, total, nil
}

// SearchBooksFTS выполняет полнотекстовый поиск по индексу FTS5.
func (r *BookRepository) SearchBooksFTS(ctx context.Context, query string, offset, limit int) ([]models.Book, int, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return r.ListBooks(ctx, offset, limit)
	}

	// Подготавливаем слова для префиксного поиска
	words := strings.Fields(query)
	var formattedTerms []string
	for _, w := range words {
		cleaned := strings.Map(func(r rune) rune {
			if strings.ContainsRune(`"*-+^:()'`, r) {
				return -1
			}
			return r
		}, w)
		if cleaned != "" {
			formattedTerms = append(formattedTerms, cleaned+"*")
		}
	}
	if len(formattedTerms) == 0 {
		return nil, 0, nil
	}
	ftsQuery := strings.Join(formattedTerms, " ")

	var total int
	err := r.pool.Reader.GetContext(ctx, &total, `
		SELECT COUNT(*) FROM books_fts WHERE books_fts MATCH ?
	`, ftsQuery)
	if err != nil {
		return nil, 0, fmt.Errorf("count fts: %w", err)
	}

	var books []models.Book
	querySQL := `
		SELECT b.*
		FROM books b
		JOIN books_fts fts ON b.id = fts.book_id
		WHERE books_fts MATCH ?
		ORDER BY bm25(books_fts)
		LIMIT ? OFFSET ?
	`
	err = r.pool.Reader.SelectContext(ctx, &books, querySQL, ftsQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("select fts: %w", err)
	}

	for i := range books {
		_ = r.pool.Reader.SelectContext(ctx, &books[i].Authors, `
			SELECT a.id, a.name, a.sort_name, ba.role, ba.author_order
			FROM authors a
			JOIN book_authors ba ON a.id = ba.author_id
			WHERE ba.book_id = ?
			ORDER BY ba.author_order ASC
		`, books[i].ID)

		_ = r.pool.Reader.SelectContext(ctx, &books[i].Series, `
			SELECT s.id, s.name, s.sort_name, bs.series_index
			FROM series s
			JOIN book_series bs ON s.id = bs.series_id
			WHERE bs.book_id = ?
		`, books[i].ID)

		_ = r.pool.Reader.SelectContext(ctx, &books[i].Files, `
			SELECT * FROM book_files WHERE book_id = ?
		`, books[i].ID)
	}

	return books, total, nil
}

// SetCoverCached обновляет флаг наличия закешированной обложки.
func (r *BookRepository) SetCoverCached(ctx context.Context, bookID string, cached bool) error {
	val := 0
	if cached {
		val = 1
	}
	_, err := r.pool.Writer.ExecContext(ctx, `UPDATE books SET cover_cached = ?, updated_at = ? WHERE id = ?`,
		val, time.Now().UTC(), bookID)
	return err
}

// DTO для навигации OPDS

type LetterCount struct {
	Letter string `db:"letter" json:"letter"`
	Count  int    `db:"count" json:"count"`
}

type AuthorWithCount struct {
	models.Author
	BookCount int `db:"book_count" json:"book_count"`
}

type SeriesWithCount struct {
	models.Series
	BookCount int `db:"book_count" json:"book_count"`
}

type CategoryWithCount struct {
	CategoryRU string `db:"category_ru" json:"category_ru"`
	CategoryEN string `db:"category_en" json:"category_en"`
	BookCount  int    `db:"book_count" json:"book_count"`
}

type GenreWithCount struct {
	models.Genre
	BookCount int `db:"book_count" json:"book_count"`
}

// GetAuthorsAlphabet возвращает список начальных букв авторов с числом авторов.
func (r *BookRepository) GetAuthorsAlphabet(ctx context.Context) ([]LetterCount, error) {
	var list []LetterCount
	err := r.pool.Reader.SelectContext(ctx, &list, `
		SELECT UPPER(SUBSTR(sort_name, 1, 1)) AS letter, COUNT(*) AS count
		FROM authors
		GROUP BY letter
		ORDER BY letter ASC
	`)
	return list, err
}

// GetAuthorsByLetter возвращает список авторов на заданную букву.
func (r *BookRepository) GetAuthorsByLetter(ctx context.Context, letter string, offset, limit int) ([]AuthorWithCount, int, error) {
	letter = strings.ToUpper(letter)
	var total int
	err := r.pool.Reader.GetContext(ctx, &total, `
		SELECT COUNT(*) FROM authors WHERE UPPER(SUBSTR(sort_name, 1, 1)) = ?
	`, letter)
	if err != nil {
		return nil, 0, err
	}

	var authors []AuthorWithCount
	err = r.pool.Reader.SelectContext(ctx, &authors, `
		SELECT a.id, a.name, a.sort_name, COUNT(ba.book_id) AS book_count
		FROM authors a
		LEFT JOIN book_authors ba ON a.id = ba.author_id
		WHERE UPPER(SUBSTR(a.sort_name, 1, 1)) = ?
		GROUP BY a.id, a.name, a.sort_name
		ORDER BY a.sort_name ASC
		LIMIT ? OFFSET ?
	`, letter, limit, offset)
	return authors, total, err
}

// GetAuthorByID возвращает автора по ID.
func (r *BookRepository) GetAuthorByID(ctx context.Context, id string) (*models.Author, error) {
	var a models.Author
	err := r.pool.Reader.GetContext(ctx, &a, `SELECT * FROM authors WHERE id = ?`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

// GetAuthorBooks возвращает книги автора с деталями.
func (r *BookRepository) GetAuthorBooks(ctx context.Context, authorID string) ([]models.Book, error) {
	var books []models.Book
	err := r.pool.Reader.SelectContext(ctx, &books, `
		SELECT b.*
		FROM books b
		JOIN book_authors ba ON b.id = ba.book_id
		WHERE ba.author_id = ?
		ORDER BY b.title ASC
	`, authorID)
	if err != nil {
		return nil, err
	}
	r.enrichBooksWithDetails(ctx, books)
	return books, nil
}

// GetSeriesAlphabet возвращает начальные буквы серий.
func (r *BookRepository) GetSeriesAlphabet(ctx context.Context) ([]LetterCount, error) {
	var list []LetterCount
	err := r.pool.Reader.SelectContext(ctx, &list, `
		SELECT UPPER(SUBSTR(name, 1, 1)) AS letter, COUNT(*) AS count
		FROM series
		GROUP BY letter
		ORDER BY letter ASC
	`)
	return list, err
}

// GetSeriesByLetter возвращает серии на заданную букву.
func (r *BookRepository) GetSeriesByLetter(ctx context.Context, letter string, offset, limit int) ([]SeriesWithCount, int, error) {
	letter = strings.ToUpper(letter)
	var total int
	err := r.pool.Reader.GetContext(ctx, &total, `
		SELECT COUNT(*) FROM series WHERE UPPER(SUBSTR(name, 1, 1)) = ?
	`, letter)
	if err != nil {
		return nil, 0, err
	}

	var series []SeriesWithCount
	err = r.pool.Reader.SelectContext(ctx, &series, `
		SELECT s.id, s.name, s.sort_name, COUNT(bs.book_id) AS book_count
		FROM series s
		LEFT JOIN book_series bs ON s.id = bs.series_id
		WHERE UPPER(SUBSTR(s.name, 1, 1)) = ?
		GROUP BY s.id, s.name, s.sort_name
		ORDER BY s.name ASC
		LIMIT ? OFFSET ?
	`, letter, limit, offset)
	return series, total, err
}

// GetSeriesByID возвращает серию по ID.
func (r *BookRepository) GetSeriesByID(ctx context.Context, id string) (*models.Series, error) {
	var s models.Series
	err := r.pool.Reader.GetContext(ctx, &s, `SELECT * FROM series WHERE id = ?`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// GetSeriesBooks возвращает книги серии, отсортированные по индексу в серии.
func (r *BookRepository) GetSeriesBooks(ctx context.Context, seriesID string) ([]models.Book, error) {
	var books []models.Book
	err := r.pool.Reader.SelectContext(ctx, &books, `
		SELECT b.*
		FROM books b
		JOIN book_series bs ON b.id = bs.book_id
		WHERE bs.series_id = ?
		ORDER BY bs.series_index ASC, b.title ASC
	`, seriesID)
	if err != nil {
		return nil, err
	}
	r.enrichBooksWithDetails(ctx, books)
	return books, nil
}

// GetGenreCategories возвращает список верхнеуровневых категорий жанров.
func (r *BookRepository) GetGenreCategories(ctx context.Context) ([]CategoryWithCount, error) {
	var list []CategoryWithCount
	err := r.pool.Reader.SelectContext(ctx, &list, `
		SELECT g.category_ru, g.category_en, COUNT(DISTINCT bg.book_id) AS book_count
		FROM genres g
		JOIN book_genres bg ON g.code = bg.genre_code
		GROUP BY g.category_ru, g.category_en
		ORDER BY g.category_ru ASC
	`)
	return list, err
}

// GetGenresByCategory возвращает подкатегории жанров внутри категории.
func (r *BookRepository) GetGenresByCategory(ctx context.Context, categoryRU string) ([]GenreWithCount, error) {
	var list []GenreWithCount
	err := r.pool.Reader.SelectContext(ctx, &list, `
		SELECT g.code, g.name_ru, g.name_en, g.category_ru, g.category_en, COUNT(DISTINCT bg.book_id) AS book_count
		FROM genres g
		JOIN book_genres bg ON g.code = bg.genre_code
		WHERE g.category_ru = ?
		GROUP BY g.code
		ORDER BY g.name_ru ASC
	`, categoryRU)
	return list, err
}

// GetGenreByCode возвращает жанр по коду.
func (r *BookRepository) GetGenreByCode(ctx context.Context, code string) (*models.Genre, error) {
	var g models.Genre
	err := r.pool.Reader.GetContext(ctx, &g, `SELECT * FROM genres WHERE code = ?`, code)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &g, nil
}

// GetGenreBooks возвращает книги заданного жанра.
func (r *BookRepository) GetGenreBooks(ctx context.Context, genreCode string, offset, limit int) ([]models.Book, int, error) {
	var total int
	err := r.pool.Reader.GetContext(ctx, &total, `
		SELECT COUNT(DISTINCT book_id) FROM book_genres WHERE genre_code = ?
	`, genreCode)
	if err != nil {
		return nil, 0, err
	}

	var books []models.Book
	err = r.pool.Reader.SelectContext(ctx, &books, `
		SELECT b.*
		FROM books b
		JOIN book_genres bg ON b.id = bg.book_id
		WHERE bg.genre_code = ?
		ORDER BY b.title ASC
		LIMIT ? OFFSET ?
	`, genreCode, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	r.enrichBooksWithDetails(ctx, books)
	return books, total, nil
}

// GetRecentBooks возвращает новинки библиотеки (сортировка по created_at DESC).
func (r *BookRepository) GetRecentBooks(ctx context.Context, offset, limit int) ([]models.Book, int, error) {
	return r.ListBooks(ctx, offset, limit)
}

// GetBookFileByFormat возвращает метаданные конкретного файла книги для скачивания/стриминга.
func (r *BookRepository) GetBookFileByFormat(ctx context.Context, bookID, format string) (*models.BookFile, error) {
	var file models.BookFile
	err := r.pool.Reader.GetContext(ctx, &file, `
		SELECT * FROM book_files WHERE book_id = ? AND format = ? LIMIT 1
	`, bookID, strings.ToLower(format))
	if err != nil {
		if err == sql.ErrNoRows {
			// Пробуем искать fb2.zip если запрошен fb2
			if strings.ToLower(format) == "fb2" {
				errZip := r.pool.Reader.GetContext(ctx, &file, `
					SELECT * FROM book_files WHERE book_id = ? AND format = 'fb2.zip' LIMIT 1
				`, bookID)
				if errZip == nil {
					return &file, nil
				}
			}
			return nil, nil
		}
		return nil, err
	}
	return &file, nil
}

func (r *BookRepository) enrichBooksWithDetails(ctx context.Context, books []models.Book) {
	for i := range books {
		_ = r.pool.Reader.SelectContext(ctx, &books[i].Authors, `
			SELECT a.id, a.name, a.sort_name, ba.role, ba.author_order
			FROM authors a
			JOIN book_authors ba ON a.id = ba.author_id
			WHERE ba.book_id = ?
			ORDER BY ba.author_order ASC
		`, books[i].ID)

		_ = r.pool.Reader.SelectContext(ctx, &books[i].Series, `
			SELECT s.id, s.name, s.sort_name, bs.series_index
			FROM series s
			JOIN book_series bs ON s.id = bs.series_id
			WHERE bs.book_id = ?
		`, books[i].ID)

		_ = r.pool.Reader.SelectContext(ctx, &books[i].Files, `
			SELECT * FROM book_files WHERE book_id = ?
		`, books[i].ID)
	}
}

// FindFileBySHA256 находит файл книги по SHA-256 хешу.
func (r *BookRepository) FindFileBySHA256(ctx context.Context, sha256 string) (*models.BookFile, error) {
	var file models.BookFile
	err := r.pool.Reader.GetContext(ctx, &file, `SELECT * FROM book_files WHERE sha256 = ? LIMIT 1`, sha256)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find file by sha256: %w", err)
	}
	return &file, nil
}

// GetBookFileByID возвращает запись файла книги по ID.
func (r *BookRepository) GetBookFileByID(ctx context.Context, fileID string) (*models.BookFile, error) {
	var file models.BookFile
	err := r.pool.Reader.GetContext(ctx, &file, `SELECT * FROM book_files WHERE id = ? LIMIT 1`, fileID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get book file by id: %w", err)
	}
	return &file, nil
}

// AddBookFile прикрепляет новый формат файла к существующей книге.
func (r *BookRepository) AddBookFile(ctx context.Context, file *models.BookFile) error {
	if file.ID == "" {
		file.ID = uuid.NewString()
	}
	if file.CreatedAt.IsZero() {
		file.CreatedAt = time.Now().UTC()
	}
	_, err := r.pool.Writer.ExecContext(ctx, `
		INSERT INTO book_files (id, book_id, format, file_path, archive_inner_path, file_size, sha256, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, file.ID, file.BookID, file.Format, file.FilePath, file.ArchiveInnerPath, file.FileSize, file.SHA256, file.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert book file: %w", err)
	}
	return nil
}

// UpdateBookFile обновляет путь, размер и хеш файла книги (например, при замене файла).
func (r *BookRepository) UpdateBookFile(ctx context.Context, file *models.BookFile) error {
	_, err := r.pool.Writer.ExecContext(ctx, `
		UPDATE book_files
		SET format = ?, file_path = ?, archive_inner_path = ?, file_size = ?, sha256 = ?
		WHERE id = ?
	`, file.Format, file.FilePath, file.ArchiveInnerPath, file.FileSize, file.SHA256, file.ID)
	return err
}

// DeleteBookFile удаляет запись о файле из базы данных.
func (r *BookRepository) DeleteBookFile(ctx context.Context, fileID string) error {
	_, err := r.pool.Writer.ExecContext(ctx, `DELETE FROM book_files WHERE id = ?`, fileID)
	return err
}

// DeleteBook удаляет книгу и все связанные связи и файлы из базы данных (каскадно).
func (r *BookRepository) DeleteBook(ctx context.Context, bookID string) error {
	tx, err := r.pool.Writer.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Удаляем из FTS5
	_, _ = tx.ExecContext(ctx, `DELETE FROM books_fts WHERE book_id = ?`, bookID)

	// Удаляем саму книгу (каскадные внешние ключи удалят book_authors, book_series, book_genres, book_files)
	_, err = tx.ExecContext(ctx, `DELETE FROM books WHERE id = ?`, bookID)
	if err != nil {
		return fmt.Errorf("delete book: %w", err)
	}

	return tx.Commit()
}

// FindBookByTitleAndAuthor выполняет поиск книги по точному или нормализованному названию и автору.
func (r *BookRepository) FindBookByTitleAndAuthor(ctx context.Context, title, authorName string) (*models.Book, error) {
	title = strings.TrimSpace(title)
	authorName = strings.TrimSpace(authorName)
	if title == "" {
		return nil, nil
	}

	var bookID string
	var err error

	if authorName != "" {
		query := `
			SELECT b.id
			FROM books b
			JOIN book_authors ba ON b.id = ba.book_id
			JOIN authors a ON ba.author_id = a.id
			WHERE LOWER(TRIM(b.title)) = LOWER(TRIM(?))
			  AND (LOWER(TRIM(a.name)) = LOWER(TRIM(?)) OR LOWER(TRIM(a.sort_name)) = LOWER(TRIM(?)))
			LIMIT 1
		`
		err = r.pool.Reader.GetContext(ctx, &bookID, query, title, authorName, authorName)
	} else {
		query := `SELECT id FROM books WHERE LOWER(TRIM(title)) = LOWER(TRIM(?)) LIMIT 1`
		err = r.pool.Reader.GetContext(ctx, &bookID, query, title)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return r.GetBookByID(ctx, bookID)
}

