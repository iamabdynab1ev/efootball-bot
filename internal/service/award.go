package service

import (
	"context"
	"efootball-bot/internal/logger"
	"efootball-bot/internal/repository"
	"fmt"
)

type AwardService struct {
	awardRepo  repository.AwardRepository
	leagueRepo repository.LeagueRepository
	achievRepo repository.AchievementRepository
}

func NewAwardService(
	awardRepo repository.AwardRepository,
	leagueRepo repository.LeagueRepository,
	achievRepo repository.AchievementRepository,
) *AwardService {
	return &AwardService{awardRepo: awardRepo, leagueRepo: leagueRepo, achievRepo: achievRepo}
}

func (s *AwardService) FinalizeLeague(ctx context.Context, leagueID int64) error {
	league, err := s.leagueRepo.GetByID(ctx, leagueID)
	if err != nil || league == nil {
		return fmt.Errorf("league not found")
	}

	members, err := s.leagueRepo.GetMembers(ctx, leagueID)
	if err != nil {
		return err
	}
	if len(members) == 0 {
		return nil
	}

	// champion = first position
	var champion = members[0]
	for _, m := range members {
		if m.Position != nil && champion.Position != nil && *m.Position < *champion.Position {
			champion = m
		}
	}

	// top_scorer = most goals_for
	topScorer := members[0]
	for _, m := range members[1:] {
		if m.GoalsFor > topScorer.GoalsFor {
			topScorer = m
		}
	}

	seasonID := league.SeasonID

	if err := s.awardRepo.CreateAward(ctx, seasonID, leagueID, "champion", champion.UserID, int(champion.Points)); err != nil {
		return err
	}
	if err := s.awardRepo.CreateAward(ctx, seasonID, leagueID, "top_scorer", topScorer.UserID, int(topScorer.GoalsFor)); err != nil {
		return err
	}

	if err := s.achievRepo.Award(ctx, champion.UserID, "league_champion", &leagueID); err != nil {
		logger.FromContext(ctx).Error("award league_champion achievement", "user_id", champion.UserID, "league_id", leagueID, "err", err)
	}

	return nil
}
