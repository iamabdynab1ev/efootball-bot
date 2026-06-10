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
}
