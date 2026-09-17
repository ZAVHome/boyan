-- Третья миграция: Карантин дубликатов, прогресс чтения и полки пользователей

CREATE TABLE IF NOT EXISTS quarantine (
    id TEXT PRIMARY KEY,
    file_path TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    sha256 TEXT NOT NULL,
    format TEXT NOT NULL,
    parsed_title TEXT NOT NULL,
    parsed_authors TEXT NOT NULL,
    existing_book_id TEXT REFERENCES books(id) ON DELETE SET NULL,
    conflict_type TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_quarantine_sha256 ON quarantine(sha256);
CREATE INDEX IF NOT EXISTS idx_quarantine_existing ON quarantine(existing_book_id);

CREATE TABLE IF NOT EXISTS read_progress (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    format TEXT NOT NULL DEFAULT '',
    progress_percent REAL NOT NULL DEFAULT 0.0,
    position TEXT NOT NULL DEFAULT '',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, book_id)
);

CREATE TABLE IF NOT EXISTS user_shelves (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    shelf_type TEXT NOT NULL,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, book_id, shelf_type)
);

CREATE INDEX IF NOT EXISTS idx_user_shelves_user ON user_shelves(user_id);
CREATE INDEX IF NOT EXISTS idx_user_shelves_type ON user_shelves(user_id, shelf_type);
