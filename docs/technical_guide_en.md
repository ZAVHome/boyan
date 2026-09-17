# Technical Guide: Boyan

<img src="assets/logo.svg" align="right" width="90" alt="Boyan" />

Technical and architectural documentation for system administrators, DevOps engineers, and backend/frontend developers working with **Boyan**.

---

## 📑 Table of Contents

1. [Architectural Overview (Strict Headless Architecture)](#1-architectural-overview)
2. [Technology Stack & System Components](#2-technology-stack--system-components)
3. [Database Schema & Full-Text Search Engine](#3-database-schema--full-text-search-engine)
4. [Ingestion Pipeline & File Processing](#4-ingestion-pipeline--file-processing)
5. [OPDS Protocols Specification (v1.2 & v2.0)](#5-opds-protocols-specification-v12--v20)
6. [REST API Reference & OpenAPI/Swagger Docs](#6-rest-api-reference--openapiswagger-docs)
7. [Calibre Library Importer Subsystem](#7-calibre-library-importer-subsystem)
8. [Telegram Bot Architecture](#8-telegram-bot-architecture)
9. [Installation, Deployment & Operations Guide](#9-installation-deployment--operations-guide)
10. [Configuration Reference (config.yaml)](#10-configuration-reference-configyaml)

---

## 1. Architectural Overview

Boyan strictly enforces a **Headless Architecture**, separating data storage and business logic from user interfaces:

```text
                               ┌────────────────────────┐
                               │      OPDS Clients      │
                               │ PocketBook / KOReader  │
                               └───────────┬────────────┘
                                           │ OPDS v1.2 / v2.0
┌───────────────────────┐                  ▼                  ┌───────────────────────┐
│     Web Desktop       │◄─────────►┌─────────────┐◄─────────►│     Web Mobile        │
│   Vue 3 SPA (:3000)   │  REST API │    Boyan    │  REST API │    Vue 3 PWA (:3001)  │
└───────────────────────┘           │   Backend   │           └───────────────────────┘
                                    │  Go (:8080) │
┌───────────────────────┐           └──────┬──────┘           ┌───────────────────────┐
│     Telegram Bot      │◄─────────────────┼─────────────────►│   Calibre Importer    │
│ Bidirectional Ingest  │                  ▼                  │  Reads metadata.db    │
└───────────────────────┘         ┌─────────────────┐         └───────────────────────┘
                                  │   SQLite WAL    │
                                  │   + FTS5 Search │
                                  └─────────────────┘
```

### Key Architectural Tenets

1. **Autonomous Go Core (Headless):** The Go backend does not render monolithic HTML templates. It exposes a clean REST API compliant with OpenAPI 3.0 / Swagger 2.0 and native OPDS protocol feeds.
2. **Ultra-Low Memory Footprint:** The Go service consumes merely **4.5–5.7 MB RAM** in production under systemd. It can easily run on low-power single-board computers (Raspberry Pi, Orange Pi) or cloud instances with as little as 128 MB RAM.
3. **CGO-Free Builds:** Utilizing `modernc.org/sqlite` ensures 100% pure Go compilation with zero GCC or CGO toolchain requirements, making cross-compilation for any target (Linux amd64/arm64, Windows, macOS) instantaneous and trouble-free.

---

## 2. Technology Stack & System Components

| Component | Technology | Role |
| :--- | :--- | :--- |
| **Backend Core** | Go 1.23+ / 1.26+ | High-throughput asynchronous service & business logic |
| **HTTP Router** | `go-chi/chi/v5` | Lightweight routing, middleware (CORS, Recoverer, RequestID, Slog) |
| **Database** | SQLite 3 (`modernc.org/sqlite`) | WAL (Write-Ahead Logging), foreign keys, FTS5 full-text search |
| **ORM / Data Access** | `jmoiron/sqlx` | Type-safe SQL execution without heavy ORM abstraction overhead |
| **Streaming Parsers** | `encoding/xml`, `archive/zip` | In-memory FB2 and FB2.ZIP parsing with zero temporary disk files |
| **Cover Processing** | `golang.org/x/image` | Thumbnail image scaling and disk LRU cache manager |
| **File Watcher** | `fsnotify/fsnotify` | Background file monitor with debounce settle delay |
| **Desktop Frontend** | Vue 3 + Tailwind CSS + Pinia | Responsive SPA for desktop and tablet displays (port 3000) |
| **Mobile Frontend** | Vue 3 + Vite PWA + IndexedDB | Touch-first PWA with offline storage and E-Ink mode (port 3001) |
| **API Documentation** | Swagger UI (`http-swagger/v2`) | Interactive API explorer hosted at `/api/v1/docs` |

---

## 3. Database Schema & Full-Text Search Engine

The SQLite database is initialized through automatic migrations embedded in binary assets via `embed.FS` (`backend/internal/storage/migrations/001_init.sql`).

### SQLite Configuration

- `PRAGMA journal_mode=WAL;` — Concurrent multi-goroutine reads without writer blocking.
- `PRAGMA synchronous=NORMAL;` — Optimal balance between high disk write throughput and crash resilience.
- `PRAGMA foreign_keys=ON;` — Cascading relational integrity.
- `PRAGMA busy_timeout=5000;` — Graceful wait time (up to 5s) before returning busy errors.

### Core Tables

- `books`: Core book metadata (title, annotation, language, release date, path, file size, SHA-256 hash, format, cover existence).
- `authors`, `book_authors`: Author records and Many-to-Many junction mappings.
- `series`, `book_series`: Book series / cycles and volume sequence indices.
- `tags`, `book_tags`: Canonical genre codes and localized names.
- `quarantine`: Isolated files (duplicates or corrupted archives) with reasons and review statuses.
- `users`: User profiles with `bcrypt` password hashing and role-based access (`admin`, `user`).
- `shelves`: Virtual user shelves (`reading`, `to-read`, `finished`).
- `reading_progress`: Reading state tracking (current chapter, completion percentage, last update timestamp).

### Full-Text Search (FTS5)

A virtual table `books_fts` is configured with the `porter` tokenizer and synchronized via triggers on `INSERT`, `UPDATE`, and `DELETE`:

```sql
CREATE VIRTUAL TABLE books_fts USING fts5(
    title,
    annotation,
    author_names,
    series_names,
    content='books',
    content_rowid='id'
);
```

Searches are executed using `books_fts MATCH ?` with prefix matching (`*`), allowing instant matching across Russian and English word forms.

---

## 4. Ingestion Pipeline & File Processing

```text
[New File On Disk] ──► [fsnotify] ──► [Settle Delay: 500ms] ──► [SHA-256 Checksum]
                                                                        │
                        ┌───────────────────────────────────────────────┴─────────────────┐
                        │ Unique                                                          │ Duplicate
                        ▼                                                                 ▼
                [Streaming Parser]                                                [Quarantine Table]
           (FB2 XML / ZIP In-Memory)                                              (Reason: duplicate)
                        │
         ┌──────────────┴──────────────┐
         ▼ Success                     ▼ Parsing Error
  [Write to SQLite WAL]       [Quarantine Table]
  [Cover LRU Cache Write]     (Reason: corrupted)
  [Update FTS5 Index]
```

### Key Engineering Safeguards

1. **Settle Delay (Debouncing):** When large archives are uploaded over FTP, WebDAV, or Samba, file system events are debounced with a 500ms delay to prevent parsing partially transferred files.
2. **In-Memory Streaming Without Disk Spooling:** `.fb2.zip` archives are opened via an in-memory `zip.Reader`. The parser extracts the XML stream directly into RAM buffers without ever writing temporary files to `/tmp`. Strict path normalization prevents Zip Slip vulnerabilities.
3. **Automatic Encoding Detection:** The `html/charset` module dynamically transcodes input streams from `windows-1251`, `cp866`, or `koi8-r` into valid UTF-8.
4. **Cover LRU Disk Cache:** Cover images are scaled to a maximum dimension of 400x500px and persisted in the cache directory. When disk usage exceeds the configured quota (default: 500 MB), the least recently accessed images are evicted automatically.

---

## 5. OPDS Protocols Specification (v1.2 & v2.0)

### OPDS v1.2 (Atom/XML)

- **Endpoint:** `GET /opds/v1/feed.xml`
- **MIME Type:** `application/atom+xml;profile=opds-catalog;kind=acquisition`
- **Structure:**
  - Navigation links (`rel="subsection"`) pointing to authors, series, recent releases, and random selections.
  - OpenSearch descriptor link:

    ```xml
    <link rel="search" href="/opds/v1/search.xml" type="application/opensearchdescription+xml"/>
    ```

  - Acquisition links within `<entry>` tags map to proper MIME types:
    - FB2: `application/x-fictionbook+xml`
    - FB2.ZIP: `application/x-fictionbook+zip`
    - EPUB: `application/epub+zip`
  - Cover image link: `rel="http://opds-spec.org/image"` pointing to `/api/v1/covers/{id}`.

### OPDS v2.0 (JSON-LD)

- **Endpoint:** `GET /opds/v2/catalog.json`
- **MIME Type:** `application/opds+json`
- Implements the modern OPDS 2.0 schema, including `metadata`, `links`, `navigation`, and a `publications` collection.

### Multilingual Feeds & Catalog Localization (Backend i18n)

Boyan supports dynamic, on-the-fly OPDS feed localization (see [ADR-15](decisions/15_backend_i18n_architecture.md)):

- **E-Ink Readers (PocketBook, KOReader, Moon+ Reader):** Readers can configure feed language directly in the catalog URL:
  - `GET /opds/v1/feed.xml?lang=en` — all sections (*By Authors*, *By Series*, *Recent Additions*, *Search Results*) and the OpenSearch descriptor are served in English.
  - `GET /opds/v1/feed.xml?lang=ru` — all sections are served in Russian.
- **Header-based Detection:** Clients sending `Accept-Language: en-US,en;q=0.9` automatically receive English feeds.
- **Default Language:** Configured in `config.yaml` via `server.default_language: "ru"`.

---

## 6. REST API Reference & OpenAPI/Swagger Docs

All REST endpoints are prefixed with `/api/v1/`. Interactive documentation and test forms are available at:
👉 `http://localhost:8080/api/v1/docs/index.html`

### Multilingual Error Formatting (Backend i18n)

REST API error responses use a standardized hybrid payload:

```json
{
  "error": "Invalid username or password",
  "code": "AUTH_INVALID_CREDENTIALS"
}
```

- The `error` field contains human-readable text in the client's language (resolved via `?lang=` or `Accept-Language`).
- The `code` field contains a stable machine-readable identifier in `SCREAMING_SNAKE_CASE`.
- External and lightweight clients can display `error` directly, while SPA/PWA clients can map `code` to `vue-i18n` strings when desired.

### Route Groups

#### System

- `GET /health` — Service health and database connection verification.
- `GET /api/v1/ping` — Fast network ping.

#### Authentication & Profiles

- `POST /api/v1/auth/register` — Create a new user account.
- `POST /api/v1/auth/login` — Authenticate and receive a JWT bearer token.
- `GET /api/v1/auth/me` — Inspect profile of the authenticated user.

#### Catalog

- `GET /api/v1/books` — Paginated book listing with `search`, `page`, and `limit` parameters.
- `GET /api/v1/books/{id}` — Full book metadata and relation hierarchy.
- `GET /api/v1/books/{id}/download` — Stream original book archive.
- `GET /api/v1/covers/{id}` — Cover thumbnail delivery (supports `If-None-Match` and `304 Not Modified`).

#### Taxonomy

- `GET /api/v1/authors` — Authors list with associated book counts.
- `GET /api/v1/series` — Book series listing.
- `GET /api/v1/tags` — Canonical genre taxonomy.

#### Shelves & Reading Progress

- `GET /api/v1/shelves` — User books partitioned by shelf (`reading`, `to-read`, `finished`).
- `POST /api/v1/shelves/{shelf}/books/{bookId}` — Move book to a shelf.
- `GET /api/v1/progress/{bookId}` — Retrieve reading position.
- `POST /api/v1/progress/{bookId}` — Record reading progress (`chapter`, `percent`).

#### Administration

- **Dashboard & Host Metrics:**
  - `GET /api/v1/admin/dashboard` — Aggregated system overview: host metrics (RAM, CPU, goroutines, uptime, disk stats), SQLite engine (file size, WAL status, FTS5), cover thumbnail cache, and Telegram bot health.
  - `POST /api/v1/admin/maintenance/checkpoint` — Explicit SQLite WAL checkpoint (`PRAGMA wal_checkpoint(TRUNCATE)`).
  - `POST /api/v1/admin/maintenance/purge-cache` — Purge thumbnail image cache on disk.
  - `GET /api/v1/admin/logs` — In-memory ring buffer log viewer (last 200 `slog` events) with severity level and substring filtering.
- **User Management:**
  - `GET /api/v1/admin/users` — Paginated user directory with username and role search.
  - `POST /api/v1/admin/users` — Create user with specified role (`admin`, `user`, `restricted`).
  - `PUT /api/v1/admin/users/{id}` — Update user role and active status.
  - `PUT /api/v1/admin/users/{id}/password` — Administrative password override.
  - `DELETE /api/v1/admin/users/{id}` — Delete user account (with self-deletion guard).
  - `POST /api/v1/auth/register` — Public user registration (enabled when `allow_public_registration` is true).
- **Book Curation & Metadata:**
  - `GET /api/v1/admin/books` — Extended book inventory with disk paths and raw metadata.
  - `PUT /api/v1/admin/books/{id}` — Update book metadata (title, authors, series, genres, annotation, publisher, language, year) with SQLite FTS5 re-indexing.
  - `DELETE /api/v1/admin/books/{id}?delete_files=true|false` — Remove book with optional physical disk file deletion.
  - `POST /api/v1/admin/books/batch` — Batch actions (bulk deletion, bulk genre/series assignment, cover regeneration).
  - `POST /api/v1/admin/books/{id}/regenerate-cover` — Force extract cover from the original book file.
- **Storage & Task Manager:**
  - `POST /api/v1/admin/tasks/scan-watch` — Trigger background scan of incoming directory (`watch_dir`).
  - `POST /api/v1/admin/tasks/rescan-library` — Trigger background rescan of the full library (`library_dir`).
  - `GET /api/v1/admin/tasks` — Inspect active and completed background tasks and their progress.
  - `POST /api/v1/admin/tasks/{id}/cancel` — Cancel a running background task.
  - `POST /api/v1/admin/import/calibre` — Trigger batch Calibre library import.
- **Quarantine Moderation:**
  - `GET /api/v1/admin/quarantine` — List quarantined duplicate files.
  - `POST /api/v1/admin/quarantine/{id}/restore` — Force import quarantined file into the library.
  - `DELETE /api/v1/admin/quarantine/{id}` — Permanently delete file from quarantine and disk.
- **System Settings:**
  - `GET /api/v1/admin/settings` — Fetch active system configuration.
  - `PUT /api/v1/admin/settings` — Validate and persist updated settings to `config.yaml`.
  - `POST /api/v1/admin/settings/reload` — Hot reload internal services without server restart.

---

## 7. Calibre Library Importer Subsystem

The `boyan/internal/importer/calibre` module imports existing collections from Calibre libraries.

- **Read-Only Concurrency:** The Calibre `metadata.db` database is opened in strict read-only mode:

  ```go
  db, err := sqlx.Open("sqlite", "file:"+dbPath+"?mode=ro")
  ```

  This eliminates any risk of database corruption even if the Calibre desktop app is actively running.
- **Metadata Ingestion:**
  - Extracts authors from `authors` and `books_authors_link`.
  - Extracts series and sequence indices from `series` and `books_series_link`.
  - Extracts tags from `tags`.
  - Strips HTML tags (`<p>`, `<div>`, `<span>`) from `comments` fields.
  - Links all physical book format files listed in `data`.
- **Execution Options:**
  1. CLI Flag:

     ```bash
     boyan --import-calibre /path/to/calibre/library
     ```

  2. REST API:

     ```bash
     curl -X POST http://localhost:8080/api/v1/admin/import/calibre \
          -H "Authorization: Bearer <ADMIN_TOKEN>" \
          -H "Content-Type: application/json" \
          -d '{"calibre_dir": "/var/books/calibre"}'
     ```

---

## 8. Telegram Bot Architecture

The Telegram daemon (`boyan/internal/telegram`) communicates directly with the Telegram Bot API using native Go HTTP long-polling (`getUpdates`):

- **Zero Heavy Frameworks:** Minimal memory overhead (< 1 MB RAM).
- **Dual Pipeline:**
  1. **Outbound Flow (Search & Download):**
     - Handles `/search <query>` or raw text queries.
     - Builds dynamic `InlineKeyboardMarkup` buttons encoded as `download:<format>:<book_id>`.
     - Streams book files via `sendDocument`.
  2. **Inbound Flow (Book Ingestion):**
     - Captures file documents sent to the bot (`message.Document`).
     - Downloads payload via Telegram `getFile`.
     - Passes file to the standard `watcher.ProcessFile` pipeline.
     - Sends immediate user feedback confirming success or notifying of duplicates.

---

## 9. Installation, Deployment & Operations Guide

> 📘 **Comprehensive Production VPS Deployment Guide:**  
> For step-by-step instructions on setting up a Linux VPS with an unprivileged system user, Nginx Reverse Proxy, Certbot SSL, and zero-build release deployment, see:  
> 🚀 **[Linux VPS Installation Guide](vps_installation_guide_en.md)**.

### Option A: Linux / WSL2 Deployment Under systemd

1. **Build the Linux Binary:**

   ```bash
   cd backend
   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ../bin/boyan cmd/server/main.go
   ```

2. **Directory Structure Setup (FHS 3.0 Standard):**

   ```bash
   sudo mkdir -p /opt/boyan /var/lib/boyan/data /var/lib/boyan/library /var/lib/boyan/import /var/lib/boyan/quarantine /var/cache/boyan/covers
   sudo cp ../bin/boyan /opt/boyan/boyan
   sudo cp ../config.example.yaml /opt/boyan/config.yaml
   ```

3. **Register systemd Service (`/etc/systemd/system/boyan.service`):**

   ```ini
   [Unit]
   Description=Boyan Server
   After=network.target

   [Service]
   Type=simple
   User=boyan
   Group=boyan
   WorkingDirectory=/opt/boyan
   ExecStart=/opt/boyan/boyan --config /opt/boyan/config.yaml
   Restart=always
   RestartSec=3s
   StateDirectory=boyan
   CacheDirectory=boyan
   ProtectSystem=full
   ProtectHome=true
   NoNewPrivileges=true
   PrivateTmp=true
   LimitNOFILE=65535

   [Install]
   WantedBy=multi-user.target
   ```

4. **Start & Verify:**

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable boyan
   sudo systemctl start boyan
   sudo systemctl status boyan
   ```

### Option B: Docker Compose Deployment

A complete multi-container configuration is provided in `docker-compose.yml`:

```bash
cp config.example.yaml config.yaml
docker compose up -d --build
```

Exposed ports:

- `8080` — Core Go Backend & OPDS feeds
- `3000` — Web Desktop Frontend
- `3001` — Web Mobile PWA Frontend

---

## 10. Configuration Reference (config.yaml)

```yaml
server:
  host: "0.0.0.0"               # Service binding interface
  port: 8080                    # HTTP API & OPDS port
  read_timeout_sec: 15          # Client request read timeout
  write_timeout_sec: 60         # File streaming write timeout
  cors_allowed_origins:
    - "*"                       # Allowed CORS origins

storage:
  db_path: "/var/lib/boyan/boyan.db"       # SQLite database location
  books_dir: "/var/books"                  # Library root directory
  cover_cache_dir: "/var/cache/boyan/covers" # Cover thumbnail cache folder
  cover_cache_max_mb: 500                  # Max cover cache quota in MB

auth:
  jwt_secret: "CHANGE_THIS_SECRET_KEY"     # JWT signature secret
  token_ttl_hours: 720                     # Token validity period (30 days)

watcher:
  enabled: true                 # Enable directory auto-scan daemon
  settle_delay_ms: 500          # File write settle debounce delay

telegram:
  enabled: false                # Enable Telegram bot daemon
  bot_token: ""                 # Token from @BotFather
  allowed_chat_ids: []          # Whitelist of Telegram Chat IDs (empty = public)
```
