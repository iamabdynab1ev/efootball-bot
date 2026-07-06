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

// Видимость приложения целиком (любая страница): вкладка на переднем плане →
// уведомления доставляются внутри приложения (звук + колокольчик/тост), а
// системный web-push не дублирует их. TTL тот же, что у фокуса чата.
var appFocus sync.Map // userID int64 → time.Time

// isAppVisible — вкладка приложения сейчас на переднем плане у пользователя.
func isAppVisible(userID int64) bool {
	v, ok := appFocus.Load(userID)
	if !ok {
		return false
	}
	t, _ := v.(time.Time)
	return time.Since(t) < focusTTL
}

// handleAppFocus — POST /api/app/focus {"on": true|false}: клиент шлёт on=true
// при видимой вкладке и каждые ~60с, on=false — при сворачивании/уходе.
func (s *Server) handleAppFocus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		On bool `json:"on"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	uid := currentUserID(r)
	if req.On {
		appFocus.Store(uid, time.Now())
	} else {
		appFocus.Delete(uid)
	}
	jsonOK(w, map[string]any{"ok": true})
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
