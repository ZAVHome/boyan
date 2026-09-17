package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"boyan/internal/auth"
	"boyan/internal/config"
	"boyan/internal/models"
	"boyan/internal/parsers/cover"
	"boyan/internal/storage"
	"boyan/internal/watcher"

	"github.com/go-chi/chi/v5"
)

func setupTestApp(t *testing.T) (*config.Config, *storage.DBPool, *storage.BookRepository, *storage.UserRepository, *storage.QuarantineRepository, *storage.ProgressRepository, *watcher.Watcher, *cover.CoverCache) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	ctx := context.Background()

	pool, err := storage.NewSQLitePool(ctx, dbPath, 5000, 10000)
	if err != nil {
		t.Fatalf("failed to create sqlite pool: %v", err)
	}

	if err := storage.RunMigrations(ctx, pool.Writer); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Storage.LibraryDir = filepath.Join(tempDir, "library")
	cfg.Storage.WatchDir = filepath.Join(tempDir, "import")
	cfg.Metadata.CoverCacheDir = filepath.Join(tempDir, "cache", "covers")

	userRepo := storage.NewUserRepository(pool)
	bookRepo := storage.NewBookRepository(pool)
	quarantineRepo := storage.NewQuarantineRepository(pool)
	progressRepo := storage.NewProgressRepository(pool)

	coverCache, _ := cover.NewCoverCache(cfg.Metadata.CoverCacheDir, 50)
	watcherInstance := watcher.NewWatcher(cfg, bookRepo, quarantineRepo, coverCache)

	_ = userRepo.EnsureAdminUser(ctx, "admin", "adminpassword")

	return cfg, pool, bookRepo, userRepo, quarantineRepo, progressRepo, watcherInstance, coverCache
}

func TestAuthFlow(t *testing.T) {
	cfg, pool, _, userRepo, _, _, _, _ := setupTestApp(t)
	defer pool.Close()

	authH := NewAuthHandler(cfg, userRepo)

	// 1. Успешный вход
	loginBody, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "adminpassword",
	})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	authH.Login(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(w.Body).Decode(&loginResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}
	if loginResp.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if loginResp.User.Username != "admin" {
		t.Errorf("expected username 'admin', got %s", loginResp.User.Username)
	}

	// Проверяем установку cookie
	cookies := w.Result().Cookies()
	var tokenCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "boyan_token" {
			tokenCookie = c
			break
		}
	}
	if tokenCookie == nil || tokenCookie.Value == "" {
		t.Fatal("expected boyan_token cookie to be set")
	}

	// 2. Проверка Me через контекст
	claims, err := auth.ValidateToken(loginResp.Token, cfg.Server.JWTSecret)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	meReq := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	meReq = meReq.WithContext(auth.WithUser(meReq.Context(), claims))
	meW := httptest.NewRecorder()
	authH.Me(meW, meReq)

	if meW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on Me, got %d", meW.Code)
	}

	// 3. Неверный пароль
	badLogin, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "wrongpassword",
	})
	badReq := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(badLogin))
	badReq.Header.Set("Content-Type", "application/json")
	badW := httptest.NewRecorder()
	authH.Login(badW, badReq)

	if badW.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for bad login, got %d", badW.Code)
	}
}

func TestProgressAndShelves(t *testing.T) {
	_, pool, bookRepo, userRepo, _, progressRepo, _, _ := setupTestApp(t)
	defer pool.Close()

	ctx := context.Background()
	progressH := NewProgressHandler(progressRepo, bookRepo)

	// Создаем тестового пользователя
	user, err := userRepo.CreateUser(ctx, "reader", "readerpass", models.RoleUser)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Создаем тестовую книгу
	testBook := &models.Book{
		ID:    "book-progress-1",
		Title: "Прогресс Тест",
	}
	_ = bookRepo.SaveBook(ctx, testBook, nil, nil, nil, nil)

	claims := &auth.UserClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	}

	// 1. Сохранение прогресса
	saveBody, _ := json.Marshal(map[string]any{
		"format":           "fb2",
		"progress_percent": 45.5,
		"position":         "p-15",
	})

	r := chi.NewRouter()
	r.Put("/books/{id}/progress", progressH.SaveProgress)
	r.Get("/books/{id}/progress", progressH.GetProgress)
	r.Get("/shelves/{type}", progressH.GetShelfBooks)

	putReq := httptest.NewRequest("PUT", "/books/book-progress-1/progress", bytes.NewReader(saveBody))
	putReq = putReq.WithContext(auth.WithUser(putReq.Context(), claims))
	putW := httptest.NewRecorder()
	r.ServeHTTP(putW, putReq)

	if putW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on SaveProgress, got %d: %s", putW.Code, putW.Body.String())
	}

	// 2. Получение сохраненного прогресса
	getReq := httptest.NewRequest("GET", "/books/book-progress-1/progress", nil)
	getReq = getReq.WithContext(auth.WithUser(getReq.Context(), claims))
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GetProgress, got %d", getW.Code)
	}
	var pResp storage.ReadProgress
	_ = json.NewDecoder(getW.Body).Decode(&pResp)
	if pResp.ProgressPercent != 45.5 {
		t.Errorf("expected progress 45.5, got %f", pResp.ProgressPercent)
	}

	// 3. Проверяем, что книга автоматически появилась на полке "reading"
	shelfReq := httptest.NewRequest("GET", "/shelves/reading", nil)
	shelfReq = shelfReq.WithContext(auth.WithUser(shelfReq.Context(), claims))
	shelfW := httptest.NewRecorder()
	r.ServeHTTP(shelfW, shelfReq)

	if shelfW.Code != http.StatusOK {
		t.Fatalf("expected 200 on shelf get, got %d", shelfW.Code)
	}
	var shelfResp map[string]any
	_ = json.NewDecoder(shelfW.Body).Decode(&shelfResp)
	total := int(shelfResp["total"].(float64))
	if total != 1 {
		t.Errorf("expected 1 book on reading shelf, got %d", total)
	}
}

func TestQuarantineFlow(t *testing.T) {
	cfg, pool, bookRepo, _, quarantineRepo, _, _, coverCache := setupTestApp(t)
	defer pool.Close()

	ctx := context.Background()
	quarantineH := NewQuarantineHandler(cfg, quarantineRepo, bookRepo, coverCache)

	// Добавляем элемент в карантин
	qItem := &storage.QuarantineItem{
		ID:            "q-item-1",
		FilePath:      filepath.Join(t.TempDir(), "dummy.fb2"),
		FileSize:      1024,
		SHA256:        "abc123hash",
		Format:        "fb2",
		ParsedTitle:   "Дубликат Книги",
		ParsedAuthors: "Автор",
		ConflictType:  "same_format",
	}
	_ = quarantineRepo.AddToQuarantine(ctx, qItem)

	r := chi.NewRouter()
	r.Get("/quarantine", quarantineH.ListQuarantine)
	r.Get("/quarantine/{id}", quarantineH.GetQuarantineItem)
	r.Post("/quarantine/{id}/resolve", quarantineH.ResolveQuarantine)

	// 1. Список карантина
	listReq := httptest.NewRequest("GET", "/quarantine", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 on ListQuarantine, got %d", listW.Code)
	}

	// 2. Получение элемента
	getReq := httptest.NewRequest("GET", "/quarantine/q-item-1", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 on GetQuarantineItem, got %d", getW.Code)
	}

	// 3. Отклонение (discard)
	resolveBody, _ := json.Marshal(map[string]string{"action": "discard"})
	resReq := httptest.NewRequest("POST", "/quarantine/q-item-1/resolve", bytes.NewReader(resolveBody))
	resW := httptest.NewRecorder()
	r.ServeHTTP(resW, resReq)

	if resW.Code != http.StatusOK {
		t.Fatalf("expected 200 on discard, got %d", resW.Code)
	}

	// Проверяем, что удалилось
	item, _ := quarantineRepo.GetQuarantineItem(ctx, "q-item-1")
	if item != nil {
		t.Fatal("expected item to be deleted from quarantine")
	}
}

func TestUploadBook(t *testing.T) {
	_, pool, _, _, _, _, watcherInstance, _ := setupTestApp(t)
	defer pool.Close()

	uploadH := NewUploadHandler(watcherInstance)

	// Создаем минимальный валидный FB2 XML
	fb2XML := `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <genre>sf</genre>
      <author>
        <first-name>Станислав</first-name>
        <last-name>Лем</last-name>
      </author>
      <book-title>Солярис</book-title>
      <lang>ru</lang>
    </title-info>
  </description>
  <body><section><p>Тестовый текст</p></section></body>
</FictionBook>`

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "solaris.fb2")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = part.Write([]byte(fb2XML))
	_ = writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/books/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	uploadH.UploadBook(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on upload, got %d: %s", w.Code, w.Body.String())
	}

	var res watcher.ProcessResult
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode process result: %v", err)
	}
	if res.Status != "imported" {
		t.Errorf("expected status 'imported', got '%s'", res.Status)
	}
	if res.BookID == "" {
		t.Error("expected non-empty BookID")
	}
}
