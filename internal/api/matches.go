package api

import (
	"fmt"
	"efootball-bot/internal/i18n"
	"efootball-bot/internal/logger"
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
	if body.HomeGoals < 0 || body.HomeGoals > 50 || body.AwayGoals < 0 || body.AwayGoals > 50 {
		jsonError(w, "invalid score: goals must be 0-50", http.StatusBadRequest)
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

	// Уведомляем гостя — ему нужно подтвердить или оспорить
	if awayUser, err := s.userRepo.GetByID(r.Context(), m.AwayUserID); err == nil && awayUser != nil {
		homeName := ""
		if m.HomeUser != nil {
			homeName = m.HomeUser.DisplayName
		}
		s.notifier.ResultSubmitted(homeName, awayUser.DisplayName, body.HomeGoals, body.AwayGoals, awayUser.TelegramID)
		if s.webPush != nil {
			go s.webPush.Notify([]int64{m.AwayUserID}, "⚽ Результат матча",
				homeName+" ввёл счёт "+itoa16(body.HomeGoals)+":"+itoa16(body.AwayGoals)+" — подтвердите или оспорьте", "/")
		}
		s.notifyT(r.Context(), []int64{m.AwayUserID}, models.NotifMatchResult, leagueLink(m.LeagueID),
			func(lang string) (string, string) {
				return i18n.T(lang, "match.result.title"),
					fmt.Sprintf(i18n.T(lang, "match.result.body"), homeName, body.HomeGoals, body.AwayGoals)
			})
	}
	PublishMatchUpdate(m.LeagueID, m.ID)
	s.audit(r, &models.AuditEntry{
		Action:     models.AuditSubmitResult,
		EntityType: "match",
		EntityID:   &m.ID,
		LeagueID:   &m.LeagueID,
		TargetID:   &m.AwayUserID,
		Metadata:   map[string]any{"home_goals": body.HomeGoals, "away_goals": body.AwayGoals},
	})
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

	// Best-of-X: серия ещё не завершена — игра засчитана, матч ждёт следующей.
	// ELO/награды/продвижение применяются только по завершении серии.
	if confirmed != nil && confirmed.Status != models.MatchConfirmed {
		PublishMatchUpdate(confirmed.LeagueID, confirmed.ID)
		jsonOK(w, map[string]any{
			"status":    "game_recorded",
			"home_wins": confirmed.HomeWins,
			"away_wins": confirmed.AwayWins,
			"best_of":   confirmed.BestOf,
		})
		return
	}

	// Инвалидируем кэши после подтверждения
	if confirmed != nil {
		InvalidateStandings(confirmed.LeagueID) // standings изменились
		InvalidatePlayers()                     // рейтинг скоро изменится
	}

	// Аудит-трейл подтверждения
	if confirmed != nil && confirmed.HomeGoals != nil && confirmed.AwayGoals != nil {
		logger.MatchConfirmed(r.Context(),
			confirmed.ID, confirmed.LeagueID,
			confirmed.HomeUserID, confirmed.AwayUserID,
			*confirmed.HomeGoals, *confirmed.AwayGoals,
		)
	}

	// Обновляем ELO
	if confirmed != nil && confirmed.HomeGoals != nil && confirmed.AwayGoals != nil {
		homeUser, err := s.userRepo.GetByID(r.Context(), confirmed.HomeUserID)
		if err != nil {
			logger.FromContext(r.Context()).Error("GetByID homeUser",
				"user_id", confirmed.HomeUserID, "error", err)
		}
		awayUser, err := s.userRepo.GetByID(r.Context(), confirmed.AwayUserID)
		if err != nil {
			logger.FromContext(r.Context()).Error("GetByID awayUser",
				"user_id", confirmed.AwayUserID, "error", err)
		}
		if homeUser != nil && awayUser != nil {
			s.applyEloUpdate(r.Context(), homeUser, awayUser, *confirmed.HomeGoals, *confirmed.AwayGoals)
			// Уведомляем обоих о подтверждении
			s.notifier.MatchConfirmed(
				homeUser.DisplayName, awayUser.DisplayName,
				*confirmed.HomeGoals, *confirmed.AwayGoals,
				homeUser.TelegramID, awayUser.TelegramID,
			)
			scoreLine := homeUser.DisplayName + " " + itoa16(*confirmed.HomeGoals) + ":" + itoa16(*confirmed.AwayGoals) + " " + awayUser.DisplayName
			if s.webPush != nil {
				go s.webPush.Notify([]int64{confirmed.HomeUserID, confirmed.AwayUserID},
					"✅ Матч подтверждён", scoreLine, "/")
			}
			s.notifyT(r.Context(), []int64{confirmed.HomeUserID, confirmed.AwayUserID},
				models.NotifMatchConfirmed, leagueLink(confirmed.LeagueID),
				func(lang string) (string, string) {
					return i18n.T(lang, "match.confirmed.title"), scoreLine
				})
			// Новость в группу: результат матча.
			s.newsMatchResult(r.Context(), confirmed, homeUser.DisplayName, awayUser.DisplayName)
		}
	}
	if confirmed != nil {
		PublishMatchUpdate(confirmed.LeagueID, confirmed.ID)
		meta := map[string]any{}
		if confirmed.HomeGoals != nil && confirmed.AwayGoals != nil {
			meta["home_goals"] = *confirmed.HomeGoals
			meta["away_goals"] = *confirmed.AwayGoals
		}
		s.audit(r, &models.AuditEntry{
			Action:     models.AuditConfirmMatch,
			EntityType: "match",
			EntityID:   &confirmed.ID,
			LeagueID:   &confirmed.LeagueID,
			TargetID:   &confirmed.HomeUserID,
			Metadata:   meta,
		})
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

	// Уведомляем хозяина — ему нужно переввести счёт
	if homeUser, err := s.userRepo.GetByID(r.Context(), m.HomeUserID); err == nil && homeUser != nil {
		awayName := ""
		if m.AwayUser != nil {
			awayName = m.AwayUser.DisplayName
		}
		s.notifier.MatchDisputed(homeUser.DisplayName, awayName, claimedHome, claimedAway, homeUser.TelegramID)
	}
	s.notifyT(r.Context(), []int64{m.HomeUserID}, models.NotifMatchDisputed, leagueLink(m.LeagueID),
		func(lang string) (string, string) {
			return i18n.T(lang, "match.disputed.title"),
				fmt.Sprintf(i18n.T(lang, "match.disputed.body"), claimedHome, claimedAway)
		})
	PublishMatchUpdate(m.LeagueID, m.ID)
	s.audit(r, &models.AuditEntry{
		Action:     models.AuditDisputeMatch,
		EntityType: "match",
		EntityID:   &m.ID,
		LeagueID:   &m.LeagueID,
		TargetID:   &m.HomeUserID,
		Metadata:   map[string]any{"claimed_home": claimedHome, "claimed_away": claimedAway},
	})
	jsonOK(w, map[string]string{"status": "disputed"})
}
