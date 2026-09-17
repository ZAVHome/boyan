package api

import (
	"net/http"

	"boyan/internal/api/handlers"
	customMiddleware "boyan/internal/api/middleware"
	"boyan/internal/config"
	opdsv1 "boyan/internal/opds/v1"
	opdsv2 "boyan/internal/opds/v2"
	"boyan/internal/parsers/cover"
	"boyan/internal/services"
	"boyan/internal/storage"
	"boyan/internal/watcher"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
) *chi.Mux {
	r := chi.NewRouter()

	// Глобальные middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(customMiddleware.SlogLogger)
	r.Use(middleware.Recoverer)
	r.Use(customMiddleware.CORS(cfg.Server.CORSAllowedOrigins))
	r.Use(customMiddleware.JWTAuthMiddleware(cfg.Server.JWTSecret))

	// Инициализация сервисов
	streamer := services.NewStreamer(bookRepo, cfg.Storage.LibraryDir, cfg.OPDS.StreamFromZIP)

	// Инициализация хендлеров
	healthH := handlers.NewHealthHandler(pool)
	authH := handlers.NewAuthHandler(cfg, userRepo)
	booksH := handlers.NewBooksHandler(bookRepo, coverCache)
	uploadH := handlers.NewUploadHandler(watcherInstance)
	quarantineH := handlers.NewQuarantineHandler(cfg, quarantineRepo, bookRepo, coverCache)
	progressH := handlers.NewProgressHandler(progressRepo, bookRepo)

	// Системные эндпоинты
	r.Get("/health", healthH.HealthCheck)

	// REST API v1 ветка
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", healthH.Ping)

		// Аутентификация
		r.Post("/auth/login", authH.Login)
		r.Post("/auth/logout", authH.Logout)

		// Профиль пользователя
		r.With(customMiddleware.RequireAuth).Get("/auth/me", authH.Me)

		// Книги и обложки
		r.Get("/books", booksH.ListBooks)
		r.Get("/books/{id}", booksH.GetBook)
		r.Get("/covers/{id}", booksH.GetCover)

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

		// Карантин дубликатов (требует прав администратора)
		r.Group(func(r chi.Router) {
			r.Use(customMiddleware.RequireAdmin)

			r.Get("/quarantine", quarantineH.ListQuarantine)
			r.Get("/quarantine/{id}", quarantineH.GetQuarantineItem)
			r.Post("/quarantine/{id}/resolve", quarantineH.ResolveQuarantine)
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
