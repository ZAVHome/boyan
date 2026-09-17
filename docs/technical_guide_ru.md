# Техническое руководство: «Боян»

<img src="assets/logo.svg" align="right" width="90" alt="Боян" />

Техническая документация для системных администраторов, DevOps-инженеров и разработчиков проекта **«Боян»**.

---

## 📑 Содержание

1. [Архитектурный обзор (Strict Headless Architecture)](#1-архитектурный-обзор)
2. [Стек технологий и компоненты системы](#2-стек-технологий-и-компоненты-системы)
3. [Схема базы данных и полнотекстовый поиск](#3-схема-базы-данных-и-полнотекстовый-поиск)
4. [Конвейер инжеста и обработка файлов](#4-конвейер-инжеста-и-обработка-файлов)
5. [Спецификация протоколов OPDS v1.2 и v2.0](#5-спецификация-протоколов-opds-v12-и-v20)
6. [REST API и документация OpenAPI/Swagger](#6-rest-api-и-документация-openapiswagger)
7. [Подсистема импорта библиотек Calibre](#7-подсистема-импорта-библиотек-calibre)
8. [Архитектура Telegram-бота](#8-архитектура-telegram-бота)
9. [Руководство по установке, развертыванию и эксплуатации](#9-руководство-по-установке-развертыванию-и-эксплуатации)
10. [Справочник конфигурационного файла (config.yaml)](#10-справочник-конфигурационного-файла-configyaml)

---

## 1. Архитектурный обзор

Проект спроектирован в соответствии с принципом **Strict Headless Architecture** (строгое разделение бэкенда и пользовательских интерфейсов):

```text
                               ┌────────────────────────┐
                               │     Клиенты OPDS       │
                               │ PocketBook / KOReader  │
                               └───────────┬────────────┘
                                           │ OPDS v1.2 / v2.0
┌───────────────────────┐                  ▼                  ┌───────────────────────┐
│     Web Desktop       │◄─────────►┌─────────────┐◄─────────►│     Web Mobile        │
│   Vue 3 SPA (:3000)   │  REST API │    Боян     │  REST API │    Vue 3 PWA (:3001)  │
└───────────────────────┘           │   Бэкенд    │           └───────────────────────┘
                                    │  Go (:8080) │
┌───────────────────────┐           └──────┬──────┘           ┌───────────────────────┐
│     Telegram-бот      │◄─────────────────┼─────────────────►│   Calibre Importer    │
│  Двусторонний инжест  │                  ▼                  │  Чтение metadata.db   │
└───────────────────────┘         ┌─────────────────┐         └───────────────────────┘
                                  │   SQLite WAL    │
                                  │   + FTS5 Search │
                                  └─────────────────┘
```

### Ключевые архитектурные решения

1. **Автономный Go-бэкенд (Headless):** бэкенд не содержит монолитных HTML-шаблонов. Он предоставляет исключительно REST API (спецификация OpenAPI 3.0 / Swagger 2.0) и эндпоинты протоколов OPDS.
2. **Экстремальная легковесность:** сервер на Go потребляет всего **4.5–5.7 МБ RAM** в боевом режиме под управлением systemd, что позволяет запускать его на самых слабых одноплатных компьютерах (Raspberry Pi, Orange Pi) и виртуальных машинах с объемом ОЗУ от 128 МБ.
3. **CGO-Free сборка:** драйвер базы данных `modernc.org/sqlite` компилируется на 100% на чистом Go без зависимости от GCC/CGO, обеспечивая тривиальную кросс-компиляцию под любую ОС и архитектуру (Linux amd64/arm64, Windows, macOS).

---

## 2. Стек технологий и компоненты системы

| Компонент | Технология | Назначение |
| :--- | :--- | :--- |
| **Backend Core** | Go 1.23+ / 1.26+ | Многопоточный микросервер, бизнес-логика |
| **HTTP Маршрутизатор** | `go-chi/chi/v5` | Легковесный роутинг, middleware (CORS, Recoverer, RequestID, Slog) |
| **База данных** | SQLite 3 (`modernc.org/sqlite`) | Режим WAL (Write-Ahead Logging), внешние ключи, FTS5 поиск |
| **ORM / Data Access** | `jmoiron/sqlx` | Типизированная работа с SQL без тяжелых ORM-прослоек |
| **Потоковые парсеры** | `encoding/xml`, `archive/zip` | Парсинг FB2 и FB2.ZIP на лету без распаковки на диск |
| **Обработка обложек** | `golang.org/x/image` | Ресайз миниатюр, дисковый LRU-кэш |
| **Файловый монитор** | `fsnotify/fsnotify` | Inotify-демон автоимпорта новых файлов с settle-delay |
| **Desktop Frontend** | Vue 3 + Tailwind CSS + Pinia | Адаптивный SPA для ПК и планшетов (порт 3000) |
| **Mobile Frontend** | Vue 3 + Vite PWA + IndexedDB | Сенсорный PWA с офлайн-хранилищем и E-Ink профилем (порт 3001) |
| **API Документация** | Swagger UI (`http-swagger/v2`) | Интерактивный каталог API на `/api/v1/docs` |

---

## 3. Схема базы данных и полнотекстовый поиск

База данных SQLite инициализируется автоматическими миграциями из встроенной файловой системы `embed.FS` (`backend/internal/storage/migrations/001_init.sql`).

### Режим работы SQLite

- `PRAGMA journal_mode=WAL;` — параллельное чтение несколькими горутинами без блокировки писателем.
- `PRAGMA synchronous=NORMAL;` — высокая производительность при надежной сохранности данных.
- `PRAGMA foreign_keys=ON;` — каскадное обеспечение ссылочной целостности.
- `PRAGMA busy_timeout=5000;` — ожидание освобождения блокировки до 5 секунд.

### Основные таблицы

- `books`: метаданные (название, аннотация, язык, дата, путь к файлу, размер, хэш SHA-256, формат, наличие обложки).
- `authors`, `book_authors`: авторы (ФИО) и связи Many-to-Many.
- `series`, `book_series`: серии/циклы книг и порядковый номер тома.
- `tags`, `book_tags`: жанры книги (с каноническими кодами и русскими названиями).
- `quarantine`: изолированные файлы (дубликаты по SHA-256 или поврежденные архивы) с указанием причины и статуса.
- `users`: учетные записи с безопасным хешированием паролей `bcrypt` и ролями (`admin`, `user`).
- `shelves`: персональные полки пользователя (`reading`, `to-read`, `finished`).
- `reading_progress`: текущий прогресс (глава, процент прочтения, временная метка).

### Полнотекстовый поиск (FTS5)

Создана виртуальная таблица `books_fts` с токенизатором `porter` и триггерами на добавление, обновление и удаление:

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

Запросы используют синтаксис `books_fts MATCH ?` с автоматическим добавлением префиксного поиска `*`, что позволяет находить книги по неполным словам и словоформам на русском и английском языках.

---

## 4. Конвейер инжеста и обработка файлов

```text
[Новый файл на диске] ──► [fsnotify] ──► [Settle Delay: 500ms] ──► [SHA-256 Checksum]
                                                                          │
                        ┌─────────────────────────────────────────────────┴───────────────────┐
                        │ Уникален                                                            │ Дубликат
                        ▼                                                                     ▼
                [Потоковый парсер]                                                    [Таблица quarantine]
           (FB2 XML / ZIP In-Memory)                                                  (Причина: duplicate)
                        │
         ┌──────────────┴──────────────┐
         ▼ Успешно                     ▼ Ошибка XML
  [Запись в SQLite WAL]       [Таблица quarantine]
  [Кэширование обложки LRU]   (Причина: corrupted)
  [Обновление FTS5 индекса]
```

### Особенности реализации

1. **Settle Delay (Антидребезг):** при копировании больших файлов по сети (FTP/Samba) события файловой системы группируются с задержкой в 500 мс, гарантируя, что парсер не начнет чтение недописанного файла.
2. **Потоковое чтение ZIP без дискового оверхеда:** файл `.fb2.zip` открывается потоком `zip.Reader` в оперативной памяти. Извлекается только поток XML, без создания временных файлов на диске. Реализована строгая проверка пути (защита от уязвимости Zip Slip).
3. **Автодетектирование кодировок:** модуль `html/charset` декодирует исходные данные из `windows-1251`, `cp866`, `koi8-r` в UTF-8 без потерь символов.
4. **LRU-кэш обложек:** обложки масштабируются до максимального разрешения 400x500px и сохраняются в директорию кэша. Демон очистки отслеживает суммарный размер папки и при превышении лимита (по умолчанию 500 МБ) удаляет наименее востребованные миниатюры.

---

## 5. Спецификация протоколов OPDS v1.2 и v2.0

### OPDS v1.2 (Atom/XML)

- **Эндпоинт:** `GET /opds/v1/feed.xml`
- **MIME-тип:** `application/atom+xml;profile=opds-catalog;kind=acquisition`
- **Структура фида:**
  - Навигационные ссылки (`rel="subsection"`) на каталоги авторов, серий, новинок и случайных книг.
  - Ссылка на OpenSearch дескриптор:

    ```xml
    <link rel="search" href="/opds/v1/search.xml" type="application/opensearchdescription+xml"/>
    ```

  - В элементах `<entry>` ссылки на скачивание книги формируются с корректными MIME-типами:
    - FB2: `application/x-fictionbook+xml`
    - FB2.ZIP: `application/x-fictionbook+zip`
    - EPUB: `application/epub+zip`
  - Ссылка на обложку: `rel="http://opds-spec.org/image"` с ссылкой на `/api/v1/covers/{id}`.

### OPDS v2.0 (JSON-LD)

- **Эндпоинт:** `GET /opds/v2/catalog.json`
- **MIME-тип:** `application/opds+json`
- Формирует спецификацию OPDS 2.0 с объектами `metadata`, `links`, `navigation` и массивом публикаций `publications`.

### Мультиязычность и локализация каталогов (Backend i18n)

Бэкенд поддерживает динамическую локализацию фидов OPDS v1.2 и v2.0 на лету (подробности в [ADR-15](decisions/15_backend_i18n_architecture.md)):

- **E-Ink читалки (PocketBook, KOReader, Moon+ Reader):** поддерживают прямое переключение языка через URL фида:
  - `GET /opds/v1/feed.xml?lang=en` — все разделы (*By Authors*, *By Series*, *Recent Additions*, *Search Results*) и дескриптор OpenSearch отдаются на английском языке.
  - `GET /opds/v1/feed.xml?lang=ru` — разделы отдаются на русском языке.
- **Определение языка по заголовкам:** клиенты, отправляющие `Accept-Language: en-US,en;q=0.9`, автоматически получают англоязычные фиды.
- **Язык по умолчанию:** настраивается в `config.yaml` параметром `server.default_language: "ru"`.

---

## 6. REST API и документация OpenAPI/Swagger

Все эндпоинты начинаются с префикса `/api/v1/`. Интерактивная документация доступна по адресу:
👉 `http://localhost:8080/api/v1/docs/index.html`

### Мультиязычность и формат ошибок REST API

Ошибки REST API отдаются в стандартизированном гибридном формате:

```json
{
  "error": "Неверное имя пользователя или пароль",
  "code": "AUTH_INVALID_CREDENTIALS"
}
```

- Поле `error` содержит локализованный текст на языке пользователя (на основе `?lang=` или заголовка `Accept-Language`).
- Поле `code` содержит стабильный машиночитаемый идентификатор в `SCREAMING_SNAKE_CASE`.
- Это позволяет простым внешним клиентам выводить текст ошибки напрямую, а умным клиентам (Vue 3 SPA/PWA) при необходимости переопределять сообщения через словарь `vue-i18n`.

### Основные группы маршрутов

#### Системные

- `GET /health` — проверка работоспособности сервиса и статуса подключения к БД.
- `GET /api/v1/ping` — быстрый ping.

#### Аутентификация и пользователи

- `POST /api/v1/auth/register` — регистрация нового пользователя.
- `POST /api/v1/auth/login` — вход в систему, возвращает JWT-токен.
- `GET /api/v1/auth/me` — получение профиля текущего пользователя (требуется заголовок `Authorization: Bearer <token>`).

#### Каталог книг

- `GET /api/v1/books` — получение списка книг с пагинацией (`page`, `limit`) и поиском (`search`).
- `GET /api/v1/books/{id}` — получение подробной карточки книги со всеми связями.
- `GET /api/v1/books/{id}/download` — скачивание исходного файла книги.
- `GET /api/v1/covers/{id}` — отдача обложки (поддерживает заголовок `If-None-Match` и ответ `304 Not Modified`).

#### Таксономия

- `GET /api/v1/authors` — список авторов с количеством книг.
- `GET /api/v1/series` — список серий/циклов.
- `GET /api/v1/tags` — список жанров.

#### Полки и прогресс

- `GET /api/v1/shelves` — книги пользователя, сгруппированные по полкам (`reading`, `to-read`, `finished`).
- `POST /api/v1/shelves/{shelf}/books/{bookId}` — добавление/перемещение книги на полку.
- `GET /api/v1/progress/{bookId}` — получение сохраненного прогресса чтения.
- `POST /api/v1/progress/{bookId}` — сохранение прогресса (`chapter`, `percent`).

#### Администрирование

- `GET /api/v1/admin/quarantine` — список файлов в карантине.
- `POST /api/v1/admin/quarantine/{id}/restore` — принудительное восстановление файла в каталог.
- `DELETE /api/v1/admin/quarantine/{id}` — удаление файла из карантина и с диска.
- `POST /api/v1/admin/import/calibre` — запуск пакетного импорта библиотеки Calibre.

---

## 7. Подсистема импорта библиотек Calibre

Модуль `boyan/internal/importer/calibre` позволяет мигрировать или синхронизировать коллекцию из Calibre.

- **Безопасный доступ:** база Calibre `metadata.db` открывается в строго монопольном режиме чтения SQLite:

  ```go
  db, err := sqlx.Open("sqlite", "file:"+dbPath+"?mode=ro")
  ```

  Это исключает риск повреждения файла `metadata.db` работающим приложением Calibre.
- **Маппинг данных:**
  - Авторы из таблицы `authors` и связи `books_authors_link`.
  - Серии и порядковые номера из `series` и `books_series_link`.
  - Жанры и теги из `tags`.
  - Аннотации из `comments` с автоматической очисткой от HTML-тегов (`<p>`, `<div>`, `<span>`).
  - Форматы из таблицы `data`: поддержка одновременного учета нескольких форматов одного издания.
- **Способы запуска:**
  1. Через аргумент командной строки:

     ```bash
     boyan --import-calibre /path/to/calibre/library
     ```

  2. Через REST API:

     ```bash
     curl -X POST http://localhost:8080/api/v1/admin/import/calibre \
          -H "Authorization: Bearer <ADMIN_JWT_TOKEN>" \
          -H "Content-Type: application/json" \
          -d '{"calibre_dir": "/var/books/calibre"}'
     ```

---

## 8. Архитектура Telegram-бота

Демон бота (`boyan/internal/telegram`) работает на базе стандартного HTTP-клиента Go в режиме Long Polling (`getUpdates`):

- **Независимость от фреймворков:** минимальный оверхед по памяти (менее 1 МБ дополнительного ОЗУ).
- **Двунаправленная логика:**
  1. **Исходящий трафик (Поиск и выдача):**
     - Обработка команды `/search <запрос>` или произвольного текста.
     - Формирование клавиатуры `InlineKeyboardMarkup` с кнопками `download:<format>:<book_id>`.
     - Отправка документа методом `sendDocument` в потоковом режиме.
  2. **Входящий трафик (Инжест книг):**
     - Перехват сообщений с вложенными документами (`message.Document`).
     - Скачивание файла по `file_id` через `getFile`.
     - Передача файла в стандартный конвейер `watcher.ProcessFile`.
     - Отправка пользователю понятного ответа об успешном добавлении или обнаружении дубликата.

---

## 9. Руководство по установке, развертыванию и эксплуатации

> 📘 **Полное руководство по развертыванию на боевых серверах:**  
> Подробные пошаговые инструкции по настройке VPS с безопасным пользователем, Nginx Reverse Proxy, Certbot SSL и автоматической сборкой см. в документе:  
> 🚀 **[Руководство по установке на Linux VPS](vps_installation_guide_ru.md)**.

### Вариант А: Развертывание в Linux / WSL2 под управлением systemd

1. **Компиляция бинарника:**

   ```bash
   cd backend
   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ../bin/boyan cmd/server/main.go
   ```

2. **Создание каталогов:**

   ```bash
   sudo mkdir -p /opt/boyan /var/lib/boyan /var/cache/boyan/covers /var/books
   sudo cp ../bin/boyan /opt/boyan/boyan
   sudo cp ../config.example.yaml /opt/boyan/config.yaml
   ```

3. **Регистрация службы systemd (`/etc/systemd/system/boyan.service`):**

   ```ini
   [Unit]
   Description=Boyan Server
   After=network.target

   [Service]
   Type=simple
   User=root
   WorkingDirectory=/opt/boyan
   ExecStart=/opt/boyan/boyan --config /opt/boyan/config.yaml
   Restart=always
   RestartSec=5
   LimitNOFILE=65535

   [Install]
   WantedBy=multi-user.target
   ```

4. **Запуск и проверка:**

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable boyan
   sudo systemctl start boyan
   sudo systemctl status boyan
   ```

### Вариант Б: Развертывание через Docker Compose

В корне репозитория подготовлен `docker-compose.yml`:

```bash
# Копирование конфигурации
cp config.example.yaml config.yaml

# Сборка и запуск контейнеров
docker compose up -d --build
```

Доступные порты:

- `8080` — Core Go Backend & OPDS
- `3000` — Web Desktop Frontend
- `3001` — Web Mobile PWA Frontend

---

## 10. Справочник конфигурационного файла (config.yaml)

```yaml
server:
  host: "0.0.0.0"               # IP адрес привязки сервиса
  port: 8080                    # Порт HTTP API и OPDS
  read_timeout_sec: 15          # Таймаут на чтение запроса клиентом
  write_timeout_sec: 60         # Таймаут на отдачу больших файлов
  cors_allowed_origins:
    - "*"                       # Разрешенные источники CORS

storage:
  db_path: "/var/lib/boyan/boyan.db"       # Путь к файлу базы данных SQLite
  books_dir: "/var/books"                  # Корневая папка библиотеки с книгами
  cover_cache_dir: "/var/cache/boyan/covers" # Папка для кэширования миниатюр обложек
  cover_cache_max_mb: 500                  # Максимальный размер кэша обложек в МБ

auth:
  jwt_secret: "CHANGE_THIS_SECRET_KEY"     # Секретный ключ подписи JWT
  token_ttl_hours: 720                     # Время жизни токена (30 дней)

watcher:
  enabled: true                 # Включение демона автосканирования папки книг
  settle_delay_ms: 500          # Задержка антидребезга перед началом парсинга

telegram:
  enabled: false                # Включение Telegram-бота
  bot_token: ""                 # Токен бота от @BotFather
  allowed_chat_ids: []          # Белый список ID пользователей (пусто = доступно всем)
```
