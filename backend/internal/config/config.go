package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config описывает полную конфигурацию сервиса.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Logging  LoggingConfig  `yaml:"logging"`
	Database DatabaseConfig `yaml:"database"`
	Storage  StorageConfig  `yaml:"storage"`
	OPDS     OPDSConfig     `yaml:"opds"`
	Metadata MetadataConfig `yaml:"metadata"`
	Admin    AdminConfig    `yaml:"admin"`
	Telegram TelegramConfig `yaml:"telegram"`
}

type TelegramConfig struct {
	Enabled        bool    `yaml:"enabled"`
	BotToken       string  `yaml:"bot_token"`
	AllowedUserIDs []int64 `yaml:"allowed_user_ids"`
}

type ServerConfig struct {
	Host               string   `yaml:"host"`
	Port               int      `yaml:"port"`
	BaseURL            string   `yaml:"base_url"`
	DefaultLanguage    string   `yaml:"default_language"`
	JWTSecret          string   `yaml:"jwt_secret"`
	JWTExpirationHours int      `yaml:"jwt_expiration_hours"`
	CORSAllowedOrigins []string `yaml:"cors_allowed_origins"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // text, json
}

type DatabaseConfig struct {
	Driver string       `yaml:"driver"` // sqlite
	SQLite SQLiteConfig `yaml:"sqlite"`
}

type SQLiteConfig struct {
	Path          string `yaml:"path"`
	EnableWAL     bool   `yaml:"enable_wal"`
	BusyTimeoutMS int    `yaml:"busy_timeout_ms"`
	CacheSizeKB   int    `yaml:"cache_size_kb"`
}

type StorageConfig struct {
	LibraryDir   string        `yaml:"library_dir"`
	WatchDir     string        `yaml:"watch_dir"`
	PathTemplate string        `yaml:"path_template"`
	Watcher      WatcherConfig `yaml:"watcher"`
}

type WatcherConfig struct {
	Enabled                 bool `yaml:"enabled"`
	SettleDelaySeconds      int  `yaml:"settle_delay_seconds"`
	DeleteSourceAfterImport bool `yaml:"delete_source_after_import"`
	QuarantineDuplicates    bool `yaml:"quarantine_duplicates"`
}

type OPDSConfig struct {
	Title                 string `yaml:"title"`
	Subtitle              string `yaml:"subtitle"`
	PageSize              int    `yaml:"page_size"`
	EnableOPDSv1          bool   `yaml:"enable_opds_v1"`
	EnableOPDSv2          bool   `yaml:"enable_opds_v2"`
	AllowAnonymousReading bool   `yaml:"allow_anonymous_reading"`
	StreamFromZIP         bool   `yaml:"stream_from_zip"`
}

type MetadataConfig struct {
	AutoEnrichOnImport bool     `yaml:"auto_enrich_on_import"`
	ProvidersPriority  []string `yaml:"providers_priority"`
	CacheCoversLocally bool     `yaml:"cache_covers_locally"`
	CoverThumbnailSize int      `yaml:"cover_thumbnail_size"`
	CoverCacheMaxMB    int      `yaml:"cover_cache_max_mb"`
	CoverCacheDir      string   `yaml:"cover_cache_dir"`
}

type AdminConfig struct {
	DefaultUsername string `yaml:"default_username"`
	DefaultPassword string `yaml:"default_password"`
}

// DefaultConfig возвращает конфигурацию с разумными значениями по умолчанию.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:               "0.0.0.0",
			Port:               8080,
			BaseURL:            "http://localhost:8080",
			DefaultLanguage:    "ru",
			JWTSecret:          "boyan-secret-default-change-me-in-production-32b",
			JWTExpirationHours: 168,
			CORSAllowedOrigins: []string{"http://localhost:3000", "http://localhost:3001"},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			SQLite: SQLiteConfig{
				Path:          "./data/opds.db",
				EnableWAL:     true,
				BusyTimeoutMS: 5000,
				CacheSizeKB:   64000,
			},
		},
		Storage: StorageConfig{
			LibraryDir:   "./library",
			WatchDir:     "./import",
			PathTemplate: "{Author}/{Series}/{SeriesIndex:02d} - {Title}.{ext}",
			Watcher: WatcherConfig{
				Enabled:                 true,
				SettleDelaySeconds:      5,
				DeleteSourceAfterImport: false,
				QuarantineDuplicates:    true,
			},
		},
		OPDS: OPDSConfig{
			Title:                 "Боян",
			Subtitle:              "Каталог электронных книг",
			PageSize:              50,
			EnableOPDSv1:          true,
			EnableOPDSv2:          true,
			AllowAnonymousReading: false,
			StreamFromZIP:         true,
		},
		Metadata: MetadataConfig{
			AutoEnrichOnImport: true,
			ProvidersPriority:  []string{"fantlab", "livelib", "openlibrary"},
			CacheCoversLocally: true,
			CoverThumbnailSize: 400,
			CoverCacheMaxMB:    500,
			CoverCacheDir:      "./data/cache/covers",
		},
		Admin: AdminConfig{
			DefaultUsername: "admin",
			DefaultPassword: "adminpassword",
		},
		Telegram: TelegramConfig{
			Enabled:        false,
			BotToken:       "",
			AllowedUserIDs: []int64{},
		},
	}
}

// Load загружает конфигурацию из указанного пути с поддержкой ENV переопределений.
func Load(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	// 1. Попытка прочитать файл
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			// Если файл не найден, пробуем config.example.yaml
			if os.IsNotExist(err) && configPath == "config.yaml" {
				if exData, exErr := os.ReadFile("config.example.yaml"); exErr == nil {
					data = exData
				} else {
					return nil, fmt.Errorf("read config file: %w", err)
				}
			} else {
				return nil, fmt.Errorf("read config file: %w", err)
			}
		}

		if len(data) > 0 {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parse yaml config: %w", err)
			}
		}
	}

	// 2. ENV переопределения (12-Factor App)
	applyEnvOverrides(cfg)

	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if host := os.Getenv("BOYAN_SERVER_HOST"); host != "" {
		cfg.Server.Host = host
	}
	if baseURL := os.Getenv("BOYAN_SERVER_BASE_URL"); baseURL != "" {
		cfg.Server.BaseURL = baseURL
	}
	if defLang := os.Getenv("BOYAN_SERVER_DEFAULT_LANGUAGE"); defLang != "" {
		cfg.Server.DefaultLanguage = defLang
	}
	if portStr := os.Getenv("BOYAN_SERVER_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			cfg.Server.Port = p
		}
	}
	if dbPath := os.Getenv("BOYAN_DATABASE_SQLITE_PATH"); dbPath != "" {
		cfg.Database.SQLite.Path = dbPath
	}
	if libDir := os.Getenv("BOYAN_STORAGE_LIBRARY_DIR"); libDir != "" {
		cfg.Storage.LibraryDir = libDir
	}
	if watchDir := os.Getenv("BOYAN_STORAGE_WATCH_DIR"); watchDir != "" {
		cfg.Storage.WatchDir = watchDir
	}
	if secret := os.Getenv("BOYAN_SERVER_JWT_SECRET"); secret != "" {
		cfg.Server.JWTSecret = secret
	}
	if origins := os.Getenv("BOYAN_SERVER_CORS_ORIGINS"); origins != "" {
		cfg.Server.CORSAllowedOrigins = strings.Split(origins, ",")
	}
	if adminUser := os.Getenv("BOYAN_ADMIN_USERNAME"); adminUser != "" {
		cfg.Admin.DefaultUsername = adminUser
	}
	if adminPass := os.Getenv("BOYAN_ADMIN_PASSWORD"); adminPass != "" {
		cfg.Admin.DefaultPassword = adminPass
	}
	if tgEnabled := os.Getenv("BOYAN_TELEGRAM_ENABLED"); tgEnabled != "" {
		cfg.Telegram.Enabled = tgEnabled == "true" || tgEnabled == "1"
	}
	if tgToken := os.Getenv("BOYAN_TELEGRAM_BOT_TOKEN"); tgToken != "" {
		cfg.Telegram.BotToken = tgToken
	}
	if tgUsers := os.Getenv("BOYAN_TELEGRAM_ALLOWED_USER_IDS"); tgUsers != "" {
		parts := strings.Split(tgUsers, ",")
		var ids []int64
		for _, p := range parts {
			if id, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64); err == nil {
				ids = append(ids, id)
			}
		}
		cfg.Telegram.AllowedUserIDs = ids
	}
}
