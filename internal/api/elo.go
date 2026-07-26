package api

import (
	"context"
	"efootball-bot/internal/logger"
	"efootball-bot/internal/models"
)

// applyEloUpdate recalculates ELO for both players and persists new ratings.
func (s *Server) applyEloUpdate(ctx context.Context, homeUser, awayUser *models.User, homeGoals, awayGoals int16) {
	newHome, newAway := s.eloSvc.Calculate(
		ctx,
		homeUser.ID, awayUser.ID,
		homeUser.Rating, awayUser.Rating,
		homeUser.TeamPower, awayUser.TeamPower,
		homeGoals, awayGoals,
	)
	if err := s.userRepo.UpdateRating(ctx, homeUser.ID, newHome); err != nil {
		logger.FromContext(ctx).Error("update elo rating", "user_id", homeUser.ID, "err", err)
	}
	if err := s.userRepo.UpdateRating(ctx, awayUser.ID, newAway); err != nil {
		logger.FromContext(ctx).Error("update elo rating", "user_id", awayUser.ID, "err", err)
	}
	if err := s.userRepo.RecalculateAllRanks(ctx); err != nil {
		logger.FromContext(ctx).Error("recalculate ranks after elo update", "err", err)
	}

	// Рейтинговые вехи (идемпотентно — unique-индекс достижений). О новых —
	// уведомляем: колокольчик + SSE, клиент показывает celebration.
	if s.achievRepo != nil {
		milestone := func(uid int64, code string) {
			inserted, err := s.achievRepo.Award(ctx, uid, code, nil)
			if err != nil || !inserted {
				return
			}
			name, icon := code, "🏅"
			if a, err := s.achievRepo.GetByCode(ctx, code); err == nil && a != nil {
				name = a.NameRu
				if a.Icon != "" {
					icon = a.Icon
				}
			}
			s.notify(ctx, []int64{uid}, models.NotifAward,
				"🏅 Новое достижение!", icon+" «"+name+"»", "/trophies")
		}
		for uid, r := range map[int64]int{homeUser.ID: newHome, awayUser.ID: newAway} {
			if r >= 1200 {
				milestone(uid, "elo_1200")
			}
			if r >= 1300 {
				milestone(uid, "elo_1300")
			}
		}
	}
}
