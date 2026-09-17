package i18n

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseAcceptLanguage(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "Empty header",
			header:   "",
			expected: "",
		},
		{
			name:     "Direct ru",
			header:   "ru-RU,ru;q=0.9",
			expected: LocaleRU,
		},
		{
			name:     "Direct en",
			header:   "en-US,en;q=0.9",
			expected: LocaleEN,
		},
		{
			name:     "English preferred over Russian by weight",
			header:   "ru;q=0.5, en-US;q=0.9, en;q=0.8",
			expected: LocaleEN,
		},
		{
			name:     "Russian preferred over English by weight",
			header:   "en;q=0.3, ru-RU;q=0.9, ru;q=0.8",
			expected: LocaleRU,
		},
		{
			name:     "Unknown language with ru fallback",
			header:   "de-DE,de;q=0.9, fr;q=0.8, ru;q=0.5",
			expected: LocaleRU,
		},
		{
			name:     "Unsupported language only",
			header:   "zh-CN,zh;q=0.9, ja;q=0.8",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAcceptLanguage(tt.header)
			if got != tt.expected {
				t.Errorf("ParseAcceptLanguage(%q) = %q, want %q", tt.header, got, tt.expected)
			}
		})
	}
}

func TestDetectLocaleCascade(t *testing.T) {
	// 1. Query parameter ?lang= has priority over Accept-Language
	req1 := httptest.NewRequest(http.MethodGet, "/opds/v1/feed.xml?lang=en", nil)
	req1.Header.Set("Accept-Language", "ru-RU,ru;q=0.9")
	if loc := DetectLocale(req1, LocaleRU); loc != LocaleEN {
		t.Errorf("expected %q, got %q", LocaleEN, loc)
	}

	// 2. Query parameter ?locale= has priority
	req2 := httptest.NewRequest(http.MethodGet, "/opds/v1/feed.xml?locale=ru", nil)
	req2.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if loc := DetectLocale(req2, LocaleEN); loc != LocaleRU {
		t.Errorf("expected %q, got %q", LocaleRU, loc)
	}

	// 3. Header used if no query param
	req3 := httptest.NewRequest(http.MethodGet, "/opds/v1/feed.xml", nil)
	req3.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if loc := DetectLocale(req3, LocaleRU); loc != LocaleEN {
		t.Errorf("expected %q, got %q", LocaleEN, loc)
	}

	// 4. Default locale used if header is empty
	req4 := httptest.NewRequest(http.MethodGet, "/opds/v1/feed.xml", nil)
	if loc := DetectLocale(req4, LocaleRU); loc != LocaleRU {
		t.Errorf("expected %q, got %q", LocaleRU, loc)
	}
}

func TestTranslator_T(t *testing.T) {
	trRU := NewTranslator(LocaleRU)
	trEN := NewTranslator(LocaleEN)

	// OPDS keys
	if trRU.T("opds.authors") != "По авторам" {
		t.Errorf("unexpected RU opds.authors: %q", trRU.T("opds.authors"))
	}
	if trEN.T("opds.authors") != "By Authors" {
		t.Errorf("unexpected EN opds.authors: %q", trEN.T("opds.authors"))
	}

	// Formatted keys
	if got := trRU.T("opds.search_results", "Толстой"); got != "Результаты поиска: Толстой" {
		t.Errorf("unexpected RU formatted: %q", got)
	}
	if got := trEN.T("opds.search_results", "Tolstoy"); got != "Search Results: Tolstoy" {
		t.Errorf("unexpected EN formatted: %q", got)
	}

	// API Error codes
	if trRU.T("AUTH_INVALID_CREDENTIALS") != "Неверное имя пользователя или пароль" {
		t.Errorf("unexpected RU error message")
	}
	if trEN.T("AUTH_INVALID_CREDENTIALS") != "Invalid username or password" {
		t.Errorf("unexpected EN error message")
	}
}

func TestMiddleware(t *testing.T) {
	mw := Middleware(LocaleRU)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tr := FromContext(r.Context())
		if tr.Locale() != LocaleEN {
			t.Errorf("expected locale %q in context, got %q", LocaleEN, tr.Locale())
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(tr.T("opds.authors")))
	}))

	req := httptest.NewRequest(http.MethodGet, "/?lang=en", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "By Authors" {
		t.Errorf("handler response = %q, want %q", rec.Body.String(), "By Authors")
	}
}

func TestWithLocale(t *testing.T) {
	ctx := context.Background()
	ctx = WithLocale(ctx, LocaleEN)
	tr := FromContext(ctx)
	if tr.Locale() != LocaleEN {
		t.Errorf("expected %q, got %q", LocaleEN, tr.Locale())
	}
}
