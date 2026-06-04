package api

import (
	"context"
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
	_ = s.userRepo.UpdateRating(ctx, homeUser.ID, newHome)
	_ = s.userRepo.UpdateRating(ctx, awayUser.ID, newAway)
	_ = s.userRepo.RecalculateAllRanks(ctx)
}
