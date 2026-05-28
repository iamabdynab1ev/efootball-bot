package api

import (
	"efootball-bot/internal/models"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleGetMatch(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	m, err := s.matchRepo.GetByID(r.Context(), id)
	if err != nil || m == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	uid := currentUserID(r)
	if m.HomeUserID != uid && m.AwayUserID != uid {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}
	jsonOK(w, matchDTO(m))
}

func (s *Server) handleSubmitResult(w http.ResponseWriter, r *http.Request) {
	matchID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	var body struct {
		HomeGoals int16 `json:"home_goals"`
		AwayGoals int16 `json:"away_goals"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}

	m, err := s.matchRepo.GetByID(r.Context(), matchID)
	if err != nil || m == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if m.HomeUserID != currentUserID(r) {
		jsonError(w, "only home player can submit result", http.StatusForbidden)
		return
	}
	if m.Status != models.MatchScheduled && m.Status != models.MatchDisputed {
		jsonError(w, "match is not ready for result submission", http.StatusBadRequest)
		return
	}

	// Вызываем repo напрямую — API идентифицирует по userID, не telegramID
	if err := s.matchRepo.ClaimResult(r.Context(), matchID, body.HomeGoals, body.AwayGoals); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, map[string]string{"status": "pending_confirm"})
}

func (s *Server) handleConfirmMatch(w http.ResponseWriter, r *http.Request) {
	matchID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	m, err := s.matchRepo.GetByID(r.Context(), matchID)
	if err != nil || m == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if m.AwayUserID != currentUserID(r) {
		jsonError(w, "only away player can confirm", http.StatusForbidden)
		return
	}

	confirmed, err := s.matchSvc.Confirm(r.Context(), matchID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// ELO обновляем вручную через Calculate
	if confirmed != nil && confirmed.HomeGoals != nil && confirmed.AwayGoals != nil {
		homeUser, _ := s.userRepo.GetByID(r.Context(), confirmed.HomeUserID)
		awayUser, _ := s.userRepo.GetByID(r.Context(), confirmed.AwayUserID)
		if homeUser != nil && awayUser != nil {
			newHome, newAway := s.eloSvc.Calculate(
				r.Context(),
				homeUser.ID, awayUser.ID,
				homeUser.Rating, awayUser.Rating,
				homeUser.TeamPower, awayUser.TeamPower,
				*confirmed.HomeGoals, *confirmed.AwayGoals,
			)
			_ = s.userRepo.UpdateRating(r.Context(), homeUser.ID, newHome)
			_ = s.userRepo.UpdateRating(r.Context(), awayUser.ID, newAway)
			_ = s.userRepo.RecalculateAllRanks(r.Context())
		}
	}
	jsonOK(w, map[string]string{"status": "confirmed"})
}

func (s *Server) handleDisputeMatch(w http.ResponseWriter, r *http.Request) {
	matchID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	m, err := s.matchRepo.GetByID(r.Context(), matchID)
	if err != nil || m == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if m.AwayUserID != currentUserID(r) {
		jsonError(w, "only away player can dispute", http.StatusForbidden)
		return
	}

	claimedHome := int16(0)
	claimedAway := int16(0)
	if m.ClaimedHome != nil {
		claimedHome = *m.ClaimedHome
	}
	if m.ClaimedAway != nil {
		claimedAway = *m.ClaimedAway
	}

	if _, err := s.matchSvc.Dispute(r.Context(), matchID, claimedHome, claimedAway); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, map[string]string{"status": "disputed"})
}
