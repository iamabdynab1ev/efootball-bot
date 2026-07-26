package service

import (
	"context"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"fmt"
	"log"
	"strings"
	"time"
)

type ReminderNotifier interface {
	SendReminder(ctx context.Context, telegramID int64, msg string) error
}

// GroupPublisher — шина групповых уведомлений (Telegram-группа/WhatsApp);
// сюда уходит дайджест несыгранных матчей перед дедлайном.
type GroupPublisher interface {
	Publish(text string)
}

type ReminderService struct {
	deadlineRepo repository.DeadlineRepository
	matchRepo    repository.MatchRepository
	leagueRepo   repository.LeagueRepository
	userRepo     repository.UserRepository
	notifier     ReminderNotifier
	notif        *NotificationService // in-app/push/TG по состоянию получателя
	groups       GroupPublisher       // может быть nil — тогда дайджест не шлём
}

// SetGroups подключает шину групповых уведомлений.
func (s *ReminderService) SetGroups(g GroupPublisher) { s.groups = g }

// SetNotifications подключает единый канал уведомлений (колокольчик + push).
func (s *ReminderService) SetNotifications(n *NotificationService) { s.notif = n }

func NewReminderService(
	deadlineRepo repository.DeadlineRepository,
	matchRepo repository.MatchRepository,
	leagueRepo repository.LeagueRepository,
	userRepo repository.UserRepository,
	notifier ReminderNotifier,
) *ReminderService {
	return &ReminderService{
		deadlineRepo: deadlineRepo,
		matchRepo:    matchRepo,
		leagueRepo:   leagueRepo,
		userRepo:     userRepo,
		notifier:     notifier,
	}
}

func (s *ReminderService) CheckAndSend(ctx context.Context) error {
	now := time.Now()
	deadlines, err := s.deadlineRepo.GetPendingReminders(ctx, now)
	if err != nil {
		return err
	}

	for _, dl := range deadlines {
		is1h := !dl.Reminder1hSent && dl.Deadline.Before(now.Add(65*time.Minute))
		is24h := !is1h && !dl.Reminder24hSent && dl.Deadline.Before(now.Add(25*time.Hour))
		if !is1h && !is24h {
			continue
		}

		// Матчи в зоне дедлайна: тур или стадия плей-офф.
		var matches []*models.Match
		var mErr error
		if dl.Stage != "" {
			matches, mErr = s.matchRepo.GetMatchesByStage(ctx, dl.LeagueID, dl.Stage)
		} else {
			matches, mErr = s.matchRepo.GetScheduleForLeague(ctx, dl.LeagueID, dl.Round)
		}
		if mErr != nil {
			log.Printf("reminder: get matches: %v", mErr)
			continue
		}

		left := humanLeft(dl.Deadline.Sub(now))
		scope := dl.ScopeLabel()
		leagueName := ""
		if lg, lErr := s.leagueRepo.GetByID(ctx, dl.LeagueID); lErr == nil && lg != nil {
			leagueName = lg.Name
		}

		notified := map[int64]bool{}
		names := map[int64]string{}
		var pending []string
		for _, m := range matches {
			// Напоминаем всем, чей матч ещё не закрыт (включая неподтверждённые).
			if m.Status == models.MatchConfirmed || m.Status == models.MatchCancelled {
				continue
			}
			for _, uid := range []int64{m.HomeUserID, m.AwayUserID} {
				if notified[uid] {
					continue
				}
				notified[uid] = true
				user, uErr := s.userRepo.GetByID(ctx, uid)
				if uErr != nil || user == nil {
					continue
				}
				names[uid] = user.DisplayName
			}
			h, a := names[m.HomeUserID], names[m.AwayUserID]
			if h == "" {
				h = "?"
			}
			if a == "" {
				a = "?"
			}
			pending = append(pending, "• "+h+" — "+a)
		}

		if len(notified) > 0 {
			ids := make([]int64, 0, len(notified))
			for uid := range notified {
				ids = append(ids, uid)
			}
			title := "⏰ " + scope + ": осталось " + left
			body := "«" + leagueName + "»: сыграйте и отправьте счёт до дедлайна — иначе результат закроет автоматика."
			if s.notif != nil {
				// Единый канал: in-app + push + Telegram по состоянию получателя.
				s.notif.Notify(ctx, ids, "system", title, body,
					fmt.Sprintf("/leagues/details?id=%d&tab=my", dl.LeagueID))
			} else if s.notifier != nil {
				for uid := range notified {
					user, uErr := s.userRepo.GetByID(ctx, uid)
					if uErr != nil || user == nil || !user.HasTelegram() {
						continue
					}
					if err := s.notifier.SendReminder(ctx, user.TelegramID, title+" — "+body); err != nil {
						log.Printf("reminder: send to %d: %v", user.TelegramID, err)
					}
				}
			}
		}

		// Дайджест в общую группу: кто ещё не сыграл к дедлайну.
		if s.groups != nil && len(pending) > 0 {
			s.groups.Publish(fmt.Sprintf(
				"⏰ «%s» · %s: осталось %s!\nЕщё не закрыли матчи:\n%s",
				leagueName, scope, left, strings.Join(pending, "\n"),
			))
		}

		if is1h {
			if err := s.deadlineRepo.MarkReminderSent(ctx, dl.ID, false); err != nil {
				log.Printf("reminder: mark 1h sent (deadline %d): %v", dl.ID, err)
			}
		}
		if is24h {
			if err := s.deadlineRepo.MarkReminderSent(ctx, dl.ID, true); err != nil {
				log.Printf("reminder: mark 24h sent (deadline %d): %v", dl.ID, err)
			}
		}
	}
	return nil
}

// humanLeft — «2 ч 15 мин» / «45 мин» — честное оставшееся время вместо
// шаблонных «24 часа», которые врут при коротких дедлайнах.
func humanLeft(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	switch {
	case h >= 24:
		return fmt.Sprintf("%d дн %d ч", h/24, h%24)
	case h > 0:
		return fmt.Sprintf("%d ч %d мин", h, m)
	default:
		return fmt.Sprintf("%d мин", m)
	}
}
