package models

import "time"

// ChatRoom — комната чата турнира. GroupName=="" — общий чат лиги; иначе чат
// конкретной группы. Архивируется (а не удаляется) по завершении турнира.
type ChatRoom struct {
	ID        int64     `json:"id"`
	LeagueID  int64     `json:"league_id"`
	GroupName string    `json:"group_name"`
	Title     string    `json:"title"`
	Archived  bool      `json:"archived"`
	CreatedAt time.Time `json:"created_at"`
}

// ChatMessage — сообщение в комнате. AuthorName/AuthorClub заполняются при
// выборке/доставке (JOIN users) для рендера без доп. запросов на фронте.
type ChatMessage struct {
	ID         int64     `json:"id"`
	RoomID     int64     `json:"room_id"`
	UserID     *int64    `json:"user_id,omitempty"`
	AuthorName string    `json:"author_name"`
	AuthorClub string    `json:"author_club,omitempty"`
	Body       string    `json:"body"`
	Deleted    bool      `json:"deleted"`
	CreatedAt  time.Time `json:"created_at"`
}
