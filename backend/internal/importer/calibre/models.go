package calibre

import "time"

// CalibreBook представляет запись книги из таблицы `books` в Calibre metadata.db.
type CalibreBook struct {
	ID          int64     `db:"id"`
	Title       string    `db:"title"`
	Sort        string    `db:"sort"`
	Timestamp   *string   `db:"timestamp"`
	Pubdate     *string   `db:"pubdate"`
	SeriesIndex float64   `db:"series_index"`
	Path        string    `db:"path"`
	HasCover    bool      `db:"has_cover"`
	CreatedAt   time.Time
}

// CalibreAuthor представляет автора из таблицы `authors`.
type CalibreAuthor struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
	Sort string `db:"sort"`
}

// CalibreSeries представляет серию книги из таблицы `series`.
type CalibreSeries struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
	Sort string `db:"sort"`
}

// CalibreTag представляет тег (жанр) из таблицы `tags`.
type CalibreTag struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

// CalibreData представляет файл книги определенного формата из таблицы `data`.
type CalibreData struct {
	ID               int64  `db:"id"`
	Book             int64  `db:"book"`
	Format           string `db:"format"`
	UncompressedSize int64  `db:"uncompressed_size"`
	Name             string `db:"name"`
}

// CalibreComment представляет аннотацию/описание книги из таблицы `comments`.
type CalibreComment struct {
	Book int64  `db:"book"`
	Text string `db:"text"`
}

// CalibreIdentifier представляет идентификаторы (ISBN, Goodreads и др.) из таблицы `identifiers`.
type CalibreIdentifier struct {
	Book int64  `db:"book"`
	Type string `db:"type"`
	Val  string `db:"val"`
}
