package service

import (
	"context"
	"efootball-bot/internal/repository"
	"fmt"
	"log"
	"time"
)

type ReminderNotifier interface {
	SendReminder(ctx context.Context, telegramID int64, msg string) error
}

type ReminderService struct {
	deadlineRepo repository.DeadlineRepository
	matchRepo    repository.MatchRepository
	leagueRepo   repository.LeagueRepository
	userRepo     repository.UserRepository
	notifier     ReminderNotifier
}

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
				if err != nil || user == nil || !user.HasTelegram() {
					continue
				}
				msg := "⏰ Напоминание: дедлайн раунда " + fmt.Sprint(dl.Round) + " через " + hoursLeft + "!"
				if err := s.notifier.SendReminder(ctx, user.TelegramID, msg); err != nil {
					log.Printf("reminder: send to %d: %v", user.TelegramID, err)
				}
			}
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
