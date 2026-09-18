package services_test

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"boyan/internal/models"
	"boyan/internal/services"
	"boyan/internal/storage"
)

func TestStreamer_StreamFB2FromZIP(t *testing.T) {
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp("", "streamer_test_*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	libraryDir := filepath.Join(tmpDir, "library")
	_ = os.MkdirAll(libraryDir, 0755)

	pool, err := storage.NewSQLitePool(ctx, dbPath, 5000, 16000)
	if err != nil {
		t.Fatalf("NewSQLitePool: %v", err)
	}
	defer pool.Close()

	_ = storage.RunMigrations(ctx, pool.Writer)
	bookRepo := storage.NewBookRepository(pool)

	// Создаем тестовый .fb2.zip файл внутри libraryDir
	zipRelativePath := "books/testbook.fb2.zip"
	zipFullPath := filepath.Join(libraryDir, zipRelativePath)
	_ = os.MkdirAll(filepath.Dir(zipFullPath), 0755)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("inner_book.fb2")
	expectedFB2Content := "<FictionBook><body><p>Hello Streaming World!</p></body></FictionBook>"
	_, _ = w.Write([]byte(expectedFB2Content))
	_ = zw.Close()

	if err := os.WriteFile(zipFullPath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("write zip file: %v", err)
	}

	// Сохраняем метаданные книги в БД
	book := &models.Book{
		ID:    "stream-book-1",
		Title: "Стриминг тест",
	}
	file := &models.BookFile{
		Format:   "fb2.zip",
		FilePath: zipRelativePath,
		FileSize: int64(len(buf.Bytes())),
		SHA256:   "aabbccdd",
	}
	_ = bookRepo.SaveBook(ctx, book, nil, nil, nil, file)

	// Инициализируем стример с включенной опцией распаковки на лету (streamFromZIP=true)
	streamer := services.NewStreamer(bookRepo, libraryDir, true)

	// Делаем запрос на получение fb2 (сервер должен распаковать inner_book.fb2 на лету из .fb2.zip)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download/fb2", nil)
	streamer.ServeBookFile(rec, req, "stream-book-1", "fb2")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for streamed fb2, got %d: %s", rec.Code, rec.Body.String())
	}

	if rec.Header().Get("Content-Type") != "application/x-fictionbook+xml" {
		t.Errorf("unexpected content-type: %s", rec.Header().Get("Content-Type"))
	}

	body := rec.Body.String()
	if body != expectedFB2Content {
		t.Errorf("streamed content mismatch: expected '%s', got '%s'", expectedFB2Content, body)
	}
}

func TestStreamer_ServeBookFile_AbsolutePath(t *testing.T) {
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp("", "streamer_abs_test_*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	libraryDir := filepath.Join(tmpDir, "library")
	_ = os.MkdirAll(libraryDir, 0755)

	pool, err := storage.NewSQLitePool(ctx, dbPath, 5000, 16000)
	if err != nil {
		t.Fatalf("NewSQLitePool: %v", err)
	}
	defer pool.Close()

	_ = storage.RunMigrations(ctx, pool.Writer)
	bookRepo := storage.NewBookRepository(pool)

	// Создаем тестовый файл с абсолютным путем (как делает Calibre импорт)
	absFilePath := filepath.Join(libraryDir, "author", "book.fb2")
	_ = os.MkdirAll(filepath.Dir(absFilePath), 0755)
	expectedContent := "<FictionBook><body><p>Direct File Content</p></body></FictionBook>"
	if err := os.WriteFile(absFilePath, []byte(expectedContent), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	book := &models.Book{
		ID:    "stream-book-abs",
		Title: "Абсолютный путь",
	}
	file := &models.BookFile{
		Format:   "fb2",
		FilePath: absFilePath, // Сохраняем абсолютный путь!
		FileSize: int64(len(expectedContent)),
		SHA256:   "11223344",
	}
	_ = bookRepo.SaveBook(ctx, book, nil, nil, nil, file)

	streamer := services.NewStreamer(bookRepo, libraryDir, false)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download/fb2", nil)
	streamer.ServeBookFile(rec, req, "stream-book-abs", "fb2")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for absolute path fb2, got %d: %s", rec.Code, rec.Body.String())
	}

	if rec.Body.String() != expectedContent {
		t.Errorf("content mismatch: expected '%s', got '%s'", expectedContent, rec.Body.String())
	}
}
