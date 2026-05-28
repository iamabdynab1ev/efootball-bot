// schedule.go - Лига учун матчларни генерация қилиш ва бошқариш хизмати
package service

import (
	"context"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"errors"
)

type ScheduleService struct {
	matchRepo  repository.MatchRepository
	leagueRepo repository.LeagueRepository
}

func NewScheduleService(mr repository.MatchRepository, lr repository.LeagueRepository) *ScheduleService {
	return &ScheduleService{matchRepo: mr, leagueRepo: lr}
}

func (s *ScheduleService) GenerateSchedule(ctx context.Context, leagueID int64, double bool) error {
	members, err := s.leagueRepo.GetMembers(ctx, leagueID)
	if err != nil {
		return err
	}
	if len(members) < 2 {
		return errors.New("not enough approved members to generate schedule")
	}

	userIDs := make([]int64, len(members))
	for i, m := range members {
		userIDs[i] = m.UserID
	}

	matches := roundRobin(leagueID, userIDs, double)
	return s.matchRepo.CreateBatch(ctx, matches)
}

// roundRobin генерирует матчи по алгоритму round-robin
func roundRobin(leagueID int64, players []int64, double bool) []*models.Match {
	n := len(players)
	// Если нечётное — добавляем "пустышку" (bye)
	if n%2 != 0 {
		players = append(players, -1)
		n++
	}

	rounds := n - 1
	var matches []*models.Match
	roundNum := int16(1)

	for r := 0; r < rounds; r++ {
		for i := 0; i < n/2; i++ {
			home := players[i]
			away := players[n-1-i]
			if home == -1 || away == -1 {
				continue // пропуск (bye)
			}
			matches = append(matches, &models.Match{
				LeagueID:   leagueID,
				HomeUserID: home,
				AwayUserID: away,
				Round:      roundNum,
			})
		}
		roundNum++

		// Ротация (первый элемент фиксирован)
		last := players[n-1]
		copy(players[2:], players[1:n-1])
		players[1] = last
	}

	// Второй круг — меняем дом/выезд
	if double {
		firstCircle := matches // уже созданные
		for _, m := range firstCircle {
			matches = append(matches, &models.Match{
				LeagueID:   leagueID,
				HomeUserID: m.AwayUserID,
				AwayUserID: m.HomeUserID,
				Round:      m.Round + int16(rounds),
			})
		}
	}

	return matches
}
