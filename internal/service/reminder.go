package service

import (
	"context"
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
	groups       GroupPublisher // может быть nil — тогда дайджест не шлём
}

// SetGroups подключает шину групповых уведомлений.
func (s *ReminderService) SetGroups(g GroupPublisher) { s.groups = g }

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
		is1h := dl.Deadline.Before(now.Add(65*time.Minute)) && dl.Deadline.After(now.Add(55*time.Minute))
		is24h := dl.Deadline.Before(now.Add(25*time.Hour)) && dl.Deadline.After(now.Add(23*time.Hour))

		if (!is1h || dl.Reminder1hSent) && (!is24h || dl.Reminder24hSent) {
			continue
		}

		matches, err := s.matchRepo.GetScheduleForLeague(ctx, dl.LeagueID, dl.Round)
		if err != nil {
			log.Printf("reminder: get schedule: %v", err)
			continue
		}

		var hoursLeft string
		if is1h {
			hoursLeft = "1 час"
		} else {
			hoursLeft = "24 часа"
		}

		notified := map[int64]bool{}
		names := map[int64]string{} // имена для группового дайджеста
		var pending []string        // пары, не сыгравшие к дедлайну
		for _, m := range matches {
			if m.Status != "scheduled" {
				continue
			}
			for _, uid := range []int64{m.HomeUserID, m.AwayUserID} {
				if notified[uid] {
					continue
				}
				notified[uid] = true
				user, err := s.userRepo.GetByID(ctx, uid)
				if err != nil || user == nil {
					continue
				}
				names[uid] = user.DisplayName
				if !user.HasTelegram() {
					continue
				}
				msg := "⏰ Напоминание: дедлайн раунда " + fmt.Sprint(dl.Round) + " через " + hoursLeft + "!"
				if err := s.notifier.SendReminder(ctx, user.TelegramID, msg); err != nil {
					log.Printf("reminder: send to %d: %v", user.TelegramID, err)
				}
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

		// Дайджест в общую группу: кто ещё не сыграл к дедлайну.
		if s.groups != nil && len(pending) > 0 {
			leagueName := ""
			if lg, err := s.leagueRepo.GetByID(ctx, dl.LeagueID); err == nil && lg != nil {
				leagueName = " «" + lg.Name + "»"
			}
			s.groups.Publish(fmt.Sprintf(
				"⏰ Напоминание%s: до дедлайна раунда %d осталось %s!\nЕщё не сыграли:\n%s",
				leagueName, dl.Round, hoursLeft, strings.Join(pending, "\n"),
			))
		}

		if is1h && !dl.Reminder1hSent {
			if err := s.deadlineRepo.MarkReminderSent(ctx, dl.ID, false); err != nil {
				log.Printf("reminder: mark 1h sent (deadline %d): %v", dl.ID, err)
			}
		}
		if is24h && !dl.Reminder24hSent {
			if err := s.deadlineRepo.MarkReminderSent(ctx, dl.ID, true); err != nil {
				log.Printf("reminder: mark 24h sent (deadline %d): %v", dl.ID, err)
			}
		}
	}
	return nil
}
