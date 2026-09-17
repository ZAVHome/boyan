package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	customMiddleware "boyan/internal/api/middleware"
	"boyan/internal/models"
	"boyan/internal/storage"
)

func TestOPDSAuth(t *testing.T) {
	ctx := context.Background()
	tmpDir, _ := os.MkdirTemp("", "auth_test_*")
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	pool, _ := storage.NewSQLitePool(ctx, dbPath, 5000, 16000)
	defer pool.Close()

	_ = storage.RunMigrations(ctx, pool.Writer)
	userRepo := storage.NewUserRepository(pool)

	_, _ = userRepo.CreateUser(ctx, "reader", "password123", models.RoleUser)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	staticToken := "secret-url-token"
	authMW := customMiddleware.OPDSAuth(userRepo, false, staticToken)
	secured := authMW(handler)

	// 1. Без авторизации -> 401 Unauthorized
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/opds/v1/feed.xml", nil)
	secured.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", rec1.Code)
	}

	// 2. С правильным URL токеном -> 200 OK
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/opds/v1/feed.xml?token=secret-url-token", nil)
	secured.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200 with valid token, got %d", rec2.Code)
	}

	// 3. С правильным Basic Auth -> 200 OK
	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/opds/v1/feed.xml", nil)
	req3.SetBasicAuth("reader", "password123")
	secured.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Errorf("expected 200 with valid Basic Auth, got %d", rec3.Code)
	}

	// 4. С неверным Basic Auth -> 401 Unauthorized
	rec4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodGet, "/opds/v1/feed.xml", nil)
	req4.SetBasicAuth("reader", "wrongpass")
	secured.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with invalid password, got %d", rec4.Code)
	}
}
