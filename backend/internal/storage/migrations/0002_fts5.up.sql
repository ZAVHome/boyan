-- Вторая миграция: Виртуальная таблица полнотекстового поиска SQLite FTS5

CREATE VIRTUAL TABLE IF NOT EXISTS books_fts USING fts5(
    book_id UNINDEXED,
    title,
    original_title,
    annotation,
    author_names,
    series_names,
    tokenize='unicode61 remove_diacritics 2'
);
