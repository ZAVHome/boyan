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
func (b *Bot) handleStart(chatID int64, lang Lang) error {
	msgs := GetMessages(lang)
	return b.client.SendMessage(chatID, msgs.StartHelp, nil)
}

// handleSearch выполняет полнотекстовый поиск по книгам библиотеки.
func (b *Bot) handleSearch(ctx context.Context, chatID int64, query string, lang Lang) error {
	query = strings.TrimSpace(query)
	msgs := GetMessages(lang)

	if query == "" {
		return b.client.SendMessage(chatID, msgs.SearchEmptyQuery, nil)
	}

	books, total, err := b.bookRepo.SearchBooksFTS(ctx, query, 0, 5)
	if err != nil {
		slog.Error("Telegram bot search failed", "query", query, "err", err)
		return b.client.SendMessage(chatID, msgs.SearchError, nil)
	}

	if total == 0 || len(books) == 0 {
		return b.client.SendMessage(chatID, msgs.SearchNotFound(html.EscapeString(query)), nil)
	}

	intro := msgs.SearchFoundIntro(total, len(books))
	_ = b.client.SendMessage(chatID, intro, nil)

	for _, book := range books {
		b.sendBookCard(chatID, book, lang)
	}

	return nil
}

func (b *Bot) sendBookCard(chatID int64, book models.Book, lang Lang) {
	msgs := GetMessages(lang)
	authorNames := make([]string, 0, len(book.Authors))
	for _, a := range book.Authors {
		authorNames = append(authorNames, a.Name)
	}
	authorsStr := strings.Join(authorNames, ", ")
	if authorsStr == "" {
		authorsStr = msgs.UnknownAuthor
	}

	var text strings.Builder
	text.WriteString(fmt.Sprintf("📖 <b>%s</b>\n", html.EscapeString(book.Title)))
	text.WriteString(fmt.Sprintf("✍️ <i>%s</i>\n", html.EscapeString(authorsStr)))

	if len(book.Series) > 0 {
		s := book.Series[0]
		text.WriteString(msgs.SeriesPrefix(html.EscapeString(s.Name), s.Index))
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
func (b *Bot) handleCallbackQuery(ctx context.Context, cq *CallbackQuery, lang Lang) error {
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
	msgs := GetMessages(lang)

	_ = b.client.AnswerCallbackQuery(cq.ID, msgs.PreparingDownload)

	fileRecord, err := b.bookRepo.GetBookFileByFormat(ctx, bookID, format)
	if err != nil || fileRecord == nil {
		return b.client.SendMessage(cq.Message.Chat.ID, msgs.FileNotFound, nil)
	}

	fullPath := fileRecord.FilePath
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(b.cfg.Storage.LibraryDir, fullPath)
	}

	file, err := os.Open(fullPath)
	if err != nil {
		slog.Error("Failed to open book file for telegram", "path", fullPath, "err", err)
		return b.client.SendMessage(cq.Message.Chat.ID, msgs.FileReadError, nil)
	}
	defer file.Close()

	var bookTitle string
	if book, err := b.bookRepo.GetBookByID(ctx, bookID); err == nil && book != nil {
		bookTitle = html.EscapeString(book.Title)
	}
	caption := msgs.HappyReading(bookTitle)
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
