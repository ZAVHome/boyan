-- Инициализация схемы базы данных Next-Gen OPDS Suite (Boyan)

CREATE TABLE IF NOT EXISTS books (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    original_title TEXT DEFAULT '',
    annotation TEXT DEFAULT '',
    language TEXT DEFAULT 'ru',
    published_date TEXT DEFAULT '',
    publisher TEXT DEFAULT '',
    isbn TEXT DEFAULT '',
    cover_cached INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_books_title ON books(title);
CREATE INDEX IF NOT EXISTS idx_books_created_at ON books(created_at);

CREATE TABLE IF NOT EXISTS authors (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    sort_name TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_authors_name ON authors(name);
CREATE INDEX IF NOT EXISTS idx_authors_sort_name ON authors(sort_name);

CREATE TABLE IF NOT EXISTS book_authors (
    book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    author_id TEXT NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
    role TEXT DEFAULT 'author',
    author_order INTEGER DEFAULT 0,
    PRIMARY KEY (book_id, author_id, role)
);

CREATE INDEX IF NOT EXISTS idx_book_authors_book ON book_authors(book_id);
CREATE INDEX IF NOT EXISTS idx_book_authors_author ON book_authors(author_id);

CREATE TABLE IF NOT EXISTS series (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    sort_name TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_series_name ON series(name);

CREATE TABLE IF NOT EXISTS book_series (
    book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    series_id TEXT NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    series_index REAL DEFAULT 0.0,
    PRIMARY KEY (book_id, series_id)
);

CREATE INDEX IF NOT EXISTS idx_book_series_book ON book_series(book_id);
CREATE INDEX IF NOT EXISTS idx_book_series_series ON book_series(series_id);

CREATE TABLE IF NOT EXISTS genres (
    code TEXT PRIMARY KEY,
    name_ru TEXT NOT NULL,
    name_en TEXT NOT NULL,
    category_ru TEXT NOT NULL,
    category_en TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS book_genres (
    book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    genre_code TEXT NOT NULL REFERENCES genres(code) ON DELETE CASCADE,
    PRIMARY KEY (book_id, genre_code)
);

CREATE INDEX IF NOT EXISTS idx_book_genres_book ON book_genres(book_id);
CREATE INDEX IF NOT EXISTS idx_book_genres_genre ON book_genres(genre_code);

CREATE TABLE IF NOT EXISTS book_files (
    id TEXT PRIMARY KEY,
    book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    format TEXT NOT NULL,
    file_path TEXT NOT NULL,
    archive_inner_path TEXT DEFAULT '',
    file_size INTEGER NOT NULL,
    sha256 TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_book_files_book ON book_files(book_id);
CREATE INDEX IF NOT EXISTS idx_book_files_sha256 ON book_files(sha256);

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user',
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
