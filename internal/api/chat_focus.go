package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

// «Фокус» чата — какие комнаты прямо сейчас открыты у пользователей.
// Нужен для умных уведомлений: сообщение из чата, который человек читает
// в эту секунду, не должно дублироваться колокольчиком и push-ом.
// Хранится в памяти процесса: TTL короткий, потеря при рестарте безвредна.
type focusEntry struct {
	roomID int64
	at     time.Time
}

const focusTTL = 90 * time.Second

var chatFocus sync.Map // userID int64 → focusEntry

// isFocusedOn — открыт ли у пользователя данный чат прямо сейчас (вкладка видима).
func isFocusedOn(userID, roomID int64) bool {
	v, ok := chatFocus.Load(userID)
	if !ok {
		return false
	}
	e, _ := v.(focusEntry)
	return e.roomID == roomID && time.Since(e.at) < focusTTL
}

// handleChatFocus — POST /api/chat/rooms/{roomId}/focus {"on": true|false}.
// Клиент шлёт on=true при открытии чата и каждые ~60с, on=false — при уходе
// из чата или сворачивании вкладки.
func (s *Server) handleChatFocus(w http.ResponseWriter, r *http.Request) {
	roomID, err := strconv.ParseInt(chi.URLParam(r, "roomId"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	var req struct {
		On bool `json:"on"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	uid := currentUserID(r)
	if req.On {
		chatFocus.Store(uid, focusEntry{roomID: roomID, at: time.Now()})
	} else if v, ok := chatFocus.Load(uid); ok {
		if e, _ := v.(focusEntry); e.roomID == roomID {
			chatFocus.Delete(uid)
		}
	}
	jsonOK(w, map[string]any{"ok": true})
}
