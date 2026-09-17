package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"boyan/internal/api"
	"boyan/internal/config"
	"boyan/internal/parsers/cover"
	"boyan/internal/storage"
)

const Version = "0.1.0"

func main() {
	var (
		configPath  string
		showVersion bool
	)

	flag.StringVar(&configPath, "config", "config.yaml", "Path to YAML configuration file")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("Next-Gen OPDS Suite (Boyan) v%s\n", Version)
		return
	}

	// 1. Загрузка конфигурации
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error loading config: %v\n", err)
		os.Exit(1)
	}

	// 2. Инициализация структурированного логирования slog
	setupLogger(cfg.Logging)

	slog.Info("Starting Next-Gen OPDS Suite (Boyan)",
		"version", Version,
		"config", configPath,
		"log_level", cfg.Logging.Level,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Подключение к SQLite (WAL mode, pure-Go)
	pool, err := storage.NewSQLitePool(
		ctx,
		cfg.Database.SQLite.Path,
		cfg.Database.SQLite.BusyTimeoutMS,
		cfg.Database.SQLite.CacheSizeKB,
	)
	if err != nil {
		slog.Error("Failed to initialize SQLite database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// 4. Применение автоматических миграций схемы
	if err := storage.RunMigrations(ctx, pool.Writer); err != nil {
		slog.Error("Failed to apply database migrations", "err", err)
		os.Exit(1)
	}

	// 5. Инициализация репозиториев
	userRepo := storage.NewUserRepository(pool)
	bookRepo := storage.NewBookRepository(pool)

	// Автоматическое создание учетной записи администратора по умолчанию
	if cfg.Admin.DefaultUsername != "" && cfg.Admin.DefaultPassword != "" {
		if err := userRepo.EnsureAdminUser(ctx, cfg.Admin.DefaultUsername, cfg.Admin.DefaultPassword); err != nil {
			slog.Warn("Could not ensure admin user", "err", err)
		}
	}

	// 6. Инициализация кеша обложек
	coverCache, err := cover.NewCoverCache(cfg.Metadata.CoverCacheDir, cfg.Metadata.CoverCacheMaxMB)
	if err != nil {
		slog.Error("Failed to initialize cover cache", "err", err)
		os.Exit(1)
	}

	// 7. Сборка HTTP роутера
	router := api.NewRouter(cfg, pool, bookRepo, coverCache)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 8. Запуск сервера в отдельной горутине
	go func() {
		slog.Info("HTTP server is listening",
			"addr", addr,
			"health_url", fmt.Sprintf("http://localhost:%d/health", cfg.Server.Port),
			"api_url", fmt.Sprintf("http://localhost:%d/api/v1/ping", cfg.Server.Port),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server failed", "err", err)
			os.Exit(1)
		}
	}()

	// 9. Перехват системных сигналов завершения (Graceful Shutdown)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "err", err)
	}

	slog.Info("Server stopped successfully")
}

func setupLogger(cfg config.LoggingConfig) {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}
