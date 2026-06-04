package service

import (
	"context"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
)

type AchievementService struct {
	achievRepo repository.AchievementRepository
	matchRepo  repository.MatchRepository
}

func NewAchievementService(achievRepo repository.AchievementRepository, matchRepo repository.MatchRepository) *AchievementService {
	return &AchievementService{achievRepo: achievRepo, matchRepo: matchRepo}
}

// CheckAndAward runs achievement checks after a match is confirmed. Designed to run in a goroutine.
func (s *AchievementService) CheckAndAward(ctx context.Context, userID, leagueID int64, match *models.Match) {
	isHome := match.HomeUserID == userID
	var goalsFor, goalsAgainst int16
	if match.HomeGoals != nil && match.AwayGoals != nil {
		if isHome {
			goalsFor = *match.HomeGoals
			goalsAgainst = *match.AwayGoals
		} else {
			goalsFor = *match.AwayGoals
			goalsAgainst = *match.HomeGoals
		}
	}
	won := goalsFor > goalsAgainst
	leagueIDPtr := &leagueID

	// first_win
	if won {
		if has, _ := s.achievRepo.HasAchievement(ctx, userID, "first_win", nil); !has {
			_ = s.achievRepo.Award(ctx, userID, "first_win", nil)
		}
	}

	// hat_trick
	if goalsFor >= 3 {
		_ = s.achievRepo.Award(ctx, userID, "hat_trick", leagueIDPtr)
	}

	// streaks — check last 10 confirmed matches across all leagues
	history, err := s.matchRepo.GetUserMatchHistory(ctx, userID, 10, 0, 0)
	if err == nil {
		streak := countWinStreak(history, userID)
		if streak >= 10 {
			_ = s.achievRepo.Award(ctx, userID, "streak_10", nil)
		} else if streak >= 5 {
			_ = s.achievRepo.Award(ctx, userID, "streak_5", nil)
		} else if streak >= 3 {
			_ = s.achievRepo.Award(ctx, userID, "streak_3", nil)
		}
	}

	// scorer_10 — goals in this league
	leagueHistory, err := s.matchRepo.GetUserMatchHistory(ctx, userID, 100, 0, leagueID)
	if err == nil {
		var totalGoals int16
		for _, m := range leagueHistory {
			if m.HomeUserID == userID && m.HomeGoals != nil {
				totalGoals += *m.HomeGoals
			} else if m.AwayUserID == userID && m.AwayGoals != nil {
				totalGoals += *m.AwayGoals
			}
		}
		if totalGoals >= 10 {
			_ = s.achievRepo.Award(ctx, userID, "scorer_10", leagueIDPtr)
		}
	}

	// clean_sheet_5 — last 5 in this league had 0 goals against
	last5, err := s.matchRepo.GetUserMatchHistory(ctx, userID, 5, 0, leagueID)
	if err == nil && len(last5) >= 5 {
		allClean := true
		for _, m := range last5 {
			var against int16
			if m.HomeUserID == userID && m.AwayGoals != nil {
				against = *m.AwayGoals
			} else if m.AwayUserID == userID && m.HomeGoals != nil {
				against = *m.HomeGoals
			} else {
				allClean = false
				break
			}
			if against > 0 {
				allClean = false
				break
			}
		}
		if allClean {
			_ = s.achievRepo.Award(ctx, userID, "clean_sheet_5", leagueIDPtr)
		}
	}

	// veteran — 50 total confirmed matches
	allHistory, err := s.matchRepo.GetUserMatchHistory(ctx, userID, 51, 0, 0)
	if err == nil && len(allHistory) >= 50 {
		if has, _ := s.achievRepo.HasAchievement(ctx, userID, "veteran", nil); !has {
			_ = s.achievRepo.Award(ctx, userID, "veteran", nil)
		}
	}
}

func countWinStreak(matches []*models.Match, userID int64) int {
	streak := 0
	for _, m := range matches {
		if m.HomeGoals == nil || m.AwayGoals == nil {
			break
		}
		var won bool
		if m.HomeUserID == userID {
			won = *m.HomeGoals > *m.AwayGoals
		} else {
			won = *m.AwayGoals > *m.HomeGoals
		}
		if !won {
			break
		}
		streak++
	}
	return streak
}
