# Walkthrough: Результаты реализации Этапа 1 (Core Engine, Storage & WSL2 Deployment)

Мы успешно завершили реализацию **Этапа 1** проекта **Next-Gen OPDS Suite (Boyan)**.

---

## 1. Что было реализовано

### Архитектура бэкенда (Go 1.23+ / Headless)

* **Точка входа:** [backend/cmd/server/main.go](file:///d:/git/AntiGravity/boyan/backend/cmd/server/main.go) — инициализация `log/slog`, загрузка конфигурации, пула SQLite, применение встроенных миграций, автосоздание администратора по умолчанию, запуск HTTP-сервера и Graceful Shutdown.
* **Конфигурация:** [backend/internal/config/config.go](file:///d:/git/AntiGravity/boyan/backend/internal/config/config.go) — загрузка параметров из `config.yaml` с откатом к `config.example.yaml` и поддержкой переопределения через переменные окружения (стандарт 12-Factor App).
* **Доменные модели:** [backend/internal/models/book.go](file:///d:/git/AntiGravity/boyan/backend/internal/models/book.go) (`Book`, `Author`, `Series`, `Genre`, `BookFile`, `AuthorDetail`, `SeriesDetail`) и [backend/internal/models/user.go](file:///d:/git/AntiGravity/boyan/backend/internal/models/user.go).

### Хранилище данных SQLite (WAL + FTS5)

* **Пул соединений:** [backend/internal/storage/sqlite.go](file:///d:/git/AntiGravity/boyan/backend/internal/storage/sqlite.go) на базе чистого Go-драйвера `modernc.org/sqlite` (без зависимости от CGO и GCC) с раздельными пулами: 1 писатель (`SetMaxOpenConns(1)` во избежание `SQLITE_BUSY`) и пул параллельных читателей (`SetMaxOpenConns(10)`).
* **Встроенные миграции:** [backend/internal/storage/migrator.go](file:///d:/git/AntiGravity/boyan/backend/internal/storage/migrator.go) через директиву `//go:embed migrations/*.sql`:
  * `0001_init.up.sql`: таблицы книг, авторов, серий, жанров, файлов форматов, пользователей и индексы связей.
  * `0002_fts5.up.sql`: виртуальная таблица полнотекстового поиска `books_fts` с токенайзером `unicode61` (поддержка поиска по подстроке/префиксу для русского и английского языков).
* **Репозитории:**
  * [backend/internal/storage/book_repo.go](file:///d:/git/AntiGravity/boyan/backend/internal/storage/book_repo.go) — транзакционное сохранение книг со всеми связями, выборка по ID, пагинация и полнотекстовый поиск `SearchBooksFTS`.
  * [backend/internal/storage/user_repo.go](file:///d:/git/AntiGravity/boyan/backend/internal/storage/user_repo.go) — управление пользователями, безопасное хеширование паролей `bcrypt`, метод `EnsureAdminUser` для первого старта.

### Парсеры форматов и обработка обложек

* **Потоковый парсер FB2:** [backend/internal/parsers/fb2/parser.go](file:///d:/git/AntiGravity/boyan/backend/internal/parsers/fb2/parser.go) — парсинг XML с автоматическим декодированием любых кодировок (`windows-1251`, `cp866`, `koi8-r`, `utf-8`) через `golang.org/x/net/html/charset`, извлечение авторов, серий с индексами, аннотаций и бинарной обложки Base64.
* **Нормализатор жанров:** [backend/internal/parsers/fb2/genres.go](file:///d:/git/AntiGravity/boyan/backend/internal/parsers/fb2/genres.go) — справочник жанров FictionBook с автоматическим переводом на русский и английский языки.
* **Чтение ZIP на лету:** [backend/internal/parsers/zip/reader.go](file:///d:/git/AntiGravity/boyan/backend/internal/parsers/zip/reader.go) — извлечение FB2 из архивов `.fb2.zip` напрямую из памяти без распаковки временных файлов на диск, с защитой от уязвимости Zip Slip.
* **Ресайз обложек:** [backend/internal/parsers/cover/processor.go](file:///d:/git/AntiGravity/boyan/backend/internal/parsers/cover/processor.go) — масштабирование обложек средствами чистого Go (`draw.BiLinear`).
* **Дисковый кеш обложек с LRU:** [backend/internal/parsers/cover/cache.go](file:///d:/git/AntiGravity/boyan/backend/internal/parsers/cover/cache.go) — сохранение миниатюр на диске с отслеживанием времени последнего обращения и автоматической фоновой LRU-очисткой при превышении квоты (параметр `cover_cache_max_mb`).

### HTTP API на базе `go-chi/chi/v5`

* **Роутер и Middleware:** [backend/internal/api/router.go](file:///d:/git/AntiGravity/boyan/backend/internal/api/router.go) — структурированное логирование `slog` с `Request-ID`, защита от паник `Recoverer`, `RealIP`, CORS.
* **Эндпоинты:**
  * `GET /health` — проверка статуса сервиса и подключения к SQLite.
  * `GET /api/v1/ping` — быстрый ping.
  * `GET /api/v1/books` — получение списка книг с пагинацией и поиском.
  * `GET /api/v1/books/{id}` — получение книги со всеми метаданными.
  * `GET /api/v1/covers/{id}` — отдача миниатюры обложки с поддержкой `ETag` и `304 Not Modified`.

### Автоматизация развертывания в WSL2

* **Скрипт сборки под Linux:** [scripts/build-wsl.ps1](file:///d:/git/AntiGravity/boyan/scripts/build-wsl.ps1) — компиляция статического бинарника `bin/boyan-linux` (`GOOS=linux GOARCH=amd64 CGO_ENABLED=0`) размером всего **11.7 МБ**.
* **Служба systemd:** [scripts/wsl/boyan.service](file:///d:/git/AntiGravity/boyan/scripts/wsl/boyan.service).
* **Скрипт установки в WSL2:** [scripts/wsl/deploy.sh](file:///d:/git/AntiGravity/boyan/scripts/wsl/deploy.sh) — создание каталогов в `/opt/boyan`, копирование файлов, регистрация и запуск службы через `systemctl`.
* **Скрипт управления службой:** [scripts/wsl/service.sh](file:///d:/git/AntiGravity/boyan/scripts/wsl/service.sh) — команды `status`, `logs`, `restart`, `stop`, `start`.
* **Единый запуск из Windows:** [scripts/deploy-to-wsl.ps1](file:///d:/git/AntiGravity/boyan/scripts/deploy-to-wsl.ps1).

---

## 2. Результаты верификации и тестирования

### Автоматизированные тесты

Все юнит-тесты бэкенда (`go test -v ./...`) успешно пройдены:

* `boyan/internal/parsers/cover`:
  * `TestProcessCover`: масштабирование изображения 1000x800 до 400x320 — **PASS**
  * `TestCoverCache_LRU`: сохранение в кеш, проверка квоты и автоматическое удаление старых файлов по LRU — **PASS**
* `boyan/internal/parsers/fb2`:
  * `TestParseFB2_UTF8`: извлечение названия, авторов, серий с индексами, жанров и обложки — **PASS**
  * `TestParseFB2_Windows1251`: корректное декодирование кириллицы из CP1251 — **PASS**
  * `TestBase64Decode`: валидация декодирования бинарников — **PASS**
* `boyan/internal/parsers/zip`:
  * `TestFindFB2InZip`: потоковое нахождение и чтение файла `.fb2` внутри ZIP-архива — **PASS**
* `boyan/internal/storage`:
  * `TestStorage_FullFlow`: инициализация SQLite WAL, применение миграций из `embed.FS`, создание дефолтного админа с bcrypt, CRUD операции над книгами со связями, полнотекстовый поиск FTS5 по автору, аннотации и серии, параллельное чтение 50 горутинами — **PASS**
* `boyan/internal/api/handlers`:
  * `TestAPIHandlers`: проверка статус-кодов для `/health`, `/api/v1/ping`, `/api/v1/books` (пагинация), `/api/v1/books/{id}` и 404 — **PASS**

### Верификация службы в Ubuntu WSL2

Служба успешно развернута и проверена:

```text
● boyan.service - Next-Gen OPDS Suite (Boyan) Server
     Loaded: loaded (/etc/systemd/system/boyan.service; enabled; preset: enabled)
     Active: active (running)
   Main PID: 1314997 (boyan)
     Memory: 5.3M (peak: 5.4M)
```

Ответ сервера на хосте Windows:

```bash
$ curl -s http://localhost:8080/health
{"db":"connected","service":"Next-Gen OPDS Suite (Boyan)","status":"ok","timestamp":"2026-09-17T11:09:06Z","version":"0.1.0"}

$ curl -s http://localhost:8080/api/v1/ping
{"pong":true,"timestamp":"2026-09-17T11:09:06Z"}
```

> [!NOTE]
> **Потребление ресурсов:**
> Фактическое потребление памяти работающим процессом сервера со SQLite WAL, FTS5 и HTTP-роутером составляет **5.3 МБ RAM**, что в 4–10 раз лучше установленного лимита проекта (20–50 МБ).
