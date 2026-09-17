package telegram

import (
	"fmt"
	"strings"
)

// Lang представляет поддерживаемый язык интерфейса Telegram-бота.
type Lang string

const (
	LangRU Lang = "ru"
	LangEN Lang = "en"
)

// NormalizeLang нормализует код языка (ISO 639-1 / BCP 47) к поддерживаемым LangRU или LangEN.
func NormalizeLang(code string) Lang {
	code = strings.ToLower(strings.TrimSpace(code))
	if strings.HasPrefix(code, "ru") || strings.HasPrefix(code, "be") || strings.HasPrefix(code, "uk") {
		return LangRU
	}
	if strings.HasPrefix(code, "en") {
		return LangEN
	}
	if code != "" {
		// Для любых прочих нерусскоязычных локалей (de, fr, es, etc.) используем международный английский
		return LangEN
	}
	return LangRU
}

// UserLang определяет язык по объекту пользователя Telegram.
func UserLang(u *User) Lang {
	if u == nil {
		return LangRU
	}
	return NormalizeLang(u.LanguageCode)
}

// Messages содержит локализованные строковые шаблоны для Telegram-бота.
type Messages struct {
	AccessDenied        string
	StartHelp           string
	LangSwitched        string
	SearchEmptyQuery    string
	SearchError         string
	SearchNotFound      func(query string) string
	SearchFoundIntro    func(total int, count int) string
	UnknownAuthor       string
	SeriesPrefix        func(name string, index float64) string
	PreparingDownload   string
	FileNotFound        string
	FileReadError       string
	HappyReading        func(title string) string
	UnsupportedFormat   func(filename string) string
	AnalyzingBook       func(filename string) string
	DownloadTgError     string
	DownloadFileError   string
	SaveTempError       string
	ProcessError        func(err string) string
	ImportSuccess       func(filename string) string
	FormatAttached      string
	Quarantined         string
	Skipped             func(filename string) string
	CallbackRestricted  string
	UnknownStatus       func(status string) string
}

var ruMessages = Messages{
	AccessDenied: "⛔ Доступ к библиотеке ограничен администратором.",
	StartHelp: `👋 <b>Добро пожаловать в библиотеку «Боян»!</b>

Я помогу вам искать, скачивать и пополнять вашу домашнюю библиотеку.

🔍 <b>Поиск книг:</b>
Просто напишите название книги, автора или серию прямо в этот чат (или используйте команду <code>/search Название</code>).

📥 <b>Загрузка книг:</b>
Отправьте мне файл книги (поддерживаются <b>FB2, FB2.ZIP, EPUB, MOBI, PDF, DJVU</b>), и я автоматически добавлю её в библиотеку и проверю на дубликаты.

🌐 <b>Смена языка / Language:</b>
Используйте команду <code>/lang en</code> или <code>/lang ru</code> для ручного переключения языка.`,
	LangSwitched:     "🇷🇺 Язык интерфейса переключен на <b>Русский</b>.",
	SearchEmptyQuery: "Пожалуйста, укажите поисковый запрос (например: <code>/search Булгаков</code>).",
	SearchError:      "❌ Произошла ошибка при выполнении поиска.",
	SearchNotFound: func(query string) string {
		return fmt.Sprintf("🔍 По запросу «<i>%s</i>» ничего не найдено.", query)
	},
	SearchFoundIntro: func(total int, count int) string {
		return fmt.Sprintf("📚 Найдено книг: <b>%d</b> (показаны первые %d):", total, count)
	},
	UnknownAuthor: "Неизвестный автор",
	SeriesPrefix: func(name string, index float64) string {
		if index > 0 {
			return fmt.Sprintf("📚 Серия: %s #%.0f\n", name, index)
		}
		return fmt.Sprintf("📚 Серия: %s\n", name)
	},
	PreparingDownload: "Подготовка книги к отправке...",
	FileNotFound:      "❌ Файл книги не найден на сервере.",
	FileReadError:     "❌ Не удалось прочитать файл книги на сервере.",
	HappyReading: func(title string) string {
		if title != "" {
			return fmt.Sprintf("📖 <b>%s</b>\nПриятного чтения от <b>Бояна</b>! 📚", title)
		}
		return "Приятного чтения от <b>Бояна</b>! 📚"
	},
	UnsupportedFormat: func(filename string) string {
		return fmt.Sprintf("⚠️ Формат файла <b>%s</b> не поддерживается.\n\nПоддерживаемые форматы: <b>FB2, FB2.ZIP, EPUB, MOBI, PDF, DJVU</b>.", filename)
	},
	AnalyzingBook: func(filename string) string {
		return fmt.Sprintf("⏳ Скачиваю и анализирую книгу <b>%s</b>...", filename)
	},
	DownloadTgError:   "❌ Не удалось получить файл от Telegram.",
	DownloadFileError: "❌ Ошибка скачивания файла книги.",
	SaveTempError:     "❌ Ошибка сохранения временного файла.",
	ProcessError: func(err string) string {
		return fmt.Sprintf("❌ Ошибка при обработке книги: %s", err)
	},
	ImportSuccess: func(filename string) string {
		return fmt.Sprintf("✅ Книга <b>%s</b> успешно добавлена в библиотеку!", filename)
	},
	FormatAttached: "📎 Формат книги успешно прикреплён к уже существующей записи в каталоге!",
	Quarantined:    "⚠️ Обнаружен дубликат книги или конфликт версий. Файл направлен в <b>карантин</b> на модерацию администратором.",
	Skipped: func(filename string) string {
		return fmt.Sprintf("ℹ️ Точная копия файла <b>%s</b> уже есть в каталоге библиотеки.", filename)
	},
	CallbackRestricted: "Доступ ограничен.",
	UnknownStatus: func(status string) string {
		return fmt.Sprintf("ℹ️ Статус обработки: %s", status)
	},
}

var enMessages = Messages{
	AccessDenied: "⛔ Access to the library is restricted by the administrator.",
	StartHelp: `👋 <b>Welcome to Boyan Digital Library!</b>

I will help you search, download, and enrich your home eBook collection.

🔍 <b>Search books:</b>
Simply type a book title, author, or series directly into this chat (or use <code>/search Title</code>).

📥 <b>Upload books:</b>
Send me any eBook file (supported formats: <b>FB2, FB2.ZIP, EPUB, MOBI, PDF, DJVU</b>), and I will automatically import it into the library and check for duplicates.

🌐 <b>Language / Смена языка:</b>
Use <code>/lang en</code> or <code>/lang ru</code> to manually switch languages.`,
	LangSwitched:     "🇬🇧 Interface language switched to <b>English</b>.",
	SearchEmptyQuery: "Please provide a search query (e.g.: <code>/search Tolkien</code>).",
	SearchError:      "❌ An error occurred while searching.",
	SearchNotFound: func(query string) string {
		return fmt.Sprintf("🔍 No books found matching «<i>%s</i>».", query)
	},
	SearchFoundIntro: func(total int, count int) string {
		return fmt.Sprintf("📚 Books found: <b>%d</b> (showing first %d):", total, count)
	},
	UnknownAuthor: "Unknown author",
	SeriesPrefix: func(name string, index float64) string {
		if index > 0 {
			return fmt.Sprintf("📚 Series: %s #%.0f\n", name, index)
		}
		return fmt.Sprintf("📚 Series: %s\n", name)
	},
	PreparingDownload: "Preparing book for delivery...",
	FileNotFound:      "❌ Book file not found on the server.",
	FileReadError:     "❌ Failed to read book file on the server.",
	HappyReading: func(title string) string {
		if title != "" {
			return fmt.Sprintf("📖 <b>%s</b>\nEnjoy reading with <b>Boyan</b>! 📚", title)
		}
		return "Enjoy reading with <b>Boyan</b>! 📚"
	},
	UnsupportedFormat: func(filename string) string {
		return fmt.Sprintf("⚠️ File format of <b>%s</b> is not supported.\n\nSupported formats: <b>FB2, FB2.ZIP, EPUB, MOBI, PDF, DJVU</b>.", filename)
	},
	AnalyzingBook: func(filename string) string {
		return fmt.Sprintf("⏳ Downloading and analyzing <b>%s</b>...", filename)
	},
	DownloadTgError:   "❌ Failed to retrieve file from Telegram.",
	DownloadFileError: "❌ Error downloading book file.",
	SaveTempError:     "❌ Error saving temporary file.",
	ProcessError: func(err string) string {
		return fmt.Sprintf("❌ Error processing book: %s", err)
	},
	ImportSuccess: func(filename string) string {
		return fmt.Sprintf("✅ Book <b>%s</b> successfully added to library!", filename)
	},
	FormatAttached: "📎 Book format successfully attached to existing catalog entry!",
	Quarantined:    "⚠️ Duplicate book or version conflict detected. File moved to <b>quarantine</b> for moderation.",
	Skipped: func(filename string) string {
		return fmt.Sprintf("ℹ️ An exact duplicate of <b>%s</b> already exists in the library catalog.", filename)
	},
	CallbackRestricted: "Access restricted.",
	UnknownStatus: func(status string) string {
		return fmt.Sprintf("ℹ️ Processing status: %s", status)
	},
}

// GetMessages возвращает словарь сообщений для заданного языка.
func GetMessages(lang Lang) Messages {
	if lang == LangEN {
		return enMessages
	}
	return ruMessages
}
