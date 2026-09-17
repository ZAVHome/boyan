package v1_test

import (
	"context"
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

func setupOPDSTestServer(t *testing.T) (http.Handler, *storage.BookRepository, func()) {
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp("", "opds_test_*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	coverDir := filepath.Join(tmpDir, "covers")

	pool, err := storage.NewSQLitePool(ctx, dbPath, 5000, 16000)
	if err != nil {
		t.Fatalf("NewSQLitePool: %v", err)
	}

	if err := storage.RunMigrations(ctx, pool.Writer); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	bookRepo := storage.NewBookRepository(pool)
	userRepo := storage.NewUserRepository(pool)
	quarantineRepo := storage.NewQuarantineRepository(pool)
	progressRepo := storage.NewProgressRepository(pool)
	coverCache, _ := cover.NewCoverCache(coverDir, 10)

	cfg := config.DefaultConfig()
	cfg.OPDS.AllowAnonymousReading = true // Для тестов без обязательного Basic Auth
	watcherInstance := watcher.NewWatcher(cfg, bookRepo, quarantineRepo, coverCache)
	router := api.NewRouter(cfg, pool, bookRepo, userRepo, quarantineRepo, progressRepo, coverCache, watcherInstance)

	cleanup := func() {
		pool.Close()
		os.RemoveAll(tmpDir)
	}

	return router, bookRepo, cleanup
}

func TestOPDSv1_NavigationAndFeeds(t *testing.T) {
	router, bookRepo, cleanup := setupOPDSTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Создаем тестовую серию и книги
	book := &models.Book{
		ID:         "opds-book-1",
		Title:      "Ночной Дозор",
		Annotation: "История противостояния Иных.",
		Language:   "ru",
	}
	authors := []models.AuthorDetail{
		{Author: models.Author{Name: "Сергей Лукьяненко", SortName: "Лукьяненко, Сергей"}, Role: "author", Order: 1},
	}
	series := []models.SeriesDetail{
		{Series: models.Series{Name: "Дозоры"}, Index: 1.0},
	}
	genres := []models.Genre{
		{Code: "sf_fantasy", NameRU: "Фэнтези", CategoryRU: "Фантастика", CategoryEN: "Sci-Fi"},
	}
	files := &models.BookFile{
		Format:   "fb2",
		FilePath: "books/dozor.fb2",
		FileSize: 500000,
		SHA256:   "11223344",
	}

	err := bookRepo.SaveBook(ctx, book, authors, series, genres, files)
	if err != nil {
		t.Fatalf("SaveBook failed: %v", err)
	}

	// 1. Проверка главного каталога /opds/v1/feed.xml
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/opds/v1/feed.xml", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for feed.xml, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<feed xmlns="http://www.w3.org/2005/Atom"`) {
		t.Errorf("missing Atom namespace in feed.xml")
	}
	if !strings.Contains(body, "По авторам") || !strings.Contains(body, "По сериям") {
		t.Errorf("missing standard navigation sections in feed.xml")
	}

	// 2. Проверка раздела авторов /opds/v1/authors
	recAuthors := httptest.NewRecorder()
	reqAuthors := httptest.NewRequest(http.MethodGet, "/opds/v1/authors", nil)
	router.ServeHTTP(recAuthors, reqAuthors)
	if recAuthors.Code != http.StatusOK {
		t.Fatalf("expected 200 for /opds/v1/authors, got %d", recAuthors.Code)
	}
	if !strings.Contains(recAuthors.Body.String(), "Л") {
		t.Errorf("expected letter 'Л' in authors feed, got %s", recAuthors.Body.String())
	}

	// 3. Проверка авторов на букву 'Л'
	recL := httptest.NewRecorder()
	reqL := httptest.NewRequest(http.MethodGet, "/opds/v1/authors/alpha/%D0%9B", nil)
	router.ServeHTTP(recL, reqL)
	if recL.Code != http.StatusOK {
		t.Fatalf("expected 200 for letter Л, got %d", recL.Code)
	}
	if !strings.Contains(recL.Body.String(), "Лукьяненко") {
		t.Errorf("expected Lukyanenko in letter Л list")
	}

	// 4. Проверка раздела серий /opds/v1/series
	recSeries := httptest.NewRecorder()
	reqSeries := httptest.NewRequest(http.MethodGet, "/opds/v1/series", nil)
	router.ServeHTTP(recSeries, reqSeries)
	if recSeries.Code != http.StatusOK {
		t.Fatalf("expected 200 for /opds/v1/series, got %d", recSeries.Code)
	}

	// 5. Проверка новинок /opds/v1/recent
	recRecent := httptest.NewRecorder()
	reqRecent := httptest.NewRequest(http.MethodGet, "/opds/v1/recent", nil)
	router.ServeHTTP(recRecent, reqRecent)
	if recRecent.Code != http.StatusOK {
		t.Fatalf("expected 200 for /opds/v1/recent, got %d", recRecent.Code)
	}
	recentBody := recRecent.Body.String()
	if !strings.Contains(recentBody, "Ночной Дозор") {
		t.Errorf("expected 'Ночной Дозор' in recent books")
	}
	// Проверка расширений Calibre для E-Ink
	if !strings.Contains(recentBody, "<calibre:series>Дозоры</calibre:series>") {
		t.Errorf("missing calibre:series tag in recent feed: %s", recentBody)
	}
	if !strings.Contains(recentBody, "<calibre:series_index>1</calibre:series_index>") {
		t.Errorf("missing calibre:series_index tag in recent feed: %s", recentBody)
	}

	// 6. Проверка OpenSearch дескриптора
	recOS := httptest.NewRecorder()
	reqOS := httptest.NewRequest(http.MethodGet, "/opds/v1/opensearch.xml", nil)
	router.ServeHTTP(recOS, reqOS)
	if recOS.Code != http.StatusOK {
		t.Fatalf("expected 200 for opensearch.xml, got %d", recOS.Code)
	}
	if !strings.Contains(recOS.Body.String(), "<OpenSearchDescription") {
		t.Errorf("invalid opensearch.xml: %s", recOS.Body.String())
	}

	// 7. Проверка поиска /opds/v1/search?q=Дозор
	recSearch := httptest.NewRecorder()
	reqSearch := httptest.NewRequest(http.MethodGet, "/opds/v1/search?q=%D0%94%D0%BE%D0%B7%D0%BE%D1%80", nil)
	router.ServeHTTP(recSearch, reqSearch)
	if recSearch.Code != http.StatusOK {
		t.Fatalf("expected 200 for search, got %d", recSearch.Code)
	}
	if !strings.Contains(recSearch.Body.String(), "Ночной Дозор") {
		t.Errorf("search result does not contain book: %s", recSearch.Body.String())
	}
}
