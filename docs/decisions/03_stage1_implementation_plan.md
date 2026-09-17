# План реализации проекта Next-Gen OPDS Suite (Boyan) — Этап 1: Core Engine, Storage & WSL2

## 1. Обзор задачи и контекст

Разработка ядра автономного Headless-бэкенда электронной библиотеки **Next-Gen OPDS Suite** («Боян») на языке Go, ориентированного на минимальное потребление оперативной памяти (20–50 МБ RAM), быструю работу со SQLite (WAL-режим) без CGO-зависимостей, глубокую поддержку русскоязычных форматов книг (FB2 / FB2.ZIP с любыми кодировками) и нативное развертывание в виде `systemd`-сервиса в среде **WSL2 (Ubuntu)**.

### Согласованные решения в ходе интервью:
1. **Объем первого спринта:** Базовое ядро Go-бэкенда (конфигурация, хранилище SQLite, потоковый парсер FB2/ZIP, генератор превью обложек) + полный комплект скриптов для развертывания сервера в локальном WSL2 под управлением `systemd`.
2. **HTTP-роутер:** **`go-chi/chi/v5`** — идиоматичный, без оверхеда памяти, с нативной изоляцией sub-routers (`/api/v1/`, `/opds/v1/`, `/opds/v2/`) и прямой поддержкой потоковой отдачи `io.Copy(w, reader)`.
3. **Слой БД:** **`sqlx` + встроенные SQL-миграции (`embed.FS`)** на базе чистого Go-драйвера **`modernc.org/sqlite`** (без необходимости GCC/MinGW). Раздельный пул соединений: 1 писатель (`SetMaxOpenConns(1)`), $N$ читателей для предотвращения `SQLITE_BUSY`.
4. **Полнотекстовый поиск:** Таблицы **SQLite FTS5** с токенайзером `unicode61` и префиксным поиском для русского и английского языков.
5. **Парсинг FB2 и кодировки:** Потоковый XML-парсер с автоматическим декодером любых кодировок (`windows-1251`, `cp866`, `koi8-r`, `utf-8`, `iso-8859-*`) через `golang.org/x/net/html/charset`.
6. **Кеш обложек и защита диска:** Превью обложек генерируются на лету через чистый Go (`golang.org/x/image/draw`), сохраняются в локальный дисковый кеш с заголовками `ETag` / `304 Not Modified`. Механизм LRU-очистки защищает диск от переполнения по настраиваемому лимиту памяти (параметр `metadata.cover_cache_max_mb`).
7. **Модель файлов:** Смешанный режим — поддержка отдельных файлов (`.fb2`, `.epub`, `.fb2.zip`), архитектурно схема БД уже содержит поле `archive_inner_path` для будущих массивных библиотечных архивов.
8. **Авторизация:** Создание дефолтного администратора (`admin`) при первом старте с хэшированием пароля `bcrypt`.
9. **WSL2 Деплой:** Кросс-компиляция под `linux/amd64`, unit-файл `boyan.service` и автоматизированные bash-скрипты установки и управления службой через `systemctl`.

---

## 2. Предлагаемые изменения и архитектура файлов

```
backend/
├── cmd/
│   └── server/
│       └── main.go                 # Точка входа: инициализация slog, config, db, http server, graceful shutdown
├── internal/
│   ├── config/
│   │   └── config.go               # Структуры конфигурации, загрузка из YAML + переопределение ENV
│   ├── models/
│   │   ├── book.go                 # Доменные модели: Book, Author, Series, Genre, BookFile
│   │   └── user.go                 # Модели: User, Role, Session
│   ├── storage/
│   │   ├── migrations/
│   │   │   ├── 0001_init.up.sql    # Инициализация схемы SQLite: таблицы, внешние ключи, индексы
│   │   │   └── 0002_fts5.up.sql    # Виртуальная таблица FTS5 books_fts и триггеры синхронизации
│   │   ├── migrator.go             # Легковесный автоприменитель миграций из embed.FS
│   │   ├── sqlite.go               # Пул соединений SQLite (1 writer, N readers, WAL, PRAGMA)
│   │   ├── book_repo.go            # Репозиторий книг, авторов, серий и файлов на sqlx
│   │   └── user_repo.go            # Репозиторий пользователей и проверка хешей паролей bcrypt
│   ├── parsers/
│   │   ├── fb2/
│   │   │   ├── parser.go           # Потоковый парсер FB2 с поддержкой CP1251/UTF-8 и извлечением base64 обложки
│   │   │   └── genres.go           # Справочник и нормализатор русских/английских жанров FictionBook
│   │   ├── zip/
│   │   │   └── reader.go           # Извлечение FB2 из архива .fb2.zip на лету в памяти
│   │   └── cover/
│   │       ├── processor.go        # Pure-Go ресайз обложек в WebP/JPEG
│   │       └── cache.go            # Дисковый кеш обложек с LRU-очисткой по квоте размера
│   └── api/
│       ├── router.go               # Сборка роутера chi/v5, подключение middleware
│       ├── middleware/
│       │   ├── logger.go           # Middleware структурированного логирования slog с Request-ID
│       │   └── recoverer.go        # Graceful recovery от паник
│       └── handlers/
│           ├── health.go           # Эндпоинты /health, /api/v1/ping
│           └── books.go            # Эндпоинты получения книг и стриминга обложек / файлов
├── go.mod                          # Определение модуля boyan/backend
└── go.sum                          # Хеши зависимостей

scripts/
├── wsl/
│   ├── boyan.service               # Systemd unit-файл для автозапуска в WSL2
│   ├── deploy.sh                   # Скрипт развертывания бинарника и конфига в /opt/boyan и регистрация сервиса
│   └── service.sh                  # Скрипт управления (status, logs, restart, stop)
└── build-wsl.ps1                   # PowerShell-скрипт кросс-компиляции Go под linux/amd64
```

---

## 3. Детальный состав модулей

### Слой 1: Конфигурация и точка входа
- **`backend/internal/config/config.go`**:
  Парсинг YAML файла (по умолчанию `config.yaml` с откатом к `config.example.yaml`). Поддержка переопределения через ENV переменные (`BOYAN_SERVER_PORT`, `BOYAN_DATABASE_SQLITE_PATH` и т.д.).
- **`backend/cmd/server/main.go`**:
  Настройка `slog.NewTextHandler` (или `JSONHandler`), инициализация пула SQLite, применение миграций, инициализация админа, запуск HTTP-сервера на `chi`, перехват сигналов `SIGINT` и `SIGTERM` для плавного завершения (Graceful Shutdown) с закрытием соединений БД.

---

### Слой 2: База данных SQLite и миграции
- **`backend/internal/storage/sqlite.go`**:
  Открытие базы через драйвер `modernc.org/sqlite`.
  Выполнение обязательных PRAGMA:
  - `PRAGMA journal_mode = WAL;`
  - `PRAGMA synchronous = NORMAL;`
  - `PRAGMA foreign_keys = ON;`
  - `PRAGMA busy_timeout = 5000;`
  - `PRAGMA cache_size = -64000;` (64 МБ кеша страниц в памяти)
  Пул: `writerDB.SetMaxOpenConns(1)` (строго 1 писатель), `readerDB.SetMaxOpenConns(10)`.
- **`backend/internal/storage/migrations/0001_init.up.sql`**:
  Создание таблиц:
  - `books`: id (UUID/TEXT), title, original_title, annotation, language, published_date, publisher, isbn, cover_cached, created_at, updated_at
  - `authors`: id, name, sort_name
  - `book_authors`: book_id, author_id, role, author_order (PRIMARY KEY `book_id, author_id, role`)
  - `series`: id, name, sort_name
  - `book_series`: book_id, series_id, series_index (REAL, например `1.0`, `2.5`)
  - `genres`: code (PK), name_ru, name_en, category_ru, category_en
  - `book_genres`: book_id, genre_code
  - `book_files`: id, book_id, format (`fb2`, `epub`, и т.д.), file_path, archive_inner_path, file_size, sha256, created_at
  - `users`: id, username, password_hash, role (`admin`, `user`), is_active, created_at
- **`backend/internal/storage/migrations/0002_fts5.up.sql`**:
  Виртуальная таблица полнотекстового поиска:
  - `CREATE VIRTUAL TABLE books_fts USING fts5(title, original_title, annotation, tokenize='unicode61 remove_diacritics 2');`
  Триггеры автоматической синхронизации FTS при вставке, обновлении и удалении записей в `books`.

---

### Слой 3: Парсер FB2/ZIP, кодировки и обработчик обложек
- **`backend/internal/parsers/fb2/parser.go`**:
  Парсинг XML с использованием `xml.NewDecoder(r)`.
  Настройка `decoder.CharsetReader = charset.NewReaderLabel` для прозрачного декодирования кодировок Windows-1251, CP866, KOI8-R, ISO-8859-5 в UTF-8.
  Извлечение:
  - `<description><title-info>`: `book-title`, `author` (first, middle, last), `genre`, `annotation`, `sequence` (name + number), `lang`, `date`, `coverpage`.
  - `<publish-info>`: `publisher`, `year`, `isbn`.
  - `<binary id="...">`: поиск бинарной обложки, декодирование `base64`.
- **`backend/internal/parsers/zip/reader.go`**:
  Потоковое открытие `zip.NewReader`. Поиск внутри архива файла с расширением `.fb2`. Потоковая передача `io.Reader` в парсер FB2 без записи на диск.
- **`backend/internal/parsers/cover/cache.go`**:
  Дисковый кеш обложек в `data/cache/covers/`.
  Генерация миниатюр: сохранение в JPEG/WebP заданного максимального размера (400px по умолчанию) с использованием высококачественной билинейной интерполяции `draw.BiLinear`.
  Механизм очистки LRU (Least Recently Used): проверка суммарного объема кеша; если размер превышает `cover_cache_max_mb`, удаляются файлы с наиболее старым временем доступа `atime`.

---

### Слой 4: HTTP API и роутер (chi/v5)
- **`backend/internal/api/router.go`**:
  Маршрутизатор `chi.NewRouter()` с цепочкой middleware:
  - `middleware.RequestID`
  - `middleware.RealIP`
  - Структурированное логирование запросов через `slog`
  - `middleware.Recoverer`
  - CORS middleware
  Маршруты:
  - `GET /health` — проверка доступности сервера и БД
  - `GET /api/v1/ping` — статус сервиса и версия
  - `GET /api/v1/covers/{id}` — отдача превью обложки с поддержкой заголовков кеширования `ETag` и `304 Not Modified`
  - `GET /api/v1/books` — получение списка книг с пагинацией

---

### Слой 5: Скрипты развертывания в WSL2
- **`scripts/build-wsl.ps1`**:
  PowerShell-скрипт компиляции Go под Linux:
  `$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -ldflags="-s -w" -o bin/boyan-linux ./cmd/server`
- **`scripts/wsl/boyan.service`**:
  Unit-файл systemd для фонового автозапуска сервиса.
- **`scripts/wsl/deploy.sh`**:
  Bash-скрипт для Ubuntu WSL2:
  - Создание директорий `/opt/boyan`, `/opt/boyan/data`, `/opt/boyan/library`, `/opt/boyan/import`.
  - Копирование скомпилированного бинарника `boyan-linux` и `config.yaml`.
  - Установка unit-файла в `/etc/systemd/system/boyan.service`.
  - `systemctl daemon-reload && systemctl enable --now boyan`.
- **`scripts/wsl/service.sh`**:
  Утилита быстрого управления в WSL2: `./service.sh status`, `logs`, `restart`, `stop`.

---

## 4. План верификации (Verification Plan)

### Автоматизированные тесты (Go Tests)
1. **Тесты парсера FB2/ZIP (`internal/parsers/fb2`):**
   - Парсинг тестового FB2-файла в кодировке UTF-8.
   - Парсинг тестового FB2-файла в кодировке Windows-1251 (проверка корректности кириллицы).
   - Потоковый парсинг архива `.fb2.zip` без распаковки на диск.
   - Корректное извлечение метаданных: автор, название, серия, дробный номер серии, аннотация, жанры.
   - Извлечение Base64 обложки и ресайз до 400px.
   - Проверка LRU-очистки кеша при превышении квоты.
2. **Тесты базы данных и FTS5 (`internal/storage`):**
   - Создание базы SQLite в режиме WAL с `modernc.org/sqlite`.
   - Применение миграций `0001_init` и `0002_fts5`.
   - Вставка тестовых книг и проверка автозаполнения виртуальной таблицы FTS5.
   - Поиск книги по названию и автору через полнотекстовый индекс.
   - Проверка параллельного чтения (10 горутин) при активной транзакции записи.
3. **Тесты API (`internal/api`):**
   - Запрос `GET /health` (проверка ответа `{"status":"ok","db":"connected"}`).
   - Запрос `GET /api/v1/covers/{id}` с проверкой заголовка `ETag` и повторный запрос с `If-None-Match` (проверка кода `304 Not Modified`).

### Ручная верификация в Windows и WSL2
1. **Локальный запуск на Windows:**
   - Компиляция `go build ./cmd/server` под Windows.
   - Запуск `server.exe --config config.yaml`.
   - Проверка создания базы данных `data/opds.db` и появления админа в логах.
2. **Развертывание и проверка в WSL2 (Ubuntu):**
   - Выполнение скрипта сборки `scripts/build-wsl.ps1`.
   - Запуск развертывания в Ubuntu WSL2 через `wsl -d Ubuntu -- bash /mnt/d/git/AntiGravity/boyan/scripts/wsl/deploy.sh`.
   - Проверка статуса сервиса через `wsl -d Ubuntu -- systemctl status boyan`.
   - Проверка ответа сервера из хостовой Windows: `curl http://localhost:8080/health`.
