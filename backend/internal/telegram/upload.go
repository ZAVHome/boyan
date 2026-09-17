package telegram

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

var supportedExtensions = []string{
	".fb2",
	".fb2.zip",
	".epub",
	".mobi",
	".pdf",
	".djvu",
}

// handleDocument обрабатывает входящий файл книги, загруженный пользователем в чат.
func (b *Bot) handleDocument(ctx context.Context, msg *Message, lang Lang) error {
	doc := msg.Document
	if doc == nil {
		return nil
	}
	msgs := GetMessages(lang)

	lowerName := strings.ToLower(doc.FileName)
	supported := false
	for _, ext := range supportedExtensions {
		if strings.HasSuffix(lowerName, ext) || strings.HasSuffix(lowerName, ".zip") {
			supported = true
			break
		}
	}

	if !supported {
		return b.client.SendMessage(msg.Chat.ID, msgs.UnsupportedFormat(html.EscapeString(doc.FileName)), nil)
	}

	_ = b.client.SendMessage(msg.Chat.ID, msgs.AnalyzingBook(html.EscapeString(doc.FileName)), nil)

	// 1. Получаем путь к файлу в Telegram
	tgFile, err := b.client.GetFile(doc.FileID)
	if err != nil {
		slog.Error("Telegram getFile failed", "file_id", doc.FileID, "err", err)
		return b.client.SendMessage(msg.Chat.ID, msgs.DownloadTgError, nil)
	}

	// 2. Скачиваем байты книги
	fileBytes, err := b.client.DownloadFile(tgFile.FilePath)
	if err != nil {
		slog.Error("Telegram download file failed", "path", tgFile.FilePath, "err", err)
		return b.client.SendMessage(msg.Chat.ID, msgs.DownloadFileError, nil)
	}

	// 3. Сохраняем во временный файл для передачи в пайплайн инжеста
	tempDir := os.TempDir()
	tempFilePath := filepath.Join(tempDir, fmt.Sprintf("tg_upload_%s_%s", doc.FileID, filepath.Base(doc.FileName)))
	if err := os.WriteFile(tempFilePath, fileBytes, 0644); err != nil {
		slog.Error("Failed to write temp book file", "path", tempFilePath, "err", err)
		return b.client.SendMessage(msg.Chat.ID, msgs.SaveTempError, nil)
	}
	defer func() {
		_ = os.Remove(tempFilePath)
	}()

	// 4. Передаем в Ingest pipeline (детекция, парсинг, дедупликация, сохранение)
	res, err := b.watcherInstance.ProcessFile(ctx, tempFilePath, doc.FileName)
	if err != nil {
		slog.Error("Telegram ingest process file failed", "file", doc.FileName, "err", err)
		return b.client.SendMessage(msg.Chat.ID, msgs.ProcessError(html.EscapeString(err.Error())), nil)
	}

	// 5. Оповещаем пользователя о результате
	switch res.Status {
	case "imported":
		return b.client.SendMessage(msg.Chat.ID, msgs.ImportSuccess(html.EscapeString(doc.FileName)), nil)

	case "format_attached":
		return b.client.SendMessage(msg.Chat.ID, msgs.FormatAttached, nil)

	case "quarantined":
		return b.client.SendMessage(msg.Chat.ID, msgs.Quarantined, nil)

	case "skipped":
		return b.client.SendMessage(msg.Chat.ID, msgs.Skipped(html.EscapeString(doc.FileName)), nil)

	default:
		return b.client.SendMessage(msg.Chat.ID, msgs.UnknownStatus(html.EscapeString(res.Status)), nil)
	}
}
