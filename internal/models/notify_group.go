package models

import "time"

// NotifyGroup — подключённая группа для новостей турнира (Telegram или WhatsApp).
// Бот может быть подключён к нескольким группам; лига маршрутизирует свои
// новости в конкретную группу через leagues.notify_group_id.
type NotifyGroup struct {
	ID        int64     `db:"id"`
	Channel   string    `db:"channel"` // "telegram" | "whatsapp"
	ChatID    string    `db:"chat_id"` // TG chat_id (как текст) или WA jid
	Title     string    `db:"title"`
	Enabled   bool      `db:"enabled"`
	CreatedAt time.Time `db:"created_at"`
}
