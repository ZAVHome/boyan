package storage_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"boyan/internal/models"
	"boyan/internal/storage"
)

func TestStorage_FullFlow(t *testing.T) {
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp("", "sqlite_test_*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// 1. Инициализация пула SQLite
	pool, err := storage.NewSQLitePool(ctx, dbPath, 5000, 16000)
	if err != nil {
		t.Fatalf("NewSQLitePool failed: %v", err)
	}
	defer pool.Close()

	// 2. Применение миграций
	if err := storage.RunMigrations(ctx, pool.Writer); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// 3. Тест репозитория пользователей и создания admin
	userRepo := storage.NewUserRepository(pool)
	if err := userRepo.EnsureAdminUser(ctx, "admin", "secretpass123"); err != nil {
		t.Fatalf("EnsureAdminUser failed: %v", err)
	}

	adminUser, err := userRepo.GetByUsername(ctx, "admin")
	if err != nil {
		t.Fatalf("GetByUsername failed: %v", err)
	}
	if adminUser == nil || adminUser.Role != models.RoleAdmin {
		t.Fatalf("invalid admin user: %+v", adminUser)
	}
	if !userRepo.VerifyPassword(adminUser, "secretpass123") {
		t.Errorf("expected password verification to succeed")
	}
	if userRepo.VerifyPassword(adminUser, "wrongpass") {
		t.Errorf("expected password verification to fail for wrong password")
	}

	// 4. Тест сохранения и поиска книг через FTS5
	bookRepo := storage.NewBookRepository(pool)

	book1 := &models.Book{
		ID:            "book-uuid-1",
		Title:         "Мастер и Маргарита",
		OriginalTitle: "The Master and Margarita",
		Annotation:    "Культовый роман о визите дьявола в Москву.",
		Language:      "ru",
		PublishedDate: "1967",
		Publisher:     "Художественная литература",
	}
	authors1 := []models.AuthorDetail{
		{
			Author: models.Author{Name: "Михаил Афанасьевич Булгаков", SortName: "Булгаков, Михаил"},
			Role:   "author",
			Order:  1,
		},
	}
	genres1 := []models.Genre{
		{Code: "prose_classic", NameRU: "Классическая проза", NameEN: "Classic Prose", CategoryRU: "Проза", CategoryEN: "Prose"},
	}
	file1 := &models.BookFile{
		Format:   "fb2",
		FilePath: "books/master.fb2",
		FileSize: 102400,
		SHA256:   "abcd1234efgh5678",
	}

	err = bookRepo.SaveBook(ctx, book1, authors1, nil, genres1, file1)
	if err != nil {
		t.Fatalf("SaveBook 1 failed: %v", err)
	}

	book2 := &models.Book{
		ID:         "book-uuid-2",
		Title:      "Пикник на обочине",
		Annotation: "Фантастическая повесть братьев Стругацких о Зоне и сталкерах.",
		Language:   "ru",
	}
	authors2 := []models.AuthorDetail{
		{Author: models.Author{Name: "Аркадий Стругацкий"}, Role: "author", Order: 1},
		{Author: models.Author{Name: "Борис Стругацкий"}, Role: "author", Order: 2},
	}
	series2 := []models.SeriesDetail{
		{Series: models.Series{Name: "Мир Полудня"}, Index: 4.0},
	}

	err = bookRepo.SaveBook(ctx, book2, authors2, series2, nil, nil)
	if err != nil {
		t.Fatalf("SaveBook 2 failed: %v", err)
	}

	// 5. Проверка выборки книги по ID
	loadedBook1, err := bookRepo.GetBookByID(ctx, "book-uuid-1")
	if err != nil {
		t.Fatalf("GetBookByID failed: %v", err)
	}
	if loadedBook1 == nil || loadedBook1.Title != "Мастер и Маргарита" {
		t.Fatalf("loaded book mismatch: %+v", loadedBook1)
	}
	if len(loadedBook1.Authors) != 1 || loadedBook1.Authors[0].Name != "Михаил Афанасьевич Булгаков" {
		t.Fatalf("loaded book authors mismatch: %+v", loadedBook1.Authors)
	}
	if len(loadedBook1.Files) != 1 || loadedBook1.Files[0].Format != "fb2" {
		t.Fatalf("loaded book files mismatch: %+v", loadedBook1.Files)
	}

	// 6. Проверка полнотекстового поиска FTS5
	// Поиск по автору
	results, total, err := bookRepo.SearchBooksFTS(ctx, "Булгаков", 0, 10)
	if err != nil {
		t.Fatalf("SearchBooksFTS 'Булгаков' failed: %v", err)
	}
	if total != 1 || len(results) != 1 || results[0].ID != "book-uuid-1" {
		t.Errorf("expected 1 result for 'Булгаков', got %d", total)
	}

	// Поиск по слову из аннотации ("сталкер")
	results, total, err = bookRepo.SearchBooksFTS(ctx, "сталкер", 0, 10)
	if err != nil {
		t.Fatalf("SearchBooksFTS 'сталкер' failed: %v", err)
	}
	if total != 1 || len(results) != 1 || results[0].ID != "book-uuid-2" {
		t.Errorf("expected 1 result for 'сталкер', got %d", total)
	}

	// Поиск по названию серии ("Мир")
	results, total, err = bookRepo.SearchBooksFTS(ctx, "Мир", 0, 10)
	if err != nil {
		t.Fatalf("SearchBooksFTS 'Мир' failed: %v", err)
	}
	if total != 1 || len(results) != 1 || results[0].ID != "book-uuid-2" {
		t.Errorf("expected 1 result for 'Мир', got %d", total)
	}

	// 7. Проверка авторов и серий
	authorsList, aTotal, err := bookRepo.ListAuthors(ctx, "", "", 0, 10)
	if err != nil {
		t.Fatalf("ListAuthors failed: %v", err)
	}
	if aTotal < 2 || len(authorsList) < 2 {
		t.Errorf("expected at least 2 authors, got total=%d, len=%d", aTotal, len(authorsList))
	}

	authorsAlphabet, err := bookRepo.GetAuthorsAlphabet(ctx)
	if err != nil {
		t.Fatalf("GetAuthorsAlphabet failed: %v", err)
	}
	if len(authorsAlphabet) == 0 {
		t.Errorf("expected non-empty authors alphabet")
	}

	seriesList, sTotal, err := bookRepo.ListSeries(ctx, "", "", 0, 10)
	if err != nil {
		t.Fatalf("ListSeries failed: %v", err)
	}
	if sTotal != 1 || len(seriesList) != 1 {
		t.Errorf("expected 1 series, got total=%d", sTotal)
	}

	// 8. Проверка сортировки книг
	booksSortedByTitle, _, err := bookRepo.ListBooksSorted(ctx, 0, 10, "title", "asc")
	if err != nil {
		t.Fatalf("ListBooksSorted by title failed: %v", err)
	}
	if len(booksSortedByTitle) != 2 {
		t.Errorf("expected 2 books, got %d", len(booksSortedByTitle))
	}

	booksSortedBySeries, _, err := bookRepo.ListBooksSorted(ctx, 0, 10, "series", "asc")
	if err != nil {
		t.Fatalf("ListBooksSorted by series failed: %v", err)
	}
	if len(booksSortedBySeries) != 2 {
		t.Errorf("expected 2 books, got %d", len(booksSortedBySeries))
	}
	// Книга с серией ("book-uuid-2" имеет серию "Мир") должна быть первой
	if len(booksSortedBySeries[0].Series) == 0 {
		t.Errorf("expected book with series to be first, got: %s", booksSortedBySeries[0].Title)
	}

	booksSortedByYear, _, err := bookRepo.ListBooksSorted(ctx, 0, 10, "year", "desc")
	if err != nil {
		t.Fatalf("ListBooksSorted by year failed: %v", err)
	}
	if len(booksSortedByYear) != 2 {
		t.Errorf("expected 2 books, got %d", len(booksSortedByYear))
	}

	// 9. Проверка параллельного чтения в WAL-режиме
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				_, _, _ = bookRepo.ListBooks(ctx, 0, 5)
				time.Sleep(5 * time.Millisecond)
			}
			done <- true
		}()
	}
	for i := 0; i < 5; i++ {
		<-done
	}

	// 10. Проверка поиска с фильтрацией (SearchBooksWithFilter)
	// Фильтр по жанру
	booksByGenre, gTotal, err := bookRepo.SearchBooksWithFilter(ctx, storage.BookFilter{
		Genre: "prose_classic",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("SearchBooksWithFilter by genre failed: %v", err)
	}
	if gTotal != 1 || len(booksByGenre) != 1 || booksByGenre[0].ID != "book-uuid-1" {
		t.Errorf("expected 1 book for genre 'prose_classic', got %d", gTotal)
	}

	// Фильтр по издательству
	booksByPub, pTotal, err := bookRepo.SearchBooksWithFilter(ctx, storage.BookFilter{
		Publisher: "Художественная",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("SearchBooksWithFilter by publisher failed: %v", err)
	}
	if pTotal != 1 || len(booksByPub) != 1 || booksByPub[0].ID != "book-uuid-1" {
		t.Errorf("expected 1 book for publisher 'Художественная', got %d", pTotal)
	}

	// Фильтр по году издания
	booksByYear, yTotal, err := bookRepo.SearchBooksWithFilter(ctx, storage.BookFilter{
		Year:  "1967",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("SearchBooksWithFilter by year failed: %v", err)
	}
	if yTotal != 1 || len(booksByYear) != 1 || booksByYear[0].ID != "book-uuid-1" {
		t.Errorf("expected 1 book for year '1967', got %d", yTotal)
	}

	// Фильтр по языку
	booksByLang, lTotal, err := bookRepo.SearchBooksWithFilter(ctx, storage.BookFilter{
		Language: "ru",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("SearchBooksWithFilter by language failed: %v", err)
	}
	if lTotal != 2 || len(booksByLang) != 2 {
		t.Errorf("expected 2 books for language 'ru', got %d", lTotal)
	}

	// Комбинированный фильтр: FTS-запрос + фильтр по жанру
	combinedBooks, cTotal, err := bookRepo.SearchBooksWithFilter(ctx, storage.BookFilter{
		Query: "Булгаков",
		Genre: "prose_classic",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("SearchBooksWithFilter combined failed: %v", err)
	}
	if cTotal != 1 || len(combinedBooks) != 1 || combinedBooks[0].ID != "book-uuid-1" {
		t.Errorf("expected 1 book for combined query and genre filter, got %d", cTotal)
	}

	// Несуществующий год издания
	noBooks, nTotal, err := bookRepo.SearchBooksWithFilter(ctx, storage.BookFilter{
		Year:  "1812",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("SearchBooksWithFilter non-matching year failed: %v", err)
	}
	if nTotal != 0 || len(noBooks) != 0 {
		t.Errorf("expected 0 books for year '1812', got %d", nTotal)
	}
}
