package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// handleLeagueGroupsList — GET /api/leagues/{id}/groups
func (s *Server) handleLeagueGroupsList(w http.ResponseWriter, r *http.Request) {
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	members, err := s.leagueRepo.GetMembers(r.Context(), leagueID)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	groups := map[string][]map[string]any{}
	for _, m := range members {
		if m.GroupName == "" {
			continue
		}
		groups[m.GroupName] = append(groups[m.GroupName], memberDTO(m))
	}
	jsonOK(w, map[string]any{"groups": groups})
}

// handleGroupStandings — GET /api/leagues/{id}/groups/{group}/standings
func (s *Server) handleGroupStandings(w http.ResponseWriter, r *http.Request) {
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	group := chi.URLParam(r, "group")
	members, err := s.leagueRepo.GetMembers(r.Context(), leagueID)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	rows := make([]map[string]any, 0)
	for _, m := range members {
		if m.GroupName == group {
			rows = append(rows, memberDTO(m))
		}
	}
	jsonOK(w, map[string]any{"standings": rows})
}

// handleGroupSchedule — GET /api/leagues/{id}/groups/{group}/schedule
func (s *Server) handleGroupSchedule(w http.ResponseWriter, r *http.Request) {
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	group := chi.URLParam(r, "group")
	matches, err := s.matchRepo.GetAllForLeague(r.Context(), leagueID)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	result := make([]map[string]any, 0)
	for _, m := range matches {
		if m.Stage == group {
			result = append(result, matchDTO(m))
		}
	}
	jsonOK(w, map[string]any{"matches": result})
}
