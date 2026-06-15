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
	return s.finalize(ctx, leagueID, nil)
}

// FinalizeLeagueWithChampion подводит итоги, но чемпион задаётся явно — для
// форматов на выбывание (двойная элиминация), где победитель определяется
// сеткой, а не позицией в таблице.
func (s *AwardService) FinalizeLeagueWithChampion(ctx context.Context, leagueID, championUserID int64) error {
	return s.finalize(ctx, leagueID, &championUserID)
}

func (s *AwardService) finalize(ctx context.Context, leagueID int64, championOverride *int64) error {
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

	// champion = явный (сетка) либо первая позиция в таблице.
	champion := members[0]
	for _, m := range members {
		if m.Position != nil && champion.Position != nil && *m.Position < *champion.Position {
			champion = m
		}
	}
	championID := champion.UserID
	championPoints := int(champion.Points)
	if championOverride != nil {
		championID = *championOverride
		championPoints = 0
		for _, m := range members {
			if m.UserID == championID {
				championPoints = int(m.Points)
			}
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

	if err := s.awardRepo.CreateAward(ctx, seasonID, leagueID, "champion", championID, championPoints); err != nil {
		return err
	}
	if err := s.awardRepo.CreateAward(ctx, seasonID, leagueID, "top_scorer", topScorer.UserID, int(topScorer.GoalsFor)); err != nil {
		return err
	}

	if err := s.achievRepo.Award(ctx, championID, "league_champion", &leagueID); err != nil {
		logger.FromContext(ctx).Error("award league_champion achievement", "user_id", championID, "league_id", leagueID, "err", err)
	}

	return nil
}
