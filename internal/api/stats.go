package api

import (
	"efootball-bot/internal/repository"
	"math"
	"net/http"
)

func statEntryDTO(e *repository.StatEntry) map[string]any {
	club := ""
	if e.FavoriteClub != nil {
		club = *e.FavoriteClub
	}
	return map[string]any{
		"user_id":       e.UserID,
		"display_name":  e.DisplayName,
		"favorite_club": club,
		"rank":          e.Rank,
		"rating":        e.Rating,
		"played":        e.Played,
		"wins":          e.Wins,
		"draws":         e.Draws,
		"losses":        e.Losses,
		"goals_for":     e.GoalsFor,
		"goals_against": e.GoalsAgainst,
		"team_power":    e.TeamPower,
		"win_rate":      math.Round(e.WinRate*10) / 10,
		"avg_goals":     math.Round(e.AvgGoals*100) / 100,
		"streak":        e.Streak,
	}
}

func (s *Server) handleStatWinRate(w http.ResponseWriter, r *http.Request) {
	entries, err := s.statsRepo.GetWinRate(r.Context(), 5)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	rows := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, statEntryDTO(e))
	}
	jsonOK(w, rows)
}

func (s *Server) handleStatStreaks(w http.ResponseWriter, r *http.Request) {
	entries, err := s.statsRepo.GetStreaks(r.Context())
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	rows := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, statEntryDTO(e))
	}
	jsonOK(w, rows)
}

func (s *Server) handleStatAvgGoals(w http.ResponseWriter, r *http.Request) {
	entries, err := s.statsRepo.GetAvgGoals(r.Context(), 5)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	rows := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, statEntryDTO(e))
	}
	jsonOK(w, rows)
}

func (s *Server) handleStatTeamPower(w http.ResponseWriter, r *http.Request) {
	entries, err := s.statsRepo.GetTeamPower(r.Context())
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	rows := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, statEntryDTO(e))
	}
	jsonOK(w, rows)
}

func (s *Server) handleStatActivity(w http.ResponseWriter, r *http.Request) {
	entries, err := s.statsRepo.GetActivity(r.Context())
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	rows := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, statEntryDTO(e))
	}
	jsonOK(w, rows)
}
