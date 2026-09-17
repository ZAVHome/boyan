package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"boyan/internal/models"
	"boyan/internal/storage"

	"github.com/google/uuid"
)

func TestCalculateFileSHA256(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")
	content := []byte("Hello, Boyan Deduplicator!")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	hash, size, err := CalculateFileSHA256(filePath)
	if err != nil {
		t.Fatalf("CalculateFileSHA256 failed: %v", err)
	}
	if size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), size)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}

	// Повторный расчет должен дать тот же хеш
	hash2, _, _ := CalculateFileSHA256(filePath)
	if hash != hash2 {
		t.Errorf("hashes mismatch: %s != %s", hash, hash2)
	}
}

func setupTestStorage(t *testing.T) (*storage.DBPool, *storage.BookRepository) {
	dbPath := filepath.Join(t.TempDir(), "test_dedup.db")
	ctx := context.Background()

	pool, err := storage.NewSQLitePool(ctx, dbPath, 5000, 10000)
	if err != nil {
		t.Fatalf("failed to create sqlite pool: %v", err)
	}

	if err := storage.RunMigrations(ctx, pool.Writer); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	bookRepo := storage.NewBookRepository(pool)
	return pool, bookRepo
}

func TestDeduplicatorScenarios(t *testing.T) {
	pool, bookRepo := setupTestStorage(t)
	defer pool.Close()

	ctx := context.Background()
	dedup := NewDeduplicator(bookRepo)

	// Создаем тестовый файл 1
	tempDir := t.TempDir()
	file1Path := filepath.Join(tempDir, "book1.fb2")
	_ = os.WriteFile(file1Path, []byte("<FictionBook>Content 1</FictionBook>"), 0644)
	hash1, size1, _ := CalculateFileSHA256(file1Path)

	// 1. Проверяем, пока в базе пусто -> DuplicateNone
	res, err := dedup.CheckDuplicate(ctx, file1Path, "Тестовая Книга", "Автор Тестовый", "fb2")
	if err != nil {
		t.Fatalf("CheckDuplicate failed: %v", err)
	}
	if res.Type != DuplicateNone {
		t.Fatalf("expected DuplicateNone, got %v", res.Type)
	}

	// Сохраняем книгу 1 в базу
	book1 := &models.Book{
		ID:    uuid.NewString(),
		Title: "Тестовая Книга",
	}
	authors1 := []models.AuthorDetail{{Author: models.Author{Name: "Автор Тестовый"}}}
	file1 := &models.BookFile{
		ID:        uuid.NewString(),
		BookID:    book1.ID,
		Format:    "fb2",
		FilePath:  "books/book1.fb2",
		FileSize:  size1,
		SHA256:    hash1,
		CreatedAt: time.Now().UTC(),
	}
	if err := bookRepo.SaveBook(ctx, book1, authors1, nil, nil, file1); err != nil {
		t.Fatalf("SaveBook failed: %v", err)
	}

	// 2. Проверяем тот же самый файл -> DuplicateExactHash
	res2, err := dedup.CheckDuplicate(ctx, file1Path, "Другое Название", "Другой Автор", "fb2")
	if err != nil {
		t.Fatalf("CheckDuplicate failed: %v", err)
	}
	if res2.Type != DuplicateExactHash {
		t.Errorf("expected DuplicateExactHash, got %v", res2.Type)
	}

	// 3. Создаем файл с другим содержимым, но с тем же названием, автором и форматом -> DuplicateSameFormat
	file2Path := filepath.Join(tempDir, "book1_v2.fb2")
	_ = os.WriteFile(file2Path, []byte("<FictionBook>Content 2 Different Edition</FictionBook>"), 0644)

	res3, err := dedup.CheckDuplicate(ctx, file2Path, "Тестовая Книга", "Автор Тестовый", "fb2")
	if err != nil {
		t.Fatalf("CheckDuplicate failed: %v", err)
	}
	if res3.Type != DuplicateSameFormat {
		t.Errorf("expected DuplicateSameFormat, got %v", res3.Type)
	}

	// 4. Создаем файл с тем же автором и названием, но новым форматом (epub) -> DuplicateNewFormat
	file3Path := filepath.Join(tempDir, "book1.epub")
	_ = os.WriteFile(file3Path, []byte("EPUB Content"), 0644)

	res4, err := dedup.CheckDuplicate(ctx, file3Path, "Тестовая Книга", "Автор Тестовый", "epub")
	if err != nil {
		t.Fatalf("CheckDuplicate failed: %v", err)
	}
	if res4.Type != DuplicateNewFormat {
		t.Errorf("expected DuplicateNewFormat, got %v", res4.Type)
	}
}
