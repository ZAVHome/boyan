package calibre

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"boyan/internal/config"
	"boyan/internal/parsers/cover"
	"boyan/internal/storage"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func setupTestCalibreDB(t *testing.T, dir string) {
	t.Helper()

	dbPath := filepath.Join(dir, "metadata.db")
	db, err := sqlx.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to create test metadata.db: %v", err)
	}
	defer db.Close()

	schema := `
		CREATE TABLE books (
			id INTEGER PRIMARY KEY,
			title TEXT NOT NULL,
			sort TEXT,
			timestamp TEXT,
			pubdate TEXT,
			series_index REAL DEFAULT 1.0,
			path TEXT NOT NULL,
			has_cover BOOL DEFAULT 0
		);

		CREATE TABLE authors (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			sort TEXT
		);

		CREATE TABLE books_authors_link (
			id INTEGER PRIMARY KEY,
			book INTEGER NOT NULL,
			author INTEGER NOT NULL
		);

		CREATE TABLE series (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			sort TEXT
		);

		CREATE TABLE books_series_link (
			id INTEGER PRIMARY KEY,
			book INTEGER NOT NULL,
			series INTEGER NOT NULL
		);

		CREATE TABLE tags (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);

		CREATE TABLE books_tags_link (
			id INTEGER PRIMARY KEY,
			book INTEGER NOT NULL,
			tag INTEGER NOT NULL
		);

		CREATE TABLE comments (
			id INTEGER PRIMARY KEY,
			book INTEGER NOT NULL,
			text TEXT
		);

		CREATE TABLE data (
			id INTEGER PRIMARY KEY,
			book INTEGER NOT NULL,
			format TEXT NOT NULL,
			uncompressed_size INTEGER,
			name TEXT NOT NULL
		);

		CREATE TABLE identifiers (
			id INTEGER PRIMARY KEY,
			book INTEGER NOT NULL,
			type TEXT NOT NULL,
			val TEXT NOT NULL
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO books (id, title, sort, series_index, path, has_cover) 
		VALUES (1, 'Мастер и Маргарита', 'Мастер и Маргарита', 1.0, 'Mikhail Bulgakov/Master i Margarita (1)', 1);

		INSERT INTO authors (id, name, sort) VALUES (10, 'Михаил Булгаков', 'Булгаков, Михаил');
		INSERT INTO books_authors_link (book, author) VALUES (1, 10);

		INSERT INTO series (id, name, sort) VALUES (20, 'Классика', 'Классика');
		INSERT INTO books_series_link (book, series) VALUES (1, 20);

		INSERT INTO tags (id, name) VALUES (30, 'Роман'), (31, 'Мистика');
		INSERT INTO books_tags_link (book, tag) VALUES (1, 30), (1, 31);

		INSERT INTO comments (book, text) VALUES (1, '<p>Великий роман <b>Михаила Булгакова</b>.</p>');
		INSERT INTO identifiers (book, type, val) VALUES (1, 'isbn', '978-5-17-090620-8');

		INSERT INTO data (id, book, format, uncompressed_size, name) 
		VALUES (101, 1, 'EPUB', 1024, 'Master i Margarita - Mikhail Bulgakov'),
		       (102, 1, 'FB2', 2048, 'Master i Margarita - Mikhail Bulgakov');
	`)
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// Создаем файлы книги и обложку на диске
	bookDir := filepath.Join(dir, "Mikhail Bulgakov", "Master i Margarita (1)")
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		t.Fatalf("failed to create book directory: %v", err)
	}

	coverPath := filepath.Join(bookDir, "cover.jpg")
	_ = os.WriteFile(coverPath, []byte("fake-jpeg-cover-data"), 0644)

	epubPath := filepath.Join(bookDir, "Master i Margarita - Mikhail Bulgakov.epub")
	_ = os.WriteFile(epubPath, []byte("fake-epub-file-content"), 0644)

	fb2Path := filepath.Join(bookDir, "Master i Margarita - Mikhail Bulgakov.fb2")
	_ = os.WriteFile(fb2Path, []byte("fake-fb2-file-content"), 0644)
}

func TestCalibreImporter(t *testing.T) {
	tempDir := t.TempDir()
	calibreDir := filepath.Join(tempDir, "calibre_library")
	if err := os.MkdirAll(calibreDir, 0755); err != nil {
		t.Fatal(err)
	}

	setupTestCalibreDB(t, calibreDir)

	// Подготовка Boyan storage
	cfg := config.DefaultConfig()
	cfg.Database.SQLite.Path = filepath.Join(tempDir, "boyan_test.db")
	cfg.Storage.LibraryDir = filepath.Join(tempDir, "boyan_library")
	cfg.Metadata.CoverCacheDir = filepath.Join(tempDir, "cover_cache")

	ctx := context.Background()
	pool, err := storage.NewSQLitePool(
		ctx,
		cfg.Database.SQLite.Path,
		cfg.Database.SQLite.BusyTimeoutMS,
		cfg.Database.SQLite.CacheSizeKB,
	)
	if err != nil {
		t.Fatalf("failed to init db pool: %v", err)
	}
	defer pool.Close()

	if err := storage.RunMigrations(ctx, pool.Writer); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	bookRepo := storage.NewBookRepository(pool)
	coverCache, err := cover.NewCoverCache(cfg.Metadata.CoverCacheDir, cfg.Metadata.CoverCacheMaxMB)
	if err != nil {
		t.Fatalf("failed to init cover cache: %v", err)
	}
	importer := NewImporter(cfg, bookRepo, coverCache)

	// 1. Первый импорт (копирование файлов)
	stats, err := importer.ImportLibrary(ctx, calibreDir, ImportOptions{CopyFiles: true})
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}

	if stats.TotalCalibreBooks != 1 {
		t.Errorf("expected 1 total calibre book, got %d", stats.TotalCalibreBooks)
	}
	if stats.ImportedBooks != 1 {
		t.Errorf("expected 1 imported book, got %d", stats.ImportedBooks)
	}
	if len(stats.Errors) > 0 {
		t.Errorf("unexpected import errors: %v", stats.Errors)
	}

	// Проверяем, что книга есть в репозитории Бояна
	books, total, err := bookRepo.ListBooks(ctx, 0, 10)
	if err != nil {
		t.Fatalf("failed to list books: %v", err)
	}
	if total != 1 || len(books) != 1 {
		t.Fatalf("expected 1 book in Boyan, got total %d, items %d", total, len(books))
	}

	importedBook := books[0]
	if importedBook.Title != "Мастер и Маргарита" {
		t.Errorf("expected title 'Мастер и Маргарита', got '%s'", importedBook.Title)
	}
	if importedBook.Annotation != "Великий роман Михаила Булгакова." {
		t.Errorf("expected cleaned annotation, got '%s'", importedBook.Annotation)
	}
	if !importedBook.CoverCached {
		t.Errorf("expected cover to be cached")
	}

	// Проверяем форматы
	fullBook, err := bookRepo.GetBookByID(ctx, importedBook.ID)
	if err != nil {
		t.Fatalf("failed to get full book: %v", err)
	}
	if len(fullBook.Files) != 2 {
		t.Errorf("expected 2 formats (epub, fb2), got %d", len(fullBook.Files))
	}

	// 2. Повторный импорт (проверка идемпотентности и пропуска уже импортированных книг/форматов)
	stats2, err := importer.ImportLibrary(ctx, calibreDir, ImportOptions{CopyFiles: true})
	if err != nil {
		t.Fatalf("second import failed: %v", err)
	}
	if stats2.ImportedBooks != 0 {
		t.Errorf("expected 0 newly imported books on second run, got %d", stats2.ImportedBooks)
	}
}
