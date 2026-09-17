package i18n

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type contextKey string

const (
	localeKey contextKey = "boyan_locale"

	LocaleRU = "ru"
	LocaleEN = "en"

	DefaultLocale = LocaleRU
)

// SupportedLocales содержит перечень поддерживаемых языков.
var SupportedLocales = map[string]bool{
	LocaleRU: true,
	LocaleEN: true,
}

// Translator инкапсулирует локализацию для конкретного языка.
type Translator struct {
	locale string
}

// NewTranslator создает новый экземпляр переводчика для указанного языка.
func NewTranslator(locale string) *Translator {
	loc := NormalizeLocale(locale)
	return &Translator{locale: loc}
}

// Locale возвращает активный язык переводчика.
func (t *Translator) Locale() string {
	if t == nil || t.locale == "" {
		return DefaultLocale
	}
	return t.locale
}

// T возвращает перевод по ключу с опциональным форматированием аргументов.
func (t *Translator) T(key string, args ...any) string {
	loc := t.Locale()
	dict, ok := dictionaries[loc]
	if !ok {
		dict = dictionaries[DefaultLocale]
	}

	val, found := dict[key]
	if !found {
		// Fallback на дефолтный словарь
		if fallbackDict, okFall := dictionaries[DefaultLocale]; okFall {
			val, found = fallbackDict[key]
		}
	}

	if !found {
		val = key
	}

	if len(args) > 0 {
		return fmt.Sprintf(val, args...)
	}
	return val
}

// Sprintf алиас для T с аргументами.
func (t *Translator) Sprintf(key string, args ...any) string {
	return t.T(key, args...)
}

// NormalizeLocale нормализует строку локали к поддерживаемым значениям (ru, en).
func NormalizeLocale(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return DefaultLocale
	}
	if strings.HasPrefix(raw, "ru") {
		return LocaleRU
	}
	if strings.HasPrefix(raw, "en") {
		return LocaleEN
	}
	if SupportedLocales[raw] {
		return raw
	}
	return DefaultLocale
}

// ParseAcceptLanguage анализирует заголовок Accept-Language с учетом весов качества q=.
func ParseAcceptLanguage(header string) string {
	if header == "" {
		return ""
	}

	type langPref struct {
		lang string
		q    float64
	}

	var prefs []langPref
	parts := strings.Split(header, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		subparts := strings.Split(p, ";")
		lang := strings.TrimSpace(strings.ToLower(subparts[0]))
		q := 1.0
		if len(subparts) > 1 {
			for _, param := range subparts[1:] {
				param = strings.TrimSpace(param)
				if strings.HasPrefix(param, "q=") {
					if val, err := strconv.ParseFloat(param[2:], 64); err == nil {
						q = val
					}
				}
			}
		}
		prefs = append(prefs, langPref{lang: lang, q: q})
	}

	// Сортировка по убыванию q
	for i := 0; i < len(prefs); i++ {
		for j := i + 1; j < len(prefs); j++ {
			if prefs[j].q > prefs[i].q {
				prefs[i], prefs[j] = prefs[j], prefs[i]
			}
		}
	}

	for _, pref := range prefs {
		if strings.HasPrefix(pref.lang, "ru") {
			return LocaleRU
		}
		if strings.HasPrefix(pref.lang, "en") {
			return LocaleEN
		}
	}

	return ""
}

// DetectLocale определяет язык запроса по каскаду:
// 1. Query-параметр ?lang= или ?locale=
// 2. HTTP-заголовок Accept-Language
// 3. Значение по умолчанию (defaultLocale)
func DetectLocale(r *http.Request, defaultLocale string) string {
	// 1. Проверяем query-параметр (критично для E-Ink читалок и ручного переключения)
	if q := r.URL.Query().Get("lang"); q != "" {
		if loc := NormalizeLocale(q); loc != "" {
			return loc
		}
	}
	if q := r.URL.Query().Get("locale"); q != "" {
		if loc := NormalizeLocale(q); loc != "" {
			return loc
		}
	}

	// 2. Проверяем заголовок Accept-Language
	if accept := r.Header.Get("Accept-Language"); accept != "" {
		if loc := ParseAcceptLanguage(accept); loc != "" {
			return loc
		}
	}

	// 3. Fallback на дефолтный язык
	if defaultLocale != "" {
		return NormalizeLocale(defaultLocale)
	}
	return DefaultLocale
}

// Middleware возвращает HTTP-middleware, определяющий язык запроса и инжектирующий Translator в context.Context.
func Middleware(defaultLocale string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			loc := DetectLocale(r, defaultLocale)
			ctx := context.WithValue(r.Context(), localeKey, NewTranslator(loc))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// FromContext извлекает Translator из контекста запроса.
func FromContext(ctx context.Context) *Translator {
	if ctx == nil {
		return NewTranslator(DefaultLocale)
	}
	if t, ok := ctx.Value(localeKey).(*Translator); ok && t != nil {
		return t
	}
	return NewTranslator(DefaultLocale)
}

// WithLocale возвращает новый контекст с установленной локалью.
func WithLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, localeKey, NewTranslator(locale))
}

// Словари переводов
var dictionaries = map[string]map[string]string{
	LocaleRU: {
		// OPDS v1.2 / v2.0 Разделы
		"opds.authors":             "По авторам",
		"opds.authors_desc":        "Поиск книг по алфавитному каталогу авторов",
		"opds.series":              "По сериям и циклам",
		"opds.series_desc":         "Книжные циклы и сериалы с нумерацией частей",
		"opds.genres":              "По жанрам и категориям",
		"opds.genres_desc":         "Тематическое дерево жанров FictionBook",
		"opds.recent":              "Новые поступления",
		"opds.recent_desc":         "Недавно добавленные в библиотеку книги",
		"opds.authors_alpha":       "Авторы по алфавиту",
		"opds.authors_letter":      "Авторы на букву %s",
		"opds.author_books":        "Книги автора",
		"opds.series_alpha":        "Серии по алфавиту",
		"opds.series_letter":       "Серии на букву %s",
		"opds.series_books":        "Книги серии",
		"opds.genre_categories":    "Категории жанров",
		"opds.search_results":      "Результаты поиска: %s",
		"opds.search_desc":         "Поиск по каталогу библиотеки",
		"opds.search_query_desc":   "Поиск по запросу '%s'",
		"opds.auth_realm":          "Боян",

		// Ошибки и коды REST API
		"AUTH_REQUIRED":            "Требуется авторизация",
		"AUTH_INVALID_CREDENTIALS": "Неверное имя пользователя или пароль",
		"AUTH_USER_INACTIVE":       "Учетная запись пользователя отключена",
		"AUTH_ADMIN_REQUIRED":      "Требуются права администратора",
		"AUTH_FIELDS_REQUIRED":     "Имя пользователя и пароль обязательны для заполнения",
		"AUTH_TOKEN_GEN_FAILED":    "Не удалось сформировать токен авторизации",
		"AUTH_LOGGED_OUT":          "Успешный выход из системы",

		"BOOK_NOT_FOUND":           "Книга не найдена",
		"COVER_NOT_FOUND":          "Обложка не найдена",
		"BOOK_ID_REQUIRED":         "Идентификатор книги обязателен",
		"FILE_NOT_FOUND":           "Файл не найден на диске",
		"FORMAT_NOT_FOUND":         "Формат файла не найден для этой книги",
		"ZIP_STREAM_FAILED":        "Ошибка потокового чтения из архива: %v",
		"INVALID_JSON":             "Некорректный формат данных JSON",
		"DB_ERROR":                 "Ошибка базы данных",
		"SEARCH_FAILED":            "Ошибка выполнения поискового запроса",

		"UPLOAD_FAILED":            "Ошибка обработки книги: %v",
		"UPLOAD_MISSING_FILE":      "Отсутствует файл в форме запроса ('file')",
		"UPLOAD_PARSE_FAILED":      "Не удалось разобрать multipart форму: %v",
		"UPLOAD_TEMP_FAILED":       "Не удалось создать временный файл",
		"UPLOAD_SAVE_FAILED":       "Не удалось сохранить загруженный файл",

		"QUARANTINE_NOT_FOUND":     "Элемент карантина не найден",
		"QUARANTINE_ID_REQUIRED":   "Идентификатор элемента карантина обязателен",
		"QUARANTINE_INVALID_ACTION": "Недопустимое действие. Разрешены: 'discard', 'replace', 'attach_format', 'keep_both'",
		"QUARANTINE_NO_EXISTING":   "Связанная книга не найдена для выполнения операции",
		"QUARANTINE_DISCARDED":     "Элемент карантина успешно удален",
		"QUARANTINE_REPLACED":      "Файл книги успешно заменен",
		"QUARANTINE_ATTACHED":      "Формат успешно прикреплен к книге",
		"QUARANTINE_KEPT_BOTH":     "Импортировано как новая книга",
		"QUARANTINE_LIST_FAILED":   "Не удалось загрузить список карантина",
		"QUARANTINE_PARSE_FAILED":  "Не удалось разобрать файл карантина: %v",
		"QUARANTINE_COPY_FAILED":   "Не удалось скопировать файл в библиотеку: %v",
		"QUARANTINE_SAVE_FAILED":   "Не удалось сохранить новую книгу: %v",

		"SHELF_INVALID_TYPE":       "Недопустимый тип полки. Допустимо: 'reading', 'finished' или 'favorite'",
		"SHELF_ADD_FAILED":         "Не удалось добавить книгу на полку",
		"SHELF_REMOVE_FAILED":      "Не удалось удалить книгу с полки",
		"SHELF_GET_FAILED":         "Не удалось получить список книг на полке",
		"PROGRESS_SAVE_FAILED":     "Не удалось сохранить прогресс чтения",
		"SHELF_REQUIRED":           "Необходимо указать тип полки",
		"BOOK_OR_SHELF_REQUIRED":   "Необходимо указать ID книги и тип полки",

		"CALIBRE_PATH_REQUIRED":    "Путь к библиотеке Calibre обязателен",

		// Watcher & Импорт
		"WATCHER_IMPORTED":         "Новая книга успешно импортирована",
		"WATCHER_EXACT_DUPLICATE":  "Обнаружен точный дубликат (SHA-256); отправлен в карантин",
		"WATCHER_SAME_FORMAT":      "Книга в данном формате уже существует с другим хешем; отправлена в карантин",
		"WATCHER_EXACT_SKIPPED":    "Точная копия файла уже существует в библиотеке",
		"WATCHER_FORMAT_ATTACHED":  "Новый формат %s прикреплен к существующей книге %s",
	},
	LocaleEN: {
		// OPDS v1.2 / v2.0 Sections
		"opds.authors":             "By Authors",
		"opds.authors_desc":        "Browse books by author alphabetical index",
		"opds.series":              "By Series",
		"opds.series_desc":         "Book series and cycles with part numbering",
		"opds.genres":              "By Genres & Categories",
		"opds.genres_desc":         "Thematic FictionBook genre tree",
		"opds.recent":              "Recent Additions",
		"opds.recent_desc":         "Recently added books to the library",
		"opds.authors_alpha":       "Authors by Alphabet",
		"opds.authors_letter":      "Authors starting with %s",
		"opds.author_books":        "Author's Books",
		"opds.series_alpha":        "Series by Alphabet",
		"opds.series_letter":       "Series starting with %s",
		"opds.series_books":        "Series Books",
		"opds.genre_categories":    "Genre Categories",
		"opds.search_results":      "Search Results: %s",
		"opds.search_desc":         "Search library catalog",
		"opds.search_query_desc":   "Search query '%s'",
		"opds.auth_realm":          "Boyan",

		// REST API Errors and Codes
		"AUTH_REQUIRED":            "Authentication required",
		"AUTH_INVALID_CREDENTIALS": "Invalid username or password",
		"AUTH_USER_INACTIVE":       "User account is inactive",
		"AUTH_ADMIN_REQUIRED":      "Administrator privileges required",
		"AUTH_FIELDS_REQUIRED":     "Username and password are required",
		"AUTH_TOKEN_GEN_FAILED":    "Failed to generate authentication token",
		"AUTH_LOGGED_OUT":          "Logged out successfully",

		"BOOK_NOT_FOUND":           "Book not found",
		"COVER_NOT_FOUND":          "Cover not found",
		"BOOK_ID_REQUIRED":         "Book ID is required",
		"FILE_NOT_FOUND":           "File not found on disk",
		"FORMAT_NOT_FOUND":         "File format not found for this book",
		"ZIP_STREAM_FAILED":        "Failed to stream from archive: %v",
		"INVALID_JSON":             "Invalid JSON payload",
		"DB_ERROR":                 "Database error",
		"SEARCH_FAILED":            "Search failed",

		"UPLOAD_FAILED":            "Failed to process book: %v",
		"UPLOAD_MISSING_FILE":      "Missing 'file' in form-data",
		"UPLOAD_PARSE_FAILED":      "Failed to parse multipart form: %v",
		"UPLOAD_TEMP_FAILED":       "Failed to create temp file",
		"UPLOAD_SAVE_FAILED":       "Failed to save uploaded file",

		"QUARANTINE_NOT_FOUND":     "Quarantine item not found",
		"QUARANTINE_ID_REQUIRED":   "Quarantine item ID is required",
		"QUARANTINE_INVALID_ACTION": "Invalid action. Allowed: 'discard', 'replace', 'attach_format', 'keep_both'",
		"QUARANTINE_NO_EXISTING":   "No existing book linked to perform this action",
		"QUARANTINE_DISCARDED":     "Quarantine item discarded",
		"QUARANTINE_REPLACED":      "Book file replaced successfully",
		"QUARANTINE_ATTACHED":      "Format attached successfully",
		"QUARANTINE_KEPT_BOTH":     "Imported as new book",
		"QUARANTINE_LIST_FAILED":   "Failed to list quarantine items",
		"QUARANTINE_PARSE_FAILED":  "Failed to parse quarantine file: %v",
		"QUARANTINE_COPY_FAILED":   "Failed to copy file to library: %v",
		"QUARANTINE_SAVE_FAILED":   "Failed to save new book: %v",

		"SHELF_INVALID_TYPE":       "Invalid shelf type. Allowed: 'reading', 'finished', or 'favorite'",
		"SHELF_ADD_FAILED":         "Failed to add book to shelf",
		"SHELF_REMOVE_FAILED":      "Failed to remove book from shelf",
		"SHELF_GET_FAILED":         "Failed to get shelf books",
		"PROGRESS_SAVE_FAILED":     "Failed to save reading progress",
		"SHELF_REQUIRED":           "Shelf type is required",
		"BOOK_OR_SHELF_REQUIRED":   "Missing book ID or shelf type",

		"CALIBRE_PATH_REQUIRED":    "Path to Calibre library is required",

		// Watcher & Import
		"WATCHER_IMPORTED":         "Successfully imported new book",
		"WATCHER_EXACT_DUPLICATE":  "Exact SHA-256 match found; sent to quarantine",
		"WATCHER_SAME_FORMAT":      "Same book and format already exists with different hash; sent to quarantine",
		"WATCHER_EXACT_SKIPPED":    "Exact file already exists in library",
		"WATCHER_FORMAT_ATTACHED":  "New format %s attached to existing book %s",
	},
}
