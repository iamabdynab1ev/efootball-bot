package api

import (
	"context"
	"efootball-bot/internal/models"
	"encoding/json"
	"net/http"
	"strconv"
)

// leagueLink — внутренняя ссылка на страницу лиги (для перехода из уведомления).
func leagueLink(leagueID int64) string {
	return "/leagues/details?id=" + strconv.FormatInt(leagueID, 10)
}

// notify персистит и доставляет уведомление адресатам. no-op, если сервис не
// подключён — вызовы безопасны из любого хендлера.
func (s *Server) notify(ctx context.Context, userIDs []int64, typ, title, body, link string) {
	if s.notifSvc == nil {
		return
	}
	s.notifSvc.Notify(ctx, userIDs, typ, title, body, link)
}

// notifyT — локализованное уведомление: build(lang) строит (title, body)
// на языке каждого получателя (users.language).
func (s *Server) notifyT(ctx context.Context, userIDs []int64, typ, link string, build func(lang string) (string, string)) {
	if s.notifSvc == nil {
		return
	}
	s.notifSvc.NotifyT(ctx, userIDs, typ, link, build)
}

// PublishNotification — колбэк живой доставки: шлёт уведомление в личный
// SSE-топик получателя. Вызывается NotificationService после записи в БД.
func (s *Server) PublishNotification(n *models.Notification) {
	publishEvent(topicUser(n.UserID), "notification", n)
}

// handleListNotifications — GET /api/notifications?before=&limit= — лента
// пользователя + счётчик непрочитанных.
func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	if s.notifSvc == nil {
		jsonOK(w, map[string]any{"notifications": []any{}, "unread": 0})
		return
	}
	uid := currentUserID(r)
	before := atoi64(r.URL.Query().Get("before"))
	limit := int(atoi64(r.URL.Query().Get("limit")))
	list, unread, err := s.notifSvc.List(r.Context(), uid, before, limit)
	if err != nil {
		jsonErrorLog(w, r, "db error", http.StatusInternalServerError, err)
		return
	}
	if list == nil {
		list = []*models.Notification{}
	}
	jsonOK(w, map[string]any{"notifications": list, "unread": unread})
}

// handleMarkNotificationsRead — POST /api/notifications/read — помечает
// прочитанными переданные ids, либо все при {"all":true}.
func (s *Server) handleMarkNotificationsRead(w http.ResponseWriter, r *http.Request) {
	if s.notifSvc == nil {
		jsonOK(w, map[string]bool{"ok": true})
		return
	}
	var body struct {
		IDs []int64 `json:"ids"`
		All bool    `json:"all"`
	}
	// Тело опционально; пустое — ничего не помечаем (защита от случайного mark-all).
	_ = json.NewDecoder(r.Body).Decode(&body)

	uid := currentUserID(r)
	var err error
	switch {
	case len(body.IDs) > 0:
		err = s.notifSvc.MarkRead(r.Context(), uid, body.IDs)
	case body.All:
		err = s.notifSvc.MarkRead(r.Context(), uid, nil)
	}
	if err != nil {
		jsonErrorLog(w, r, "db error", http.StatusInternalServerError, err)
		return
	}
	jsonOK(w, map[string]bool{"ok": true})
}
