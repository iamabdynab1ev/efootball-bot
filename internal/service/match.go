// match.go - Матчларни бошқариш учун сервис
package service

import (
	"context"
	"efootball-bot/internal/logger"
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

// OnLeaguesChanged — хук для инвалидации кэша списка лиг (internal/api.InvalidateLeagues),
// устанавливается из cmd/bot/main.go во избежание цикла импортов service↔api.
var OnLeaguesChanged func()

// AutomationNotifier — минимальный интерфейс уведомлений, которые может
// инициировать MatchService после автоматики (например, генерация плей-офф).
// Реализуется *api.TelegramNotifier (структурно, без прямого импорта).
type AutomationNotifier interface {
	DrawGenerated(leagueName string, telegramIDs []int64)
	GroupStageComplete(leagueName string, telegramIDs []int64)
}

type MatchService struct {
	matchRepo  repository.MatchRepository
	leagueRepo repository.LeagueRepository
	achievSvc  *AchievementService
	playoffSvc *PlayoffService
	deSvc      *DoubleElimService
	awardSvc   *AwardService
	notifier   AutomationNotifier
}

func NewMatchService(mr repository.MatchRepository, lr repository.LeagueRepository) *MatchService {
	return &MatchService{matchRepo: mr, leagueRepo: lr}
}

func (s *MatchService) SetAchievementService(a *AchievementService) { s.achievSvc = a }
func (s *MatchService) SetPlayoffService(p *PlayoffService)         { s.playoffSvc = p }
func (s *MatchService) SetDoubleElimService(d *DoubleElimService)   { s.deSvc = d }
func (s *MatchService) SetAwardService(a *AwardService)             { s.awardSvc = a }
func (s *MatchService) SetNotifier(n AutomationNotifier)            { s.notifier = n }
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

	if err := s.finalizeResult(ctx, match); err != nil {
		return nil, err
	}

	return match, nil
}

// AdminResolve — администратор принудительно выставляет итоговый счёт
// (например, разрешая спор). Применяет ту же постобработку, что и обычное
// подтверждение: обновление таблицы (с учётом стадии), продвижение сетки
// плей-офф, ачивки и автоматику (авто-плей-офф / авто-завершение лиги).
func (s *MatchService) AdminResolve(ctx context.Context, matchID int64, homeGoals, awayGoals int16, adminID int64, note string) (*models.Match, error) {
	if err := s.matchRepo.AdminResolve(ctx, matchID, homeGoals, awayGoals, adminID, note); err != nil {
		return nil, err
	}

	match, err := s.matchRepo.GetByID(ctx, matchID)
	if err != nil || match == nil {
		return nil, fmt.Errorf("reload match after admin resolve: %w", err)
	}

	if err := s.finalizeResult(ctx, match); err != nil {
		return nil, err
	}

	return match, nil
}

// finalizeResult — общая постобработка подтверждённого результата матча:
// обновление таблицы (только для стадий, влияющих на турнирную таблицу),
// продвижение сетки плей-офф, начисление ачивок и запуск автоматики.
func (s *MatchService) finalizeResult(ctx context.Context, match *models.Match) error {
	if match.HomeGoals == nil || match.AwayGoals == nil {
		return nil
	}

	league, err := s.leagueRepo.GetByID(ctx, match.LeagueID)
	if err != nil {
		return fmt.Errorf("get league: %w", err)
	}
	roundsType := ""
	if league != nil {
		roundsType = league.RoundsType
	}

	if models.IsTableStage(roundsType, match.Stage) {
		if err := s.leagueRepo.ApplyMatchResultStats(ctx, match.LeagueID, match.HomeUserID, match.AwayUserID, *match.HomeGoals, *match.AwayGoals); err != nil {
			return fmt.Errorf("apply match stats: %w", err)
		}
		if err := s.leagueRepo.RecalculateTable(ctx, match.LeagueID); err != nil {
			return fmt.Errorf("recalculate table: %w", err)
		}
		// H2H: уточняем позиции для команд с одинаковыми очками/GD/GF
		if err := RecalculatePositionsH2H(ctx, match.LeagueID, s.leagueRepo, s.matchRepo); err != nil {
			return fmt.Errorf("recalculate h2h: %w", err)
		}
	}

	// Продвижение сетки: двойная элиминация — по графу de_nodes, остальные
	// форматы — по стадийной сетке bracket_slots.
	if models.GetFormat(roundsType) == models.FormatDoubleElim {
		if s.deSvc != nil {
			champion, _, err := s.deSvc.AdvanceByMatch(ctx, match)
			if err != nil {
				logger.FromContext(ctx).Error("AdvanceDoubleElim failed", "match_id", match.ID, "error", err)
			}
			if champion != nil {
				champ := *champion
				lid := match.LeagueID
				logger.Go("de-finalize", func() { s.finalizeBracketChampion(context.Background(), lid, champ) })
			}
		}
	} else if s.playoffSvc != nil {
		// no-op для не-кубковых матчей — BracketSlot == nil
		if err := s.playoffSvc.AdvanceBracket(ctx, match); err != nil {
			logger.FromContext(ctx).Error("AdvanceBracket failed", "match_id", match.ID, "error", err)
		}
	}

	// context.Background() — вызывающий контекст (HTTP/бот) может закрыться раньше горутин.
	if s.achievSvc != nil {
		logger.Go("achievements-home", func() {
			s.achievSvc.CheckAndAward(context.Background(), match.HomeUserID, match.LeagueID, match)
		})
		logger.Go("achievements-away", func() {
			s.achievSvc.CheckAndAward(context.Background(), match.AwayUserID, match.LeagueID, match)
		})
	}

	if league != nil {
		logger.Go("match-automation", func() {
			s.runAutomation(context.Background(), match, league)
		})
	}

	return nil
}

// runAutomation запускает автоматику после подтверждения результата:
//  1. Если групповой этап завершён → уведомляет о готовности к генерации плей-офф
//     (генерацию с выбором числа проходящих команд запускает администратор вручную).
//  2. Если финал завершён → закрывает турнир (FINISHED) и подводит итоги сезона.
func (s *MatchService) runAutomation(ctx context.Context, match *models.Match, league *models.League) {
	isGroupFormat := models.GetFormat(league.RoundsType) == models.FormatGroupsPlayoff
	if models.IsGroupStage(match.Stage) && isGroupFormat {
		remaining, err := s.matchRepo.CountUnconfirmedLeagueMatches(ctx, match.LeagueID)
		if err != nil || remaining > 0 {
			return
		}
		members, err := s.leagueRepo.GetMembers(ctx, match.LeagueID)
		if err != nil {
			return
		}
		logger.FromContext(ctx).Info("group stage complete, awaiting admin playoff generation", "league_id", match.LeagueID)
		if s.notifier != nil {
			s.notifier.GroupStageComplete(league.Name, memberTelegramIDs(members))
		}
	}

	if match.Stage == models.StageFinal {
		// Сначала награды (идемпотентны), потом статус: крэш между шагами
		// не оставит лигу FINISHED без наград.
		if s.awardSvc != nil {
			if err := s.awardSvc.FinalizeLeague(ctx, match.LeagueID); err != nil {
				logger.FromContext(ctx).Error("auto finalize league awards failed", "league_id", match.LeagueID, "error", err)
			}
		}
		if err := s.leagueRepo.SetLeagueStatus(ctx, match.LeagueID, string(models.LeagueFinished)); err != nil {
			logger.FromContext(ctx).Error("auto finish league failed", "league_id", match.LeagueID, "error", err)
			return
		}
		logger.FromContext(ctx).Info("league finished automatically", "league_id", match.LeagueID)
		if OnLeaguesChanged != nil {
			OnLeaguesChanged()
		}
	}
}

// finalizeBracketChampion закрывает турнир на выбывание с явным чемпионом
// (победитель гранд-финала). Сначала награды (идемпотентны), затем статус —
// крэш между шагами не оставит лигу FINISHED без наград.
func (s *MatchService) finalizeBracketChampion(ctx context.Context, leagueID, championID int64) {
	if s.awardSvc != nil {
		if err := s.awardSvc.FinalizeLeagueWithChampion(ctx, leagueID, championID); err != nil {
			logger.FromContext(ctx).Error("finalize double-elim awards failed", "league_id", leagueID, "error", err)
		}
	}
	if err := s.leagueRepo.SetLeagueStatus(ctx, leagueID, string(models.LeagueFinished)); err != nil {
		logger.FromContext(ctx).Error("finish double-elim league failed", "league_id", leagueID, "error", err)
		return
	}
	logger.FromContext(ctx).Info("double-elim league finished", "league_id", leagueID, "champion", championID)
	if OnLeaguesChanged != nil {
		OnLeaguesChanged()
	}
}

// memberTelegramIDs извлекает Telegram ID из списка участников лиги.
func memberTelegramIDs(members []*models.LeagueMember) []int64 {
	ids := make([]int64, 0, len(members))
	for _, m := range members {
		if m.User != nil {
			ids = append(ids, m.User.TelegramID)
		}
	}
	return ids
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
