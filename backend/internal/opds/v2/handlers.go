package v2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"boyan/internal/config"
	"boyan/internal/i18n"
	"boyan/internal/storage"
)

// OPDSv2Handler обслуживает запросы OPDS v2.0 (JSON-LD).
type OPDSv2Handler struct {
	cfg  *config.Config
	repo *storage.BookRepository
}

func NewOPDSv2Handler(cfg *config.Config, repo *storage.BookRepository) *OPDSv2Handler {
	return &OPDSv2Handler{cfg: cfg, repo: repo}
}

// Catalog формирует корневой каталог OPDS v2.0.
func (h *OPDSv2Handler) Catalog(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	tr := i18n.FromContext(r.Context())

	feed := Feed{
		Metadata: Metadata{
			Title:       h.cfg.OPDS.Title,
			Description: h.cfg.OPDS.Subtitle,
			Modified:    time.Now().UTC().Format(time.RFC3339),
		},
		Links: []Link{
			{Rel: "self", Href: baseURL + "/opds/v2/catalog.json", Type: MimeTypeOPDSJSON},
			{Rel: "start", Href: baseURL + "/opds/v2/catalog.json", Type: MimeTypeOPDSJSON},
			{
				Rel:       "search",
				Href:      baseURL + "/opds/v2/search{?query}",
				Type:      MimeTypeOPDSJSON,
				Templated: true,
			},
		},
		Navigation: []Link{
			{Rel: "subsection", Href: baseURL + "/opds/v1/authors", Type: "application/atom+xml", Title: tr.T("opds.authors")},
			{Rel: "subsection", Href: baseURL + "/opds/v1/series", Type: "application/atom+xml", Title: tr.T("opds.series")},
			{Rel: "subsection", Href: baseURL + "/opds/v1/genres", Type: "application/atom+xml", Title: tr.T("opds.genres")},
			{Rel: "subsection", Href: baseURL + "/opds/v1/recent", Type: "application/atom+xml", Title: tr.T("opds.recent")},
		},
	}

	books, _, err := h.repo.GetRecentBooks(r.Context(), 0, 50)
	if err == nil {
		for _, b := range books {
			feed.Publications = append(feed.Publications, BookToPublication(&b, baseURL))
		}
	}

	w.Header().Set("Content-Type", MimeTypeOPDSJSON+"; charset=utf-8")
	_ = json.NewEncoder(w).Encode(feed)
}

// Search выполняет поиск книг по протоколу OPDS v2.0.
func (h *OPDSv2Handler) Search(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	tr := i18n.FromContext(r.Context())
	query := r.URL.Query().Get("query")
	if query == "" {
		query = r.URL.Query().Get("q")
	}

	feed := Feed{
		Metadata: Metadata{
			Title:       tr.T("opds.search_results", query),
			Description: tr.T("opds.search_query_desc", query),
			Modified:    time.Now().UTC().Format(time.RFC3339),
		},
		Links: []Link{
			{Rel: "self", Href: fmt.Sprintf("%s/opds/v2/search?query=%s", baseURL, query), Type: MimeTypeOPDSJSON},
			{Rel: "start", Href: baseURL + "/opds/v2/catalog.json", Type: MimeTypeOPDSJSON},
		},
	}

	books, _, err := h.repo.SearchBooksFTS(r.Context(), query, 0, 50)
	if err == nil {
		for _, b := range books {
			feed.Publications = append(feed.Publications, BookToPublication(&b, baseURL))
		}
	}

	w.Header().Set("Content-Type", MimeTypeOPDSJSON+"; charset=utf-8")
	_ = json.NewEncoder(w).Encode(feed)
}

func (h *OPDSv2Handler) getBaseURL(r *http.Request) string {
	if h.cfg.Server.BaseURL != "" && h.cfg.Server.BaseURL != "http://localhost:8080" {
		return h.cfg.Server.BaseURL
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}
