# Реестр архитектурных решений (Architecture Decision Records — ADR)

В данной директории сохраняются аналитические сравнения, обоснования выбора технологий и детальные планы реализации, подготовленные и согласованные в ходе разработки комплекса **«Боян» (Boyan)**.

## Список документов

| № | Документ | Статус | Краткое описание |
| :--- | :--- | :--- | :--- |
| **01** | [01_http_router_comparison.md](01_http_router_comparison.md) | **Утверждено** | Сравнительный анализ HTTP-роутеров (`go-chi/chi/v5` vs `labstack/echo/v4` vs `net/http`). Утвержден `go-chi/chi/v5`. |
| **02** | [02_db_layer_comparison.md](02_db_layer_comparison.md) | **Утверждено** | Сравнительный анализ слоев работы с SQLite (`sqlx` + SQL миграции vs `GORM` vs `database/sql`). Утвержден `sqlx` + встроенные SQL-миграции (`embed.FS`). |
| **03** | [03_stage1_implementation_plan.md](03_stage1_implementation_plan.md) | **Выполнено** | Детальный план реализации Этапа 1: Core Engine, Storage (SQLite WAL+FTS5), парсер FB2/ZIP с любыми кодировками, кеш обложек с LRU и скрипты развертывания `systemd` в WSL2. |
| **04** | [04_stage1_walkthrough.md](04_stage1_walkthrough.md) | **Выполнено** | Отчет о реализации и результатах верификации Этапа 1 (тесты, развертывание в WSL2, потребление 5.3 МБ RAM). |
| **05** | [05_stage2_implementation_plan.md](05_stage2_implementation_plan.md) | **Выполнено** | Детальный план реализации Этапа 2: OPDS v1.2 (Atom/XML), OPDS v2.0 (JSON-LD), навигационные каналы, поиск OpenSearch, авторизация E-Ink и стриминг книг из ZIP. |
| **06** | [06_stage2_walkthrough.md](06_stage2_walkthrough.md) | **Выполнено** | Отчет о реализации и результатах верификации Этапа 2 (OPDS v1.2, OPDS v2.0, тесты, проверка в WSL2). |
| **07** | [07_stage3_implementation_plan.md](07_stage3_implementation_plan.md) | **Выполнено** | Детальный план реализации Этапа 3: REST API Gateway, JWT Auth, демон автоимпорта (fsnotify) папки /import и карантин дубликатов. |
| **08** | [08_stage3_walkthrough.md](08_stage3_walkthrough.md) | **Выполнено** | Отчет о реализации и результатах верификации Этапа 3 (REST API Gateway, JWT, Watcher fsnotify, дедупликация, 5.7 МБ RAM в WSL2). |
| **09** | [09_stage4_implementation_plan.md](09_stage4_implementation_plan.md) | **Выполнено** | Детальный план реализации Этапа 4: Web Desktop Frontend (SPA на Vue 3, Vite, Tailwind CSS, TypeScript, Pinia, онлайн-читалка, модерация карантина и 4 темы). |
| **10** | [10_stage4_walkthrough.md](10_stage4_walkthrough.md) | **Выполнено** | Отчет о реализации и результатах верификации Этапа 4 (Web Desktop SPA, универсальная читалка FB2/EPUB, темы Light/Dark/OLED/Sepia, браузерные тесты). |
| **11** | [11_stage5_implementation_plan.md](11_stage5_implementation_plan.md) | **Выполнено** | Детальный план реализации Этапа 5: Web Mobile PWA Frontend (Touch-first PWA, E-Ink режим, IndexedDB офлайн-чтение, свайпы и жесты). |
| **12** | [12_stage5_walkthrough.md](12_stage5_walkthrough.md) | **Выполнено** | Отчет о реализации и результатах верификации Этапа 5 (Touch-first PWA, 5 тем включая E-Ink без анимаций, IndexedDB офлайн-библиотека, мобильная читалка FB2/EPUB). |
| **13** | [13_stage6_implementation_plan.md](13_stage6_implementation_plan.md) | **Выполнено** | Детальный план реализации Этапа 6: Импорт библиотек Calibre (`metadata.db`) и автономный двусторонний Telegram-бот (поиск, скачивание и приём входящих книг). |
| **14** | [14_stage6_walkthrough.md](14_stage6_walkthrough.md) | **Выполнено** | Отчет о реализации и результатах верификации Этапа 6 (Calibre Importer, REST API, двусторонний Telegram-бот, развертывание в WSL2, 4.5 МБ RAM). |
| **15** | [15_backend_i18n_architecture.md](15_backend_i18n_architecture.md) | **Утверждено** | Архитектура серверной интернационализации (Backend i18n): каскадное определение языка (`?lang=` / `Accept-Language` / `config.yaml`), гибридные ошибки REST API (`error` + `code`) и динамическая локализация OPDS v1.2 / v2.0 для E-Ink читалок и внешних клиентов. |
