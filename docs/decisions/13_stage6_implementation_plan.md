# План реализации проекта Next-Gen OPDS Suite (Boyan) — Этап 6: Calibre Importer & Двусторонний Telegram-бот

## 1. Обзор задачи и цели этапа

Цель **Этапа 6** — реализовать интеграцию с внешними библиотеками Calibre и двусторонний Telegram-бот для каталогизации, поиска, скачивания и входящей загрузки книг прямо из мессенджера:

1. **Calibre Library Importer (`backend/internal/importer/calibre`):**
   - Чтение и парсинг базы метаданных Calibre `metadata.db` (SQLite 3 в режиме Read-Only).
   - Извлечение авторов, серий, тегов/жанров, комментариев/аннотаций, идентификаторов (ISBN, Goodreads) и форматов файлов.
   - Сопоставление структуры файлов Calibre: `<calibre_path>/<author_folder>/<book_folder>/<file>.<format>` и обложек `cover.jpg`.
   - Интеграция с репозиториями Бояна: импорт книг с дедупликацией (пропуск дубликатов, объединение форматов).
   - Поддержка CLI-команды: `boyan --import-calibre /path/to/calibre` и REST API эндпоинта: `POST /api/v1/admin/import/calibre`.

2. **Двусторонний автономный Telegram-бот (`backend/internal/telegram`):**
   - Реализация нативного long-polling клиента к Telegram Bot API без тяжелых внешних зависимостей.
   - **Исходящий сценарий (Поиск и скачивание):**
     - Команды `/start`, `/help`, `/search <запрос>` или прямой ввод текста в чат.
     - Полнотекстовый поиск по базе SQLite FTS5 (названия, авторы, серии).
     - Выдача результатов с inline-кнопками для выбора формата скачивания (`[📖 FB2]`, `[📘 EPUB]`, `[📕 MOBI]`).
     - Отправка файла книги прямо в чат (`sendDocument`) с обложкой (`thumb`) и описанием.
   - **Входящий сценарий (Загрузка книг из чата):**
     - Приём отправленного пользователем файла (`.fb2`, `.fb2.zip`, `.epub`, `.mobi`, `.pdf`).
     - Автоматическое скачивание через `getFile` и передача в Ingest pipeline (`Watcher.ProcessFile`).
     - Оповещение пользователя о статусе: успешно импортировано, формат прикреплен к книге или отправлен в карантин.
   - **Безопасность и разграничение доступа:**
     - Поддержка белого списка `allowed_user_ids` в `config.yaml` / ENV.
     - Режим открытого бота или приватного семейного архива.

---

## 2. Архитектура и структура модулей

```
backend/
├── cmd/server/main.go               # Регистрация CLI флага --import-calibre и запуск Telegram бота
├── internal/
│   ├── config/config.go             # Добавление TelegramConfig в конфигурацию
│   ├── importer/
│   │   └── calibre/
│   │       ├── reader.go            # Чтение metadata.db Calibre через SQLite
│   │       ├── models.go            # Модели Calibre (Book, Author, Series, Tag, Data)
│   │       ├── importer.go          # Сервис импорта в репозиторий Boyan
│   │       └── importer_test.go     # Юнит-тесты с синтетической базой Calibre
│   ├── telegram/
│   │   ├── bot.go                   # Жизненный цикл бота (long polling, dispatcher)
│   │   ├── client.go                # HTTP клиент к Telegram Bot API (getUpdates, sendMessage, sendDocument)
│   │   ├── handlers.go              # Обработчики команд, поиска и inline кнопок
│   │   ├── upload.go                # Обработчик входящих файлов документов
│   │   └── bot_test.go              # Мок-тесты логики бота
│   └── api/
│       └── handlers/
│           └── calibre.go           # REST эндпоинт POST /api/v1/admin/import/calibre
```

---

## 3. План верификации

### Автоматизированные тесты
1. `importer_test.go`: создание временной базы Calibre `metadata.db` со связанными таблицами, выполнение импорта и проверка создания книг в базе Бояна.
2. `bot_test.go`: тестирование диспетчеризации сообщений, проверки прав `allowed_user_ids`, генерации inline-клавиатур форматов и обработки входящих файлов.
3. Общие тесты: `go test -v -race ./...`

### Интеграционная проверка
1. Проверка работы CLI `boyan --import-calibre <dir>` с импортом тестовой библиотеки Calibre.
2. Проверка работы REST API `POST /api/v1/admin/import/calibre`.
3. Валидация запуска демона Telegram-бота при указании токена и проверка gracefully shutdown при остановке сервиса.
