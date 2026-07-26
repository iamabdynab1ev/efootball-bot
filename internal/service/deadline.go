package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"efootball-bot/internal/logger"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
)

// DeadlineService — честное исполнение дедлайнов туров и стадий плей-офф.
// По истечении срока автоматика закрывает несыгранные матчи:
//   - счёт был отправлен, но не подтверждён → авто-подтверждение заявленного счёта
//     (соперник имел всё время дедлайна, чтобы оспорить);
//   - счёт не отправлен вовсе: тур — техническая ничья 0:0 (по одному баллу каждому),
//     плей-офф — техническая победа 1:0 игрока с лучшими показателями группового
//     этапа (ничьих в кубке не бывает, сетка должна двигаться);
//   - спорные матчи автоматика НЕ решает — их разбирает администратор.
type DeadlineService struct {
	deadlineRepo repository.DeadlineRepository
	matchRepo    repository.MatchRepository
	leagueRepo   repository.LeagueRepository
	userRepo     repository.UserRepository
	matchSvc     *MatchService
	notif        *NotificationService
	groups       GroupPublisher // может быть nil
}

func NewDeadlineService(
	deadlineRepo repository.DeadlineRepository,
	matchRepo repository.MatchRepository,
	leagueRepo repository.LeagueRepository,
	userRepo repository.UserRepository,
	matchSvc *MatchService,
) *DeadlineService {
	return &DeadlineService{
		deadlineRepo: deadlineRepo,
		matchRepo:    matchRepo,
		leagueRepo:   leagueRepo,
		userRepo:     userRepo,
		matchSvc:     matchSvc,
	}
}

func (s *DeadlineService) SetNotifications(n *NotificationService) { s.notif = n }
func (s *DeadlineService) SetGroups(g GroupPublisher)              { s.groups = g }

// EnforceDue обрабатывает все истёкшие дедлайны. Идемпотентно: обработанный
// дедлайн помечается processed_at и не трогается повторно.
func (s *DeadlineService) EnforceDue(ctx context.Context) error {
	due, err := s.deadlineRepo.DueUnprocessed(ctx, time.Now())
	if err != nil {
		return fmt.Errorf("due deadlines: %w", err)
	}
	for _, dl := range due {
		if err := s.enforceOne(ctx, dl); err != nil {
			logger.FromContext(ctx).Error("deadline enforce failed",
				"league_id", dl.LeagueID, "scope", dl.ScopeLabel(), "error", err)
			continue // не блокируем остальные дедлайны
		}
		if err := s.deadlineRepo.MarkProcessed(ctx, dl.ID); err != nil {
			logger.FromContext(ctx).Error("deadline mark processed", "id", dl.ID, "error", err)
		}
	}
	return nil
}

func (s *DeadlineService) enforceOne(ctx context.Context, dl *models.RoundDeadline) error {
	var matches []*models.Match
	var err error
	if dl.Stage != "" {
		matches, err = s.matchRepo.GetMatchesByStage(ctx, dl.LeagueID, dl.Stage)
	} else {
		matches, err = s.matchRepo.GetScheduleForLeague(ctx, dl.LeagueID, dl.Round)
	}
	if err != nil {
		return fmt.Errorf("load matches: %w", err)
	}

	league, err := s.leagueRepo.GetByID(ctx, dl.LeagueID)
	if err != nil || league == nil {
		return fmt.Errorf("league not found")
	}

	// Показатели группового этапа для выбора технического победителя в плей-офф.
	seedRank := map[int64]int{}
	if dl.Stage != "" {
		members, mErr := s.leagueRepo.GetMembers(ctx, dl.LeagueID)
		if mErr == nil {
			ranked := approvedMembers(members)
			sortMembersByStanding(ranked)
			for i, m := range ranked {
				seedRank[m.UserID] = i // меньше — сильнее
			}
		}
	}

	scopeLabel := dl.ScopeLabel()
	var digest []string
	var disputed []string

	// Имена для дайджеста и уведомлений (ленивый кэш на дедлайн).
	nameCache := map[int64]string{}
	nameOf := func(uid int64) string {
		if n, ok := nameCache[uid]; ok {
			return n
		}
		n := "—"
		if u, uErr := s.userRepo.GetByID(ctx, uid); uErr == nil && u != nil {
			n = u.DisplayName
		}
		nameCache[uid] = n
		return n
	}

	for _, m := range matches {
		switch m.Status {
		case models.MatchScheduled:
			if dl.Stage != "" {
				// Плей-офф: техническая победа лучшего сида — сетка должна двигаться.
				homeWins := seedRank[m.HomeUserID] <= seedRank[m.AwayUserID]
				hg, ag := int16(1), int16(0)
				if !homeWins {
					hg, ag = 0, 1
				}
				if _, rErr := s.matchSvc.AdminResolve(ctx, m.ID, hg, ag, 0, "техническая победа: дедлайн "+scopeLabel); rErr != nil {
					logger.FromContext(ctx).Error("deadline tech win", "match_id", m.ID, "error", rErr)
					continue
				}
				winner := m.HomeUserID
				if !homeWins {
					winner = m.AwayUserID
				}
				s.notifyPair(ctx, m, "⏱ "+scopeLabel+": время вышло",
					fmt.Sprintf("Матч не сыгран к дедлайну — техническая победа %d:%d по показателям группового этапа.", hg, ag),
					league.ID)
				digest = append(digest, fmt.Sprintf("• %s — %s: тех. победа (%d:%d), проходит сид №%d",
					nameOf(m.HomeUserID), nameOf(m.AwayUserID), hg, ag, seedRank[winner]+1))
			} else {
				// Тур: техническая ничья 0:0 — по одному баллу каждому.
				if _, rErr := s.matchSvc.AdminResolve(ctx, m.ID, 0, 0, 0, "техническая ничья: дедлайн "+scopeLabel); rErr != nil {
					logger.FromContext(ctx).Error("deadline tech draw", "match_id", m.ID, "error", rErr)
					continue
				}
				s.notifyPair(ctx, m, "⏱ "+scopeLabel+": время вышло",
					"Счёт не отправлен к дедлайну — техническая ничья 0:0, по 1 баллу каждому.",
					league.ID)
				digest = append(digest, fmt.Sprintf("• %s — %s: тех. ничья 0:0",
					nameOf(m.HomeUserID), nameOf(m.AwayUserID)))
			}

		case models.MatchPendingConfirm:
			// Счёт отправлен — соперник не ответил за весь срок: авто-подтверждение.
			if _, cErr := s.matchSvc.Confirm(ctx, m.ID); cErr != nil {
				logger.FromContext(ctx).Error("deadline auto-confirm", "match_id", m.ID, "error", cErr)
				continue
			}
			s.notifyPair(ctx, m, "⏱ "+scopeLabel+": счёт подтверждён автоматически",
				"Соперник не подтвердил и не оспорил счёт до дедлайна — заявленный результат засчитан.",
				league.ID)
			digest = append(digest, fmt.Sprintf("• %s — %s: авто-подтверждение заявленного счёта",
				nameOf(m.HomeUserID), nameOf(m.AwayUserID)))

		case models.MatchDisputed:
			// Спор автоматика не решает — оставляем администратору.
			disputed = append(disputed, fmt.Sprintf("• %s — %s", nameOf(m.HomeUserID), nameOf(m.AwayUserID)))
		}
	}

	// Групповой дайджест: что сделала автоматика + что ждёт админа.
	if s.groups != nil && (len(digest) > 0 || len(disputed) > 0) {
		text := fmt.Sprintf("⏱ «%s» · %s: дедлайн истёк.", league.Name, scopeLabel)
		if len(digest) > 0 {
			text += "\nАвтоматика закрыла матчи:\n" + strings.Join(digest, "\n")
		}
		if len(disputed) > 0 {
			text += "\n⚠️ Споры ждут решения администратора:\n" + strings.Join(disputed, "\n")
		}
		s.groups.Publish(text)
	}
	return nil
}

func (s *DeadlineService) notifyPair(ctx context.Context, m *models.Match, title, body string, leagueID int64) {
	if s.notif == nil {
		return
	}
	link := fmt.Sprintf("/leagues/details?id=%d&tab=my", leagueID)
	s.notif.Notify(ctx, []int64{m.HomeUserID, m.AwayUserID}, models.NotifMatchConfirmed, title, body, link)
}

