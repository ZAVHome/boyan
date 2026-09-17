package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"time"
)

// Модели Telegram Bot API

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

type Message struct {
	MessageID int64     `json:"message_id"`
	From      *User     `json:"from,omitempty"`
	Chat      Chat      `json:"chat"`
	Date      int64     `json:"date"`
	Text      string    `json:"text,omitempty"`
	Document  *Document `json:"document,omitempty"`
	Caption   string    `json:"caption,omitempty"`
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

type Chat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"` // "private", "group", "supergroup", "channel"
	Title string `json:"title,omitempty"`
}

type Document struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

type TelegramFile struct {
	FileID   string `json:"file_id"`
	FilePath string `json:"file_path,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
}

type apiResponse[T any] struct {
	OK          bool   `json:"ok"`
	Result      T      `json:"result"`
	Description string `json:"description,omitempty"`
}

// Client реализует HTTP-клиент для взаимодействия с Telegram Bot API.
type Client struct {
	token      string
	baseURL    string
	fileURL    string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token:      token,
		baseURL:    fmt.Sprintf("https://api.telegram.org/bot%s", token),
		fileURL:    fmt.Sprintf("https://api.telegram.org/file/bot%s", token),
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// GetUpdates получает новые события с поддержкой Long Polling.
func (c *Client) GetUpdates(offset int64, limit int, timeoutSeconds int) ([]Update, error) {
	params := url.Values{}
	if offset > 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if timeoutSeconds > 0 {
		params.Set("timeout", strconv.Itoa(timeoutSeconds))
	}

	endpoint := fmt.Sprintf("%s/getUpdates?%s", c.baseURL, params.Encode())
	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("getUpdates request: %w", err)
	}
	defer resp.Body.Close()

	var apiResp apiResponse[[]Update]
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode getUpdates: %w", err)
	}
	if !apiResp.OK {
		return nil, fmt.Errorf("telegram api error: %s", apiResp.Description)
	}

	return apiResp.Result, nil
}

// SendMessage отправляет текстовое сообщение в чат с опциональной inline-клавиатурой.
func (c *Client) SendMessage(chatID int64, text string, replyMarkup *InlineKeyboardMarkup) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	if replyMarkup != nil {
		payload["reply_markup"] = replyMarkup
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/sendMessage", c.baseURL)
	resp, err := c.httpClient.Post(endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("sendMessage request: %w", err)
	}
	defer resp.Body.Close()

	var apiResp apiResponse[any]
	_ = json.NewDecoder(resp.Body).Decode(&apiResp)
	if !apiResp.OK {
		return fmt.Errorf("sendMessage error: %s", apiResp.Description)
	}

	return nil
}

// AnswerCallbackQuery подтверждает нажатие inline-кнопки.
func (c *Client) AnswerCallbackQuery(queryID string, text string) error {
	payload := map[string]any{
		"callback_query_id": queryID,
	}
	if text != "" {
		payload["text"] = text
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/answerCallbackQuery", c.baseURL)
	resp, err := c.httpClient.Post(endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("answerCallbackQuery request: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// SendDocument отправляет файл книги в чат с подписью.
func (c *Client) SendDocument(chatID int64, fileName string, fileData io.Reader, caption string) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	_ = writer.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	if caption != "" {
		_ = writer.WriteField("caption", caption)
		_ = writer.WriteField("parse_mode", "HTML")
	}

	part, err := writer.CreateFormFile("document", filepath.Base(fileName))
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, fileData); err != nil {
		return fmt.Errorf("copy document content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	endpoint := fmt.Sprintf("%s/sendDocument", c.baseURL)
	req, err := http.NewRequest("POST", endpoint, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sendDocument request: %w", err)
	}
	defer resp.Body.Close()

	var apiResp apiResponse[any]
	_ = json.NewDecoder(resp.Body).Decode(&apiResp)
	if !apiResp.OK {
		return fmt.Errorf("sendDocument error: %s", apiResp.Description)
	}

	return nil
}

// GetFile получает путь к загруженному пользователем файлу.
func (c *Client) GetFile(fileID string) (*TelegramFile, error) {
	endpoint := fmt.Sprintf("%s/getFile?file_id=%s", c.baseURL, url.QueryEscape(fileID))
	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("getFile request: %w", err)
	}
	defer resp.Body.Close()

	var apiResp apiResponse[TelegramFile]
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode getFile: %w", err)
	}
	if !apiResp.OK {
		return nil, fmt.Errorf("getFile error: %s", apiResp.Description)
	}

	return &apiResp.Result, nil
}

// DownloadFile скачивает бинарные данные файла из Telegram CDN.
func (c *Client) DownloadFile(filePath string) ([]byte, error) {
	downloadURL := fmt.Sprintf("%s/%s", c.fileURL, filePath)
	resp, err := c.httpClient.Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("download file request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download file HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
