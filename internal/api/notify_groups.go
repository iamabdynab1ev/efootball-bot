package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// handleListNotifyGroups — GET /api/admin/notify-groups — все подключённые
// группы (Telegram/WhatsApp) для управления и привязки к лигам.
func (s *Server) handleListNotifyGroups(w http.ResponseWriter, r *http.Request) {
	if s.notifyGroupRepo == nil {
		jsonOK(w, []any{})
		return
	}
	groups, err := s.notifyGroupRepo.List(r.Context())
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	out := make([]map[string]any, 0, len(groups))
	for _, g := range groups {
		out = append(out, map[string]any{
			"id": g.ID, "channel": g.Channel, "chat_id": g.ChatID,
			"title": g.Title, "enabled": g.Enabled,
		})
	}
	jsonOK(w, out)
}

// handleToggleNotifyGroup — POST /api/admin/notify-groups/{id}/toggle — вкл/выкл
// группу (выключенная не получает новостей и недоступна для привязки к лиге).
func (s *Server) handleToggleNotifyGroup(w http.ResponseWriter, r *http.Request) {
	if s.notifyGroupRepo == nil {
		jsonError(w, "not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := s.notifyGroupRepo.SetEnabled(r.Context(), id, body.Enabled); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]any{"status": "ok", "enabled": body.Enabled})
}

// handleDeleteNotifyGroup — DELETE /api/admin/notify-groups/{id} — убрать группу
// из реестра (лиги с этой группой откатятся на «во все группы»).
func (s *Server) handleDeleteNotifyGroup(w http.ResponseWriter, r *http.Request) {
	if s.notifyGroupRepo == nil {
		jsonError(w, "not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := s.notifyGroupRepo.Delete(r.Context(), id); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]any{"status": "deleted"})
}

// handleSetLeagueNotifyGroup — POST /api/admin/leagues/{id}/notify-group —
// маршрут новостей лиги: {"group_id": N} — слать в группу N; {"group_id": null}
// — во все включённые группы (как по умолчанию).
func (s *Server) handleSetLeagueNotifyGroup(w http.ResponseWriter, r *http.Request) {
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	var body struct {
		GroupID *int64 `json:"group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := s.leagueRepo.SetNotifyGroup(r.Context(), leagueID, body.GroupID); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	InvalidateLeagues()
	jsonOK(w, map[string]any{"status": "ok", "group_id": body.GroupID})
}
