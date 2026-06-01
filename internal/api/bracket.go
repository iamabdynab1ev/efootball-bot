package api

import (
	"efootball-bot/internal/models"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// handleBracket — GET /api/leagues/{id}/bracket
func (s *Server) handleBracket(w http.ResponseWriter, r *http.Request) {
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}

	slots, err := s.bracketRepo.GetAllSlots(r.Context(), leagueID)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	// Group by stage keeping order
	type stageGroup struct {
		Stage string           `json:"stage"`
		Label string           `json:"label"`
		Slots []map[string]any `json:"slots"`
	}

	stageMap := map[string]*stageGroup{}
	order := []string{}

	for _, slot := range slots {
		if _, exists := stageMap[slot.Stage]; !exists {
			label := models.StageLabel[slot.Stage]
			if label == "" {
				label = slot.Stage
			}
			stageMap[slot.Stage] = &stageGroup{Stage: slot.Stage, Label: label}
			order = append(order, slot.Stage)
		}
		stageMap[slot.Stage].Slots = append(stageMap[slot.Stage].Slots, bracketSlotDTO(slot))
	}

	result := make([]stageGroup, 0, len(order))
	for _, stage := range order {
		result = append(result, *stageMap[stage])
	}

	jsonOK(w, map[string]any{"stages": result})
}

// handleAdminPlayoff — POST /api/admin/leagues/{id}/playoff
func (s *Server) handleAdminPlayoff(w http.ResponseWriter, r *http.Request) {
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}

	var body struct {
		TopK int `json:"top_k"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.TopK <= 0 {
		body.TopK = 8 // default
	}

	if err := s.playoffSvc.GeneratePlayoff(r.Context(), leagueID, body.TopK); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, map[string]string{"status": "playoff_generated"})
}

func bracketSlotDTO(s *models.BracketSlot) map[string]any {
	m := map[string]any{
		"slot":      s.Slot,
		"stage":     s.Stage,
		"home_name": s.HomeName,
		"away_name": s.AwayName,
	}
	if s.HomeUserID != nil {
		m["home_user_id"] = *s.HomeUserID
	}
	if s.AwayUserID != nil {
		m["away_user_id"] = *s.AwayUserID
	}
	if s.MatchID != nil {
		m["match_id"] = *s.MatchID
	}
	if s.WinnerUserID != nil {
		m["winner_user_id"] = *s.WinnerUserID
		m["winner_name"] = s.WinnerName
	}
	if s.HomeGoals != nil {
		m["home_goals"] = *s.HomeGoals
	}
	if s.AwayGoals != nil {
		m["away_goals"] = *s.AwayGoals
	}
	if s.MatchStatus != "" {
		m["match_status"] = s.MatchStatus
	}
	return m
}
