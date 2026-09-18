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
	return r.ListBooksSorted(ctx, offset, limit, "recent")
}

// buildBookOrderClause формирует SQL-выражение сортировки книг.
func buildBookOrderClause(sort, direction string) string {
	sort = strings.TrimSpace(strings.ToLower(sort))
	direction = strings.TrimSpace(strings.ToLower(direction))

	dir := "ASC"
	if direction == "desc" || (direction == "" && (sort == "recent" || sort == "year")) {
		dir = "DESC"
	} else if direction == "asc" {
		dir = "ASC"
	}

	switch sort {
	case "title":
		return fmt.Sprintf("ORDER BY b.title COLLATE NOCASE %s", dir)
	case "author":
		return fmt.Sprintf(`ORDER BY 
			CASE WHEN (SELECT a.id FROM authors a JOIN book_authors ba ON a.id = ba.author_id WHERE ba.book_id = b.id ORDER BY ba.author_order ASC LIMIT 1) IS NOT NULL THEN 0 ELSE 1 END,
			(SELECT COALESCE(NULLIF(a.sort_name, ''), a.name) FROM authors a JOIN book_authors ba ON a.id = ba.author_id WHERE ba.book_id = b.id ORDER BY ba.author_order ASC LIMIT 1) COLLATE NOCASE %s,
			b.title COLLATE NOCASE ASC`, dir)
	case "series":
		return fmt.Sprintf(`ORDER BY 
			CASE WHEN (SELECT s.id FROM series s JOIN book_series bs ON s.id = bs.series_id WHERE bs.book_id = b.id LIMIT 1) IS NOT NULL THEN 0 ELSE 1 END,
			(SELECT COALESCE(NULLIF(s.name, ''), s.sort_name) FROM series s JOIN book_series bs ON s.id = bs.series_id WHERE bs.book_id = b.id LIMIT 1) COLLATE NOCASE %s,
			(SELECT bs.series_index FROM book_series bs WHERE bs.book_id = b.id LIMIT 1) ASC,
			b.title COLLATE NOCASE ASC`, dir)
	case "year":
		return fmt.Sprintf(`ORDER BY 
			CASE WHEN b.published_date IS NOT NULL AND TRIM(b.published_date) != '' THEN 0 ELSE 1 END,
			b.published_date %s,
			b.title COLLATE NOCASE ASC`, dir)
	case "recent":
		fallthrough
	default:
		return fmt.Sprintf("ORDER BY b.created_at %s", dir)
	}
}

// ListBooksSorted возвращает список книг с пагинацией и заданной сортировкой.
func (r *BookRepository) ListBooksSorted(ctx context.Context, offset, limit int, sort string, direction ...string) ([]models.Book, int, error) {
	var total int
	err := r.pool.Reader.GetContext(ctx, &total, `SELECT COUNT(*) FROM books`)
	if err != nil {
		return nil, 0, fmt.Errorf("count books: %w", err)
	}

	dir := ""
	if len(direction) > 0 {
		dir = direction[0]
	}
	orderClause := buildBookOrderClause(sort, dir)

	var books []models.Book
	query := fmt.Sprintf(`SELECT b.* FROM books b %s LIMIT ? OFFSET ?`, orderClause)
	err = r.pool.Reader.SelectContext(ctx, &books, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list books: %w", err)
	}

	r.enrichBooksWithDetails(ctx, books)
	return books, total, nil
}

// BookFilter задает параметры фильтрации, поиска и пагинации книг.
type BookFilter struct {
	Query     string
	Genre     string
	Publisher string
	Year      string
	Language  string
	Sort      string
	Direction string
	Offset    int
	Limit     int
}

// SearchBooksWithFilter выполняет фильтрацию и поиск книг по различным критериям.
func (r *BookRepository) SearchBooksWithFilter(ctx context.Context, filter BookFilter) ([]models.Book, int, error) {
	query := strings.TrimSpace(filter.Query)
	genre := strings.TrimSpace(filter.Genre)
	publisher := strings.TrimSpace(filter.Publisher)
	year := strings.TrimSpace(filter.Year)
	language := strings.TrimSpace(filter.Language)

	if query == "" && genre == "" && publisher == "" && year == "" && language == "" {
		return r.ListBooksSorted(ctx, filter.Offset, filter.Limit, filter.Sort, filter.Direction)
	}

	var joins []string
	var whereClauses []string
	var args []any

	if query != "" {
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
			return []models.Book{}, 0, nil
		}
		joins = append(joins, "JOIN books_fts fts ON b.id = fts.book_id")
		whereClauses = append(whereClauses, "books_fts MATCH ?")
		args = append(args, strings.Join(formattedTerms, " "))
	}

	if genre != "" {
		joins = append(joins, "JOIN book_genres bg ON b.id = bg.book_id")
		whereClauses = append(whereClauses, "bg.genre_code = ?")
		args = append(args, genre)
	}

	if publisher != "" {
		whereClauses = append(whereClauses, "b.publisher LIKE ?")
		args = append(args, "%"+publisher+"%")
	}

	if language != "" {
		whereClauses = append(whereClauses, "b.language = ?")
		args = append(args, language)
	}

	if year != "" {
		whereClauses = append(whereClauses, "b.published_date LIKE ?")
		args = append(args, "%"+year+"%")
	}

	if len(whereClauses) == 0 {
		return r.ListBooksSorted(ctx, filter.Offset, filter.Limit, filter.Sort, filter.Direction)
	}

	joinSQL := strings.Join(joins, " ")
	whereSQL := strings.Join(whereClauses, " AND ")

	countSQL := fmt.Sprintf(`
		SELECT COUNT(DISTINCT b.id)
		FROM books b
		%s
		WHERE %s
	`, joinSQL, whereSQL)

	var total int
	err := r.pool.Reader.GetContext(ctx, &total, countSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("count books with filter: %w", err)
	}
	if total == 0 {
		return []models.Book{}, 0, nil
	}

	orderClause := "ORDER BY b.created_at DESC"
	if query != "" && (filter.Sort == "" || filter.Sort == "relevance") {
		orderClause = "ORDER BY bm25(books_fts)"
	} else {
		orderClause = buildBookOrderClause(filter.Sort, filter.Direction)
	}

	selectSQL := fmt.Sprintf(`
		SELECT DISTINCT b.*
		FROM books b
		%s
		WHERE %s
		%s
		LIMIT ? OFFSET ?
	`, joinSQL, whereSQL, orderClause)

	selectArgs := make([]any, len(args), len(args)+2)
	copy(selectArgs, args)
	selectArgs = append(selectArgs, filter.Limit, filter.Offset)

	var books []models.Book
	err = r.pool.Reader.SelectContext(ctx, &books, selectSQL, selectArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("select books with filter: %w", err)
	}

	r.enrichBooksWithDetails(ctx, books)
	return books, total, nil
}

// SearchBooksFTS выполняет полнотекстовый поиск по индексу FTS5.
func (r *BookRepository) SearchBooksFTS(ctx context.Context, query string, offset, limit int) ([]models.Book, int, error) {
	return r.SearchBooksFTSSorted(ctx, query, offset, limit, "recent", "desc")
}

// SearchBooksFTSSorted выполняет полнотекстовый поиск с указанным порядком сортировки.
func (r *BookRepository) SearchBooksFTSSorted(ctx context.Context, query string, offset, limit int, sort string, direction ...string) ([]models.Book, int, error) {
	dir := ""
	if len(direction) > 0 {
		dir = direction[0]
	}
	return r.SearchBooksWithFilter(ctx, BookFilter{
		Query:     query,
		Sort:      sort,
		Direction: dir,
		Offset:    offset,
		Limit:     limit,
	})
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
		SELECT UPPER(SUBSTR(COALESCE(NULLIF(sort_name, ''), name), 1, 1)) AS letter, COUNT(*) AS count
		FROM authors
		WHERE (sort_name IS NOT NULL AND sort_name != '') OR (name IS NOT NULL AND name != '')
		GROUP BY letter
		ORDER BY letter ASC
	`)
	return list, err
}

// ListAuthors возвращает список авторов с фильтрацией по букве или поисковой строке.
func (r *BookRepository) ListAuthors(ctx context.Context, letter, query string, offset, limit int) ([]AuthorWithCount, int, error) {
	letter = strings.TrimSpace(letter)
	query = strings.TrimSpace(query)

	var whereClauses []string
	var countArgs []any
	var selectArgs []any

	if query != "" {
		whereClauses = append(whereClauses, "(a.name LIKE ? OR a.sort_name LIKE ?)")
		searchTerm := "%" + query + "%"
		countArgs = append(countArgs, searchTerm, searchTerm)
		selectArgs = append(selectArgs, searchTerm, searchTerm)
	} else if letter != "" {
		upperLetter := strings.ToUpper(letter)
		lowerLetter := strings.ToLower(letter)
		whereClauses = append(whereClauses, "SUBSTR(COALESCE(NULLIF(a.sort_name, ''), a.name), 1, 1) IN (?, ?)")
		countArgs = append(countArgs, upperLetter, lowerLetter)
		selectArgs = append(selectArgs, upperLetter, lowerLetter)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM authors a %s`, whereSQL)
	var total int
	err := r.pool.Reader.GetContext(ctx, &total, countSQL, countArgs...)
	if err != nil {
		return nil, 0, err
	}

	selectSQL := fmt.Sprintf(`
		SELECT a.id, a.name, a.sort_name, COUNT(ba.book_id) AS book_count
		FROM authors a
		LEFT JOIN book_authors ba ON a.id = ba.author_id
		%s
		GROUP BY a.id, a.name, a.sort_name
		ORDER BY COALESCE(NULLIF(a.sort_name, ''), a.name) COLLATE NOCASE ASC
		LIMIT ? OFFSET ?
	`, whereSQL)

	selectArgs = append(selectArgs, limit, offset)
	var authors []AuthorWithCount
	err = r.pool.Reader.SelectContext(ctx, &authors, selectSQL, selectArgs...)
	return authors, total, err
}

// GetAuthorsByLetter возвращает список авторов на заданную букву.
func (r *BookRepository) GetAuthorsByLetter(ctx context.Context, letter string, offset, limit int) ([]AuthorWithCount, int, error) {
	return r.ListAuthors(ctx, letter, "", offset, limit)
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
		SELECT UPPER(SUBSTR(COALESCE(NULLIF(name, ''), sort_name), 1, 1)) AS letter, COUNT(*) AS count
		FROM series
		WHERE (name IS NOT NULL AND name != '') OR (sort_name IS NOT NULL AND sort_name != '')
		GROUP BY letter
		ORDER BY letter ASC
	`)
	return list, err
}

// ListSeries возвращает список серий с фильтрацией по букве или поисковой строке.
func (r *BookRepository) ListSeries(ctx context.Context, letter, query string, offset, limit int) ([]SeriesWithCount, int, error) {
	letter = strings.TrimSpace(letter)
	query = strings.TrimSpace(query)

	var whereClauses []string
	var countArgs []any
	var selectArgs []any

	if query != "" {
		whereClauses = append(whereClauses, "(s.name LIKE ? OR s.sort_name LIKE ?)")
		searchTerm := "%" + query + "%"
		countArgs = append(countArgs, searchTerm, searchTerm)
		selectArgs = append(selectArgs, searchTerm, searchTerm)
	} else if letter != "" {
		upperLetter := strings.ToUpper(letter)
		lowerLetter := strings.ToLower(letter)
		whereClauses = append(whereClauses, "SUBSTR(COALESCE(NULLIF(s.name, ''), s.sort_name), 1, 1) IN (?, ?)")
		countArgs = append(countArgs, upperLetter, lowerLetter)
		selectArgs = append(selectArgs, upperLetter, lowerLetter)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM series s %s`, whereSQL)
	var total int
	err := r.pool.Reader.GetContext(ctx, &total, countSQL, countArgs...)
	if err != nil {
		return nil, 0, err
	}

	selectSQL := fmt.Sprintf(`
		SELECT s.id, s.name, s.sort_name, COUNT(bs.book_id) AS book_count
		FROM series s
		LEFT JOIN book_series bs ON s.id = bs.series_id
		%s
		GROUP BY s.id, s.name, s.sort_name
		ORDER BY COALESCE(NULLIF(s.name, ''), s.sort_name) COLLATE NOCASE ASC
		LIMIT ? OFFSET ?
	`, whereSQL)

	selectArgs = append(selectArgs, limit, offset)
	var series []SeriesWithCount
	err = r.pool.Reader.SelectContext(ctx, &series, selectSQL, selectArgs...)
	return series, total, err
}

// GetSeriesByLetter возвращает серии на заданную букву.
func (r *BookRepository) GetSeriesByLetter(ctx context.Context, letter string, offset, limit int) ([]SeriesWithCount, int, error) {
	return r.ListSeries(ctx, letter, "", offset, limit)
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

// GetAllFB2Files возвращает все файлы формата FB2 из библиотеки для сервисных задач и ремонта.
func (r *BookRepository) GetAllFB2Files(ctx context.Context) ([]models.BookFile, error) {
	var files []models.BookFile
	err := r.pool.Reader.SelectContext(ctx, &files, `
		SELECT * FROM book_files
		WHERE LOWER(format) = 'fb2' OR LOWER(file_path) LIKE '%.fb2'
		ORDER BY created_at ASC
	`)
	return files, err
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

// GetBookFiles возвращает список всех файлов книги.
func (r *BookRepository) GetBookFiles(ctx context.Context, bookID string) ([]models.BookFile, error) {
	var files []models.BookFile
	err := r.pool.Reader.SelectContext(ctx, &files, `SELECT * FROM book_files WHERE book_id = ?`, bookID)
	if err != nil {
		return nil, fmt.Errorf("get book files: %w", err)
	}
	return files, nil
}

// UpdateBookMetadata обновляет метаданные книги и ее связи в единой транзакции.
func (r *BookRepository) UpdateBookMetadata(ctx context.Context, bookID string, req models.UpdateBookMetadataRequest) error {
	tx, err := r.pool.Writer.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	// 1. Обновление таблицы books
	res, err := tx.ExecContext(ctx, `
		UPDATE books SET 
			title = ?, original_title = ?, annotation = ?, language = ?, 
			publisher = ?, published_date = ?, isbn = ?, updated_at = ?
		WHERE id = ?
	`, req.Title, req.OriginalTitle, req.Annotation, req.Language,
		req.Publisher, req.PublishedDate, req.ISBN, now, bookID)
	if err != nil {
		return fmt.Errorf("update book table: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}

	// 2. Обновление авторов
	_, err = tx.ExecContext(ctx, `DELETE FROM book_authors WHERE book_id = ?`, bookID)
	if err != nil {
		return fmt.Errorf("delete book authors: %w", err)
	}

	var authorNames []string
	for i, a := range req.Authors {
		trimmed := strings.TrimSpace(a.Name)
		if trimmed == "" {
			continue
		}
		authorNames = append(authorNames, trimmed)

		var authorID string
		err = tx.GetContext(ctx, &authorID, `SELECT id FROM authors WHERE name = ? LIMIT 1`, trimmed)
		if err != nil {
			if err == sql.ErrNoRows {
				authorID = uuid.NewString()
				_, err = tx.ExecContext(ctx, `
					INSERT INTO authors (id, name, sort_name) VALUES (?, ?, ?)
				`, authorID, trimmed, trimmed)
				if err != nil {
					return fmt.Errorf("insert author %s: %w", trimmed, err)
				}
			} else {
				return fmt.Errorf("lookup author: %w", err)
			}
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
		`, bookID, authorID, role, order)
		if err != nil {
			return fmt.Errorf("link book author: %w", err)
		}
	}

	// 3. Обновление серий
	_, err = tx.ExecContext(ctx, `DELETE FROM book_series WHERE book_id = ?`, bookID)
	if err != nil {
		return fmt.Errorf("delete book series: %w", err)
	}

	var seriesNames []string
	for _, s := range req.Series {
		trimmed := strings.TrimSpace(s.Name)
		if trimmed == "" {
			continue
		}
		seriesNames = append(seriesNames, trimmed)

		var seriesID string
		err = tx.GetContext(ctx, &seriesID, `SELECT id FROM series WHERE name = ? LIMIT 1`, trimmed)
		if err != nil {
			if err == sql.ErrNoRows {
				seriesID = uuid.NewString()
				_, err = tx.ExecContext(ctx, `
					INSERT INTO series (id, name, sort_name) VALUES (?, ?, ?)
				`, seriesID, trimmed, trimmed)
				if err != nil {
					return fmt.Errorf("insert series %s: %w", trimmed, err)
				}
			} else {
				return fmt.Errorf("lookup series: %w", err)
			}
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO book_series (book_id, series_id, series_index)
			VALUES (?, ?, ?)
		`, bookID, seriesID, s.Index)
		if err != nil {
			return fmt.Errorf("link book series: %w", err)
		}
	}

	// 4. Обновление жанров
	_, err = tx.ExecContext(ctx, `DELETE FROM book_genres WHERE book_id = ?`, bookID)
	if err != nil {
		return fmt.Errorf("delete book genres: %w", err)
	}

	for _, gCode := range req.Genres {
		code := strings.TrimSpace(gCode)
		if code == "" {
			continue
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO genres (code, name_ru, name_en, category_ru, category_en)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(code) DO NOTHING
		`, code, code, code, "Прочее", "Other")
		if err != nil {
			return fmt.Errorf("ensure genre %s: %w", code, err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO book_genres (book_id, genre_code)
			VALUES (?, ?)
			ON CONFLICT(book_id, genre_code) DO NOTHING
		`, bookID, code)
		if err != nil {
			return fmt.Errorf("link book genre: %w", err)
		}
	}

	// 5. Обновление FTS5
	_, _ = tx.ExecContext(ctx, `DELETE FROM books_fts WHERE book_id = ?`, bookID)
	authorsStr := strings.Join(authorNames, " ")
	seriesStr := strings.Join(seriesNames, " ")

	_, err = tx.ExecContext(ctx, `
		INSERT INTO books_fts (book_id, title, original_title, annotation, author_names, series_names)
		VALUES (?, ?, ?, ?, ?, ?)
	`, bookID, req.Title, req.OriginalTitle, req.Annotation, authorsStr, seriesStr)
	if err != nil {
		return fmt.Errorf("update fts: %w", err)
	}

	return tx.Commit()
}

// BatchDeleteBooks удаляет массив книг из БД и возвращает список путей к их файлам.
func (r *BookRepository) BatchDeleteBooks(ctx context.Context, bookIDs []string) ([]string, error) {
	if len(bookIDs) == 0 {
		return nil, nil
	}

	var filePaths []string
	query := fmt.Sprintf(`SELECT file_path FROM book_files WHERE book_id IN (%s)`, joinStringsWithComma(bookIDs))
	_ = r.pool.Reader.SelectContext(ctx, &filePaths, query)

	tx, err := r.pool.Writer.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	inClause := joinStringsWithComma(bookIDs)
	_, _ = tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM books_fts WHERE book_id IN (%s)`, inClause))
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM books WHERE id IN (%s)`, inClause))
	if err != nil {
		return nil, fmt.Errorf("delete books: %w", err)
	}

	return filePaths, tx.Commit()
}

// BatchUpdateGenres добавляет указанный жанр ко всем выбранным книгам.
func (r *BookRepository) BatchUpdateGenres(ctx context.Context, bookIDs []string, genreCode string) error {
	genreCode = strings.TrimSpace(genreCode)
	if genreCode == "" || len(bookIDs) == 0 {
		return nil
	}

	tx, err := r.pool.Writer.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO genres (code, name_ru, name_en, category_ru, category_en)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(code) DO NOTHING
	`, genreCode, genreCode, genreCode, "Прочее", "Other")
	if err != nil {
		return err
	}

	for _, bookID := range bookIDs {
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO book_genres (book_id, genre_code)
			VALUES (?, ?)
			ON CONFLICT(book_id, genre_code) DO NOTHING
		`, bookID, genreCode)
	}

	return tx.Commit()
}

// BatchUpdateSeries назначает серию выбранным книгам.
func (r *BookRepository) BatchUpdateSeries(ctx context.Context, bookIDs []string, seriesName string) error {
	seriesName = strings.TrimSpace(seriesName)
	if seriesName == "" || len(bookIDs) == 0 {
		return nil
	}

	tx, err := r.pool.Writer.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var seriesID string
	err = tx.GetContext(ctx, &seriesID, `SELECT id FROM series WHERE name = ? LIMIT 1`, seriesName)
	if err != nil {
		if err == sql.ErrNoRows {
			seriesID = uuid.NewString()
			_, err = tx.ExecContext(ctx, `
				INSERT INTO series (id, name, sort_name) VALUES (?, ?, ?)
			`, seriesID, seriesName, seriesName)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	for idx, bookID := range bookIDs {
		_, _ = tx.ExecContext(ctx, `DELETE FROM book_series WHERE book_id = ?`, bookID)
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO book_series (book_id, series_id, series_index)
			VALUES (?, ?, ?)
		`, bookID, seriesID, float64(idx+1))
	}

	return tx.Commit()
}

// StorageStats возвращает сводную статистику хранилища и форматов.
type StorageStats struct {
	TotalBooks   int            `json:"total_books"`
	TotalAuthors int            `json:"total_authors"`
	TotalSeries  int            `json:"total_series"`
	TotalFiles   int            `json:"total_files"`
	TotalBytes   int64          `json:"total_bytes"`
	FormatCounts map[string]int `json:"format_counts"`
}

// GetStorageStats собирает сводную статистику по книгам, авторам, сериям, файлам и форматам.
func (r *BookRepository) GetStorageStats(ctx context.Context) (*StorageStats, error) {
	stats := &StorageStats{
		FormatCounts: make(map[string]int),
	}

	_ = r.pool.Reader.GetContext(ctx, &stats.TotalBooks, `SELECT COUNT(*) FROM books`)
	_ = r.pool.Reader.GetContext(ctx, &stats.TotalAuthors, `SELECT COUNT(*) FROM authors`)
	_ = r.pool.Reader.GetContext(ctx, &stats.TotalSeries, `SELECT COUNT(*) FROM series`)
	_ = r.pool.Reader.GetContext(ctx, &stats.TotalFiles, `SELECT COUNT(*) FROM book_files`)
	_ = r.pool.Reader.GetContext(ctx, &stats.TotalBytes, `SELECT COALESCE(SUM(file_size), 0) FROM book_files`)

	rows, err := r.pool.Reader.QueryxContext(ctx, `SELECT format, COUNT(*) as cnt FROM book_files GROUP BY format`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var format string
			var cnt int
			if err := rows.Scan(&format, &cnt); err == nil {
				stats.FormatCounts[format] = cnt
			}
		}
	}

	return stats, nil
}

func joinStringsWithComma(items []string) string {
	var quoted []string
	for _, item := range items {
		clean := strings.ReplaceAll(item, "'", "''")
		quoted = append(quoted, fmt.Sprintf("'%s'", clean))
	}
	return strings.Join(quoted, ",")
}


