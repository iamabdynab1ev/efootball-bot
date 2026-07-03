package groupcast

import (
	"context"
	"strconv"
	"sync"

	"efootball-bot/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Ключ настройки с chat_id подключённой Telegram-группы.
const settingTGGroup = "tg_group_chat_id"

// TelegramGroup — канал «Telegram-группа». Группа подключается командой
// /connect в самой группе (админом); chat_id живёт в app_settings и
// переживает рестарты. Fallback — переменная окружения GROUP_CHAT_ID.
type TelegramGroup struct {
	bot      *tgbotapi.BotAPI
	settings repository.SettingsRepository

	mu     sync.RWMutex
	chatID int64
}

func NewTelegramGroup(bot *tgbotapi.BotAPI, settings repository.SettingsRepository, fallbackID int64) *TelegramGroup {
	t := &TelegramGroup{bot: bot, settings: settings, chatID: fallbackID}
	// Сохранённый в настройках chat_id приоритетнее env-фолбэка.
	if settings != nil {
		if vals, err := settings.GetMany(context.Background(), []string{settingTGGroup}); err == nil {
			if id, err := strconv.ParseInt(vals[settingTGGroup], 10, 64); err == nil && id != 0 {
				t.chatID = id
			}
		}
	}
	return t
}

func (t *TelegramGroup) Name() string { return "telegram" }

// ChatID — текущая подключённая группа (0 — не подключена).
func (t *TelegramGroup) ChatID() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.chatID
}

// SetChatID подключает (или отключает при id=0) группу и сохраняет выбор.
func (t *TelegramGroup) SetChatID(ctx context.Context, id int64) error {
	t.mu.Lock()
	t.chatID = id
	t.mu.Unlock()
	if t.settings == nil {
		return nil
	}
	return t.settings.Set(ctx, settingTGGroup, strconv.FormatInt(id, 10))
}

func (t *TelegramGroup) SendGroup(_ context.Context, text string) error {
	id := t.ChatID()
	if t.bot == nil || id == 0 {
		return nil // группа не подключена — тихий no-op
	}
	_, err := t.bot.Send(tgbotapi.NewMessage(id, text))
	return err
}
