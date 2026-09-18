package api

import (
	"net/http"

	_ "boyan/docs/swagger"
	"boyan/internal/api/handlers"
	customMiddleware "boyan/internal/api/middleware"
	"boyan/internal/config"
	"boyan/internal/i18n"
	"boyan/internal/importer/calibre"
	opdsv1 "boyan/internal/opds/v1"
	opdsv2 "boyan/internal/opds/v2"
	"boyan/internal/parsers/cover"
	"boyan/internal/services"
	"boyan/internal/storage"
	"boyan/internal/watcher"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// NewRouter конфигурирует и собирает главный HTTP-роутер приложения на базе go-chi.
func NewRouter(
	cfg *config.Config,
	pool *storage.DBPool,
	bookRepo *storage.BookRepository,
	userRepo *storage.UserRepository,
	quarantineRepo *storage.QuarantineRepository,
	progressRepo *storage.ProgressRepository,
	coverCache *cover.CoverCache,
	watcherInstance *watcher.Watcher,
	configPathOpt ...string,
) *chi.Mux {
	configPath := "config.yaml"
	if len(configPathOpt) > 0 && configPathOpt[0] != "" {
		configPath = configPathOpt[0]
	}

	r := chi.NewRouter()

	// Глобальные middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(customMiddleware.SlogLogger)
	r.Use(middleware.Recoverer)
	r.Use(customMiddleware.CORS(cfg.Server.CORSAllowedOrigins))
	r.Use(i18n.Middleware(cfg.Server.DefaultLanguage))
	r.Use(customMiddleware.JWTAuthMiddleware(cfg.Server.JWTSecret))

	// Инициализация сервисов
	streamer := services.NewStreamer(bookRepo, cfg.Storage.LibraryDir, cfg.OPDS.StreamFromZIP)
	calibreImporter := calibre.NewImporter(cfg, bookRepo, coverCache)
	taskManager := services.NewTaskManager()

	// Инициализация хендлеров
	healthH := handlers.NewHealthHandler(pool)
	authH := handlers.NewAuthHandler(cfg, userRepo)
	booksH := handlers.NewBooksHandler(cfg, bookRepo, coverCache)
	uploadH := handlers.NewUploadHandler(watcherInstance)
	quarantineH := handlers.NewQuarantineHandler(cfg, quarantineRepo, bookRepo, coverCache)
	progressH := handlers.NewProgressHandler(progressRepo, bookRepo)
	calibreH := handlers.NewCalibreHandler(calibreImporter, taskManager)

	// Хендлеры расширенной панели администратора
	adminDashboardH := handlers.NewAdminDashboardHandler(cfg, pool, bookRepo)
	adminUsersH := handlers.NewAdminUsersHandler(userRepo)
	adminBooksH := handlers.NewAdminBooksHandler(cfg, bookRepo, coverCache)
	adminTasksH := handlers.NewAdminTasksHandler(taskManager, watcherInstance, cfg, bookRepo)
	adminSettingsH := handlers.NewAdminSettingsHandler(cfg, configPath)

	// Системные эндпоинты
	r.Get("/health", healthH.HealthCheck)
	// Прямой доступ к обложкам без префикса /api/v1 (проксируется Vite и Nginx)
	r.Get("/covers/{id}", booksH.GetCover)

	// REST API v1 ветка
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", healthH.Ping)

		// Swagger UI документация
		r.Get("/docs/*", httpSwagger.WrapHandler)
		r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/api/v1/docs/index.html", http.StatusMovedPermanently)
		})

		// Аутентификация и регистрация
		r.Post("/auth/login", authH.Login)
		r.Post("/auth/logout", authH.Logout)
		r.Post("/auth/register", authH.Register)

		// Профиль пользователя
		r.With(customMiddleware.RequireAuth).Get("/auth/me", authH.Me)

		// Книги и обложки
		r.Get("/books", booksH.ListBooks)
		r.Get("/books/{id}", booksH.GetBook)
		r.Get("/covers/{id}", booksH.GetCover)

		// Авторы
		r.Get("/authors", booksH.ListAuthors)
		r.Get("/authors/{id}", booksH.GetAuthor)
		r.Get("/authors/{id}/books", booksH.GetAuthorBooks)

		// Серии
		r.Get("/series", booksH.ListSeries)
		r.Get("/series/{id}", booksH.GetSeries)
		r.Get("/series/{id}/books", booksH.GetSeriesBooks)

		// Стриминг и скачивание файлов книг
		r.Get("/books/{id}/download/{format}", func(w http.ResponseWriter, r *http.Request) {
			bookID := chi.URLParam(r, "id")
			format := chi.URLParam(r, "format")
			streamer.ServeBookFile(w, r, bookID, format)
		})

		// Загрузка книг (требует авторизации)
		r.With(customMiddleware.RequireAuth).Post("/books/upload", uploadH.UploadBook)

		// Прогресс чтения и пользовательские полки (требует авторизации)
		r.Group(func(r chi.Router) {
			r.Use(customMiddleware.RequireAuth)

			r.Get("/books/{id}/progress", progressH.GetProgress)
			r.Put("/books/{id}/progress", progressH.SaveProgress)
			r.Post("/books/{id}/shelf", progressH.AddToShelf)
			r.Delete("/books/{id}/shelf/{type}", progressH.RemoveFromShelf)
			r.Get("/shelves/{type}", progressH.GetShelfBooks)
		})

		// Административный раздел (требует роли admin)
		r.Group(func(r chi.Router) {
			r.Use(customMiddleware.RequireAdmin)

			// Карантин дубликатов
			r.Get("/quarantine", quarantineH.ListQuarantine)
			r.Get("/quarantine/{id}", quarantineH.GetQuarantineItem)
			r.Post("/quarantine/{id}/resolve", quarantineH.ResolveQuarantine)

			// Импорт Calibre
			r.Post("/admin/import/calibre", calibreH.ImportCalibre)

			// Дашборд, мониторинг хоста, БД и кэша
			r.Get("/admin/dashboard/stats", adminDashboardH.GetStats)
			r.Get("/admin/system/host", adminDashboardH.GetHostMetrics)
			r.Get("/admin/system/database", adminDashboardH.GetDatabaseMetrics)
			r.Post("/admin/system/database/checkpoint", adminDashboardH.CheckpointDatabase)
			r.Get("/admin/system/cache", adminDashboardH.GetCacheMetrics)
			r.Post("/admin/system/cache/purge", adminDashboardH.PurgeCache)
			r.Get("/admin/system/logs", adminDashboardH.GetLogs)
			r.Get("/admin/system/telegram", adminDashboardH.GetTelegramStatus)

			// Управление пользователями
			r.Get("/admin/users", adminUsersH.ListUsers)
			r.Post("/admin/users", adminUsersH.CreateUser)
			r.Get("/admin/users/{id}", adminUsersH.GetUser)
			r.Put("/admin/users/{id}", adminUsersH.UpdateUser)
			r.Put("/admin/users/{id}/password", adminUsersH.UpdatePassword)
			r.Delete("/admin/users/{id}", adminUsersH.DeleteUser)

			// Управление книгами, кураторство и пакетные операции
			r.Get("/admin/books", adminBooksH.ListBooks)
			r.Put("/admin/books/{id}", adminBooksH.UpdateBookMetadata)
			r.Delete("/admin/books/{id}", adminBooksH.DeleteBook)
			r.Post("/admin/books/{id}/cover", adminBooksH.RegenerateCover)
			r.Post("/admin/books/batch", adminBooksH.BatchAction)

			// Сканер хранилища и фоновые задачи
			r.Post("/admin/scanner/run", adminTasksH.RunScan)
			r.Post("/admin/storage/repair-fb2", adminTasksH.RunRepairFB2)
			r.Get("/admin/tasks", adminTasksH.ListTasks)
			r.Get("/admin/tasks/{id}", adminTasksH.GetTask)
			r.Post("/admin/tasks/{id}/cancel", adminTasksH.CancelTask)

			// Настройки сервера и перезапуск служб
			r.Get("/admin/settings", adminSettingsH.GetSettings)
			r.Put("/admin/settings", adminSettingsH.UpdateSettings)
			r.Post("/admin/services/reload", adminSettingsH.ReloadServices)
		})
	})


	// Middleware аутентификации OPDS для E-Ink читалок
	opdsAuth := customMiddleware.OPDSAuth(userRepo, cfg.OPDS.AllowAnonymousReading, cfg.Server.JWTSecret)

	// Ветка OPDS v1.2 (Atom / XML)
	if cfg.OPDS.EnableOPDSv1 {
		v1H := opdsv1.NewOPDSv1Handler(cfg, bookRepo)

		r.Route("/opds/v1", func(r chi.Router) {
			r.Use(opdsAuth)

			// Главные каталоги
			r.Get("/feed.xml", v1H.RootFeed)
			r.Get("/root.xml", v1H.RootFeed)
			r.Get("/catalog.atom", v1H.RootFeed)
			r.Get("/opensearch.xml", v1H.OpenSearchDescriptor)
			r.Get("/search", v1H.Search)

			// Авторы
			r.Get("/authors", v1H.AuthorsAlpha)
			r.Get("/authors/alpha/{letter}", v1H.AuthorsList)
			r.Get("/authors/{id}", v1H.AuthorBooks)

			// Серии
			r.Get("/series", v1H.SeriesAlpha)
			r.Get("/series/alpha/{letter}", v1H.SeriesList)
			r.Get("/series/{id}", v1H.SeriesBooks)

			// Жанры
			r.Get("/genres", v1H.GenresCategories)
			r.Get("/genres/category/{category}", v1H.GenresSubcategories)
			r.Get("/genres/{code}", v1H.GenreBooks)

			// Новинки
			r.Get("/recent", v1H.RecentBooks)
		})
	}

	// Ветка OPDS v2.0 (JSON-LD)
	if cfg.OPDS.EnableOPDSv2 {
		v2H := opdsv2.NewOPDSv2Handler(cfg, bookRepo)

		r.Route("/opds/v2", func(r chi.Router) {
			r.Use(opdsAuth)

			r.Get("/catalog.json", v2H.Catalog)
			r.Get("/search", v2H.Search)
		})
	}

	return r
}
