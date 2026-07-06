package api

import (
	"context"
	"crypto/rand"
	"efootball-bot/internal/models"
	"efootball-bot/internal/service"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

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

// directLink — ссылка на конкретный личный диалог.
func directLink(roomID int64) string {
	return "/messages?room=" + strconv.FormatInt(roomID, 10)
}

// NotifyDirectMessage — уведомляет собеседника о новом личном сообщении
// (колокольчик + web-push). Передаётся в ChatService как onDirect.
func (s *Server) NotifyDirectMessage(ctx context.Context, msg *models.ChatMessage, recipientID int64) {
	if recipientID == 0 {
		return
	}
	// Умные уведомления: получатель прямо сейчас читает этот чат — сообщение
	// доставится живьём по SSE, дублировать колокольчиком/push-ем не нужно.
	if isFocusedOn(recipientID, msg.RoomID) {
		return
	}
	preview := msg.Body
	if r := []rune(msg.Body); len(r) > 120 {
		preview = string(r[:120]) + "…"
	}
	title := "Личное сообщение"
	body := msg.AuthorName + ": " + preview
	link := directLink(msg.RoomID)
	s.notify(ctx, []int64{recipientID}, models.NotifDirect, title, body, link)
	if s.webPush != nil {
		go s.webPush.NotifyKind([]int64{recipientID}, "message", "✉️ "+msg.AuthorName, preview, link)
	}
}

// handleOpenDirect — POST /api/chat/direct {user_id} — найти/создать ЛС с
// соперником и вернуть комнату (id для открытия диалога).
func (s *Server) handleOpenDirect(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonError(w, "chat disabled", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	room, err := s.chatSvc.OpenDirect(r.Context(), currentUserID(r), req.UserID)
	if err != nil {
		writeChatErr(w, r, err)
		return
	}
	jsonOK(w, room)
}

// handleMarkChatRead — POST /api/chat/rooms/{roomId}/read {upto} — отметить
// прочитанным до сообщения upto; оповещает собеседника (для ✓✓).
func (s *Server) handleMarkChatRead(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonOK(w, map[string]any{"last_read": 0})
		return
	}
	roomID, err := strconv.ParseInt(chi.URLParam(r, "roomId"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	var req struct {
		Upto int64 `json:"upto"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	lastRead, err := s.chatSvc.MarkRead(r.Context(), currentUserID(r), roomID, req.Upto)
	if err != nil {
		writeChatErr(w, r, err)
		return
	}
	jsonOK(w, map[string]any{"last_read": lastRead})
}

// handleChatTyping — POST /api/chat/rooms/{roomId}/typing — эфемерный сигнал
// «печатает…» собеседнику.
func (s *Server) handleChatTyping(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonOK(w, map[string]bool{"ok": true})
		return
	}
	roomID, err := strconv.ParseInt(chi.URLParam(r, "roomId"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := s.chatSvc.Typing(r.Context(), currentUserID(r), roomID); err != nil {
		writeChatErr(w, r, err)
		return
	}
	jsonOK(w, map[string]bool{"ok": true})
}

// handleChatReads — GET /api/chat/rooms/{roomId}/reads — прогресс прочтения
// участников комнаты (для отметок «прочитано» в групповом чате).
func (s *Server) handleChatReads(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonOK(w, map[string]any{"reads": []any{}})
		return
	}
	roomID, err := strconv.ParseInt(chi.URLParam(r, "roomId"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	reads, err := s.chatSvc.RoomReads(r.Context(), currentUserID(r), roomID)
	if err != nil {
		writeChatErr(w, r, err)
		return
	}
	if reads == nil {
		reads = []models.RoomRead{}
	}
	jsonOK(w, map[string]any{"reads": reads})
}

// handleChatUnread — GET /api/chat/unread — всего непрочитанных ЛС (для бейджа).
func (s *Server) handleChatUnread(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonOK(w, map[string]int{"total": 0})
		return
	}
	total, err := s.chatSvc.UnreadTotal(r.Context(), currentUserID(r))
	if err != nil {
		jsonErrorLog(w, r, "db error", http.StatusInternalServerError, err)
		return
	}
	jsonOK(w, map[string]int{"total": total})
}

// handleListDirect — GET /api/chat/direct — список личных диалогов пользователя.
func (s *Server) handleListDirect(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonOK(w, map[string]any{"rooms": []any{}})
		return
	}
	rooms, err := s.chatSvc.ListDirect(r.Context(), currentUserID(r))
	if err != nil {
		jsonErrorLog(w, r, "db error", http.StatusInternalServerError, err)
		return
	}
	if rooms == nil {
		rooms = []*models.DirectRoomView{}
	}
	jsonOK(w, map[string]any{"rooms": rooms})
}

// NotifyChatMention — реакция на @упоминание: уведомление в колокольчик (персист)
// + web-push упомянутым (Telegram намеренно не трогаем). Передаётся в ChatService.
func (s *Server) NotifyChatMention(ctx context.Context, msg *models.ChatMessage, mentionedIDs []int64, leagueID int64) {
	// Умные уведомления: тем, кто прямо сейчас читает эту комнату, дубликат не шлём.
	filtered := mentionedIDs[:0:0]
	for _, id := range mentionedIDs {
		if !isFocusedOn(id, msg.RoomID) {
			filtered = append(filtered, id)
		}
	}
	mentionedIDs = filtered
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
		go s.webPush.NotifyKind(mentionedIDs, "message", "💬 "+title, body, link)
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
		Body      string `json:"body"`
		ReplyToID *int64 `json:"reply_to_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	msg, err := s.chatSvc.Send(r.Context(), currentUserID(r), roomID, body.Body, body.ReplyToID)
	if err != nil {
		writeChatErr(w, r, err)
		return
	}
	jsonOK(w, msg)
}

// handleChatReactions — GET /api/chat/rooms/{roomId}/reactions — реакции комнаты.
func (s *Server) handleChatReactions(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonOK(w, map[string]any{"reactions": []any{}})
		return
	}
	roomID, err := strconv.ParseInt(chi.URLParam(r, "roomId"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	reactions, err := s.chatSvc.RoomReactions(r.Context(), currentUserID(r), roomID)
	if err != nil {
		writeChatErr(w, r, err)
		return
	}
	if reactions == nil {
		reactions = []models.ReactionAgg{}
	}
	jsonOK(w, map[string]any{"reactions": reactions})
}

// handleAddReaction — POST /api/chat/messages/{id}/reactions {emoji}.
func (s *Server) handleAddReaction(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonError(w, "chat disabled", http.StatusServiceUnavailable)
		return
	}
	msgID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	var req struct {
		Emoji string `json:"emoji"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := s.chatSvc.AddReaction(r.Context(), currentUserID(r), msgID, req.Emoji); err != nil {
		writeChatErr(w, r, err)
		return
	}
	jsonOK(w, map[string]bool{"ok": true})
}

// handleRemoveReaction — DELETE /api/chat/messages/{id}/reactions?emoji=X.
func (s *Server) handleRemoveReaction(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonError(w, "chat disabled", http.StatusServiceUnavailable)
		return
	}
	msgID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := s.chatSvc.RemoveReaction(r.Context(), currentUserID(r), msgID, r.URL.Query().Get("emoji")); err != nil {
		writeChatErr(w, r, err)
		return
	}
	jsonOK(w, map[string]bool{"ok": true})
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

// handleDeleteOwnMessage — DELETE /api/chat/messages/{id} — пользователь удаляет
// своё сообщение (админ — любое).
func (s *Server) handleDeleteOwnMessage(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonError(w, "chat disabled", http.StatusServiceUnavailable)
		return
	}
	msgID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	uid := currentUserID(r)
	if _, err := s.chatSvc.DeleteOwnMessage(r.Context(), uid, s.isAdminCached(r, uid), msgID); err != nil {
		writeChatErr(w, r, err)
		return
	}
	jsonOK(w, map[string]bool{"ok": true})
}

// handleEditMessage — PATCH /api/chat/messages/{id} {body} — правка своего сообщения.
func (s *Server) handleEditMessage(w http.ResponseWriter, r *http.Request) {
	if s.chatSvc == nil {
		jsonError(w, "chat disabled", http.StatusServiceUnavailable)
		return
	}
	msgID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	msg, err := s.chatSvc.EditMessage(r.Context(), currentUserID(r), msgID, req.Body)
	if err != nil {
		writeChatErr(w, r, err)
		return
	}
	jsonOK(w, msg)
}

// audioExt подбирает расширение по content-type записи с телефона/браузера.
func audioExt(ct string) string {
	switch {
	case strings.Contains(ct, "webm"):
		return "webm"
	case strings.Contains(ct, "mp4"), strings.Contains(ct, "m4a"), strings.Contains(ct, "aac"):
		return "m4a"
	case strings.Contains(ct, "ogg"), strings.Contains(ct, "opus"):
		return "ogg"
	case strings.Contains(ct, "mpeg"), strings.Contains(ct, "mp3"):
		return "mp3"
	default:
		return "bin"
	}
}

// imageExt подбирает расширение по content-type изображения.
func imageExt(ct string) string {
	switch {
	case strings.Contains(ct, "jpeg"), strings.Contains(ct, "jpg"):
		return "jpg"
	case strings.Contains(ct, "png"):
		return "png"
	case strings.Contains(ct, "webp"):
		return "webp"
	case strings.Contains(ct, "gif"):
		return "gif"
	default:
		return "bin"
	}
}

// uploadMedia — общий обработчик загрузки вложения (голос/фото) в R2 + создание
// сообщения. kind — тип медиа ("audio"|"image"), ctPrefix — допустимый префикс
// content-type, maxBytes — лимит размера, folder/ext — путь в бакете.
func (s *Server) uploadMedia(w http.ResponseWriter, r *http.Request, kind, ctPrefix, folder string, maxBytes int64, ext func(string) string) {
	if s.chatSvc == nil {
		jsonError(w, "chat disabled", http.StatusServiceUnavailable)
		return
	}
	if s.media == nil || !s.media.Enabled() {
		jsonError(w, "медиа пока не настроено", http.StatusServiceUnavailable)
		return
	}
	roomID, err := strconv.ParseInt(chi.URLParam(r, "roomId"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := r.ParseMultipartForm(maxBytes + (1 << 20)); err != nil {
		jsonError(w, "invalid form", http.StatusBadRequest)
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "нет файла", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ct := hdr.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, ctPrefix) {
		jsonError(w, "неверный тип файла", http.StatusBadRequest)
		return
	}
	if hdr.Size <= 0 || hdr.Size > maxBytes {
		jsonError(w, "файл слишком большой", http.StatusBadRequest)
		return
	}

	var rnd [6]byte
	_, _ = rand.Read(rnd[:])
	key := fmt.Sprintf("%s/%d/%d-%s.%s", folder, roomID, time.Now().Unix(), hex.EncodeToString(rnd[:]), ext(ct))

	url, err := s.media.Put(r.Context(), key, file, hdr.Size, ct)
	if err != nil {
		jsonErrorLog(w, r, "не удалось загрузить", http.StatusInternalServerError, err)
		return
	}
	media := &models.ChatMedia{URL: url, Type: kind}
	if kind == "audio" {
		media.Dur, _ = strconv.ParseFloat(r.FormValue("dur"), 64)
		// Реальная форма волны, посчитанная клиентом при записи (до 64 пиков 0..1).
		if pj := r.FormValue("peaks"); pj != "" {
			var peaks []float64
			if json.Unmarshal([]byte(pj), &peaks) == nil && len(peaks) > 0 && len(peaks) <= 64 {
				for i, p := range peaks {
					peaks[i] = math.Max(0, math.Min(1, p))
				}
				media.Peaks = peaks
			}
		}
	}
	msg, err := s.chatSvc.SendMedia(r.Context(), currentUserID(r), roomID, media)
	if err != nil {
		writeChatErr(w, r, err)
		return
	}
	jsonOK(w, msg)
}

// handleSendVoice — POST /api/chat/rooms/{roomId}/voice (multipart: file, dur).
func (s *Server) handleSendVoice(w http.ResponseWriter, r *http.Request) {
	s.uploadMedia(w, r, "audio", "audio/", "voice", 5<<20, audioExt) // до 5 МБ
}

// handleSendPhoto — POST /api/chat/rooms/{roomId}/photo (multipart: file).
func (s *Server) handleSendPhoto(w http.ResponseWriter, r *http.Request) {
	s.uploadMedia(w, r, "image", "image/", "photo", 8<<20, imageExt) // до 8 МБ
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
	case errors.Is(err, service.ErrChatNotOpponents):
		jsonError(w, "писать можно только соперникам по матчу", http.StatusForbidden)
	default:
		jsonErrorLog(w, r, "db error", http.StatusInternalServerError, err)
	}
}
