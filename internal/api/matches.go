package api

import (
	"context"
	"efootball-bot/internal/logger"
	"efootball-bot/internal/models"
	"efootball-bot/internal/service"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// isGroupStage возвращает true если стадия — групповая (A, B, C, … H),
// а не плей-офф и не лига.
func isGroupStage(stage string) bool {
	switch stage {
	case models.StageLeague, models.StageR32, models.StageR16,
		models.StageQF, models.StageSF, models.StageFinal:
		return false
	}
	return stage != ""
}

// runPostConfirmAutomation запускает автоматику после подтверждения матча.
// ВАЖНО: вызывается в goroutine с context.Background() — HTTP-контекст уже закрыт.
//  1. Если групповой этап завершён → генерирует плей-офф.
//  2. Если финал завершён → закрывает турнир (FINISHED).
func (s *Server) runPostConfirmAutomation(ctx context.Context, m *models.Match) {
	league, err := s.leagueRepo.GetByID(ctx, m.LeagueID)
	if err != nil || league == nil {
		return
	}

	// Авто-плей-офф: для любого групп-формата после завершения всех групповых матчей
	isGroupFormat := league.RoundsType == "groups" || league.RoundsType == models.FormatGroupsPlayoff
	if isGroupStage(m.Stage) && isGroupFormat {
		remaining, err := s.matchRepo.CountUnconfirmedLeagueMatches(ctx, m.LeagueID)
		if err != nil || remaining > 0 {
			return
		}
		members, err := s.leagueRepo.GetMembers(ctx, m.LeagueID)
		if err != nil {
			return
		}
		cfg := service.Calculate(len(members), league.RoundsType)
		if genErr := s.groupStageSvc.GeneratePlayoffFromGroups(
			ctx, m.LeagueID, cfg.GroupAdvance, cfg.BestRunnersUp, s.bracketRepo,
		); genErr != nil {
			logger.FromContext(ctx).Error("auto playoff generation failed",
				"league_id", m.LeagueID, "error", genErr)
			return
		}
		logger.FromContext(ctx).Info("auto playoff generated", "league_id", m.LeagueID)
		InvalidateLeagues()
		s.notifier.DrawGenerated(league.Name, memberTelegramIDs(members))
	}

	// Авто-финиш: если подтверждён финальный матч
	if m.Stage == models.StageFinal {
		if err := s.leagueRepo.SetLeagueStatus(ctx, m.LeagueID, string(models.LeagueFinished)); err != nil {
			logger.FromContext(ctx).Error("auto finish league failed",
				"league_id", m.LeagueID, "error", err)
			return
		}
		logger.FromContext(ctx).Info("league finished automatically", "league_id", m.LeagueID)
		InvalidateLeagues()
	}
}

func memberTelegramIDs(members []*models.LeagueMember) []int64 {
	ids := make([]int64, 0, len(members))
	for _, m := range members {
		if m.User != nil {
			ids = append(ids, m.User.TelegramID)
		}
	}
	return ids
}

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
	}
	PublishMatchUpdate(m.LeagueID)
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

	// Продвигаем сетку плей-офф если это кубковый матч
	if confirmed != nil {
		if err := s.playoffSvc.AdvanceBracket(r.Context(), confirmed); err != nil {
			logger.FromContext(r.Context()).Error("AdvanceBracket failed",
				"match_id", matchID, "error", err)
		}
	}

	// Автоматика: авто-плей-офф после групп, авто-FINISHED после финала.
	// context.Background() — HTTP-контекст закрывается раньше чем горутина завершается.
	if confirmed != nil {
		go s.runPostConfirmAutomation(context.Background(), confirmed)
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
		}
	}
	if confirmed != nil {
		PublishMatchUpdate(confirmed.LeagueID)
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
	PublishMatchUpdate(m.LeagueID)
	jsonOK(w, map[string]string{"status": "disputed"})
}
