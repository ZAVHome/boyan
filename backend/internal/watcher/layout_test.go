package watcher

import (
	"path/filepath"
	"strings"
	"testing"

	"boyan/internal/models"
)

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"Normal Name", "Normal Name"},
		{"Invalid: / \\ * ? < > | chars", "Invalid_ _ _ _ _ _ _ _ chars"},
		{"  Trailing dots and spaces...  ", "Trailing dots and spaces"},
		{"", "Unknown"},
		{"   ", "Unknown"},
	}

	for _, c := range cases {
		got := SanitizeFilename(c.input)
		if got != c.expected {
			t.Errorf("SanitizeFilename(%q) = %q, expected %q", c.input, got, c.expected)
		}
	}
}

func TestFormatPath(t *testing.T) {
	template := "{Author}/{Series}/{SeriesIndex:02d} - {Title}.{ext}"

	book := &models.Book{
		ID:    "book-1",
		Title: "Война и мир: Том 1",
	}
	authors := []models.AuthorDetail{
		{Author: models.Author{Name: "Толстой, Лев Николаевич"}},
	}
	series := []models.SeriesDetail{
		{Series: models.Series{Name: "Собрание сочинений"}, Index: 1},
	}

	res := FormatPath(template, book, authors, series, "fb2.zip")
	// На Windows разделители могут быть '\', на Linux '/'
	expectedParts := []string{"Толстой, Лев Николаевич", "Собрание сочинений", "01 - Война и мир_ Том 1.fb2.zip"}
	expected := filepath.Join(expectedParts...)

	if res != expected {
		t.Errorf("FormatPath with series = %q, expected %q", res, expected)
	}

	// Проверка книги без серии
	resNoSeries := FormatPath(template, book, authors, nil, "fb2")
	expectedNoSeriesParts := []string{"Толстой, Лев Николаевич", "Война и мир_ Том 1.fb2"}
	expectedNoSeries := filepath.Join(expectedNoSeriesParts...)

	if resNoSeries != expectedNoSeries {
		t.Errorf("FormatPath without series = %q, expected %q", resNoSeries, expectedNoSeries)
	}

	// Проверка, что запрещенные символы убраны
	if strings.ContainsAny(res, `<>:"/\|?*`[2:]) {
		// Не считаем системные слеши пути
		cleaned := strings.ReplaceAll(res, string(filepath.Separator), "")
		if strings.ContainsAny(cleaned, `<>:"/\|?*`) {
			t.Errorf("FormatPath contains forbidden chars: %s", res)
		}
	}
}
