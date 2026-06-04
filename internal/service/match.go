// match.go - Матчларни бошқариш учун сервис
package service

import (
	"context"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"errors"
	"fmt"
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
	achievSvc  *AchievementService
}

func NewMatchService(mr repository.MatchRepository, lr repository.LeagueRepository) *MatchService {
	return &MatchService{matchRepo: mr, leagueRepo: lr}
}

func (s *MatchService) SetAchievementService(a *AchievementService) { s.achievSvc = a }
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

// Confirm — возвращает матч только если строка была реально обновлена (защита от гонки).
func (s *MatchService) Confirm(ctx context.Context, matchID int64) (*models.Match, error) {
	match, err := s.matchRepo.GetByID(ctx, matchID)
	if err != nil || match == nil {
		return nil, ErrMatchNotFound
	}
	if match.Status != models.MatchPendingConfirm {
		return nil, ErrWrongStatus
	}

	updated, err := s.matchRepo.Confirm(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if !updated {
		// Другой запрос уже подтвердил — не пересчитываем ELO
		return nil, ErrWrongStatus
	}

	match, err = s.matchRepo.GetByID(ctx, matchID)
	if err != nil || match == nil {
		return nil, fmt.Errorf("reload match after confirm: %w", err)
	}
	if match.HomeGoals != nil && match.AwayGoals != nil {
		if err := s.leagueRepo.ApplyMatchResultStats(ctx, match.LeagueID, match.HomeUserID, match.AwayUserID, *match.HomeGoals, *match.AwayGoals); err != nil {
			return nil, fmt.Errorf("apply match stats: %w", err)
		}
		if err := s.leagueRepo.RecalculateTable(ctx, match.LeagueID); err != nil {
			return nil, fmt.Errorf("recalculate table: %w", err)
		}
		// H2H: уточняем позиции для команд с одинаковыми очками/GD/GF
		if err := RecalculatePositionsH2H(ctx, match.LeagueID, s.leagueRepo, s.matchRepo); err != nil {
			return nil, fmt.Errorf("recalculate h2h: %w", err)
		}
	}

	if s.achievSvc != nil {
		go s.achievSvc.CheckAndAward(context.Background(), match.HomeUserID, match.LeagueID, match)
		go s.achievSvc.CheckAndAward(context.Background(), match.AwayUserID, match.LeagueID, match)
	}

	return match, nil
}

var ErrDisputeLimitReached = errors.New("лимит споров исчерпан, матч передан администратору")

// Dispute — гость не согласен. Максимум 3 спора на матч.
func (s *MatchService) Dispute(ctx context.Context, matchID int64, homeClaimed, awayClaimed int16) (*models.Match, error) {
	match, err := s.matchRepo.GetByID(ctx, matchID)
	if err != nil || match == nil {
		return nil, ErrMatchNotFound
	}
	if match.Status != models.MatchPendingConfirm {
		return nil, ErrWrongStatus
	}
	if match.DisputeCount >= 3 {
		return nil, ErrDisputeLimitReached
	}

	return match, s.matchRepo.Dispute(ctx, matchID, homeClaimed, awayClaimed)
}
