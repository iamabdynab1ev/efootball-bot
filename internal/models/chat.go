package models

import "time"

// ChatRoom — комната чата. Kind=="group" — чат лиги/группы (GroupName=="" —
// общий чат лиги, иначе чат группы). Kind=="direct" — личные сообщения ровно
// между двумя пользователями (DmLo<DmHi), LeagueID тогда 0.
type ChatRoom struct {
	ID        int64     `json:"id"`
	LeagueID  int64     `json:"league_id"`
	GroupName string    `json:"group_name"`
	Title     string    `json:"title"`
	Archived  bool      `json:"archived"`
	CreatedAt time.Time `json:"created_at"`
	Kind      string    `json:"kind"`
	DmLo      *int64    `json:"-"`
	DmHi      *int64    `json:"-"`
	Unread    int       `json:"unread"` // непрочитанных мной в этой комнате
}

// RoomRead — до какого сообщения участник дочитал комнату (для отметок «прочитано»
// в групповом чате: сколько человек прочитали сообщение).
type RoomRead struct {
	UserID   int64 `json:"user_id"`
	LastRead int64 `json:"last_read"`
}

// DirectRoomView — элемент списка личных диалогов: комната + собеседник +
// превью последнего сообщения (для экрана «Сообщения»).
type DirectRoomView struct {
	RoomID        int64      `json:"room_id"`
	OtherID       int64      `json:"other_id"`
	OtherName     string     `json:"other_name"`
	OtherClub     string     `json:"other_club,omitempty"`
	LastBody      string     `json:"last_body"`
	LastAt        *time.Time `json:"last_at,omitempty"`
	LastAuthorID  *int64     `json:"last_author_id,omitempty"`
	Unread        int        `json:"unread"`          // непрочитанных мной в этом диалоге
	OtherLastRead int64      `json:"other_last_read"` // до какого id дочитал собеседник (для ✓✓)
	OtherLastSeen *time.Time `json:"other_last_seen,omitempty"`
}

// ChatMember — участник комнаты (для @упоминаний, скоуп строго по комнате:
// в общей — вся лига, в групповой — только её игроки).
type ChatMember struct {
	UserID       int64  `json:"user_id"`
	DisplayName  string `json:"display_name"`
	FavoriteClub string `json:"favorite_club,omitempty"`
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
	Edited     bool      `json:"edited"`
	ReplyToID  *int64    `json:"reply_to_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ReactionAgg — агрегированная реакция на сообщение: эмодзи, сколько поставили и
// поставил ли текущий пользователь.
type ReactionAgg struct {
	MessageID int64  `json:"message_id"`
	Emoji     string `json:"emoji"`
	Count     int    `json:"count"`
	Mine      bool   `json:"mine"`
}
