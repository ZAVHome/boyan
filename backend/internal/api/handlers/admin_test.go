package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"boyan/internal/auth"
	customMiddleware "boyan/internal/api/middleware"
	"boyan/internal/config"
	"boyan/internal/models"
	"boyan/internal/parsers/cover"
	"boyan/internal/services"
	"boyan/internal/storage"
	"boyan/internal/watcher"

	"github.com/go-chi/chi/v5"
)

func setupAdminTestRouter(t *testing.T) (*chi.Mux, string, string, *config.Config, *storage.DBPool) {
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
	cfgPath := filepath.Join(tempDir, "config.yaml")
	_ = cfg.Save(cfgPath)

	userRepo := storage.NewUserRepository(pool)
	bookRepo := storage.NewBookRepository(pool)
	quarantineRepo := storage.NewQuarantineRepository(pool)
	coverCache, _ := cover.NewCoverCache(cfg.Metadata.CoverCacheDir, 50)
	watcherInstance := watcher.NewWatcher(cfg, bookRepo, quarantineRepo, coverCache)
	taskManager := services.NewTaskManager()

	adminUser, err := userRepo.CreateUser(ctx, "superadmin", "pass123", models.RoleAdmin)
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	regularUser, err := userRepo.CreateUser(ctx, "regularuser", "pass123", models.RoleUser)
	if err != nil {
		t.Fatalf("create regular user: %v", err)
	}

	adminToken, _, _ := auth.GenerateToken(adminUser, cfg.Server.JWTSecret, 24)
	userToken, _, _ := auth.GenerateToken(regularUser, cfg.Server.JWTSecret, 24)

	// Роутер
	r := chi.NewRouter()
	r.Use(customMiddleware.JWTAuthMiddleware(cfg.Server.JWTSecret))

	adminDashboardH := NewAdminDashboardHandler(cfg, pool, bookRepo)
	adminUsersH := NewAdminUsersHandler(userRepo)
	adminBooksH := NewAdminBooksHandler(cfg, bookRepo, coverCache)
	adminTasksH := NewAdminTasksHandler(taskManager, watcherInstance, cfg, bookRepo)
	adminSettingsH := NewAdminSettingsHandler(cfg, cfgPath)

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(customMiddleware.RequireAdmin)

			r.Get("/admin/dashboard/stats", adminDashboardH.GetStats)
			r.Get("/admin/system/host", adminDashboardH.GetHostMetrics)
			r.Get("/admin/system/database", adminDashboardH.GetDatabaseMetrics)
			r.Post("/admin/system/database/checkpoint", adminDashboardH.CheckpointDatabase)

			r.Get("/admin/users", adminUsersH.ListUsers)
			r.Post("/admin/users", adminUsersH.CreateUser)
			r.Get("/admin/users/{id}", adminUsersH.GetUser)
			r.Put("/admin/users/{id}", adminUsersH.UpdateUser)
			r.Delete("/admin/users/{id}", adminUsersH.DeleteUser)

			r.Get("/admin/books", adminBooksH.ListBooks)
			r.Post("/admin/storage/repair-fb2", adminTasksH.RunRepairFB2)
			r.Get("/admin/tasks", adminTasksH.ListTasks)
			r.Get("/admin/settings", adminSettingsH.GetSettings)
			r.Put("/admin/settings", adminSettingsH.UpdateSettings)
		})
	})

	return r, adminToken, userToken, cfg, pool
}

func TestAdminAPI_AccessControl(t *testing.T) {
	router, adminToken, userToken, _, pool := setupAdminTestRouter(t)
	defer pool.Close()

	// 1. Запрос без токена -> 401 Unauthorized
	req, _ := http.NewRequest("GET", "/api/v1/admin/dashboard/stats", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", rr.Code)
	}

	// 2. Запрос с токеном обычного пользователя -> 403 Forbidden
	req, _ = http.NewRequest("GET", "/api/v1/admin/dashboard/stats", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", rr.Code)
	}

	// 3. Запрос с токеном администратора -> 200 OK
	req, _ = http.NewRequest("GET", "/api/v1/admin/dashboard/stats", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminAPI_HostAndDBMetrics(t *testing.T) {
	router, adminToken, _, _, pool := setupAdminTestRouter(t)
	defer pool.Close()

	// Host metrics
	req, _ := http.NewRequest("GET", "/api/v1/admin/system/host", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("host metrics failed: %d", rr.Code)
	}

	var hostMetrics services.HostMetrics
	if err := json.Unmarshal(rr.Body.Bytes(), &hostMetrics); err != nil {
		t.Fatalf("unmarshal host metrics: %v", err)
	}
	if hostMetrics.NumCPU <= 0 || hostMetrics.GoVersion == "" {
		t.Errorf("unexpected host metrics: %+v", hostMetrics)
	}

	// DB Checkpoint
	req, _ = http.NewRequest("POST", "/api/v1/admin/system/database/checkpoint", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("db checkpoint failed: %d", rr.Code)
	}
}

func TestAdminAPI_UserManagementFlow(t *testing.T) {
	router, adminToken, _, _, pool := setupAdminTestRouter(t)
	defer pool.Close()

	// Create user
	createUserBody, _ := json.Marshal(models.CreateUserRequest{
		Username: "bob",
		Password: "bobpassword",
		Role:     models.RoleUser,
		IsActive: true,
	})
	req, _ := http.NewRequest("POST", "/api/v1/admin/users", bytes.NewReader(createUserBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create user failed: %d, body: %s", rr.Code, rr.Body.String())
	}

	var created models.User
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	if created.Username != "bob" || created.ID == "" {
		t.Fatalf("unexpected created user: %+v", created)
	}

	// Update user
	updateUserBody, _ := json.Marshal(models.UpdateUserRequest{
		Role:     models.RoleAdmin,
		IsActive: false,
	})
	req, _ = http.NewRequest("PUT", "/api/v1/admin/users/"+created.ID, bytes.NewReader(updateUserBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("update user failed: %d", rr.Code)
	}

	// Delete user
	req, _ = http.NewRequest("DELETE", "/api/v1/admin/users/"+created.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete user failed: %d", rr.Code)
	}
}

func TestAdminAPI_Settings(t *testing.T) {
	router, adminToken, _, _, pool := setupAdminTestRouter(t)
	defer pool.Close()

	// 1. GET settings
	req, _ := http.NewRequest("GET", "/api/v1/admin/settings", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get settings failed: %d", rr.Code)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal settings json failed: %v", err)
	}

	// Verify json keys are lowercase snake_case
	server, ok := res["server"].(map[string]interface{})
	if !ok || server == nil {
		t.Fatalf("expected 'server' object in settings json, got: %s", rr.Body.String())
	}
	if server["host"] == nil || server["port"] == nil {
		t.Fatalf("expected server.host and server.port, got: %+v", server)
	}

	storage, ok := res["storage"].(map[string]interface{})
	if !ok || storage == nil {
		t.Fatalf("expected 'storage' object in settings json, got: %s", rr.Body.String())
	}
	if storage["library_dir"] == nil {
		t.Fatalf("expected storage.library_dir, got: %+v", storage)
	}

	// 2. PUT settings
	updateBody := rr.Body.Bytes()
	req, _ = http.NewRequest("PUT", "/api/v1/admin/settings", bytes.NewReader(updateBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("update settings failed: %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminAPI_RepairFB2(t *testing.T) {
	router, adminToken, _, _, pool := setupAdminTestRouter(t)
	defer pool.Close()

	req, _ := http.NewRequest("POST", "/api/v1/admin/storage/repair-fb2", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d: %s", rr.Code, rr.Body.String())
	}
}


