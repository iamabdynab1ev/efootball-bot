package api

import (
	"context"
	"efootball-bot/internal/i18n"
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
			ach, _ := s.achievRepo.GetByCode(ctx, code)
			s.notifyT(ctx, []int64{uid}, models.NotifAward, "/trophies", func(lang string) (string, string) {
				name, icon := code, "🏅"
				if ach != nil {
					switch lang {
					case "uz":
						name = ach.NameUz
					case "tg":
						name = ach.NameTg
					default:
						name = ach.NameRu
					}
					if name == "" {
						name = ach.NameRu
					}
					if ach.Icon != "" {
						icon = ach.Icon
					}
				}
				return i18n.T(lang, "award.achievement.title"), icon + " «" + name + "»"
			})
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

// ApplyEloByIDs — применение ELO по id игроков. Нужен автоматике дедлайнов:
// авто-подтверждённый реальный счёт должен давать рейтинг ровно так же, как
// подтверждение вручную. Технические результаты (0:0, тех. победа) сюда
// сознательно НЕ ходят — никто не играл, рейтинг не двигается.
func (s *Server) ApplyEloByIDs(ctx context.Context, homeID, awayID int64, homeGoals, awayGoals int16) {
	home, err := s.userRepo.GetByID(ctx, homeID)
	if err != nil || home == nil {
		logger.FromContext(ctx).Error("elo by ids: home", "user_id", homeID, "err", err)
		return
	}
	away, err := s.userRepo.GetByID(ctx, awayID)
	if err != nil || away == nil {
		logger.FromContext(ctx).Error("elo by ids: away", "user_id", awayID, "err", err)
		return
	}
	s.applyEloUpdate(ctx, home, away, homeGoals, awayGoals)
}
