package service

import (
	"context"
	"efootball-bot/internal/logger"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"runtime/debug"
)

type AchievementService struct {
	achievRepo repository.AchievementRepository
	matchRepo  repository.MatchRepository
	notif      *NotificationService // уведомление о новом достижении (может быть nil)
}

func NewAchievementService(achievRepo repository.AchievementRepository, matchRepo repository.MatchRepository) *AchievementService {
	return &AchievementService{achievRepo: achievRepo, matchRepo: matchRepo}
}

// SetNotifications подключает уведомления о новых достижениях (колокольчик +
// SSE → полноэкранный celebration на клиенте).
func (s *AchievementService) SetNotifications(n *NotificationService) { s.notif = n }

// notifyAchievement — общее уведомление «🏅 Новое достижение» с именем из БД.
// Используется и AwardService (титульные достижения чемпиона).
func notifyAchievement(ctx context.Context, repo repository.AchievementRepository, notif *NotificationService, userID int64, code string) {
	if notif == nil {
		return
	}
	name := code
	icon := "🏅"
	if a, err := repo.GetByCode(ctx, code); err == nil && a != nil {
		name = a.NameRu
		if a.Icon != "" {
			icon = a.Icon
		}
	}
	notif.Notify(ctx, []int64{userID}, models.NotifAward,
		"🏅 Новое достижение!", icon+" «"+name+"»", "/trophies")
}

// award persists an achievement and logs any failure instead of swallowing it.
func (s *AchievementService) award(ctx context.Context, userID int64, code string, leagueID *int64) {
	inserted, err := s.achievRepo.Award(ctx, userID, code, leagueID)
	if err != nil {
		logger.FromContext(ctx).Error("award achievement", "user_id", userID, "achievement", code, "err", err)
		return
	}
	if inserted {
		notifyAchievement(ctx, s.achievRepo, s.notif, userID, code)
	}
}

// CheckAndAward runs achievement checks after a match is confirmed. Designed to run in a goroutine.
func (s *AchievementService) CheckAndAward(ctx context.Context, userID, leagueID int64, match *models.Match) {
	defer func() {
		if r := recover(); r != nil {
			logger.FromContext(ctx).Error("achievement check panic", "user_id", userID, "league_id", leagueID, "panic", r, "stack", string(debug.Stack()))
		}
	}()

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
		if has, err := s.achievRepo.HasAchievement(ctx, userID, "first_win", nil); err != nil {
			logger.FromContext(ctx).Error("check achievement", "user_id", userID, "achievement", "first_win", "err", err)
		} else if !has {
			s.award(ctx, userID, "first_win", nil)
		}
	}

	// hat_trick
	if goalsFor >= 3 {
		s.award(ctx, userID, "hat_trick", leagueIDPtr)
	}

	// Голевое шоу — 8+ голов в одном матче (в eFootball 5 забивают легко,
	// планка держит награду ценной).
	if goalsFor >= 8 {
		s.award(ctx, userID, "poker_5", leagueIDPtr)
	}

	// Триллер — победа в упорной перестрелке: 10+ голов на двоих И разница ≤2
	// (разгром 9:1 триллером не считается).
	if won && goalsFor+goalsAgainst >= 10 && goalsFor-goalsAgainst <= 2 {
		s.award(ctx, userID, "thriller_8", leagueIDPtr)
	}

	// streaks — check last 10 confirmed matches across all leagues
	history, err := s.matchRepo.GetUserMatchHistory(ctx, userID, 10, 0, 0)
	if err != nil {
		logger.FromContext(ctx).Error("get match history for streaks", "user_id", userID, "err", err)
	} else {
		streak := countWinStreak(history, userID)
		if streak >= 10 {
			s.award(ctx, userID, "streak_10", nil)
		} else if streak >= 5 {
			s.award(ctx, userID, "streak_5", nil)
		} else if streak >= 3 {
			s.award(ctx, userID, "streak_3", nil)
		}
	}

	// scorer_10 — goals in this league
	leagueHistory, err := s.matchRepo.GetUserMatchHistory(ctx, userID, 100, 0, leagueID)
	if err != nil {
		logger.FromContext(ctx).Error("get league history for scorer_10", "user_id", userID, "league_id", leagueID, "err", err)
	} else {
		var totalGoals int16
		for _, m := range leagueHistory {
			if m.HomeUserID == userID && m.HomeGoals != nil {
				totalGoals += *m.HomeGoals
			} else if m.AwayUserID == userID && m.AwayGoals != nil {
				totalGoals += *m.AwayGoals
			}
		}
		if totalGoals >= 10 {
			s.award(ctx, userID, "scorer_10", leagueIDPtr)
		}
	}

	// clean_sheet_5 — last 5 in this league had 0 goals against
	last5, err := s.matchRepo.GetUserMatchHistory(ctx, userID, 5, 0, leagueID)
	if err != nil {
		logger.FromContext(ctx).Error("get league history for clean_sheet_5", "user_id", userID, "league_id", leagueID, "err", err)
	} else if len(last5) >= 5 {
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
			s.award(ctx, userID, "clean_sheet_5", leagueIDPtr)
		}
	}

	// Карьерные вехи — матчи и голы одним запросом (идемпотентно по unique-индексу).
	if played, goals, err := s.matchRepo.CareerStats(ctx, userID); err != nil {
		logger.FromContext(ctx).Error("career stats", "user_id", userID, "err", err)
	} else {
		if played >= 50 {
			s.award(ctx, userID, "veteran", nil)
		}
		if played >= 100 {
			s.award(ctx, userID, "veteran_100", nil)
		}
		if played >= 200 {
			s.award(ctx, userID, "veteran_200", nil)
		}
		if goals >= 100 {
			s.award(ctx, userID, "goals_100", nil)
		}
		if goals >= 250 {
			s.award(ctx, userID, "goals_250", nil)
		}
		if goals >= 500 {
			s.award(ctx, userID, "goals_500", nil)
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
