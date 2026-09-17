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

	// 7. Проверка параллельного чтения в WAL-режиме
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
}
