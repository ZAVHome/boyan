package api

import (
	"boyan/internal/api/handlers"
	customMiddleware "boyan/internal/api/middleware"
	"boyan/internal/config"
	"boyan/internal/parsers/cover"
	"boyan/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter конфигурирует и собирает главный HTTP-роутер приложения на базе go-chi.
func NewRouter(
	cfg *config.Config,
	pool *storage.DBPool,
	bookRepo *storage.BookRepository,
	coverCache *cover.CoverCache,
) *chi.Mux {
	r := chi.NewRouter()

	// Глобальные middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(customMiddleware.SlogLogger)
	r.Use(middleware.Recoverer)
	r.Use(customMiddleware.CORS(cfg.Server.CORSAllowedOrigins))

	// Системные эндпоинты
	healthH := handlers.NewHealthHandler(pool)
	r.Get("/health", healthH.HealthCheck)

	// REST API v1 ветка
	booksH := handlers.NewBooksHandler(bookRepo, coverCache)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", healthH.Ping)

		// Книги и обложки
		r.Get("/books", booksH.ListBooks)
		r.Get("/books/{id}", booksH.GetBook)
		r.Get("/covers/{id}", booksH.GetCover)
	})

	return r
}
