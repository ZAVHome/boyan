<p align="center">
  <img src="media/Boyan_logo_transparent.svg" alt="Boyan" width="130" />
</p>

# Boyan

<p align="center">
  <img src="docs/assets/banner.jpg" alt="Boyan" width="100%" />
</p>

[![Release](https://img.shields.io/badge/release-v0.0.2--beta-blue.svg)](https://github.com/ZAVHome/boyan/releases)
[![Go Version](https://img.shields.io/badge/go-1.23%2B-00ADD8.svg?logo=go)](https://golang.org)
[![Vue 3](https://img.shields.io/badge/vue-3.5-4FC08D.svg?logo=vue.js)](https://vuejs.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![CI](https://github.com/ZAVHome/boyan/actions/workflows/ci.yml/badge.svg)](https://github.com/ZAVHome/boyan/actions)

> 📚 **Boyan** is a modern, modular, open-source Next-Gen OPDS Suite for eBook cataloging, in-browser reading, and network distribution supporting OPDS v1.2 and OPDS v2.0 protocols.

[Русская версия (README.md)](README.md)

---

## 📖 Project Documentation

| Category | Russian Guide | English Guide |
| :--- | :--- | :--- |
| **For Readers & End Users** | 📖 [Руководство пользователя](docs/user_guide_ru.md) | 📖 [User Guide](docs/user_guide_en.md) |
| **For Engineers & Admins** | 🛠️ [Техническое руководство](docs/technical_guide_ru.md) | 🛠️ [Technical Guide](docs/technical_guide_en.md) |
| **Production VPS Deployment** | 🚀 [Установка на Linux VPS](docs/vps_installation_guide_ru.md) | 🚀 [Linux VPS Deployment](docs/vps_installation_guide_en.md) |
| **Architectural Decisions** | 🏛️ [Реестр ADR](docs/decisions/README.md) | 🏛️ [Decisions Register](docs/decisions/README.md) |
| **Interactive API Explorer (Swagger)** | — | 🌐 `http://localhost:8080/api/v1/docs/index.html` |

---

## 🌟 Key Features

- **Strict Headless Backend:** Ultra-lightweight Go (Golang) core using merely **4.5–5.7 MB RAM** in production. 100% CGO-free build using pure Go SQLite driver.
- **Rich Format Support:** Native parsing of FB2, FB2.ZIP, EPUB, MOBI, PDF, DJVU. Instant in-memory streaming extraction of zipped archives with zero temporary disk files and Zip Slip protection.
- **Dual-Standard OPDS:**
  - **v1.2 (Atom/XML):** 100% compatible with PocketBook, AlReaderX, KOReader, Moon+ Reader, FBReader.
  - **v2.0 (JSON-LD):** Built for modern reading clients supporting the OPDS 2.0 standard.
  - **OpenSearch:** Built-in XML descriptor for seamless search bar integration across mobile e-reading apps.
- **Two Dedicated Frontends (Vue 3 + Tailwind CSS):**
  - **Desktop SPA (`frontends/web-desktop`):** High-density catalog, in-browser reader, duplicate & quarantine moderation, extensible i18n.
  - **Mobile PWA (`frontends/web-mobile`):** Touch gestures with swipe navigation, offline library (`IndexedDB`) for reading with zero network connectivity, and high-contrast zero-animation mode for Android E-Ink devices.
- **Mandatory Day/Night Themes:** Light, Dark, OLED Pure Black, Sepia, and E-Ink High-Contrast.
- **Two-Way Telegram Bot:** Fast book search and download via inline keyboards, plus effortless book ingestion by dropping files into the bot chat.
- **Automated Ingestion:** Background directory monitoring via `fsnotify` with debounce settle delay to prevent reading partial writes.
- **Calibre Library Compatibility:** Direct read-only import from existing Calibre collections (`metadata.db`).

---

## 📁 Repository Structure

```text
├── backend/                  # Autonomous Headless Go Backend
│   ├── cmd/server/           # Application entrypoint (main.go)
│   ├── docs/swagger/         # Swagger / OpenAPI 2.0/3.0 specifications
│   └── internal/
│       ├── api/              # Chi v5 router and REST controllers
│       ├── auth/             # JWT tokens and bcrypt password hashing
│       ├── config/           # YAML configuration & CLI flags parser
│       ├── importer/calibre/ # Calibre SQLite importer (metadata.db)
│       ├── models/           # Domain models (Book, Author, Series, Tag, User)
│       ├── opds/             # OPDS v1.2 (Atom) & v2.0 (JSON-LD) feed generators
│       ├── parsers/          # Streaming FB2/ZIP parsers & cover LRU cache
│       ├── services/         # Business logic services
│       ├── storage/          # SQLite WAL + migrations + FTS5 full-text search
│       ├── telegram/         # Two-way Telegram bot (Long Polling)
│       └── watcher/          # fsnotify file watcher & ingest pipeline
│
├── frontends/                # Independent Frontends
│   ├── web-desktop/          # Desktop & Tablet Web SPA (Vue 3, Pinia, Tailwind)
│   └── web-mobile/           # Mobile Touch-First PWA (Vue 3, PWA, IndexedDB Offline)
│
├── scripts/                  # Deployment Scripts
│   ├── build-wsl.ps1         # Cross-compile Linux binary for WSL2
│   └── wsl/                  # Systemd service installer scripts
│
├── docs/                     # Documentation Hub
│   ├── user_guide_ru.md      # User Guide (RU)
│   ├── user_guide_en.md      # User Guide (EN)
│   ├── technical_guide_ru.md # Technical & Architecture Guide (RU)
│   ├── technical_guide_en.md # Technical & Architecture Guide (EN)
│   └── decisions/            # Architecture Decision Records (ADR 01–14)
│
├── docker-compose.yml        # Multi-container orchestration
├── config.example.yaml       # Configuration template
└── Makefile                  # Build, test, and run automation
```

---

## 🚀 Quick Start

### Option 1: Production Linux VPS Deployment (Recommended)

For ultra-lightweight installation on Ubuntu / Debian VPS (~5 MB RAM footprint) with pre-configured Nginx & SSL:
👉 **[Linux VPS Installation Guide](docs/vps_installation_guide_en.md)** (one-click pre-built release deployment with zero build tools on your server).

### Option 2: Docker Compose

1. Copy the configuration template:

   ```bash
   cp config.example.yaml config.yaml
   ```

2. Start all services:

   ```bash
   docker compose up -d
   ```

3. Access points:
   - **Desktop Web Interface:** `http://localhost:3000`
   - **Mobile PWA Interface:** `http://localhost:3001`
   - **OPDS v1.2 Feed:** `http://localhost:8080/opds/v1/feed.xml`
   - **OPDS v2.0 Catalog:** `http://localhost:8080/opds/v2/catalog.json`
   - **Interactive Swagger Docs:** `http://localhost:8080/api/v1/docs/index.html`

### Option 2: Local Development

1. **Start Backend:**

   ```bash
   make run-backend
   ```

2. **Start Desktop Frontend:**

   ```bash
   make run-desktop
   ```

3. **Start Mobile PWA Frontend:**

   ```bash
   make run-mobile
   ```

---

## 🧪 Running Tests

```bash
make test
```

All system modules are thoroughly verified with unit and integration tests (FB2/ZIP streaming parsers, cover LRU cache, SQLite migrations, FTS5 search queries, duplicate quarantine, and REST API routes).

---

## 📄 License

MIT License. Free for personal and commercial use.
