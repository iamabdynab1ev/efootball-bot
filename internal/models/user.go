package models

import "time"

type User struct {
	ID          int64     `db:"id"`
	TelegramID  int64     `db:"telegram_id"`
	DisplayName string    `db:"display_name"`
	Username    *string   `db:"username"`
	IsBanned    bool      `db:"is_banned"`
	Rank        string    `db:"rank"`
	Rating      int       `db:"rating"`
	TeamPower   int       `db:"team_power"`
	Language    string    `db:"language"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
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
