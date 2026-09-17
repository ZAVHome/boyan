package telegram

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"boyan/internal/models"
)

// handleStart отправляет приветственное сообщение и инструкцию.
func (b *Bot) handleStart(chatID int64) error {
	msg := `👋 <b>Добро пожаловать в Боян — Next-Gen OPDS Suite!</b>

Я помогу вам искать, скачивать и пополнять вашу домашнюю библиотеку.

🔍 <b>Поиск книг:</b>
Просто напишите название книги, автора или серию прямо в этот чат (или используйте команду <code>/search Название</code>).

📥 <b>Загрузка книг:</b>
Отправьте мне файл книги (поддерживаются <b>FB2, FB2.ZIP, EPUB, MOBI, PDF, DJVU</b>), и я автоматически добавлю её в библиотеку и проверю на дубликаты.`

	return b.client.SendMessage(chatID, msg, nil)
}

// handleSearch выполняет полнотекстовый поиск по книгам библиотеки.
func (b *Bot) handleSearch(ctx context.Context, chatID int64, query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return b.client.SendMessage(chatID, "Пожалуйста, укажите поисковый запрос (например: <code>/search Булгаков</code>).", nil)
	}

	books, total, err := b.bookRepo.SearchBooksFTS(ctx, query, 0, 5)
	if err != nil {
		slog.Error("Telegram bot search failed", "query", query, "err", err)
		return b.client.SendMessage(chatID, "❌ Произошла ошибка при выполнении поиска.", nil)
	}

	if total == 0 || len(books) == 0 {
		return b.client.SendMessage(chatID, fmt.Sprintf("🔍 По запросу «<i>%s</i>» ничего не найдено.", html.EscapeString(query)), nil)
	}

	intro := fmt.Sprintf("📚 Найдено книг: <b>%d</b> (показаны первые %d):", total, len(books))
	_ = b.client.SendMessage(chatID, intro, nil)

	for _, book := range books {
		b.sendBookCard(chatID, book)
	}

	return nil
}

func (b *Bot) sendBookCard(chatID int64, book models.Book) {
	authorNames := make([]string, 0, len(book.Authors))
	for _, a := range book.Authors {
		authorNames = append(authorNames, a.Name)
	}
	authorsStr := strings.Join(authorNames, ", ")
	if authorsStr == "" {
		authorsStr = "Неизвестный автор"
	}

	var text strings.Builder
	text.WriteString(fmt.Sprintf("📖 <b>%s</b>\n", html.EscapeString(book.Title)))
	text.WriteString(fmt.Sprintf("✍️ <i>%s</i>\n", html.EscapeString(authorsStr)))

	if len(book.Series) > 0 {
		s := book.Series[0]
		if s.Index > 0 {
			text.WriteString(fmt.Sprintf("📚 Серия: %s #%.0f\n", html.EscapeString(s.Name), s.Index))
		} else {
			text.WriteString(fmt.Sprintf("📚 Серия: %s\n", html.EscapeString(s.Name)))
		}
	}

	if book.Annotation != "" {
		ann := book.Annotation
		if len([]rune(ann)) > 200 {
			ann = string([]rune(ann)[:197]) + "..."
		}
		text.WriteString(fmt.Sprintf("\n%s\n", html.EscapeString(ann)))
	}

	// Кнопки скачивания для каждого доступного формата
	var keyboard InlineKeyboardMarkup
	var buttons []InlineKeyboardButton

	for _, f := range book.Files {
		fmtUpper := strings.ToUpper(f.Format)
		icon := "📖"
		if strings.Contains(fmtUpper, "EPUB") {
			icon = "📘"
		} else if strings.Contains(fmtUpper, "MOBI") {
			icon = "📕"
		}

		sizeStr := formatFileSize(f.FileSize)
		label := fmt.Sprintf("%s %s (%s)", icon, fmtUpper, sizeStr)
		buttons = append(buttons, InlineKeyboardButton{
			Text:         label,
			CallbackData: fmt.Sprintf("dl:%s:%s", book.ID, f.Format),
		})
	}

	if len(buttons) > 0 {
		keyboard.InlineKeyboard = [][]InlineKeyboardButton{buttons}
	}

	_ = b.client.SendMessage(chatID, text.String(), &keyboard)
}

// handleCallbackQuery обрабатывает нажатие кнопки скачивания книги.
func (b *Bot) handleCallbackQuery(ctx context.Context, cq *CallbackQuery) error {
	data := cq.Data
	if !strings.HasPrefix(data, "dl:") {
		return nil
	}

	parts := strings.Split(data, ":")
	if len(parts) < 3 {
		return nil
	}

	bookID := parts[1]
	format := parts[2]

	_ = b.client.AnswerCallbackQuery(cq.ID, "Подготовка книги к отправке...")

	fileRecord, err := b.bookRepo.GetBookFileByFormat(ctx, bookID, format)
	if err != nil || fileRecord == nil {
		return b.client.SendMessage(cq.Message.Chat.ID, "❌ Файл книги не найден на сервере.", nil)
	}

	fullPath := fileRecord.FilePath
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(b.cfg.Storage.LibraryDir, fullPath)
	}

	file, err := os.Open(fullPath)
	if err != nil {
		slog.Error("Failed to open book file for telegram", "path", fullPath, "err", err)
		return b.client.SendMessage(cq.Message.Chat.ID, "❌ Не удалось прочитать файл книги на сервере.", nil)
	}
	defer file.Close()

	caption := fmt.Sprintf("Приятного чтения от <b>Бояна</b>! 📚")
	fileName := filepath.Base(fullPath)

	return b.client.SendDocument(cq.Message.Chat.ID, fileName, file, caption)
}

func formatFileSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.0f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}
