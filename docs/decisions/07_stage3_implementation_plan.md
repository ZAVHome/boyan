# План реализации проекта Next-Gen OPDS Suite (Boyan) — Этап 3: REST API Gateway, Ingest Daemon & Quarantine

## 1. Обзор задачи и цели этапа

Цель **Этапа 3** — создать полноценный API-шлюз для фронтендов (Desktop SPA и Mobile PWA), а также автоматизированный демон фонового инжеста книг с интеллектуальным контролем дубликатов:
1. **REST API Gateway (`/api/v1/`):**
   - Аутентификация: JWT Bearer токены + HttpOnly cookie (`/api/v1/auth/login`, `logout`, `me`).
   - Ролевой контроль (RBAC): Администратор (`admin`) и Пользователь (`user`).
   - Книги: расширенная фильтрация каталога, получение деталей, загрузка через multipart-форму (`/api/v1/books/upload`), редактирование и удаление.
   - Пользовательские полки и прогресс чтения: сохранение позиции чтения (`/api/v1/books/{id}/progress`), списки «Читаю сейчас» и «Избранное».
   - Управление авторами, сериями, жанрами.
2. **Фоновый демон автоимпорта (`Watch-Folder Daemon`):**
   - Нативный мониторинг каталога `/import` через `fsnotify`.
   - Контроль стабилизации файла (`settle_delay_seconds`) во избежание парсинга недокопированных файлов.
   - Раскладка файлов в библиотеке по конфигурируемому шаблону путей (`path_template` с санитизацией спецсимволов ОС).
3. **Интеллектуальный контроль дубликатов («Карантин»):**
   - Проверка точного совпадения по SHA-256 хешу.
   - Нечеткое сопоставление по нормализованным метаданным (`Автор + Название`).
   - Автоматическое прикрепление нового формата (если пришел EPUB к существующему FB2).
   - Помещение в очередь карантина (`quarantine`), если формат уже существует.
   - REST API управления карантином: слияние, замена, сохранение обоих или отклонение.

---

## 2. Архитектура и структура модулей

```
backend/
├── internal/
│   ├── auth/
│   │   ├── jwt.go                  # [NEW] Генерация и валидация JWT токенов (access + refresh)
│   │   └── context.go              # [NEW] Извлечение UserClaims из context.Context
│   ├── storage/
│   │   ├── migrations/
│   │   │   └── 0003_quarantine_progress.up.sql  # [NEW] Таблицы карантина и прогресса чтения
│   │   ├── quarantine_repo.go      # [NEW] Репозиторий очереди карантина дубликатов
│   │   └── progress_repo.go        # [NEW] Репозиторий прогресса чтения и полок
│   ├── watcher/
│   │   ├── watcher.go              # [NEW] Демон fsnotify с пулом воркеров и задержкой стабилизации
│   │   ├── layout.go               # [NEW] Движок именования путей по шаблону {Author}/{Series}/{Title}
│   │   └── deduplicator.go         # [NEW] Анализатор дубликатов (SHA-256 + Левенштейн метаданных)
│   └── api/
│       ├── middleware/
│       │   └── jwt_auth.go         # [NEW] Middleware проверки JWT и ролей
│       └── handlers/
│           ├── auth.go             # [NEW] Эндпоинты /api/v1/auth/login, /me, /logout
│           ├── upload.go           # [NEW] Эндпоинт загрузки файлов через браузер
│           ├── quarantine.go       # [NEW] Эндпоинты модерации дубликатов
│           └── progress.go         # [NEW] Эндпоинты отслеживания прогресса чтения
```

---

## 3. Детальный состав модулей

### Слой 1: JWT Авторизация и контекст пользователя
- `backend/internal/auth/jwt.go`:
  - `GenerateTokenPair(user *models.User, secret string, expireHours int) (*TokenPair, error)`
  - `ValidateToken(tokenString, secret string) (*UserClaims, error)`
  - Поддержка извлечения токена из заголовка `Authorization: Bearer <token>` и из cookie `boyan_token`.

### Слой 2: База данных — Карантин и Прогресс чтения
- `0003_quarantine_progress.up.sql`:
  - `quarantine`:
    - `id`, `file_path`, `file_size`, `sha256`, `title`, `authors`, `conflict_type` (`exact_hash`, `metadata_conflict`), `existing_book_id`, `created_at`
  - `read_progress`:
    - `user_id`, `book_id`, `format`, `progress_percent` (REAL), `position` (TEXT), `updated_at` (PRIMARY KEY `user_id, book_id`)
  - `user_shelves`:
    - `user_id`, `book_id`, `shelf_type` (`reading`, `finished`, `favorite`), `added_at`

### Слой 3: Ingest Daemon и Детекция дубликатов
- `backend/internal/watcher/layout.go`:
  - Шаблонизатор путей: замена макросов `{Author}`, `{Series}`, `{SeriesIndex:02d}`, `{Title}`, `{BookID}`, `{ext}`.
  - Безопасная очистка запрещенных символов Windows/Linux (`<>:"/\|?*`).
- `backend/internal/watcher/deduplicator.go`:
  - Проверка по хешу SHA-256 в `book_files`.
  - Нормализация `strings.ToLower`, удаление пунктуации и сравнение метаданных с существующими книгами.
- `backend/internal/watcher/watcher.go`:
  - Запуск монитора `fsnotify.NewWatcher` для папки `watch_dir`.
  - Ожидание завершения записи файла (контроль изменения размера и `settle_delay_seconds`).
  - Парсинг FB2/ZIP, сохранение обложки, раскладка файла в `library_dir`, сохранение в БД или отправка в карантин.

### Слой 4: REST API эндпоинты
- `/api/v1/auth/login` — авторизация по логину и паролю, выдача JWT токена.
- `/api/v1/auth/me` — профиль текущего пользователя.
- `/api/v1/books/upload` — загрузка книги напрямую через браузер с мгновенным парсингом и добавлением в библиотеку.
- `/api/v1/books/{id}/progress` — сохранение и получение прогресса чтения.
- `/api/v1/books/{id}/shelf` — добавление в избранное / статус прочтения.
- `/api/v1/quarantine` — список дубликатов для модерации.
- `/api/v1/quarantine/{id}/resolve` — разрешение конфликта (заменить существующую, объединить форматы, оставить обе, удалить).

---

## 4. План верификации

### Автоматизированные тесты
1. **Тесты JWT аутентификации:** генерация токена, валидация, проверка срока действия, доступ к защищенным эндпоинтам.
2. **Тесты шаблонизатора раскладки путей (`layout`):** проверка корректной санитизации опасных символов в названиях и сериях.
3. **Тесты детекции дубликатов (`deduplicator`):** выявление совпадений по SHA-256 и по автору/названию.
4. **Тесты загрузки книг через multipart:** отправка `.fb2` и `.fb2.zip` файла через HTTP POST `/api/v1/books/upload` и проверка появления книги в каталоге.
5. **Тесты прогресса чтения:** сохранение позиции и прогресса, получение списка прочитанных книг.
6. **Тесты разрешения карантина:** сценарии «слияние форматов» и «замена файла».

### Ручная верификация
- Копирование файла FB2/ZIP в папку `/import` и наблюдение за автоматическим появлением книги в базе и библиотеке.
- Вход в `/api/v1/auth/login` с логином `admin` и паролем `adminpassword`.
