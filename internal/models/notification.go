package models

import "time"

// Notification — внутри-приложенческое уведомление пользователя.
type Notification struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Body      string    `json:"body,omitempty"`
	Link      string    `json:"link,omitempty"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

// Типы уведомлений — единый словарь (фронт ветвит иконку/цвет по типу).
const (
	NotifMatchResult    = "match.result"     // введён счёт — подтвердите/оспорьте
	NotifMatchConfirmed = "match.confirmed"  // матч подтверждён
	NotifMatchDisputed  = "match.disputed"   // соперник оспорил — переввод счёта
	NotifAdminResolve   = "match.resolved"   // админ выставил счёт
	NotifMemberApproved = "member.approved"  // заявку приняли
	NotifMemberRejected = "member.rejected"  // заявку отклонили
	NotifTournament     = "tournament"       // старт/событие турнира
	NotifMention        = "mention"          // упоминание в чате
	NotifSystem         = "system"           // системное/рассылка
)
