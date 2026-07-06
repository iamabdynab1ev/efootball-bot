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

	// best_defense = меньше всех пропустил (при равенстве — выше в таблице).
	bestDefense := members[0]
	for _, m := range members[1:] {
		if m.GoalsAgainst < bestDefense.GoalsAgainst {
			bestDefense = m
		}
	}

	// Серебро и бронза — по позиции в таблице (кроме чемпиона).
	var runnerUp, third *int64
	var runnerPts, thirdPts int
	{
		type placed struct {
			id  int64
			pos int
			pts int
		}
		var rest []placed
		for _, m := range members {
			if m.UserID == championID || m.Position == nil {
				continue
			}
			rest = append(rest, placed{id: m.UserID, pos: int(*m.Position), pts: int(m.Points)})
		}
		for i := 0; i < len(rest); i++ {
			for j := i + 1; j < len(rest); j++ {
				if rest[j].pos < rest[i].pos {
					rest[i], rest[j] = rest[j], rest[i]
				}
			}
		}
		if len(rest) > 0 {
			runnerUp, runnerPts = &rest[0].id, rest[0].pts
		}
		if len(rest) > 1 {
			third, thirdPts = &rest[1].id, rest[1].pts
		}
	}

	seasonID := league.SeasonID

	if err := s.awardRepo.CreateAward(ctx, seasonID, leagueID, "champion", championID, championPoints); err != nil {
		return err
	}
	if err := s.awardRepo.CreateAward(ctx, seasonID, leagueID, "top_scorer", topScorer.UserID, int(topScorer.GoalsFor)); err != nil {
		return err
	}
	// Дополнительные трофеи витрины — не критичны, ошибки только логируем.
	if runnerUp != nil {
		if err := s.awardRepo.CreateAward(ctx, seasonID, leagueID, "runner_up", *runnerUp, runnerPts); err != nil {
			logger.FromContext(ctx).Error("award runner_up", "league_id", leagueID, "err", err)
		}
	}
	if third != nil {
		if err := s.awardRepo.CreateAward(ctx, seasonID, leagueID, "third_place", *third, thirdPts); err != nil {
			logger.FromContext(ctx).Error("award third_place", "league_id", leagueID, "err", err)
		}
	}
	if err := s.awardRepo.CreateAward(ctx, seasonID, leagueID, "best_defense", bestDefense.UserID, int(bestDefense.GoalsAgainst)); err != nil {
		logger.FromContext(ctx).Error("award best_defense", "league_id", leagueID, "err", err)
	}

	if err := s.achievRepo.Award(ctx, championID, "league_champion", &leagueID); err != nil {
		logger.FromContext(ctx).Error("award league_champion achievement", "user_id", championID, "league_id", leagueID, "err", err)
	}

	return nil
}
