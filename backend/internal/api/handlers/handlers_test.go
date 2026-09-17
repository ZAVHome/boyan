package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"boyan/internal/api"
	"boyan/internal/config"
	"boyan/internal/models"
	"boyan/internal/parsers/cover"
	"boyan/internal/storage"
)

func TestAPIHandlers(t *testing.T) {
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp("", "api_handlers_test_*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	coverDir := filepath.Join(tmpDir, "covers")

	pool, err := storage.NewSQLitePool(ctx, dbPath, 5000, 16000)
	if err != nil {
		t.Fatalf("NewSQLitePool failed: %v", err)
	}
	defer pool.Close()

	if err := storage.RunMigrations(ctx, pool.Writer); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	bookRepo := storage.NewBookRepository(pool)
	coverCache, err := cover.NewCoverCache(coverDir, 10)
	if err != nil {
		t.Fatalf("NewCoverCache failed: %v", err)
	}

	// Создаем тестовую книгу
	testBook := &models.Book{
		ID:    "test-book-id-123",
		Title: "Тестовая книга",
	}
	_ = bookRepo.SaveBook(ctx, testBook, nil, nil, nil, nil)

	cfg := config.DefaultConfig()
	router := api.NewRouter(cfg, pool, bookRepo, coverCache)

	// 1. Проверка /health
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /health, got %d", rec.Code)
	}
	var healthResp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &healthResp)
	if healthResp["status"] != "ok" || healthResp["db"] != "connected" {
		t.Errorf("unexpected health response: %+v", healthResp)
	}

	// 2. Проверка /api/v1/ping
	reqPing := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	recPing := httptest.NewRecorder()
	router.ServeHTTP(recPing, reqPing)

	if recPing.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/v1/ping, got %d", recPing.Code)
	}

	// 3. Проверка /api/v1/books
	reqBooks := httptest.NewRequest(http.MethodGet, "/api/v1/books?page=1&per_page=10", nil)
	recBooks := httptest.NewRecorder()
	router.ServeHTTP(recBooks, reqBooks)

	if recBooks.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/v1/books, got %d", recBooks.Code)
	}
	var booksResp struct {
		Items []models.Book `json:"items"`
		Total int           `json:"total"`
	}
	_ = json.Unmarshal(recBooks.Body.Bytes(), &booksResp)
	if booksResp.Total != 1 || len(booksResp.Items) != 1 {
		t.Errorf("expected 1 book item, got total=%d items=%d", booksResp.Total, len(booksResp.Items))
	}

	// 4. Проверка /api/v1/books/{id}
	reqBook := httptest.NewRequest(http.MethodGet, "/api/v1/books/test-book-id-123", nil)
	recBook := httptest.NewRecorder()
	router.ServeHTTP(recBook, reqBook)

	if recBook.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/v1/books/{id}, got %d", recBook.Code)
	}

	// 5. Проверка несуществующей книги (404)
	req404 := httptest.NewRequest(http.MethodGet, "/api/v1/books/non-existent", nil)
	rec404 := httptest.NewRecorder()
	router.ServeHTTP(rec404, req404)

	if rec404.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing book, got %d", rec404.Code)
	}
}
