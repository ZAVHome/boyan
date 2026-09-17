package fb2

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"boyan/internal/models"

	"golang.org/x/net/html/charset"
)

// FB2BookInfo содержит извлеченные метаданные и бинарные данные обложки.
type FB2BookInfo struct {
	Book        models.Book
	Authors     []models.AuthorDetail
	Series      []models.SeriesDetail
	Genres      []models.Genre
	CoverBytes  []byte
	CoverFormat string // "image/jpeg", "image/png"
}

// XML-структуры для десериализации метаданных FB2
type fb2FictionBook struct {
	XMLName     xml.Name       `xml:"FictionBook"`
	Description fb2Description `xml:"description"`
}

type fb2Description struct {
	TitleInfo   fb2TitleInfo   `xml:"title-info"`
	PublishInfo fb2PublishInfo `xml:"publish-info"`
}

type fb2TitleInfo struct {
	Genres     []string      `xml:"genre"`
	Authors    []fb2Author   `xml:"author"`
	BookTitle  string        `xml:"book-title"`
	Annotation fb2Annotation `xml:"annotation"`
	Date       fb2Date       `xml:"date"`
	Coverpage  fb2Coverpage  `xml:"coverpage"`
	Lang       string        `xml:"lang"`
	SrcLang    string        `xml:"src-lang"`
	Sequences  []fb2Sequence `xml:"sequence"`
}

type fb2Author struct {
	FirstName  string `xml:"first-name"`
	MiddleName string `xml:"middle-name"`
	LastName   string `xml:"last-name"`
	Nickname   string `xml:"nickname"`
}

type fb2Annotation struct {
	InnerXML string `xml:",innerxml"`
}

type fb2Date struct {
	Value string `xml:"value,attr"`
	Text  string `xml:",chardata"`
}

type fb2Coverpage struct {
	Images []fb2Image `xml:"image"`
}

type fb2Image struct {
	Href string `xml:"href,attr"`
}

type fb2Sequence struct {
	Name   string `xml:"name,attr"`
	Number string `xml:"number,attr"`
}

type fb2PublishInfo struct {
	Publisher string `xml:"publisher"`
	Year      string `xml:"year"`
	ISBN      string `xml:"isbn"`
}

var tagCleaner = regexp.MustCompile(`<[^>]*>`)

// ParseFB2 выполняет потоковый парсинг книги FictionBook с поддержкой любых кодировок.
func ParseFB2(r io.Reader) (*FB2BookInfo, error) {
	// Создаем потоковый декодер с автоопределением кодировок
	decoder := xml.NewDecoder(r)
	decoder.CharsetReader = charset.NewReaderLabel

	info := &FB2BookInfo{}
	var coverHref string
	var binaries = make(map[string][]byte)
	var binaryContentTypes = make(map[string]string)

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("xml decode token: %w", err)
		}

		switch elem := token.(type) {
		case xml.StartElement:
			if elem.Name.Local == "description" {
				var desc fb2Description
				if err := decoder.DecodeElement(&desc, &elem); err != nil {
					return nil, fmt.Errorf("decode description element: %w", err)
				}
				processDescription(&desc, info, &coverHref)
			} else if elem.Name.Local == "binary" {
				// Извлекаем атрибуты binary
				var id, contentType string
				for _, attr := range elem.Attr {
					if attr.Name.Local == "id" {
						id = attr.Value
					} else if attr.Name.Local == "content-type" {
						contentType = attr.Value
					}
				}

				// Читаем тело binary (base64)
				var rawBase64 string
				if err := decoder.DecodeElement(&rawBase64, &elem); err == nil && id != "" {
					cleaned := strings.Map(func(r rune) rune {
						if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
							return -1
						}
						return r
					}, rawBase64)

					decoded, err := base64.StdEncoding.DecodeString(cleaned)
					if err == nil {
						binaries[id] = decoded
						binaryContentTypes[id] = contentType
					}
				}
			}
		}
	}

	// Сопоставляем обложку
	matchCover(info, coverHref, binaries, binaryContentTypes)

	return info, nil
}

func processDescription(desc *fb2Description, info *FB2BookInfo, coverHref *string) {
	ti := desc.TitleInfo

	// Основные поля книги
	info.Book.Title = strings.TrimSpace(ti.BookTitle)
	if info.Book.Title == "" {
		info.Book.Title = "Без названия"
	}
	info.Book.Language = strings.TrimSpace(ti.Lang)
	if info.Book.Language == "" {
		info.Book.Language = "ru"
	}
	info.Book.Annotation = cleanAnnotation(ti.Annotation.InnerXML)

	// Дата публикации
	if ti.Date.Value != "" {
		info.Book.PublishedDate = strings.TrimSpace(ti.Date.Value)
	} else {
		info.Book.PublishedDate = strings.TrimSpace(ti.Date.Text)
	}

	// Данные издательства
	info.Book.Publisher = strings.TrimSpace(desc.PublishInfo.Publisher)
	info.Book.ISBN = strings.TrimSpace(desc.PublishInfo.ISBN)
	if info.Book.PublishedDate == "" && desc.PublishInfo.Year != "" {
		info.Book.PublishedDate = strings.TrimSpace(desc.PublishInfo.Year)
	}

	// Авторы
	for i, a := range ti.Authors {
		name, sortName := formatAuthorName(a)
		if name != "" {
			info.Authors = append(info.Authors, models.AuthorDetail{
				Author: models.Author{
					Name:     name,
					SortName: sortName,
				},
				Role:  "author",
				Order: i + 1,
			})
		}
	}

	// Серии
	for _, s := range ti.Sequences {
		name := strings.TrimSpace(s.Name)
		if name != "" {
			var idx float64
			if s.Number != "" {
				idx, _ = strconv.ParseFloat(strings.TrimSpace(s.Number), 64)
			}
			info.Series = append(info.Series, models.SeriesDetail{
				Series: models.Series{
					Name:     name,
					SortName: name,
				},
				Index: idx,
			})
		}
	}

	// Жанры
	for _, gCode := range ti.Genres {
		if strings.TrimSpace(gCode) != "" {
			info.Genres = append(info.Genres, NormalizeGenre(gCode))
		}
	}

	// Ссылка на обложку
	for _, img := range ti.Coverpage.Images {
		href := strings.TrimPrefix(img.Href, "#")
		if href != "" {
			*coverHref = href
			break
		}
	}
}

func formatAuthorName(a fb2Author) (string, string) {
	fn := strings.TrimSpace(a.FirstName)
	mn := strings.TrimSpace(a.MiddleName)
	ln := strings.TrimSpace(a.LastName)
	nn := strings.TrimSpace(a.Nickname)

	var parts []string
	if fn != "" {
		parts = append(parts, fn)
	}
	if mn != "" {
		parts = append(parts, mn)
	}
	if ln != "" {
		parts = append(parts, ln)
	}

	if len(parts) > 0 {
		fullName := strings.Join(parts, " ")
		var sortParts []string
		if ln != "" {
			sortParts = append(sortParts, ln)
		}
		if fn != "" || mn != "" {
			sortParts = append(sortParts, strings.TrimSpace(fn+" "+mn))
		}
		return fullName, strings.Join(sortParts, ", ")
	}

	if nn != "" {
		return nn, nn
	}
	return "", ""
}

func cleanAnnotation(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// Заменяем <empty-line/> и </p> на перевод строки
	replacer := strings.NewReplacer(
		"<empty-line/>", "\n\n",
		"<empty-line />", "\n\n",
		"</p>", "\n\n",
		"<p>", "",
		"<br>", "\n",
		"<br/>", "\n",
		"<br />", "\n",
	)
	clean := replacer.Replace(raw)
	// Удаляем оставшиеся теги
	clean = tagCleaner.ReplaceAllString(clean, "")
	clean = strings.ReplaceAll(clean, "\n\n\n", "\n\n")
	return strings.TrimSpace(clean)
}

func matchCover(info *FB2BookInfo, coverHref string, binaries map[string][]byte, contentTypes map[string]string) {
	// 1. Поиск по прямому совпадению id
	if coverHref != "" {
		if data, ok := binaries[coverHref]; ok {
			info.CoverBytes = data
			info.CoverFormat = contentTypes[coverHref]
			return
		}
	}

	// 2. Эвристический поиск по имени ключа (cover, cover.jpg, cover.png)
	for id, data := range binaries {
		lower := strings.ToLower(id)
		if strings.Contains(lower, "cover") {
			info.CoverBytes = data
			info.CoverFormat = contentTypes[id]
			return
		}
	}

	// 3. Если ничего не найдено, берем первый доступный бинарник изображения
	for id, data := range binaries {
		ct := contentTypes[id]
		if strings.HasPrefix(ct, "image/") || isImageHeader(data) {
			info.CoverBytes = data
			info.CoverFormat = ct
			return
		}
	}
}

func isImageHeader(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	// JPEG header: FF D8 FF
	if bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}) {
		return true
	}
	// PNG header: 89 50 4E 47
	if bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47}) {
		return true
	}
	return false
}
