# Реестр архитектурных и проектных решений (Architecture Decision Records)

В данной директории сохраняются аналитические сравнения, обоснования выбора технологий и детальные планы реализации, подготовленные и согласованные в ходе разработки комплекса **Next-Gen OPDS Suite (Boyan)**.

## Список документов:

| № | Документ | Статус | Краткое описание |
| :--- | :--- | :--- | :--- |
| **01** | [01_http_router_comparison.md](01_http_router_comparison.md) | **Утверждено** | Сравнительный анализ HTTP-роутеров (`go-chi/chi/v5` vs `labstack/echo/v4` vs `net/http`). Утвержден `go-chi/chi/v5`. |
| **02** | [02_db_layer_comparison.md](02_db_layer_comparison.md) | **Утверждено** | Сравнительный анализ слоев работы с SQLite (`sqlx` + SQL миграции vs `GORM` vs `database/sql`). Утвержден `sqlx` + встроенные SQL-миграции (`embed.FS`). |
| **03** | [03_stage1_implementation_plan.md](03_stage1_implementation_plan.md) | **Выполнено** | Детальный план реализации Этапа 1: Core Engine, Storage (SQLite WAL+FTS5), парсер FB2/ZIP с любыми кодировками, кеш обложек с LRU и скрипты развертывания `systemd` в WSL2. |
| **04** | [04_stage1_walkthrough.md](04_stage1_walkthrough.md) | **Выполнено** | Отчет о реализации и результатах верификации Этапа 1 (тесты, развертывание в WSL2, потребление 5.3 МБ RAM). |
