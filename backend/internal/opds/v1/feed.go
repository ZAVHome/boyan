package v1

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"boyan/internal/models"
	"boyan/internal/services"
)

const (
	MimeTypeAtomNavigation  = "application/atom+xml;profile=opds-catalog;kind=navigation"
	MimeTypeAtomAcquisition = "application/atom+xml;profile=opds-catalog;kind=acquisition"
	MimeTypeOpenSearch      = "application/opensearchdescription+xml"

	RelStart       = "start"
	RelSelf        = "self"
	RelUp          = "up"
	RelNext        = "next"
	RelSearch      = "search"
	RelSubsection  = "subsection"
	RelAcquisition = "http://opds-spec.org/acquisition"
	RelImage       = "http://opds-spec.org/image"
	RelThumbnail   = "http://opds-spec.org/image/thumbnail"
)

// Feed представляет корневой элемент OPDS Atom фида.
type Feed struct {
	XMLName xml.Name `xml:"feed"`
	Xmlns   string   `xml:"xmlns,attr"`
	Dc      string   `xml:"xmlns:dc,attr,omitempty"`
	Opds    string   `xml:"xmlns:opds,attr,omitempty"`
	Calibre string   `xml:"xmlns:calibre,attr,omitempty"`

	ID      string    `xml:"id"`
	Title   string    `xml:"title"`
	Updated string    `xml:"updated"`
	Icon    string    `xml:"icon,omitempty"`
	Author  *Author   `xml:"author,omitempty"`
	Links   []Link    `xml:"link"`
	Entries []Entry   `xml:"entry"`
}

// Entry представляет отдельную запись в каталоге (книгу или навигационную ссылку).
type Entry struct {
	ID          string     `xml:"id"`
	Title       string     `xml:"title"`
	Updated     string     `xml:"updated"`
	Authors     []Author   `xml:"author,omitempty"`
	Summary     *TextElem  `xml:"summary,omitempty"`
	Content     *TextElem  `xml:"content,omitempty"`
	Categories  []Category `xml:"category,omitempty"`
	Links       []Link     `xml:"link"`

	// Calibre расширения для E-Ink ридеров
	CalibreSeries      string  `xml:"calibre:series,omitempty"`
	CalibreSeriesIndex string  `xml:"calibre:series_index,omitempty"`
}

type Author struct {
	Name string `xml:"name"`
	URI  string `xml:"uri,omitempty"`
}

type TextElem struct {
	Type  string `xml:"type,attr,omitempty"`
	Value string `xml:",chardata"`
}

type Category struct {
	Term  string `xml:"term,attr"`
	Label string `xml:"label,attr,omitempty"`
}

type Link struct {
	Rel   string `xml:"rel,attr"`
	Href  string `xml:"href,attr"`
	Type  string `xml:"type,attr"`
	Title string `xml:"title,attr,omitempty"`
}

// NewFeed инициализирует фид с необходимыми пространствами имен.
func NewFeed(id, title string) *Feed {
	return &Feed{
		Xmlns:   "http://www.w3.org/2005/Atom",
		Dc:      "http://purl.org/dc/terms/",
		Opds:    "http://opds-spec.org/2010/catalog",
		Calibre: "http://calibre.kovidgoyal.net/2009/metadata",
		ID:      id,
		Title:   title,
		Updated: time.Now().UTC().Format(time.RFC3339),
	}
}

// RenderXML отдает валидный XML с прологом <?xml ...?> и UTF-8 кодировкой.
func RenderXML(w http.ResponseWriter, feed *Feed, mimeType string) {
	w.Header().Set("Content-Type", fmt.Sprintf("%s; charset=utf-8", mimeType))
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>` + "\n"))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_ = enc.Encode(feed)
}

// BookToEntry преобразует доменную модель Book в элемент Entry каталога OPDS.
func BookToEntry(book *models.Book, baseURL string) Entry {
	entry := Entry{
		ID:      fmt.Sprintf("urn:boyan:book:%s", book.ID),
		Title:   book.Title,
		Updated: book.UpdatedAt.UTC().Format(time.RFC3339),
	}

	// Авторы
	for _, a := range book.Authors {
		entry.Authors = append(entry.Authors, Author{Name: a.Name})
	}

	// Аннотация
	if book.Annotation != "" {
		entry.Content = &TextElem{
			Type:  "text/html",
			Value: book.Annotation,
		}
	}

	// Жанры
	for _, g := range book.Genres {
		label := g.NameRU
		if label == "" {
			label = g.Code
		}
		entry.Categories = append(entry.Categories, Category{
			Term:  g.Code,
			Label: label,
		})
	}

	// Серия и номер
	if len(book.Series) > 0 {
		s := book.Series[0]
		entry.CalibreSeries = s.Name
		if s.Index > 0 {
			entry.CalibreSeriesIndex = strconv.FormatFloat(s.Index, 'f', -1, 64)
		}
	}

	// Обложки (для E-Ink экранов)
	coverURL := fmt.Sprintf("%s/api/v1/covers/%s", baseURL, book.ID)
	entry.Links = append(entry.Links,
		Link{Rel: RelImage, Href: coverURL, Type: "image/jpeg"},
		Link{Rel: RelThumbnail, Href: coverURL, Type: "image/jpeg"},
	)

	// Ссылки на файлы для скачивания
	for _, f := range book.Files {
		formatMime := services.MIMETypeForFormat(f.Format)
		downloadURL := fmt.Sprintf("%s/api/v1/books/%s/download/%s", baseURL, book.ID, f.Format)
		entry.Links = append(entry.Links, Link{
			Rel:   RelAcquisition,
			Href:  downloadURL,
			Type:  formatMime,
			Title: strings.ToUpper(f.Format),
		})
	}

	return entry
}
