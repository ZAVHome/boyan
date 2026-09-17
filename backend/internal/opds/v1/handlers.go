package v1

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"boyan/internal/config"
	"boyan/internal/i18n"
	"boyan/internal/storage"

	"github.com/go-chi/chi/v5"
)

// OPDSv1Handler обслуживает все каналы протокола OPDS v1.2 (Atom/XML).
type OPDSv1Handler struct {
	cfg  *config.Config
	repo *storage.BookRepository
}

func NewOPDSv1Handler(cfg *config.Config, repo *storage.BookRepository) *OPDSv1Handler {
	return &OPDSv1Handler{cfg: cfg, repo: repo}
}

// RootFeed формирует главный навигационный каталог библиотеки.
func (h *OPDSv1Handler) RootFeed(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	tr := i18n.FromContext(r.Context())
	feed := NewFeed("urn:boyan:opds:v1:root", h.cfg.OPDS.Title)

	feed.Links = []Link{
		{Rel: RelSelf, Href: baseURL + "/opds/v1/feed.xml", Type: MimeTypeAtomNavigation},
		{Rel: RelStart, Href: baseURL + "/opds/v1/feed.xml", Type: MimeTypeAtomNavigation},
		{Rel: RelSearch, Href: baseURL + "/opds/v1/opensearch.xml", Type: MimeTypeOpenSearch},
	}

	// Разделы каталога
	feed.Entries = []Entry{
		{
			ID:      "urn:boyan:section:authors",
			Title:   tr.T("opds.authors"),
			Updated: feed.Updated,
			Content: &TextElem{Type: "text", Value: tr.T("opds.authors_desc")},
			Links: []Link{
				{Rel: RelSubsection, Href: baseURL + "/opds/v1/authors", Type: MimeTypeAtomNavigation},
			},
		},
		{
			ID:      "urn:boyan:section:series",
			Title:   tr.T("opds.series"),
			Updated: feed.Updated,
			Content: &TextElem{Type: "text", Value: tr.T("opds.series_desc")},
			Links: []Link{
				{Rel: RelSubsection, Href: baseURL + "/opds/v1/series", Type: MimeTypeAtomNavigation},
			},
		},
		{
			ID:      "urn:boyan:section:genres",
			Title:   tr.T("opds.genres"),
			Updated: feed.Updated,
			Content: &TextElem{Type: "text", Value: tr.T("opds.genres_desc")},
			Links: []Link{
				{Rel: RelSubsection, Href: baseURL + "/opds/v1/genres", Type: MimeTypeAtomNavigation},
			},
		},
		{
			ID:      "urn:boyan:section:recent",
			Title:   tr.T("opds.recent"),
			Updated: feed.Updated,
			Content: &TextElem{Type: "text", Value: tr.T("opds.recent_desc")},
			Links: []Link{
				{Rel: RelSubsection, Href: baseURL + "/opds/v1/recent", Type: MimeTypeAtomAcquisition},
			},
		},
	}

	RenderXML(w, feed, MimeTypeAtomNavigation)
}

// AuthorsAlpha отдает алфавитный указатель авторов.
func (h *OPDSv1Handler) AuthorsAlpha(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	tr := i18n.FromContext(r.Context())
	feed := NewFeed("urn:boyan:opds:v1:authors", tr.T("opds.authors_alpha"))

	feed.Links = []Link{
		{Rel: RelSelf, Href: baseURL + "/opds/v1/authors", Type: MimeTypeAtomNavigation},
		{Rel: RelUp, Href: baseURL + "/opds/v1/feed.xml", Type: MimeTypeAtomNavigation},
		{Rel: RelStart, Href: baseURL + "/opds/v1/feed.xml", Type: MimeTypeAtomNavigation},
	}

	letters, err := h.repo.GetAuthorsAlphabet(r.Context())
	if err == nil {
		for _, l := range letters {
			feed.Entries = append(feed.Entries, Entry{
				ID:      fmt.Sprintf("urn:boyan:authors:alpha:%s", l.Letter),
				Title:   fmt.Sprintf("%s (%d)", l.Letter, l.Count),
				Updated: feed.Updated,
				Links: []Link{
					{Rel: RelSubsection, Href: fmt.Sprintf("%s/opds/v1/authors/alpha/%s", baseURL, url.PathEscape(l.Letter)), Type: MimeTypeAtomNavigation},
				},
			})
		}
	}

	RenderXML(w, feed, MimeTypeAtomNavigation)
}

// AuthorsList отдает авторов на заданную букву.
func (h *OPDSv1Handler) AuthorsList(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	tr := i18n.FromContext(r.Context())
	letter := chi.URLParam(r, "letter")
	feed := NewFeed(fmt.Sprintf("urn:boyan:authors:letter:%s", letter), tr.T("opds.authors_letter", letter))

	feed.Links = []Link{
		{Rel: RelSelf, Href: fmt.Sprintf("%s/opds/v1/authors/alpha/%s", baseURL, url.PathEscape(letter)), Type: MimeTypeAtomNavigation},
		{Rel: RelUp, Href: baseURL + "/opds/v1/authors", Type: MimeTypeAtomNavigation},
		{Rel: RelStart, Href: baseURL + "/opds/v1/feed.xml", Type: MimeTypeAtomNavigation},
	}

	authors, _, err := h.repo.GetAuthorsByLetter(r.Context(), letter, 0, 200)
	if err == nil {
		for _, a := range authors {
			feed.Entries = append(feed.Entries, Entry{
				ID:      fmt.Sprintf("urn:boyan:author:%s", a.ID),
				Title:   fmt.Sprintf("%s (%d)", a.Name, a.BookCount),
				Updated: feed.Updated,
				Links: []Link{
					{Rel: RelSubsection, Href: fmt.Sprintf("%s/opds/v1/authors/%s", baseURL, a.ID), Type: MimeTypeAtomAcquisition},
				},
			})
		}
	}

	RenderXML(w, feed, MimeTypeAtomNavigation)
}

// AuthorBooks отдает книги выбранного автора.
func (h *OPDSv1Handler) AuthorBooks(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	tr := i18n.FromContext(r.Context())
	authorID := chi.URLParam(r, "id")

	author, err := h.repo.GetAuthorByID(r.Context(), authorID)
	title := tr.T("opds.author_books")
	if err == nil && author != nil {
		title = author.Name
	}

	feed := NewFeed(fmt.Sprintf("urn:boyan:author:books:%s", authorID), title)
	feed.Links = []Link{
		{Rel: RelSelf, Href: fmt.Sprintf("%s/opds/v1/authors/%s", baseURL, authorID), Type: MimeTypeAtomAcquisition},
		{Rel: RelUp, Href: baseURL + "/opds/v1/authors", Type: MimeTypeAtomNavigation},
		{Rel: RelStart, Href: baseURL + "/opds/v1/feed.xml", Type: MimeTypeAtomNavigation},
	}

	books, err := h.repo.GetAuthorBooks(r.Context(), authorID)
	if err == nil {
		for _, b := range books {
			feed.Entries = append(feed.Entries, BookToEntry(&b, baseURL))
		}
	}

	RenderXML(w, feed, MimeTypeAtomAcquisition)
}

// SeriesAlpha отдает алфавитный указатель серий.
func (h *OPDSv1Handler) SeriesAlpha(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	tr := i18n.FromContext(r.Context())
	feed := NewFeed("urn:boyan:opds:v1:series", tr.T("opds.series_alpha"))

	feed.Links = []Link{
		{Rel: RelSelf, Href: baseURL + "/opds/v1/series", Type: MimeTypeAtomNavigation},
		{Rel: RelUp, Href: baseURL + "/opds/v1/feed.xml", Type: MimeTypeAtomNavigation},
		{Rel: RelStart, Href: baseURL + "/opds/v1/feed.xml", Type: MimeTypeAtomNavigation},
	}

	letters, err := h.repo.GetSeriesAlphabet(r.Context())
	if err == nil {
		for _, l := range letters {
			feed.Entries = append(feed.Entries, Entry{
				ID:      fmt.Sprintf("urn:boyan:series:alpha:%s", l.Letter),
				Title:   fmt.Sprintf("%s (%d)", l.Letter, l.Count),
				Updated: feed.Updated,
				Links: []Link{
					{Rel: RelSubsection, Href: fmt.Sprintf("%s/opds/v1/series/alpha/%s", baseURL, url.PathEscape(l.Letter)), Type: MimeTypeAtomNavigation},
				},
			})
		}
	}

	RenderXML(w, feed, MimeTypeAtomNavigation)
}

// SeriesList отдает серии на букву.
func (h *OPDSv1Handler) SeriesList(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	tr := i18n.FromContext(r.Context())
	letter := chi.URLParam(r, "letter")
	feed := NewFeed(fmt.Sprintf("urn:boyan:series:letter:%s", letter), tr.T("opds.series_letter", letter))

	feed.Links = []Link{
		{Rel: RelSelf, Href: fmt.Sprintf("%s/opds/v1/series/alpha/%s", baseURL, url.PathEscape(letter)), Type: MimeTypeAtomNavigation},
		{Rel: RelUp, Href: baseURL + "/opds/v1/series", Type: MimeTypeAtomNavigation},
		{Rel: RelStart, Href: baseURL + "/opds/v1/feed.xml", Type: MimeTypeAtomNavigation},
	}

	seriesList, _, err := h.repo.GetSeriesByLetter(r.Context(), letter, 0, 200)
	if err == nil {
		for _, s := range seriesList {
			feed.Entries = append(feed.Entries, Entry{
				ID:      fmt.Sprintf("urn:boyan:series:%s", s.ID),
				Title:   fmt.Sprintf("%s (%d)", s.Name, s.BookCount),
				Updated: feed.Updated,
				Links: []Link{
					{Rel: RelSubsection, Href: fmt.Sprintf("%s/opds/v1/series/%s", baseURL, s.ID), Type: MimeTypeAtomAcquisition},
				},
			})
		}
	}

	RenderXML(w, feed, MimeTypeAtomNavigation)
}

// SeriesBooks отдает книги выбранной серии по порядку номеров.
func (h *OPDSv1Handler) SeriesBooks(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	tr := i18n.FromContext(r.Context())
	seriesID := chi.URLParam(r, "id")

	series, err := h.repo.GetSeriesByID(r.Context(), seriesID)
	title := tr.T("opds.series_books")
	if err == nil && series != nil {
		title = series.Name
	}

	feed := NewFeed(fmt.Sprintf("urn:boyan:series:books:%s", seriesID), title)
	feed.Links = []Link{
		{Rel: RelSelf, Href: fmt.Sprintf("%s/opds/v1/series/%s", baseURL, seriesID), Type: MimeTypeAtomAcquisition},
		{Rel: RelUp, Href: baseURL + "/opds/v1/series", Type: MimeTypeAtomNavigation},
		{Rel: RelStart, Href: baseURL + "/opds/v1/feed.xml", Type: MimeTypeAtomNavigation},
	}

	books, err := h.repo.GetSeriesBooks(r.Context(), seriesID)
	if err == nil {
		for _, b := range books {
			feed.Entries = append(feed.Entries, BookToEntry(&b, baseURL))
		}
	}

	RenderXML(w, feed, MimeTypeAtomAcquisition)
}

// GenresCategories отдает список категорий жанров.
func (h *OPDSv1Handler) GenresCategories(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	tr := i18n.FromContext(r.Context())
	feed := NewFeed("urn:boyan:opds:v1:genres", tr.T("opds.genre_categories"))

	feed.Links = []Link{
		{Rel: RelSelf, Href: baseURL + "/opds/v1/genres", Type: MimeTypeAtomNavigation},
		{Rel: RelUp, Href: baseURL + "/opds/v1/feed.xml", Type: MimeTypeAtomNavigation},
		{Rel: RelStart, Href: baseURL + "/opds/v1/feed.xml", Type: MimeTypeAtomNavigation},
	}

	categories, err := h.repo.GetGenreCategories(r.Context())
	if err == nil {
		for _, c := range categories {
			feed.Entries = append(feed.Entries, Entry{
				ID:      fmt.Sprintf("urn:boyan:genre:cat:%s", c.CategoryRU),
				Title:   fmt.Sprintf("%s (%d)", c.CategoryRU, c.BookCount),
				Updated: feed.Updated,
				Links: []Link{
					{Rel: RelSubsection, Href: fmt.Sprintf("%s/opds/v1/genres/category/%s", baseURL, url.PathEscape(c.CategoryRU)), Type: MimeTypeAtomNavigation},
				},
			})
		}
	}

	RenderXML(w, feed, MimeTypeAtomNavigation)
}

// GenresSubcategories отдает подкатегории жанров внутри категории.
func (h *OPDSv1Handler) GenresSubcategories(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	categoryRU := chi.URLParam(r, "category")
	feed := NewFeed(fmt.Sprintf("urn:boyan:genre:category:%s", categoryRU), categoryRU)

	feed.Links = []Link{
		{Rel: RelSelf, Href: fmt.Sprintf("%s/opds/v1/genres/category/%s", baseURL, url.PathEscape(categoryRU)), Type: MimeTypeAtomNavigation},
		{Rel: RelUp, Href: baseURL + "/opds/v1/genres", Type: MimeTypeAtomNavigation},
		{Rel: RelStart, Href: baseURL + "/opds/v1/feed.xml", Type: MimeTypeAtomNavigation},
	}

	genres, err := h.repo.GetGenresByCategory(r.Context(), categoryRU)
	if err == nil {
		for _, g := range genres {
			feed.Entries = append(feed.Entries, Entry{
				ID:      fmt.Sprintf("urn:boyan:genre:%s", g.Code),
				Title:   fmt.Sprintf("%s (%d)", g.NameRU, g.BookCount),
				Updated: feed.Updated,
				Links: []Link{
					{Rel: RelSubsection, Href: fmt.Sprintf("%s/opds/v1/genres/%s", baseURL, g.Code), Type: MimeTypeAtomAcquisition},
				},
			})
		}
	}

	RenderXML(w, feed, MimeTypeAtomNavigation)
}

// GenreBooks отдает книги выбранного жанра.
func (h *OPDSv1Handler) GenreBooks(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	genreCode := chi.URLParam(r, "code")

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := h.cfg.OPDS.PageSize
	offset := (page - 1) * perPage

	genre, _ := h.repo.GetGenreByCode(r.Context(), genreCode)
	title := genreCode
	if genre != nil {
		title = genre.NameRU
	}

	feed := NewFeed(fmt.Sprintf("urn:boyan:genre:books:%s", genreCode), title)
	feed.Links = []Link{
		{Rel: RelSelf, Href: fmt.Sprintf("%s/opds/v1/genres/%s?page=%d", baseURL, genreCode, page), Type: MimeTypeAtomAcquisition},
		{Rel: RelUp, Href: baseURL + "/opds/v1/genres", Type: MimeTypeAtomNavigation},
		{Rel: RelStart, Href: baseURL + "/opds/v1/feed.xml", Type: MimeTypeAtomNavigation},
	}

	books, total, err := h.repo.GetGenreBooks(r.Context(), genreCode, offset, perPage)
	if err == nil {
		for _, b := range books {
			feed.Entries = append(feed.Entries, BookToEntry(&b, baseURL))
		}
		if offset+perPage < total {
			feed.Links = append(feed.Links, Link{
				Rel:  RelNext,
				Href: fmt.Sprintf("%s/opds/v1/genres/%s?page=%d", baseURL, genreCode, page+1),
				Type: MimeTypeAtomAcquisition,
			})
		}
	}

	RenderXML(w, feed, MimeTypeAtomAcquisition)
}

// RecentBooks отдает новые поступления книг с пагинацией.
func (h *OPDSv1Handler) RecentBooks(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	tr := i18n.FromContext(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := h.cfg.OPDS.PageSize
	offset := (page - 1) * perPage

	feed := NewFeed("urn:boyan:opds:v1:recent", tr.T("opds.recent"))
	feed.Links = []Link{
		{Rel: RelSelf, Href: fmt.Sprintf("%s/opds/v1/recent?page=%d", baseURL, page), Type: MimeTypeAtomAcquisition},
		{Rel: RelUp, Href: baseURL + "/opds/v1/feed.xml", Type: MimeTypeAtomNavigation},
		{Rel: RelStart, Href: baseURL + "/opds/v1/feed.xml", Type: MimeTypeAtomNavigation},
	}

	books, total, err := h.repo.GetRecentBooks(r.Context(), offset, perPage)
	if err == nil {
		for _, b := range books {
			feed.Entries = append(feed.Entries, BookToEntry(&b, baseURL))
		}
		if offset+perPage < total {
			feed.Links = append(feed.Links, Link{
				Rel:  RelNext,
				Href: fmt.Sprintf("%s/opds/v1/recent?page=%d", baseURL, page+1),
				Type: MimeTypeAtomAcquisition,
			})
		}
	}

	RenderXML(w, feed, MimeTypeAtomAcquisition)
}

// OpenSearchDescriptor отдает XML дескриптор OpenSearch 1.1.
func (h *OPDSv1Handler) OpenSearchDescriptor(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	tr := i18n.FromContext(r.Context())
	w.Header().Set("Content-Type", MimeTypeOpenSearch+"; charset=utf-8")

	searchXML := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<OpenSearchDescription xmlns="http://a9.com/-/spec/opensearch/1.1/">
  <ShortName>%s</ShortName>
  <Description>%s</Description>
  <InputEncoding>UTF-8</InputEncoding>
  <OutputEncoding>UTF-8</OutputEncoding>
  <Url type="application/atom+xml;profile=opds-catalog;kind=acquisition" template="%s/opds/v1/search?q={searchTerms}&amp;page={startPage?}"/>
</OpenSearchDescription>`, h.cfg.OPDS.Title, tr.T("opds.search_desc"), baseURL)

	_, _ = w.Write([]byte(searchXML))
}

// Search обрабатывает поисковые запросы читалок через FTS5.
func (h *OPDSv1Handler) Search(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	tr := i18n.FromContext(r.Context())
	query := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := h.cfg.OPDS.PageSize
	offset := (page - 1) * perPage

	feed := NewFeed("urn:boyan:opds:v1:search", tr.T("opds.search_results", query))
	feed.Links = []Link{
		{Rel: RelSelf, Href: fmt.Sprintf("%s/opds/v1/search?q=%s&page=%d", baseURL, url.QueryEscape(query), page), Type: MimeTypeAtomAcquisition},
		{Rel: RelStart, Href: baseURL + "/opds/v1/feed.xml", Type: MimeTypeAtomNavigation},
		{Rel: RelSearch, Href: baseURL + "/opds/v1/opensearch.xml", Type: MimeTypeOpenSearch},
	}

	books, total, err := h.repo.SearchBooksFTS(r.Context(), query, offset, perPage)
	if err == nil {
		for _, b := range books {
			feed.Entries = append(feed.Entries, BookToEntry(&b, baseURL))
		}
		if offset+perPage < total {
			feed.Links = append(feed.Links, Link{
				Rel:  RelNext,
				Href: fmt.Sprintf("%s/opds/v1/search?q=%s&page=%d", baseURL, url.QueryEscape(query), page+1),
				Type: MimeTypeAtomAcquisition,
			})
		}
	}

	RenderXML(w, feed, MimeTypeAtomAcquisition)
}

func (h *OPDSv1Handler) getBaseURL(r *http.Request) string {
	if h.cfg.Server.BaseURL != "" && h.cfg.Server.BaseURL != "http://localhost:8080" {
		return h.cfg.Server.BaseURL
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}
