package api

import (
	"efootball-bot/internal/models"
	"net/http"
	"strconv"
	"strings"
)

// clientIP — реальный IP клиента. За прокси Render настоящий адрес лежит в
// X-Forwarded-For (берём первый, самый левый — исходный клиент).
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i != -1 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	ip := r.RemoteAddr
	if i := strings.LastIndex(ip, ":"); i != -1 {
		ip = ip[:i]
	}
	return ip
}

// audit пишет запись журнала, дозаполняя актора (из контекста) и IP (из
// запроса). no-op, если аудит не подключён — вызовы безопасны отовсюду.
func (s *Server) audit(r *http.Request, e *models.AuditEntry) {
	if s.auditSvc == nil || e == nil {
		return
	}
	if e.ActorID == nil {
		if uid := currentUserID(r); uid != 0 {
			e.ActorID = &uid
		}
	}
	if e.IP == "" {
		e.IP = clientIP(r)
	}
	s.auditSvc.Log(e)
}

// PublishAudit — колбэк живой ленты: обогащает имена из кэша юзеров (без запроса
// в БД) и публикует событие админам. Вызывается писателем AuditService после
// успешной вставки (id уже проставлен).
func (s *Server) PublishAudit(e *models.AuditEntry) {
	if !hub.hasSubscribers(topicAdmins) {
		return
	}
	if e.ActorName == "" && e.ActorID != nil {
		if u, ok := globalUserCache.get(*e.ActorID); ok && u != nil {
			e.ActorName = u.DisplayName
		}
	}
	if e.TargetName == "" && e.TargetID != nil {
		if u, ok := globalUserCache.get(*e.TargetID); ok && u != nil {
			e.TargetName = u.DisplayName
		}
	}
	publishEvent(topicAdmins, "audit", e)
}

// handleAdminAudit — GET /api/admin/audit — лента журнала действий с фильтрами
// (actor_id, target_id, league_id, action) и keyset-пагинацией (before, limit).
func (s *Server) handleAdminAudit(w http.ResponseWriter, r *http.Request) {
	if s.auditSvc == nil {
		jsonOK(w, map[string]any{"entries": []any{}})
		return
	}
	q := r.URL.Query()
	f := models.AuditFilter{
		Action:   q.Get("action"),
		BeforeID: atoi64(q.Get("before")),
		Limit:    int(atoi64(q.Get("limit"))),
	}
	if v := atoi64(q.Get("actor_id")); v > 0 {
		f.ActorID = &v
	}
	if v := atoi64(q.Get("target_id")); v > 0 {
		f.TargetID = &v
	}
	if v := atoi64(q.Get("league_id")); v > 0 {
		f.LeagueID = &v
	}

	entries, err := s.auditSvc.List(r.Context(), f)
	if err != nil {
		jsonErrorLog(w, r, "db error", http.StatusInternalServerError, err)
		return
	}
	if entries == nil {
		entries = []*models.AuditEntry{}
	}
	jsonOK(w, map[string]any{"entries": entries})
}

// atoi64 — безопасный парсинг query-параметра в int64 (0 при ошибке/пустом).
func atoi64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}
