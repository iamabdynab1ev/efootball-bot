package api

import (
	"context"
	"efootball-bot/internal/models"
	"efootball-bot/internal/service"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// PublishChat доставляет чат-событие в личные SSE-топики перечисленных
// пользователей (участников комнаты). Передаётся в ChatService как fan-out.
func (s *Server) PublishChat(userIDs []int64, eventType string, data any) {
	for _, uid := range userIDs {
		publishEvent(topicUser(uid), eventType, data)
	}
}

// chatLink — ссылка сразу на вкладку чата лиги.
func chatLink(leagueID int64) string {
	return "/leagues/details?id=" + strconv.FormatInt(leagueID, 10) + "&tab=chat"
}

// NotifyChatMention — реакция на @упоминание: уведомление в колокольчик (персист)
// + web-push упомянутым (Telegram намеренно не трогаем). Передаётся в ChatService.
func (s *Server) NotifyChatMention(ctx context.Context, msg *models.ChatMessage, mentionedIDs []int64, leagueID int64) {
	if len(mentionedIDs) == 0 {
		return
	}
	// Превью тела — по рунам, чтобы не разрезать кириллицу.
	preview := msg.Body
	if r := []rune(msg.Body); len(r) > 120 {
		preview = string(r[:120]) + "…"
	}
	title := "Вас упомянули в чате"
	body := msg.AuthorName + ": " + preview
	link := chatLink(leagueID)

	s.notify(ctx, mentionedIDs, models.NotifMention, title, body, link)
	if s.webPush != nil {
		go s.webPush.Notify(mentionedIDs, "💬 "+title, body, link)
	}
}

// handleListChatRooms — GET /api/leagues/{id}/chat/rooms — комнаты лиги,
// доступные пользователю (общая + его группа). Лениво создаёт комнаты.
func (s *Server) handleListChatRooms(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonOK(w, map[string]any{"rooms": []any{}})
		return
	}
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	rooms, err := s.chatSvc.RoomsForUser(r.Context(), currentUserID(r), leagueID)
	if err != nil {
		jsonErrorLog(w, r, "db error", http.StatusInternalServerError, err)
		return
	}
	if rooms == nil {
		rooms = []*models.ChatRoom{}
	}
	jsonOK(w, map[string]any{"rooms": rooms})
}

// handleChatMembers — GET /api/chat/rooms/{roomId}/members — участники комнаты
// для @упоминаний (скоуп строго по комнате).
func (s *Server) handleChatMembers(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonOK(w, map[string]any{"members": []any{}})
		return
	}
	roomID, err := strconv.ParseInt(chi.URLParam(r, "roomId"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	members, err := s.chatSvc.Members(r.Context(), currentUserID(r), roomID)
	if err != nil {
		writeChatErr(w, r, err)
		return
	}
	if members == nil {
		members = []*models.ChatMember{}
	}
	jsonOK(w, map[string]any{"members": members})
}

// handleChatHistory — GET /api/chat/rooms/{roomId}/messages?before=&since=&limit=
func (s *Server) handleChatHistory(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonOK(w, map[string]any{"messages": []any{}})
		return
	}
	roomID, err := strconv.ParseInt(chi.URLParam(r, "roomId"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	q := r.URL.Query()
	msgs, err := s.chatSvc.History(r.Context(), currentUserID(r), roomID,
		atoi64(q.Get("before")), atoi64(q.Get("since")), int(atoi64(q.Get("limit"))))
	if err != nil {
		writeChatErr(w, r, err)
		return
	}
	if msgs == nil {
		msgs = []*models.ChatMessage{}
	}
	jsonOK(w, map[string]any{"messages": msgs})
}

// handleSendChat — POST /api/chat/rooms/{roomId}/messages — отправка сообщения.
func (s *Server) handleSendChat(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonError(w, "chat disabled", http.StatusServiceUnavailable)
		return
	}
	roomID, err := strconv.ParseInt(chi.URLParam(r, "roomId"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	msg, err := s.chatSvc.Send(r.Context(), currentUserID(r), roomID, body.Body)
	if err != nil {
		writeChatErr(w, r, err)
		return
	}
	jsonOK(w, msg)
}

// handleAdminChatRooms — GET /api/admin/leagues/{id}/chat/rooms — все комнаты
// лиги (админ видит чаты без членства).
func (s *Server) handleAdminChatRooms(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonOK(w, map[string]any{"rooms": []any{}})
		return
	}
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := s.chatSvc.EnsureRoomsForLeague(r.Context(), leagueID); err != nil {
		jsonErrorLog(w, r, "db error", http.StatusInternalServerError, err)
		return
	}
	rooms, err := s.chatSvc.ListRooms(r.Context(), leagueID)
	if err != nil {
		jsonErrorLog(w, r, "db error", http.StatusInternalServerError, err)
		return
	}
	if rooms == nil {
		rooms = []*models.ChatRoom{}
	}
	jsonOK(w, map[string]any{"rooms": rooms})
}

// handleAdminChatHistory — GET /api/admin/chat/rooms/{roomId}/messages — без
// проверки членства (админ-просмотр любого чата).
func (s *Server) handleAdminChatHistory(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonOK(w, map[string]any{"messages": []any{}})
		return
	}
	roomID, err := strconv.ParseInt(chi.URLParam(r, "roomId"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	q := r.URL.Query()
	msgs, err := s.chatSvc.AdminMessages(r.Context(), roomID,
		atoi64(q.Get("before")), atoi64(q.Get("since")), int(atoi64(q.Get("limit"))))
	if err != nil {
		jsonErrorLog(w, r, "db error", http.StatusInternalServerError, err)
		return
	}
	if msgs == nil {
		msgs = []*models.ChatMessage{}
	}
	jsonOK(w, map[string]any{"messages": msgs})
}

// handleAdminDeleteChatMessage — DELETE /api/admin/chat/messages/{id}
func (s *Server) handleAdminDeleteChatMessage(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonError(w, "chat disabled", http.StatusServiceUnavailable)
		return
	}
	msgID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	msg, err := s.chatSvc.DeleteMessage(r.Context(), msgID)
	if err != nil {
		jsonErrorLog(w, r, "db error", http.StatusInternalServerError, err)
		return
	}
	s.audit(r, &models.AuditEntry{
		Action:     "chat.delete",
		EntityType: "chat_message",
		EntityID:   &msgID,
		LeagueID:   nil,
		Metadata:   map[string]any{"room_id": msg.RoomID},
	})
	jsonOK(w, map[string]bool{"ok": true})
}

// writeChatErr маппит доменные ошибки чата на HTTP-коды.
func writeChatErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrChatForbidden):
		jsonError(w, "нет доступа к этому чату", http.StatusForbidden)
	case errors.Is(err, service.ErrChatEmpty):
		jsonError(w, "пустое сообщение", http.StatusBadRequest)
	case errors.Is(err, service.ErrChatArchived):
		jsonError(w, "чат архивирован", http.StatusConflict)
	default:
		jsonErrorLog(w, r, "db error", http.StatusInternalServerError, err)
	}
}
