package models

import (
	"time"
)

// Book представляет основную сущность книги в библиотеке.
type Book struct {
	ID            string    `db:"id" json:"id"`
	Title         string    `db:"title" json:"title"`
	OriginalTitle string    `db:"original_title" json:"original_title,omitempty"`
	Annotation    string    `db:"annotation" json:"annotation,omitempty"`
	Language      string    `db:"language" json:"language,omitempty"`
	PublishedDate string    `db:"published_date" json:"published_date,omitempty"`
	Publisher     string    `db:"publisher" json:"publisher,omitempty"`
	ISBN          string    `db:"isbn" json:"isbn,omitempty"`
	CoverCached   bool      `db:"cover_cached" json:"cover_cached"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`

	// Связанные сущности (заполняются при выборке с деталями)
	Authors []AuthorDetail `json:"authors,omitempty"`
	Series  []SeriesDetail `json:"series,omitempty"`
	Genres  []Genre        `json:"genres,omitempty"`
	Files   []BookFile     `json:"files,omitempty"`
}

// Author представляет автора или переводчика.
type Author struct {
	ID       string `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	SortName string `db:"sort_name" json:"sort_name"`
}

// AuthorDetail связывает автора с книгой, сохраняя роль и порядок.
type AuthorDetail struct {
	Author
	Role  string `db:"role" json:"role"`   // "author", "translator"
	Order int    `db:"author_order" json:"order"`
}

// Series представляет литературную серию или цикл.
type Series struct {
	ID       string `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	SortName string `db:"sort_name" json:"sort_name"`
}

// SeriesDetail связывает серию с книгой, сохраняя номер в цикле.
type SeriesDetail struct {
	Series
	Index float64 `db:"series_index" json:"index"`
}

// Genre представляет категорию/жанр произведения.
type Genre struct {
	Code        string `db:"code" json:"code"`
	NameRU      string `db:"name_ru" json:"name_ru"`
	NameEN      string `db:"name_en" json:"name_en"`
	CategoryRU  string `db:"category_ru" json:"category_ru"`
	CategoryEN  string `db:"category_en" json:"category_en"`
}

// BookFile представляет конкретный файл формата (FB2, EPUB, MOBI и т.д.).
type BookFile struct {
	ID               string    `db:"id" json:"id"`
	BookID           string    `db:"book_id" json:"book_id"`
	Format           string    `db:"format" json:"format"` // "fb2", "fb2.zip", "epub", "mobi", "pdf"
	FilePath         string    `db:"file_path" json:"file_path"`
	ArchiveInnerPath string    `db:"archive_inner_path" json:"archive_inner_path,omitempty"`
	FileSize         int64     `db:"file_size" json:"file_size"`
	SHA256           string    `db:"sha256" json:"sha256"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

// AuthorInput описывает автора при обновлении книги администратором.
type AuthorInput struct {
	Name  string `json:"name"`
	Role  string `json:"role,omitempty"`
	Order int    `json:"order,omitempty"`
}

// SeriesInput описывает серию при обновлении книги администратором.
type SeriesInput struct {
	Name  string  `json:"name"`
	Index float64 `json:"index,omitempty"`
}

// UpdateBookMetadataRequest описывает запрос администратора на обновление метаданных книги.
type UpdateBookMetadataRequest struct {
	Title         string        `json:"title"`
	OriginalTitle string        `json:"original_title"`
	Annotation    string        `json:"annotation"`
	Language      string        `json:"language"`
	Publisher     string        `json:"publisher"`
	PublishedDate string        `json:"published_date"`
	ISBN          string        `json:"isbn"`
	Authors       []AuthorInput `json:"authors"`
	Series        []SeriesInput `json:"series"`
	Genres        []string      `json:"genres"` // Коды жанров (напр. sf_space, prose)
}

// BatchBookActionRequest описывает групповую операцию над книгами.
type BatchBookActionRequest struct {
	BookIDs     []string `json:"book_ids"`
	Action      string   `json:"action"` // "delete", "set_genre", "set_series", "regenerate_cover"
	DeleteFiles bool     `json:"delete_files,omitempty"`
	GenreCode   string   `json:"genre_code,omitempty"`
	SeriesName  string   `json:"series_name,omitempty"`
}

// BatchActionResult возвращает итоги пакетной операции.
type BatchActionResult struct {
	SuccessCount int      `json:"success_count"`
	ErrorCount   int      `json:"error_count"`
	Errors       []string `json:"errors,omitempty"`
}

