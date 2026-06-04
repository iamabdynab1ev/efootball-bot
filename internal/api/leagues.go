package api

import (
	"efootball-bot/internal/models"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleListLeagues(w http.ResponseWriter, r *http.Request) {
	if cached, ok := leaguesCache.get("all"); ok {
		w.Header().Set("Cache-Control", "public, max-age=30")
		jsonOK(w, cached)
		return
	}
	leagues, err := s.leagueRepo.GetActiveLeagues(r.Context())
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	result := make([]map[string]any, 0, len(leagues))
	for _, l := range leagues {
		result = append(result, leagueDTO(l))
	}
	leaguesCache.set("all", result)
	w.Header().Set("Cache-Control", "public, max-age=30")
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
	if cached, ok := standingsCache.get(id); ok {
		jsonOK(w, map[string]any{"standings": cached})
		return
	}
	league, err := s.leagueRepo.GetByID(r.Context(), id)
	if err != nil || league == nil {
		jsonError(w, "league not found", http.StatusNotFound)
		return
	}
	members, err := s.leagueRepo.GetMembers(r.Context(), id)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	formMap, _ := s.matchRepo.GetAllLeagueForm(r.Context(), id)
	rows := make([]map[string]any, 0, len(members))
	for _, m := range members {
		row := memberDTO(m)
		if formMap != nil {
			if form, ok := formMap[m.UserID]; ok {
				row["form"] = form
			} else {
				row["form"] = []string{}
			}
		}
		rows = append(rows, row)
	}
	standingsCache.set(id, rows)
	jsonOK(w, map[string]any{"standings": rows})
}

func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	league, err := s.leagueRepo.GetByID(r.Context(), id)
	if err != nil || league == nil {
		jsonError(w, "league not found", http.StatusNotFound)
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
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}

	var leagueFilter int64
	if raw := r.URL.Query().Get("league_id"); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil && id > 0 {
			leagueFilter = id
		}
	}

	matches, err := s.matchRepo.GetUserMatchHistory(r.Context(), currentUserID(r), limit, offset, leagueFilter)
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

func (s *Server) handleJoinLeague(w http.ResponseWriter, r *http.Request) {
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	league, err := s.leagueRepo.GetByID(r.Context(), leagueID)
	if err != nil || league == nil {
		jsonError(w, "league not found", http.StatusNotFound)
		return
	}
	if league.Status != "registration" {
		jsonError(w, "registration is closed for this league", http.StatusForbidden)
		return
	}
	if league.RegistrationDeadline != nil && time.Now().After(*league.RegistrationDeadline) {
		jsonError(w, "registration deadline has passed", http.StatusForbidden)
		return
	}
	if err := s.leagueRepo.AddMember(r.Context(), leagueID, currentUserID(r)); err != nil {
		jsonError(w, "already applied or league closed", http.StatusConflict)
		return
	}
	jsonOK(w, map[string]string{"status": "pending"})
}

func (s *Server) handleTopScorers(w http.ResponseWriter, r *http.Request) {
	leaguesWithScorers, err := s.userRepo.GetTopScorersAllLeagues(r.Context())
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	result := make([]map[string]any, 0, len(leaguesWithScorers))
	for _, lws := range leaguesWithScorers {
		scorerRows := make([]map[string]any, 0, len(lws.Scorers))
		for i, scorer := range lws.Scorers {
			if scorer.User == nil {
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
		}
		result = append(result, map[string]any{
			"league": map[string]any{
				"id":          lws.LeagueID,
				"name":        lws.LeagueName,
				"status":      lws.Status,
				"level":       lws.Level,
				"max_players": lws.MaxPlayers,
				"rounds_type": lws.RoundsType,
			},
			"scorers": scorerRows,
		})
	}

	jsonOK(w, map[string]any{"leagues": result})
}

// ── DTOs ──────────────────────────────────────────────────────────────────────

func leagueDTO(l *models.League) map[string]any {
	m := map[string]any{
		"id":            l.ID,
		"name":          l.Name,
		"status":        l.Status,
		"level":         l.Level,
		"max_players":   l.MaxPlayers,
		"rounds_type":   l.RoundsType,
		"num_groups":       l.NumGroups,
		"group_advance":    l.GroupAdvance,
		"best_runners_up":  l.BestRunnersUp,
		"current_round":    l.CurrentRound,
	}
	if l.Country != nil {
		m["country"] = *l.Country
	}
	if l.RegistrationDeadline != nil {
		m["registration_deadline"] = l.RegistrationDeadline.UTC().Format("2006-01-02T15:04:05Z")
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
		if m.User.FavoriteClub != nil {
			row["favorite_club"] = *m.User.FavoriteClub
		}
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
		if m.HomeUser.FavoriteClub != nil {
			row["home_club"] = *m.HomeUser.FavoriteClub
		}
	}
	if m.AwayUser != nil {
		row["away_name"] = m.AwayUser.DisplayName
		if m.AwayUser.FavoriteClub != nil {
			row["away_club"] = *m.AwayUser.FavoriteClub
		}
	}
	return row
}

// handleLeagueProgress — GET /api/leagues/{id}/progress — прогресс матчей лиги.
func (s *Server) handleLeagueProgress(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	remaining, err := s.matchRepo.CountUnconfirmedLeagueMatches(r.Context(), id)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]any{"remaining": remaining})
}

func (s *Server) handlePlayers(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
		limit = l
	}
	if cached, ok := playersCache.get(limit); ok {
		w.Header().Set("Cache-Control", "public, max-age=60")
		jsonOK(w, cached)
		return
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
	playersCache.set(limit, result)
	w.Header().Set("Cache-Control", "public, max-age=60")
	jsonOK(w, result)
}

