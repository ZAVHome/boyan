package v2_test

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
	opdsv2 "boyan/internal/opds/v2"
	"boyan/internal/parsers/cover"
	"boyan/internal/storage"
)

func TestOPDSv2_CatalogAndSearch(t *testing.T) {
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp("", "opds2_test_*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	coverDir := filepath.Join(tmpDir, "covers")

	pool, err := storage.NewSQLitePool(ctx, dbPath, 5000, 16000)
	if err != nil {
		t.Fatalf("NewSQLitePool: %v", err)
	}
	defer pool.Close()

	_ = storage.RunMigrations(ctx, pool.Writer)

	bookRepo := storage.NewBookRepository(pool)
	userRepo := storage.NewUserRepository(pool)
	coverCache, _ := cover.NewCoverCache(coverDir, 10)

	cfg := config.DefaultConfig()
	cfg.OPDS.AllowAnonymousReading = true
	router := api.NewRouter(cfg, pool, bookRepo, userRepo, coverCache)

	// Добавляем тестовую книгу
	testBook := &models.Book{
		ID:         "v2-book-1",
		Title:      "Гиперион",
		Annotation: "Космическая сага Дэна Симмонса.",
	}
	_ = bookRepo.SaveBook(ctx, testBook, nil, nil, nil, nil)

	// 1. Проверка каталога /opds/v2/catalog.json
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/opds/v2/catalog.json", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for catalog.json, got %d", rec.Code)
	}

	var feed opdsv2.Feed
	if err := json.Unmarshal(rec.Body.Bytes(), &feed); err != nil {
		t.Fatalf("failed to unmarshal opds v2 json: %v", err)
	}

	if feed.Metadata.Title == "" {
		t.Errorf("empty feed metadata title")
	}
	if len(feed.Publications) != 1 || feed.Publications[0].Metadata.Title != "Гиперион" {
		t.Errorf("expected publication 'Гиперион', got: %+v", feed.Publications)
	}

	// 2. Проверка поиска /opds/v2/search?query=Гиперион
	recSearch := httptest.NewRecorder()
	reqSearch := httptest.NewRequest(http.MethodGet, "/opds/v2/search?query=%D0%93%D0%B8%D0%BF%D0%B5%D1%80%D0%B8%D0%BE%D0%BD", nil)
	router.ServeHTTP(recSearch, reqSearch)

	if recSearch.Code != http.StatusOK {
		t.Fatalf("expected 200 for search, got %d", recSearch.Code)
	}

	var searchFeed opdsv2.Feed
	if err := json.Unmarshal(recSearch.Body.Bytes(), &searchFeed); err != nil {
		t.Fatalf("unmarshal search feed: %v", err)
	}
	if len(searchFeed.Publications) != 1 {
		t.Errorf("expected 1 search result, got %d", len(searchFeed.Publications))
	}
}
