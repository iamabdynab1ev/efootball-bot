package models

import "time"

type User struct {
	ID          int64     `db:"id"`
	TelegramID  int64     `db:"telegram_id"` // 0 для web-only пользователей
	DisplayName string    `db:"display_name"`
	Username    *string   `db:"username"`
	IsBanned    bool      `db:"is_banned"`
	Rank        string    `db:"rank"`
	Rating      int       `db:"rating"`
	TeamPower   int       `db:"team_power"`
	Language    string    `db:"language"`
	GoogleID    *string   `db:"google_id"`
	Email       *string   `db:"email"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func (u *User) HasTelegram() bool {
	return u.TelegramID != 0
}

func RatingStatus(rating int) string {
	switch {
	case rating >= 1500:
		return "👑 Легенда"
	case rating >= 1300:
		return "💎 Элита"
	case rating >= 1150:
		return "🌟 Про"
	case rating >= 1000:
		return "⚽ Полупро"
	default:
		return "🥉 Любитель"
	}
}
