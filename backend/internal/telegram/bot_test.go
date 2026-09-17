package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"boyan/internal/config"
	"boyan/internal/models"
)

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{2048, "2 KB"},
		{1048576, "1.0 MB"},
		{5242880, "5.0 MB"},
	}

	for _, tt := range tests {
		res := formatFileSize(tt.bytes)
		if res != tt.expected {
			t.Errorf("formatFileSize(%d): expected '%s', got '%s'", tt.bytes, tt.expected, res)
		}
	}
}

func TestUserAllowed(t *testing.T) {
	// 1. Открытый режим (allowed_user_ids пуст)
	cfgOpen := config.DefaultConfig()
	botOpen := NewBot(cfgOpen, nil, nil)

	if !botOpen.isUserAllowed(&User{ID: 12345}) {
		t.Errorf("expected user to be allowed in open mode")
	}

	// 2. Ограниченный режим
	cfgRestricted := config.DefaultConfig()
	cfgRestricted.Telegram.AllowedUserIDs = []int64{111, 222}
	botRestricted := NewBot(cfgRestricted, nil, nil)

	if !botRestricted.isUserAllowed(&User{ID: 111}) {
		t.Errorf("expected user 111 to be allowed")
	}
	if botRestricted.isUserAllowed(&User{ID: 999}) {
		t.Errorf("expected user 999 to be blocked")
	}
	if botRestricted.isUserAllowed(nil) {
		t.Errorf("expected nil user to be blocked")
	}
}

func TestTelegramClientMockServer(t *testing.T) {
	// Создаем мок HTTP сервер для Telegram API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.Contains(path, "/getUpdates") {
			resp := apiResponse[[]Update]{
				OK: true,
				Result: []Update{
					{
						UpdateID: 1001,
						Message: &Message{
							MessageID: 1,
							Chat:      Chat{ID: 999, Type: "private"},
							From:      &User{ID: 123, FirstName: "Alex"},
							Text:      "/start",
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.Contains(path, "/sendMessage") {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)

			resp := apiResponse[map[string]any]{
				OK:     true,
				Result: body,
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.Contains(path, "/answerCallbackQuery") {
			resp := apiResponse[bool]{
				OK:     true,
				Result: true,
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("dummy-token")
	client.baseURL = server.URL

	// 1. GetUpdates
	updates, err := client.GetUpdates(0, 10, 1)
	if err != nil {
		t.Fatalf("GetUpdates failed: %v", err)
	}
	if len(updates) != 1 || updates[0].UpdateID != 1001 {
		t.Fatalf("unexpected updates response: %v", updates)
	}

	// 2. SendMessage
	err = client.SendMessage(999, "Hello from test!", nil)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	// 3. AnswerCallbackQuery
	err = client.AnswerCallbackQuery("cb_123", "OK")
	if err != nil {
		t.Fatalf("AnswerCallbackQuery failed: %v", err)
	}
}

func TestBotDispatching(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := apiResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"sent": true},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := config.DefaultConfig()
	cfg.Telegram.BotToken = "mock"

	bot := NewBot(cfg, nil, nil)
	bot.client.baseURL = server.URL

	ctx := context.Background()

	// 1. Start command
	updateStart := Update{
		UpdateID: 1,
		Message: &Message{
			MessageID: 1,
			Chat:      Chat{ID: 42, Type: "private"},
			From:      &User{ID: 100, FirstName: "User"},
			Text:      "/start",
		},
	}
	bot.dispatchUpdate(ctx, updateStart)

	// 2. Help command
	updateHelp := Update{
		UpdateID: 2,
		Message: &Message{
			MessageID: 2,
			Chat:      Chat{ID: 42, Type: "private"},
			From:      &User{ID: 100, FirstName: "User"},
			Text:      "/help",
		},
	}
	bot.dispatchUpdate(ctx, updateHelp)

	// 3. Blocked user
	cfgBlocked := config.DefaultConfig()
	cfgBlocked.Telegram.AllowedUserIDs = []int64{999}
	botBlocked := NewBot(cfgBlocked, nil, nil)
	botBlocked.client.baseURL = server.URL

	botBlocked.dispatchUpdate(ctx, updateStart) // User ID is 100, not in [999]
}

func TestSendBookCardMarkup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		// Проверяем наличие inline_keyboard
		markup, ok := body["reply_markup"].(map[string]any)
		if !ok {
			t.Errorf("expected reply_markup in payload")
		}
		keyboard, ok := markup["inline_keyboard"].([]any)
		if !ok || len(keyboard) == 0 {
			t.Errorf("expected inline_keyboard buttons")
		}

		resp := apiResponse[map[string]any]{OK: true, Result: body}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := config.DefaultConfig()
	bot := NewBot(cfg, nil, nil)
	bot.client.baseURL = server.URL

	book := models.Book{
		ID:    "book-123",
		Title: "Тестовая книга",
		Authors: []models.AuthorDetail{
			{Author: models.Author{Name: "Автор Тестовый"}},
		},
		Files: []models.BookFile{
			{Format: "fb2", FileSize: 1024},
			{Format: "epub", FileSize: 2048},
		},
	}

	bot.sendBookCard(42, book)
}
