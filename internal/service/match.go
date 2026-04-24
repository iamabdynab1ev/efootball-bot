// match.go - Матчларни бошқариш учун сервис
package service

import (
	"context"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"errors"
)

var (
	ErrNotHomePlayer = errors.New("только хозяин поля вводит счёт")
	ErrNotAwayPlayer = errors.New("только гость подтверждает результат")
	ErrMatchNotFound = errors.New("матч не найден")
	ErrWrongStatus   = errors.New("матч в неверном статусе для этого действия")
)

type MatchService struct {
	matchRepo  repository.MatchRepository
	leagueRepo repository.LeagueRepository
}

func NewMatchService(mr repository.MatchRepository, lr repository.LeagueRepository) *MatchService {
	return &MatchService{matchRepo: mr, leagueRepo: lr}
}
func (s *MatchService) ClaimResult(ctx context.Context, matchID int64, callerTelegramID int64, homeGoals, awayGoals int16) (*models.Match, error) {
	match, err := s.matchRepo.GetByID(ctx, matchID)
	if err != nil || match == nil {
		return nil, ErrMatchNotFound
	}

	if match.HomeUser == nil {
		return nil, ErrMatchNotFound
	}
	if match.HomeUser.TelegramID != callerTelegramID {
		return nil, ErrNotHomePlayer
	}

	if match.Status != models.MatchScheduled && match.Status != models.MatchDisputed {
		return nil, ErrWrongStatus
	}

	if err := s.matchRepo.ClaimResult(ctx, matchID, homeGoals, awayGoals); err != nil {
		return nil, err
	}

	return s.matchRepo.GetByID(ctx, matchID)
}

func (s *MatchService) Confirm(ctx context.Context, matchID int64) (*models.Match, error) {
	match, err := s.matchRepo.GetByID(ctx, matchID)
	if err != nil || match == nil {
		return nil, ErrMatchNotFound
	}
	if match.Status != models.MatchPendingConfirm {
		return nil, ErrWrongStatus
	}
	if err := s.matchRepo.Confirm(ctx, matchID); err != nil {
		return nil, err
	}
	match, _ = s.matchRepo.GetByID(ctx, matchID)

	if match.HomeGoals != nil && match.AwayGoals != nil {
		err = s.leagueRepo.ApplyMatchResultStats(ctx, match.LeagueID, match.HomeUserID, match.AwayUserID, *match.HomeGoals, *match.AwayGoals)
		if err != nil {
			return nil, err
		}
		err = s.leagueRepo.RecalculateTable(ctx, match.LeagueID)
		if err != nil {
			return nil, err
		}
	}

	return match, nil
}

// Dispute — гость не согласен
func (s *MatchService) Dispute(ctx context.Context, matchID int64, homeClaimed, awayClaimed int16) (*models.Match, error) {
	match, err := s.matchRepo.GetByID(ctx, matchID)
	if err != nil || match == nil {
		return nil, ErrMatchNotFound
	}
	if match.Status != models.MatchPendingConfirm {
		return nil, ErrWrongStatus
	}

	return match, s.matchRepo.Dispute(ctx, matchID, homeClaimed, awayClaimed)
}
