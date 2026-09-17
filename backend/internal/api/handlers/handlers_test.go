package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"boyan/internal/api"
	"boyan/internal/config"
	"boyan/internal/models"
	"boyan/internal/parsers/cover"
	"boyan/internal/storage"
	"boyan/internal/watcher"
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
	userRepo := storage.NewUserRepository(pool)
	quarantineRepo := storage.NewQuarantineRepository(pool)
	progressRepo := storage.NewProgressRepository(pool)
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
	watcherInstance := watcher.NewWatcher(cfg, bookRepo, quarantineRepo, coverCache)
	router := api.NewRouter(cfg, pool, bookRepo, userRepo, quarantineRepo, progressRepo, coverCache, watcherInstance)

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

	// 6. Проверка прямого эндпоинта /covers/{id} и /api/v1/covers/{id}
	// Без обложки -> 404
	reqCover404 := httptest.NewRequest(http.MethodGet, "/covers/test-book-id-123", nil)
	recCover404 := httptest.NewRecorder()
	router.ServeHTTP(recCover404, reqCover404)
	if recCover404.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing cover on /covers/{id}, got %d", recCover404.Code)
	}

	// Сохраняем тестовую обложку
	_, _ = coverCache.SaveCover("test-book-id-123", []byte("fake-jpeg-cover-data"))

	// Запрос на корень /covers/{id}
	reqCoverRoot := httptest.NewRequest(http.MethodGet, "/covers/test-book-id-123", nil)
	recCoverRoot := httptest.NewRecorder()
	router.ServeHTTP(recCoverRoot, reqCoverRoot)
	if recCoverRoot.Code != http.StatusOK {
		t.Errorf("expected 200 for root /covers/{id}, got %d", recCoverRoot.Code)
	}

	// Запрос на /api/v1/covers/{id}
	reqCoverAPI := httptest.NewRequest(http.MethodGet, "/api/v1/covers/test-book-id-123", nil)
	recCoverAPI := httptest.NewRecorder()
	router.ServeHTTP(recCoverAPI, reqCoverAPI)
	if recCoverAPI.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/v1/covers/{id}, got %d", recCoverAPI.Code)
	}
}

func TestAPI_I18nErrorResponses(t *testing.T) {
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp("", "api_i18n_test_*")
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

	_ = storage.RunMigrations(ctx, pool.Writer)

	bookRepo := storage.NewBookRepository(pool)
	userRepo := storage.NewUserRepository(pool)
	quarantineRepo := storage.NewQuarantineRepository(pool)
	progressRepo := storage.NewProgressRepository(pool)
	coverCache, _ := cover.NewCoverCache(coverDir, 10)

	cfg := config.DefaultConfig()
	watcherInstance := watcher.NewWatcher(cfg, bookRepo, quarantineRepo, coverCache)
	router := api.NewRouter(cfg, pool, bookRepo, userRepo, quarantineRepo, progressRepo, coverCache, watcherInstance)

	// 1. Ошибка 404 на английском языке
	req404EN := httptest.NewRequest(http.MethodGet, "/api/v1/books/not-found-id", nil)
	req404EN.Header.Set("Accept-Language", "en-US,en;q=0.9")
	rec404EN := httptest.NewRecorder()
	router.ServeHTTP(rec404EN, req404EN)

	if rec404EN.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec404EN.Code)
	}
	var errRespEN struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	_ = json.Unmarshal(rec404EN.Body.Bytes(), &errRespEN)
	if errRespEN.Code != "BOOK_NOT_FOUND" || errRespEN.Error != "Book not found" {
		t.Errorf("expected EN BOOK_NOT_FOUND, got: %+v", errRespEN)
	}

	// 2. Ошибка 404 на русском языке
	req404RU := httptest.NewRequest(http.MethodGet, "/api/v1/books/not-found-id", nil)
	req404RU.Header.Set("Accept-Language", "ru-RU,ru;q=0.9")
	rec404RU := httptest.NewRecorder()
	router.ServeHTTP(rec404RU, req404RU)

	var errRespRU struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	_ = json.Unmarshal(rec404RU.Body.Bytes(), &errRespRU)
	if errRespRU.Code != "BOOK_NOT_FOUND" || errRespRU.Error != "Книга не найдена" {
		t.Errorf("expected RU BOOK_NOT_FOUND, got: %+v", errRespRU)
	}

	// 3. Ошибка авторизации (пустые логин/пароль) на английском языке
	reqLoginEN := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"","password":""}`))
	reqLoginEN.Header.Set("Accept-Language", "en")
	reqLoginEN.Header.Set("Content-Type", "application/json")
	recLoginEN := httptest.NewRecorder()
	router.ServeHTTP(recLoginEN, reqLoginEN)

	if recLoginEN.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty login, got %d", recLoginEN.Code)
	}
	_ = json.Unmarshal(recLoginEN.Body.Bytes(), &errRespEN)
	if errRespEN.Code != "AUTH_FIELDS_REQUIRED" || errRespEN.Error != "Username and password are required" {
		t.Errorf("expected EN AUTH_FIELDS_REQUIRED, got: %+v", errRespEN)
	}

	// 4. Ошибка авторизации на русском языке
	reqLoginRU := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"","password":""}`))
	reqLoginRU.Header.Set("Accept-Language", "ru")
	reqLoginRU.Header.Set("Content-Type", "application/json")
	recLoginRU := httptest.NewRecorder()
	router.ServeHTTP(recLoginRU, reqLoginRU)

	_ = json.Unmarshal(recLoginRU.Body.Bytes(), &errRespRU)
	if errRespRU.Code != "AUTH_FIELDS_REQUIRED" || errRespRU.Error != "Имя пользователя и пароль обязательны для заполнения" {
		t.Errorf("expected RU AUTH_FIELDS_REQUIRED, got: %+v", errRespRU)
	}
}
