package api

import (
	"efootball-bot/internal/models"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleListLeagues(w http.ResponseWriter, r *http.Request) {
	leagues, err := s.leagueRepo.GetActiveLeagues(r.Context())
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	result := make([]map[string]any, 0, len(leagues))
	for _, l := range leagues {
		result = append(result, leagueDTO(l))
	}
	jsonOK(w, result)
}

func (s *Server) handleGetLeague(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	league, err := s.leagueRepo.GetByID(r.Context(), id)
	if err != nil || league == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	jsonOK(w, leagueDTO(league))
}

func (s *Server) handleStandings(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	members, err := s.leagueRepo.GetMembers(r.Context(), id)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	rows := make([]map[string]any, 0, len(members))
	for _, m := range members {
		rows = append(rows, memberDTO(m))
	}
	jsonOK(w, map[string]any{"standings": rows})
}

func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	matches, err := s.matchRepo.GetAllForLeague(r.Context(), id)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	// Группируем по раундам
	rounds := map[int16][]map[string]any{}
	var order []int16
	seen := map[int16]bool{}
	for _, m := range matches {
		if !seen[m.Round] {
			order = append(order, m.Round)
			seen[m.Round] = true
		}
		rounds[m.Round] = append(rounds[m.Round], matchDTO(m))
	}

	result := make([]map[string]any, 0, len(order))
	for _, rnd := range order {
		result = append(result, map[string]any{
			"round":   rnd,
			"matches": rounds[rnd],
		})
	}
	jsonOK(w, map[string]any{"rounds": result})
}

func (s *Server) handleMyMatches(w http.ResponseWriter, r *http.Request) {
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	matches, err := s.matchRepo.GetUserSchedule(r.Context(), currentUserID(r), leagueID)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	result := make([]map[string]any, 0, len(matches))
	for _, m := range matches {
		result = append(result, matchDTO(m))
	}
	jsonOK(w, result)
}

func (s *Server) handleMyLeagues(w http.ResponseWriter, r *http.Request) {
	memberships, err := s.leagueRepo.GetUserLeagues(r.Context(), currentUserID(r))
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	result := make([]map[string]any, 0, len(memberships))
	for _, m := range memberships {
		row := memberDTO(m)
		if m.League != nil {
			row["league"] = leagueDTO(m.League)
		}
		result = append(result, row)
	}
	jsonOK(w, result)
}

func (s *Server) handleMyHistory(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}

	var leagueFilter int64
	if raw := r.URL.Query().Get("league_id"); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			leagueFilter = id
		}
	}

	matches, err := s.matchRepo.GetUserMatchHistory(r.Context(), currentUserID(r), limit, offset)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	result := make([]map[string]any, 0, len(matches))
	for _, m := range matches {
		if leagueFilter != 0 && m.LeagueID != leagueFilter {
			continue
		}
		result = append(result, matchDTO(m))
	}
	jsonOK(w, result)
}

func (s *Server) handleJoinLeague(w http.ResponseWriter, r *http.Request) {
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := s.leagueRepo.AddMember(r.Context(), leagueID, currentUserID(r)); err != nil {
		jsonError(w, "already applied or league closed", http.StatusConflict)
		return
	}
	jsonOK(w, map[string]string{"status": "pending"})
}

func (s *Server) handleTopScorers(w http.ResponseWriter, r *http.Request) {
	leagues, err := s.leagueRepo.GetActiveLeagues(r.Context())
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	result := make([]map[string]any, 0, len(leagues))
	for _, l := range leagues {
		scorers, err := s.userRepo.GetTopScorers(r.Context(), l.ID)
		if err != nil {
			continue
		}

		scorerRows := make([]map[string]any, 0, len(scorers))
		for i, scorer := range scorers {
			if scorer.GoalsFor == 0 || scorer.User == nil {
				continue
			}
			scorerRows = append(scorerRows, map[string]any{
				"position":     i + 1,
				"user_id":      scorer.UserID,
				"display_name": scorer.User.DisplayName,
				"rating":       scorer.User.Rating,
				"team_power":   scorer.User.TeamPower,
				"goals_for":    scorer.GoalsFor,
			})
			if len(scorerRows) >= 10 {
				break
			}
		}

		if len(scorerRows) == 0 {
			continue
		}
		result = append(result, map[string]any{
			"league":  leagueDTO(l),
			"scorers": scorerRows,
		})
	}

	jsonOK(w, map[string]any{"leagues": result})
}

// ── DTOs ──────────────────────────────────────────────────────────────────────

func leagueDTO(l *models.League) map[string]any {
	m := map[string]any{
		"id":          l.ID,
		"name":        l.Name,
		"status":      l.Status,
		"level":       l.Level,
		"max_players": l.MaxPlayers,
		"rounds_type": l.RoundsType,
	}
	if l.Country != nil {
		m["country"] = *l.Country
	}
	return m
}

func memberDTO(m *models.LeagueMember) map[string]any {
	row := map[string]any{
		"user_id":       m.UserID,
		"points":        m.Points,
		"wins":          m.Wins,
		"draws":         m.Draws,
		"losses":        m.Losses,
		"goals_for":     m.GoalsFor,
		"goals_against": m.GoalsAgainst,
		"goal_diff":     m.GoalDiff(),
		"status":        m.Status,
	}
	if m.Position != nil {
		row["position"] = *m.Position
	}
	if m.User != nil {
		row["display_name"] = m.User.DisplayName
		row["rating"] = m.User.Rating
		row["rank"] = m.User.Rank
	}
	return row
}

func matchDTO(m *models.Match) map[string]any {
	row := map[string]any{
		"id":            m.ID,
		"league_id":     m.LeagueID,
		"home_user_id":  m.HomeUserID,
		"away_user_id":  m.AwayUserID,
		"round":         m.Round,
		"status":        m.Status,
		"dispute_count": m.DisputeCount,
	}
	if m.HomeGoals != nil {
		row["home_goals"] = *m.HomeGoals
	}
	if m.AwayGoals != nil {
		row["away_goals"] = *m.AwayGoals
	}
	if m.ClaimedHome != nil {
		row["claimed_home"] = *m.ClaimedHome
	}
	if m.ClaimedAway != nil {
		row["claimed_away"] = *m.ClaimedAway
	}
	if m.PlayedAt != nil {
		row["played_at"] = m.PlayedAt
	}
	if m.HomeUser != nil {
		row["home_name"] = m.HomeUser.DisplayName
	}
	if m.AwayUser != nil {
		row["away_name"] = m.AwayUser.DisplayName
	}
	return row
}

func (s *Server) handlePlayers(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
		limit = l
	}
	users, err := s.userRepo.GetAllByRating(r.Context(), limit)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	result := make([]map[string]any, 0, len(users))
	for i, u := range users {
		m := userDTO(u)
		m["position"] = i + 1
		result = append(result, m)
	}
	jsonOK(w, result)
}

// handlePlayers уже в этом файле — перенесён сюда для удобства
var _ = json.Marshal
