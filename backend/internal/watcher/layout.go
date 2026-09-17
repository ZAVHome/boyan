package watcher

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"boyan/internal/models"
)

var (
	// invalidCharsRegex удаляет или заменяет запрещенные символы в файловых системах Windows и Linux.
	invalidCharsRegex = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	multiSpaceRegex   = regexp.MustCompile(`\s+`)
)

// SanitizeFilename очищает имя папки или файла от недопустимых символов.
func SanitizeFilename(name string) string {
	cleaned := invalidCharsRegex.ReplaceAllString(name, "_")
	cleaned = multiSpaceRegex.ReplaceAllString(cleaned, " ")
	cleaned = strings.Trim(cleaned, " ._")
	if cleaned == "" {
		return "Unknown"
	}
	// Ограничиваем длину компонента пути (например, 120 символов)
	if len([]rune(cleaned)) > 120 {
		cleaned = string([]rune(cleaned)[:120])
		cleaned = strings.Trim(cleaned, " ._")
	}
	return cleaned
}

// FormatPath формирует относительный путь для сохранения книги в библиотеке по заданному шаблону.
// Доступные макросы: {Author}, {Title}, {Series}, {SeriesIndex:02d}, {SeriesIndex}, {BookID}, {ext}
func FormatPath(
	template string,
	book *models.Book,
	authors []models.AuthorDetail,
	series []models.SeriesDetail,
	ext string,
) string {
	if template == "" {
		template = "{Author}/{Series}/{SeriesIndex:02d} - {Title}.{ext}"
	}

	authorStr := "Неизвестный автор"
	if len(authors) > 0 && authors[0].Name != "" {
		authorStr = authors[0].Name
	}

	titleStr := "Без названия"
	if book.Title != "" {
		titleStr = book.Title
	}

	seriesName := ""
	seriesIndex := 0.0
	hasSeries := false
	if len(series) > 0 && series[0].Name != "" {
		seriesName = series[0].Name
		seriesIndex = series[0].Index
		hasSeries = true
	}

	// Если серии нет, но шаблон содержит {Series}, упрощаем шаблон до формата без серии
	activeTemplate := template
	if !hasSeries {
		if strings.Contains(activeTemplate, "{Series}/") {
			activeTemplate = strings.ReplaceAll(activeTemplate, "{Series}/", "")
		} else if strings.Contains(activeTemplate, "{Series}\\") {
			activeTemplate = strings.ReplaceAll(activeTemplate, "{Series}\\", "")
		}
		// Убираем {SeriesIndex:02d} - или {SeriesIndex} -
		activeTemplate = strings.ReplaceAll(activeTemplate, "{SeriesIndex:02d} - ", "")
		activeTemplate = strings.ReplaceAll(activeTemplate, "{SeriesIndex} - ", "")
		activeTemplate = strings.ReplaceAll(activeTemplate, "{SeriesIndex:02d}", "")
		activeTemplate = strings.ReplaceAll(activeTemplate, "{SeriesIndex}", "")
		activeTemplate = strings.ReplaceAll(activeTemplate, "{Series}", "")
	}

	// Санитизируем каждую часть
	cleanAuthor := SanitizeFilename(authorStr)
	cleanTitle := SanitizeFilename(titleStr)
	cleanSeries := SanitizeFilename(seriesName)
	cleanExt := strings.TrimPrefix(ext, ".")

	seriesIndexFormatted := ""
	seriesIndex02d := ""
	if hasSeries {
		if seriesIndex == float64(int(seriesIndex)) {
			seriesIndexFormatted = fmt.Sprintf("%d", int(seriesIndex))
			seriesIndex02d = fmt.Sprintf("%02d", int(seriesIndex))
		} else {
			seriesIndexFormatted = fmt.Sprintf("%.1f", seriesIndex)
			seriesIndex02d = fmt.Sprintf("%04.1f", seriesIndex)
		}
	}

	res := activeTemplate
	res = strings.ReplaceAll(res, "{Author}", cleanAuthor)
	res = strings.ReplaceAll(res, "{Title}", cleanTitle)
	res = strings.ReplaceAll(res, "{Series}", cleanSeries)
	res = strings.ReplaceAll(res, "{SeriesIndex:02d}", seriesIndex02d)
	res = strings.ReplaceAll(res, "{SeriesIndex}", seriesIndexFormatted)
	res = strings.ReplaceAll(res, "{BookID}", book.ID)
	res = strings.ReplaceAll(res, "{ext}", cleanExt)

	// Нормализуем слеши пути
	cleanPath := filepath.Clean(filepath.FromSlash(res))
	// Убираем возможные двойные слеши и пустые папки
	parts := strings.Split(cleanPath, string(filepath.Separator))
	var filteredParts []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && p != "." {
			filteredParts = append(filteredParts, p)
		}
	}

	return filepath.Join(filteredParts...)
}
