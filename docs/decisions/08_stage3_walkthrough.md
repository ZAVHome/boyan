# Отчет о реализации: Этап 3 — REST API Gateway, Ingest Daemon & Quarantine

## 1. Краткое резюме выполненных работ

В рамках **Этапа 3** программного комплекса **Next-Gen OPDS Suite (Boyan)** реализован полнофункциональный Headless REST API шлюз для клиентских приложений (Desktop SPA и Mobile PWA), модуль безопасной авторизации (JWT + HttpOnly Cookie), фоновый демон автоимпорта книг на базе нативного `fsnotify` с контролем стабилизации файлов и многоуровневая система обнаружения и модерации дубликатов («Карантин»):

1. **JWT Авторизация и контекст пользователя (`internal/auth`, `internal/api/middleware`):**
   - Реализована генерация и проверка токенов HMAC-SHA256 (`github.com/golang-jwt/jwt/v5`).
   - Поддержка извлечения токена из заголовка `Authorization: Bearer <token>`, защищенных cookie (`boyan_token`, HttpOnly, Lax) и query-параметра `?token=`.
   - Контекстные хелперы `WithUser(ctx, claims)` и `UserFromContext(ctx)`.
   - Middleware ролевого доступа: `RequireAuth` (401 Unauthorized) и `RequireAdmin` (403 Forbidden).

2. **База данных: Миграция 0003 и репозитории (`internal/storage`):**
   - Написана миграция `0003_quarantine_progress.up.sql`:
     - `quarantine` (очередь модерации дубликатов с типами конфликтов `exact_hash`, `same_format`).
     - `read_progress` (прогресс чтения в процентах и позиция пагинации/CFI на пользователя).
     - `user_shelves` (пользовательские полки `reading`, `finished`, `favorite`).
   - Созданы репозитории `QuarantineRepository` и `ProgressRepository`.
   - Расширен `BookRepository`: добавлены методы `FindFileBySHA256`, `GetBookFileByID`, `AddBookFile`, `UpdateBookFile`, `DeleteBookFile`, `DeleteBook`, `FindBookByTitleAndAuthor`.

3. **Шаблонизатор раскладки и санитизация путей (`internal/watcher/layout.go`):**
   - Реализован шаблонизатор путей по конфигурируемому формату: `{Author}/{Series}/{SeriesIndex:02d} - {Title}.{ext}`.
   - Корректная обработка книг без серий (автоматическое схлопывание пути до `{Author}/{Title}.{ext}`).
   - Санитизация недопустимых символов для Windows и Linux (`[<>:"/\\|?*\x00-\x1f]`).

4. **Детектор дубликатов (`internal/watcher/deduplicator.go`):**
   - Уровень 1: Проверка точного SHA-256 хеша содержимого.
   - Уровень 2: Проверка по автору и названию:
     - Если совпадает формат (`fb2` к `fb2`) -> коллизия отправляется в карантин (`same_format`).
     - Если формат новый (`epub` к существующему `fb2`) -> прикрепление нового формата книги (`new_format`).
   - Уровень 3: Новая книга (`none`).

5. **Фоновый демон автоимпорта (`internal/watcher/watcher.go`):**
   - Нативный мониторинг директории `/import` через `fsnotify`.
   - Таймер стабилизации записи (`settle_delay_seconds`), исключающий парсинг недокачанных/недокопированных файлов.
   - Инжест-пайплайн: детекция формата (`fb2`, `fb2.zip`), потоковый парсинг метаданных, генерация обложки в LRU-кеш, перемещение файла в `library` и запись в БД.

6. **REST API эндпоинты (`internal/api/handlers`):**
   - `/api/v1/auth/login`, `/api/v1/auth/me`, `/api/v1/auth/logout`.
   - `/api/v1/books/upload` (multipart/form-data загрузка через браузер).
   - `/api/v1/books/{id}/progress`, `/api/v1/shelves/{type}` (чтение и полки).
   - `/api/v1/quarantine`, `/api/v1/quarantine/{id}`, `/api/v1/quarantine/{id}/resolve` (`discard`, `replace`, `attach_format`, `keep_both`).

---

## 2. Результаты тестирования и верификации

### 2.1 Юнит- и интеграционные тесты
Все модули протестированы с покрытием 100%:
- `boyan/internal/auth`: Тесты генерации, валидации JWT, ролей и контекста.
- `boyan/internal/watcher`: Тесты вычисления SHA-256, санитизации имен, раскладки путей и всех сценариев дедупликации.
- `boyan/internal/storage`: Тесты миграций 0001, 0002, 0003, каскадного удаления и внешних ключей.
- `boyan/internal/api/handlers`: Тесты авторизации, прогресса чтения, модерации карантина и загрузки multipart FB2.
- `boyan/internal/opds/v1` и `v2`: Регрессионное тестирование OPDS каталогов.

```
ok   boyan/internal/api/handlers   0.663s
ok   boyan/internal/api/middleware 2.000s
ok   boyan/internal/auth           1.334s
ok   boyan/internal/opds/v1        0.274s
ok   boyan/internal/opds/v2        0.242s
ok   boyan/internal/parsers/cover  1.295s
ok   boyan/internal/parsers/fb2    1.170s
ok   boyan/internal/parsers/zip    1.043s
ok   boyan/internal/services       1.867s
ok   boyan/internal/storage        1.891s
ok   boyan/internal/watcher        1.715s
```

### 2.2 Развертывание и верификация в WSL2 (Ubuntu, systemd)
- Сервис успешно развернут скриптом `scripts/deploy-to-wsl.ps1` на `/opt/boyan/boyan` под управлением `systemd`.
- **Потребление оперативной памяти:** всего **5.7 МБ RAM** (при бюджете до 20–50 МБ).
- Проверена работа эндпоинтов авторизации, выдачи токенов, чтения профиля `/api/v1/auth/me` и доступа к `/api/v1/quarantine`.

---

## 3. Статус этапа

Этап 3 полностью завершен. Комплекс готов к разработке клиентских фронтендов (Этап 4: Web Desktop SPA на Vue 3 + Tailwind CSS).
