package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// ── Сезоны: закрытие с церемонией, сводка, баннер ────────────────────────────

// handleAdminSeasonCurrent — GET /api/admin/seasons/current:
// активный сезон + прогресс лиг для панели «Закрыть сезон».
func (s *Server) handleAdminSeasonCurrent(w http.ResponseWriter, r *http.Request) {
	season, err := s.leagueRepo.GetOrCreateActiveSeason(r.Context())
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	leagues, err := s.leagueRepo.ListLeaguesBySeason(r.Context(), season.ID)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	total, finished := 0, 0
	names := []string{}
	for _, l := range leagues {
		if l.Status == "draft" {
			continue
		}
		total++
		if l.Status == "finished" || l.Status == "archived" {
			finished++
		} else {
			names = append(names, l.Name)
		}
	}
	jsonOK(w, map[string]any{
		"season":           map[string]any{"id": season.ID, "name": season.Name},
		"leagues_total":    total,
		"leagues_finished": finished,
		"unfinished":       names,
	})
}

// handleAdminSeasonClose — POST /api/admin/seasons/{id}/close {next_name}.
func (s *Server) handleAdminSeasonClose(w http.ResponseWriter, r *http.Request) {
	if s.seasonSvc == nil {
		jsonError(w, "not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	var body struct {
		NextName string `json:"next_name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	noms, next, err := s.seasonSvc.Close(r.Context(), id, body.NextName)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	InvalidateLeagues()
	jsonOK(w, map[string]any{
		"status":      "season_closed",
		"nominations": noms,
		"next_season": map[string]any{"id": next.ID, "name": next.Name},
	})
}

// handleSeasonSummary — GET /api/seasons/{id}/summary (публичный): данные церемонии.
func (s *Server) handleSeasonSummary(w http.ResponseWriter, r *http.Request) {
	if s.seasonSvc == nil {
		jsonError(w, "not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	summary, err := s.seasonSvc.Summary(r.Context(), id)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonOK(w, summary)
}

// handleLatestClosedSeason — GET /api/seasons/latest-closed (публичный):
// последний закрытый сезон для баннера «смотреть церемонию» на главной.
func (s *Server) handleLatestClosedSeason(w http.ResponseWriter, r *http.Request) {
	season, err := s.leagueRepo.GetLatestClosedSeason(r.Context())
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	if season == nil || season.ClosedAt == nil {
		jsonOK(w, map[string]any{"season": nil})
		return
	}
	jsonOK(w, map[string]any{"season": map[string]any{
		"id":        season.ID,
		"name":      season.Name,
		"closed_at": season.ClosedAt.UTC().Format(time.RFC3339),
	}})
}
