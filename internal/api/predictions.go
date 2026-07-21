package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"

	"github.com/go-chi/chi/v5"
)

// ── Прогнозы на матчи: виртуальные очки, таблица прогнозистов ────────────────
// Правила честности: свой матч прогнозировать нельзя; приём открыт, пока матч
// «scheduled» и не истёк дедлайн тура/стадии; чужие прогнозы видны только
// после подтверждения матча (вскрытие).

// predictionOpen — можно ли ещё ставить прогноз на матч.
func (s *Server) predictionOpen(r *http.Request, m *models.Match) bool {
	if m.Status != models.MatchScheduled {
		return false
	}
	if s.deadlineRepo == nil {
		return true
	}
	deadlines, err := s.deadlineRepo.GetDeadlines(r.Context(), m.LeagueID)
	if err != nil {
		return true // без дедлайнов приём открыт
	}
	now := time.Now()
	for _, d := range deadlines {
		match := (d.Stage != "" && d.Stage == m.Stage) ||
			(d.Stage == "" && !models.IsKnockoutStage(m.Stage) && int(d.Round) == int(m.Round))
		if match && now.After(d.Deadline) {
			return false
		}
	}
	return true
}

// handlePredict — POST /api/matches/{id}/predict {home_goals, away_goals}.
func (s *Server) handlePredict(w http.ResponseWriter, r *http.Request) {
	if s.predRepo == nil {
		jsonError(w, "not configured", http.StatusServiceUnavailable)
		return
	}
	matchID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	uid := currentUserID(r)

	var body struct {
		HomeGoals int16 `json:"home_goals"`
		AwayGoals int16 `json:"away_goals"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
		body.HomeGoals < 0 || body.AwayGoals < 0 || body.HomeGoals > 50 || body.AwayGoals > 50 {
		jsonError(w, "invalid score", http.StatusBadRequest)
		return
	}

	m, err := s.matchRepo.GetByID(r.Context(), matchID)
	if err != nil || m == nil {
		jsonError(w, "match not found", http.StatusNotFound)
		return
	}
	if m.HomeUserID == uid || m.AwayUserID == uid {
		jsonError(w, "нельзя прогнозировать свой матч", http.StatusBadRequest)
		return
	}
	if !s.predictionOpen(r, m) {
		jsonError(w, "приём прогнозов на этот матч закрыт", http.StatusBadRequest)
		return
	}
	if err := s.predRepo.Upsert(r.Context(), matchID, uid, body.HomeGoals, body.AwayGoals); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "predicted"})
}

// handleMyPredictions — GET /api/leagues/{id}/predictions/my (auth).
func (s *Server) handleMyPredictions(w http.ResponseWriter, r *http.Request) {
	if s.predRepo == nil {
		jsonOK(w, map[string]any{"predictions": []any{}})
		return
	}
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	mine, err := s.predRepo.MineByLeague(r.Context(), leagueID, currentUserID(r))
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]any{"predictions": predictionDTOs(mine, true)})
}

// handlePredictionLeaderboard — GET /api/leagues/{id}/predictions/leaderboard (публичный).
func (s *Server) handlePredictionLeaderboard(w http.ResponseWriter, r *http.Request) {
	if s.predRepo == nil {
		jsonOK(w, map[string]any{"leaderboard": []any{}})
		return
	}
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	rows, err := s.predRepo.Leaderboard(r.Context(), leagueID)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, p := range rows {
		out = append(out, map[string]any{
			"user_id": p.UserID, "name": p.DisplayName, "club": p.Club,
			"points": p.Points, "exact": p.Exact, "total": p.Total,
		})
	}
	jsonOK(w, map[string]any{"leaderboard": out})
}

// handleMatchPredictions — GET /api/matches/{id}/predictions (публичный):
// вскрытие прогнозов — только после подтверждения матча.
func (s *Server) handleMatchPredictions(w http.ResponseWriter, r *http.Request) {
	if s.predRepo == nil {
		jsonOK(w, map[string]any{"predictions": []any{}})
		return
	}
	matchID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	m, err := s.matchRepo.GetByID(r.Context(), matchID)
	if err != nil || m == nil {
		jsonError(w, "match not found", http.StatusNotFound)
		return
	}
	if m.Status != models.MatchConfirmed {
		// До вскрытия отдаём только количество — интрига сохраняется.
		n, _ := s.predRepo.CountForMatch(r.Context(), matchID)
		jsonOK(w, map[string]any{"predictions": []any{}, "count": n, "sealed": true})
		return
	}
	rows, err := s.predRepo.ByMatch(r.Context(), matchID)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]any{"predictions": predictionDTOs(rows, false), "sealed": false})
}

func predictionDTOs(rows []*repository.PredictionRow, mine bool) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, p := range rows {
		d := map[string]any{
			"match_id":   p.MatchID,
			"home_goals": p.HomeGoals,
			"away_goals": p.AwayGoals,
		}
		if !mine {
			d["user_id"] = p.UserID
			d["name"] = p.DisplayName
			d["club"] = p.Club
		}
		if p.Points != nil {
			d["points"] = *p.Points
		}
		out = append(out, d)
	}
	return out
}
