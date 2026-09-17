package v2

import (
	"fmt"
	"strings"
	"time"

	"boyan/internal/models"
	"boyan/internal/services"
)

const (
	MimeTypeOPDSJSON = "application/opds+json"
)

// Feed представляет каталог по спецификации Readium OPDS 2.0.
type Feed struct {
	Metadata     Metadata      `json:"metadata"`
	Links        []Link        `json:"links"`
	Navigation   []Link        `json:"navigation,omitempty"`
	Publications []Publication `json:"publications,omitempty"`
}

type Metadata struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Modified    string `json:"modified,omitempty"`
}

type Link struct {
	Rel        string `json:"rel"`
	Href       string `json:"href"`
	Type       string `json:"type"`
	Title      string `json:"title,omitempty"`
	Templated  bool   `json:"templated,omitempty"`
	Properties any    `json:"properties,omitempty"`
}

type Publication struct {
	Metadata PublicationMetadata `json:"metadata"`
	Links    []Link              `json:"links"`
	Images   []Link              `json:"images,omitempty"`
}

type PublicationMetadata struct {
	Type        string         `json:"@type"`
	Identifier  string         `json:"identifier"`
	Title       string         `json:"title"`
	Author      string         `json:"author,omitempty"`
	Description string         `json:"description,omitempty"`
	Language    string         `json:"language,omitempty"`
	Modified    string         `json:"modified,omitempty"`
	Published   string         `json:"published,omitempty"`
	Publisher   string         `json:"publisher,omitempty"`
	BelongsTo   *SeriesGroup   `json:"belongsTo,omitempty"`
}

type SeriesGroup struct {
	Series []SeriesItem `json:"series,omitempty"`
}

type SeriesItem struct {
	Name     string  `json:"name"`
	Position float64 `json:"position,omitempty"`
}

// BookToPublication преобразует книгу в публикацию Readium OPDS 2.0.
func BookToPublication(book *models.Book, baseURL string) Publication {
	var authorNames []string
	for _, a := range book.Authors {
		authorNames = append(authorNames, a.Name)
	}

	pub := Publication{
		Metadata: PublicationMetadata{
			Type:        "http://schema.org/Book",
			Identifier:  fmt.Sprintf("urn:boyan:book:%s", book.ID),
			Title:       book.Title,
			Author:      strings.Join(authorNames, ", "),
			Description: book.Annotation,
			Language:    book.Language,
			Modified:    book.UpdatedAt.UTC().Format(time.RFC3339),
			Published:   book.PublishedDate,
			Publisher:   book.Publisher,
		},
	}

	if len(book.Series) > 0 {
		var sItems []SeriesItem
		for _, s := range book.Series {
			sItems = append(sItems, SeriesItem{
				Name:     s.Name,
				Position: s.Index,
			})
		}
		pub.Metadata.BelongsTo = &SeriesGroup{Series: sItems}
	}

	// Ссылки на обложки
	coverURL := fmt.Sprintf("%s/api/v1/covers/%s", baseURL, book.ID)
	pub.Images = []Link{
		{Rel: "http://opds-spec.org/image", Href: coverURL, Type: "image/jpeg"},
		{Rel: "http://opds-spec.org/image/thumbnail", Href: coverURL, Type: "image/jpeg"},
	}

	// Ссылки на файлы книг
	for _, f := range book.Files {
		downloadURL := fmt.Sprintf("%s/api/v1/books/%s/download/%s", baseURL, book.ID, f.Format)
		pub.Links = append(pub.Links, Link{
			Rel:   "http://opds-spec.org/acquisition",
			Href:  downloadURL,
			Type:  services.MIMETypeForFormat(f.Format),
			Title: strings.ToUpper(f.Format),
		})
	}

	return pub
}
