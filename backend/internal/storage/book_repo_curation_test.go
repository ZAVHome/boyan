package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"boyan/internal/models"
)

func TestBookRepository_CurationAndBatch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "book_curation_test_*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	ctx := context.Background()
	pool, err := NewSQLitePool(ctx, dbPath, 5000, 10000)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer pool.Close()

	if err := RunMigrations(ctx, pool.Writer); err != nil {
		t.Fatalf("migrations: %v", err)
	}

	repo := NewBookRepository(pool)


	// 1. Создаем тестовую книгу
	book := &models.Book{
		Title:     "Old Title",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	author := models.AuthorDetail{
		Author: models.Author{Name: "Old Author"},
		Role:   "author",
		Order:  1,
	}
	series := models.SeriesDetail{
		Series: models.Series{Name: "Old Series"},
		Index:  1.0,
	}
	genre := models.Genre{
		Code:   "sf_space",
		NameRU: "Космическая фантастика",
		NameEN: "Space Sci-Fi",
	}
	file := &models.BookFile{
		Format:   "fb2",
		FilePath: filepath.Join(tempDir, "book.fb2"),
		FileSize: 1024,
		SHA256:   "hash123",
	}

	err = repo.SaveBook(ctx, book, []models.AuthorDetail{author}, []models.SeriesDetail{series}, []models.Genre{genre}, file)
	if err != nil {
		t.Fatalf("save book: %v", err)
	}

	// 2. Проверяем GetBookFiles
	files, err := repo.GetBookFiles(ctx, book.ID)
	if err != nil || len(files) != 1 || files[0].FilePath != file.FilePath {
		t.Fatalf("get book files: %v, files: %+v", err, files)
	}

	// 3. Обновляем метаданные через UpdateBookMetadata
	updateReq := models.UpdateBookMetadataRequest{
		Title:         "New Beautiful Title",
		OriginalTitle: "Original Title Trans",
		Annotation:    "Brand new annotation for test",
		Language:      "ru",
		Publisher:     "Modern Publisher",
		Authors: []models.AuthorInput{
			{Name: "New Author One", Role: "author", Order: 1},
			{Name: "New Author Two", Role: "translator", Order: 2},
		},
		Series: []models.SeriesInput{
			{Name: "New Epic Saga", Index: 2.5},
		},
		Genres: []string{"detectives", "sf_space"},
	}

	err = repo.UpdateBookMetadata(ctx, book.ID, updateReq)
	if err != nil {
		t.Fatalf("update book metadata: %v", err)
	}

	updatedBook, err := repo.GetBookByID(ctx, book.ID)
	if err != nil || updatedBook == nil {
		t.Fatalf("get updated book: %v", err)
	}

	if updatedBook.Title != "New Beautiful Title" || len(updatedBook.Authors) != 2 || len(updatedBook.Series) != 1 || len(updatedBook.Genres) != 2 {
		t.Fatalf("metadata update mismatch: %+v", updatedBook)
	}

	// Проверяем полнотекстовый поиск FTS5 после обновления
	ftsResults, total, err := repo.SearchBooksFTS(ctx, "Beautiful", 0, 10)
	if err != nil || total == 0 || len(ftsResults) == 0 {
		t.Fatalf("expected fts search match for updated title, got total=%d", total)
	}

	// 4. GetStorageStats
	stats, err := repo.GetStorageStats(ctx)
	if err != nil {
		t.Fatalf("get storage stats: %v", err)
	}
	if stats.TotalBooks != 1 || stats.TotalFiles != 1 || stats.FormatCounts["fb2"] != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	// 5. BatchUpdateGenres
	err = repo.BatchUpdateGenres(ctx, []string{book.ID}, "history")
	if err != nil {
		t.Fatalf("batch update genres: %v", err)
	}

	updatedBook, _ = repo.GetBookByID(ctx, book.ID)
	hasHistory := false
	for _, g := range updatedBook.Genres {
		if g.Code == "history" {
			hasHistory = true
			break
		}
	}
	if !hasHistory {
		t.Errorf("expected book to have genre 'history'")
	}

	// 6. BatchUpdateSeries
	err = repo.BatchUpdateSeries(ctx, []string{book.ID}, "Universe Chrono")
	if err != nil {
		t.Fatalf("batch update series: %v", err)
	}

	updatedBook, _ = repo.GetBookByID(ctx, book.ID)
	if len(updatedBook.Series) == 0 || updatedBook.Series[0].Name != "Universe Chrono" {
		t.Errorf("expected book to have series 'Universe Chrono', got: %+v", updatedBook.Series)
	}

	// 7. BatchDeleteBooks
	deletedPaths, err := repo.BatchDeleteBooks(ctx, []string{book.ID})
	if err != nil {
		t.Fatalf("batch delete books: %v", err)
	}
	if len(deletedPaths) != 1 || deletedPaths[0] != file.FilePath {
		t.Errorf("expected deleted path %s, got: %+v", file.FilePath, deletedPaths)
	}

	deletedBook, _ := repo.GetBookByID(ctx, book.ID)
	if deletedBook != nil {
		t.Errorf("expected book to be deleted from db")
	}
}
