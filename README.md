# Next-Gen OPDS Suite («Боян»)

<p align="center">
  <img src="docs/assets/banner.jpg" alt="Next-Gen OPDS Suite (Боян)" width="100%" />
</p>

[![Release](https://img.shields.io/badge/release-v0.0.1--beta-blue.svg)](https://github.com/ZAVHome/boyan/releases)
[![Go Version](https://img.shields.io/badge/go-1.23%2B-00ADD8.svg?logo=go)](https://golang.org)
[![Vue 3](https://img.shields.io/badge/vue-3.5-4FC08D.svg?logo=vue.js)](https://vuejs.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![CI](https://github.com/ZAVHome/boyan/actions/workflows/ci.yml/badge.svg)](https://github.com/ZAVHome/boyan/actions)

> 📚 Модульный программный комплекс нового поколения для каталогизации, управления, онлайн-чтения и сетевой дистрибуции электронных книг по протоколам OPDS v1.2 и OPDS v2.0.

[English version (README.en.md)](README.en.md)

---

## 📖 Документация проекта

| Раздел | Документ на русском | English Guide |
| :--- | :--- | :--- |
| **Для пользователей** | 📖 [Руководство пользователя](docs/user_guide_ru.md) | 📖 [User Guide](docs/user_guide_en.md) |
| **Для инженеров и админов** | 🛠️ [Техническое руководство](docs/technical_guide_ru.md) | 🛠️ [Technical Guide](docs/technical_guide_en.md) |
| **Развертывание на сервере** | 🚀 [Установка на Linux VPS](docs/vps_installation_guide_ru.md) | 🚀 [Linux VPS Deployment](docs/vps_installation_guide_en.md) |
| **Архитектурные решения** | 🏛️ [Реестр ADR](docs/decisions/README.md) | 🏛️ [Decisions Register](docs/decisions/README.md) |
| **Интерактивный API (Swagger)** | 🌐 `http://localhost:8080/api/v1/docs/index.html` | — |

---

## 🌟 Ключевые возможности

- **Strict Headless Backend:** легковесное ядро на Go (Golang), потребляющее всего **4.5–5.7 МБ RAM** в боевом режиме, со 100% CGO-free сборкой на базе чистого Go драйвера SQLite.
- **Поддержка форматов:** FB2, FB2.ZIP, EPUB, MOBI, PDF, DJVU. Потоковый парсинг архивов на лету без распаковки временных файлов на диск с защитой от уязвимости Zip Slip.
- **Двухстандартный OPDS:**
  - **v1.2 (Atom/XML):** 100% совместимость с PocketBook, AlReaderX, KOReader, Moon+ Reader, FBReader.
  - **v2.0 (JSON-LD):** поддержка читалок нового поколения по современному стандарту.
  - **OpenSearch:** встроенный XML-дескриптор поиска для мгновенной интеграции в приложения.
- **Два специализированных фронтенда (Vue 3 + Tailwind CSS):**
  - **Desktop SPA (`frontends/web-desktop`):** витрина, онлайн-ридер, модерация карантина и дубликатов, расширяемая интернационализация (i18n).
  - **Mobile PWA (`frontends/web-mobile`):** touch-интерфейс со свайпами, офлайн-библиотека (`IndexedDB`) для чтения без интернета и сверхконтрастный режим для E-Ink экранов.
- **Обязательные темы «День / Ночь»:** Light, Dark, OLED Black, Sepia и E-Ink Zero-Animation.
- **Двусторонний Telegram-бот:** поиск и скачивание книг прямо в чате, а также мгновенный инжест книг в каталог простой отправкой файла боту.
- **Автоматический инжест:** фоновый мониторинг папки с книгами через `fsnotify` с защитой от чтения недописанных файлов (settle-delay).
- **Совместимость с Calibre:** прямой импорт из существующих библиотек Calibre (`metadata.db`) в безопасном read-only режиме.

---

## 📁 Структура репозитория

```text
├── backend/                  # Автономный Go бэкенд (Headless Core)
│   ├── cmd/server/           # Точка входа в приложение (main.go)
│   ├── docs/swagger/         # Спецификация Swagger / OpenAPI 2.0/3.0
│   └── internal/
│       ├── api/              # Маршрутизатор Chi v5 и REST-контроллеры
│       ├── auth/             # JWT-токены и хэширование паролей bcrypt
│       ├── config/           # Парсинг YAML-конфигурации и флагов CLI
│       ├── importer/calibre/ # Импортер баз данных Calibre (metadata.db)
│       ├── models/           # Модели сущностей (Book, Author, Series, Tag, User)
│       ├── opds/             # Генераторы фидов OPDS v1.2 (Atom) и v2.0 (JSON-LD)
│       ├── parsers/          # Потоковые парсеры FB2, FB2.ZIP и LRU-кэш обложек
│       ├── services/         # Сервисный слой бизнес-логики
│       ├── storage/          # SQLite WAL + миграции + полнотекстовый поиск FTS5
│       ├── telegram/         # Двусторонний Telegram-бот (Long Polling)
│       └── watcher/          # Файловый монитор fsnotify и конвейер инжеста
│
├── frontends/                # Независимые фронтенды
│   ├── web-desktop/          # Desktop & Tablet Web SPA (Vue 3, Pinia, Tailwind)
│   └── web-mobile/           # Mobile Touch-First PWA (Vue 3, PWA, IndexedDB Offline)
│
├── scripts/                  # Скрипты развертывания
│   ├── build-wsl.ps1         # Кросс-компиляция Linux-бинарника под WSL2
│   └── wsl/                  # Скрипты установки службы systemd
│
├── docs/                     # Полная документация
│   ├── user_guide_ru.md      # Руководство пользователя (RU)
│   ├── user_guide_en.md      # User Guide (EN)
│   ├── technical_guide_ru.md # Техническое руководство (RU)
│   ├── technical_guide_en.md # Technical Guide (EN)
│   └── decisions/            # Реестр архитектурных решений (ADR 01–14)
│
├── docker-compose.yml        # Мультиконтейнерная оркестрация
├── config.example.yaml       # Пример конфигурации
└── Makefile                  # Сборка, тестирование и запуск
```

---

## 🚀 Быстрый старт

### Вариант 1: Развертывание на боевом сервере (Linux VPS) — Рекомендуемый

Для установки на Ubuntu / Debian VPS с минимальным потреблением памяти (~5 МБ RAM) и готовым Nginx:
👉 **[Подробная инструкция по установке на Linux VPS](docs/vps_installation_guide_ru.md)** (установка готового релиза в один клик без компиляторов на сервере).

### Вариант 2: Запуск через Docker Compose

1. Скопируйте файл конфигурации:

   ```bash
   cp config.example.yaml config.yaml
   ```

2. Запустите сервисы:

   ```bash
   docker compose up -d
   ```

3. Точки входа в системе:
   - **Десктопный интерфейс:** `http://localhost:3000`
   - **Мобильный PWA-интерфейс:** `http://localhost:3001`
   - **OPDS v1.2 каталог:** `http://localhost:8080/opds/v1/feed.xml`
   - **OPDS v2.0 каталог:** `http://localhost:8080/opds/v2/catalog.json`
   - **Интерактивный Swagger API:** `http://localhost:8080/api/v1/docs/index.html`

### Вариант 2: Локальная разработка

1. **Запуск бэкенда:**

   ```bash
   make run-backend
   ```

2. **Запуск десктопного фронтенда:**

   ```bash
   make run-desktop
   ```

3. **Запуск мобильного PWA фронтенда:**

   ```bash
   make run-mobile
   ```

---

## 🧪 Запуск тестов

```bash
make test
```

Все компоненты покрыты юнит- и интеграционными тестами (парсинг FB2/ZIP, LRU-кэш обложек, миграции SQLite, FTS5 поиск, обработка дубликатов и роуты REST API).

---

## 📄 Лицензия

MIT License. Свободно для личного и коммерческого использования.
