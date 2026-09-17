package telegram

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"boyan/internal/config"
	"boyan/internal/storage"
	"boyan/internal/watcher"
)

// Bot представляет автономный сервис Telegram-бота.
type Bot struct {
	cfg             *config.Config
	client          *Client
	bookRepo        *storage.BookRepository
	watcherInstance *watcher.Watcher
	allowedUsers    map[int64]bool

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
}

func NewBot(
	cfg *config.Config,
	bookRepo *storage.BookRepository,
	watcherInstance *watcher.Watcher,
) *Bot {
	allowed := make(map[int64]bool)
	for _, id := range cfg.Telegram.AllowedUserIDs {
		allowed[id] = true
	}

	return &Bot{
		cfg:             cfg,
		client:          NewClient(cfg.Telegram.BotToken),
		bookRepo:        bookRepo,
		watcherInstance: watcherInstance,
		allowedUsers:    allowed,
		stopCh:          make(chan struct{}),
	}
}

// Start запускает цикл long-polling получения обновлений от Telegram.
func (b *Bot) Start(ctx context.Context) {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return
	}
	b.running = true
	b.mu.Unlock()

	slog.Info("Telegram Bot service started", "allowed_users_count", len(b.allowedUsers))

	go b.pollLoop(ctx)
}

// Stop останавливает бота.
func (b *Bot) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.running {
		return
	}
	b.running = false
	close(b.stopCh)
	slog.Info("Telegram Bot service stopped")
}

func (b *Bot) pollLoop(ctx context.Context) {
	var offset int64 = 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-b.stopCh:
			return
		default:
		}

		updates, err := b.client.GetUpdates(offset, 100, 20)
		if err != nil {
			slog.Warn("Telegram getUpdates error, backing off...", "err", err)
			select {
			case <-time.After(3 * time.Second):
			case <-ctx.Done():
				return
			case <-b.stopCh:
				return
			}
			continue
		}

		for _, update := range updates {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}

			go func(u Update) {
				b.dispatchUpdate(ctx, u)
			}(update)
		}
	}
}

// dispatchUpdate маршрутизирует событие нужному обработчику.
func (b *Bot) dispatchUpdate(ctx context.Context, u Update) {
	// 1. Проверка прав доступа для сообщений
	if u.Message != nil {
		msg := u.Message
		if !b.isUserAllowed(msg.From) {
			_ = b.client.SendMessage(msg.Chat.ID, "⛔ Доступ к библиотеке ограничен администратором.", nil)
			return
		}

		// Обработка документа
		if msg.Document != nil {
			if err := b.handleDocument(ctx, msg); err != nil {
				slog.Error("Telegram error handling document", "err", err)
			}
			return
		}

		// Обработка команд и текста
		text := strings.TrimSpace(msg.Text)
		if strings.HasPrefix(text, "/start") || strings.HasPrefix(text, "/help") {
			_ = b.handleStart(msg.Chat.ID)
			return
		}

		if strings.HasPrefix(text, "/search") {
			query := strings.TrimSpace(strings.TrimPrefix(text, "/search"))
			_ = b.handleSearch(ctx, msg.Chat.ID, query)
			return
		}

		if text != "" {
			_ = b.handleSearch(ctx, msg.Chat.ID, text)
			return
		}
	}

	// 2. Проверка прав доступа для callback-кнопок
	if u.CallbackQuery != nil {
		cq := u.CallbackQuery
		if !b.isUserAllowed(&cq.From) {
			_ = b.client.AnswerCallbackQuery(cq.ID, "Доступ ограничен.")
			return
		}

		if err := b.handleCallbackQuery(ctx, cq); err != nil {
			slog.Error("Telegram error handling callback", "err", err)
		}
	}
}

func (b *Bot) isUserAllowed(user *User) bool {
	if len(b.allowedUsers) == 0 {
		return true // Открытый режим
	}
	if user == nil {
		return false
	}
	return b.allowedUsers[user.ID]
}
