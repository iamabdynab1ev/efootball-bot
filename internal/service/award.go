package service

import (
	"context"
	"efootball-bot/internal/logger"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"fmt"
)

type AwardService struct {
	awardRepo  repository.AwardRepository
	leagueRepo repository.LeagueRepository
	achievRepo repository.AchievementRepository
	matchRepo  repository.MatchRepository
}

func NewAwardService(
	awardRepo repository.AwardRepository,
	leagueRepo repository.LeagueRepository,
	achievRepo repository.AchievementRepository,
	matchRepo repository.MatchRepository,
) *AwardService {
	return &AwardService{awardRepo: awardRepo, leagueRepo: leagueRepo, achievRepo: achievRepo, matchRepo: matchRepo}
}

func (s *AwardService) FinalizeLeague(ctx context.Context, leagueID int64) error {
	return s.finalize(ctx, leagueID, nil)
}

// FinalizeLeagueWithChampion подводит итоги, но чемпион задаётся явно — для
// форматов на выбывание (двойная элиминация), где победитель определяется
// сеткой, а не позицией в таблице.
func (s *AwardService) FinalizeLeagueWithChampion(ctx context.Context, leagueID, championUserID int64) error {
	return s.finalize(ctx, leagueID, &championUserID)
}

func (s *AwardService) finalize(ctx context.Context, leagueID int64, championOverride *int64) error {
	league, err := s.leagueRepo.GetByID(ctx, leagueID)
	if err != nil || league == nil {
		return fmt.Errorf("league not found")
	}

	members, err := s.leagueRepo.GetMembers(ctx, leagueID)
	if err != nil {
		return err
	}
	if len(members) == 0 {
		return nil
	}

	// champion = явный (сетка) либо первая позиция в таблице.
	champion := members[0]
	for _, m := range members {
		if m.Position != nil && champion.Position != nil && *m.Position < *champion.Position {
			champion = m
		}
	}
	championID := champion.UserID
	championPoints := int(champion.Points)
	if championOverride != nil {
		championID = *championOverride
		championPoints = 0
		for _, m := range members {
			if m.UserID == championID {
				championPoints = int(m.Points)
			}
		}
	}

	// top_scorer = most goals_for
	topScorer := members[0]
	for _, m := range members[1:] {
		if m.GoalsFor > topScorer.GoalsFor {
			topScorer = m
		}
	}

	// best_defense = меньше всех пропустил (при равенстве — выше в таблице).
	bestDefense := members[0]
	for _, m := range members[1:] {
		if m.GoalsAgainst < bestDefense.GoalsAgainst {
			bestDefense = m
		}
	}

	// Серебро и бронза — по позиции в таблице (кроме чемпиона).
	var runnerUp, third *int64
	var runnerPts, thirdPts int
	{
		type placed struct {
			id  int64
			pos int
			pts int
		}
		var rest []placed
		for _, m := range members {
			if m.UserID == championID || m.Position == nil {
				continue
			}
			rest = append(rest, placed{id: m.UserID, pos: int(*m.Position), pts: int(m.Points)})
		}
		for i := 0; i < len(rest); i++ {
			for j := i + 1; j < len(rest); j++ {
				if rest[j].pos < rest[i].pos {
					rest[i], rest[j] = rest[j], rest[i]
				}
			}
		}
		if len(rest) > 0 {
			runnerUp, runnerPts = &rest[0].id, rest[0].pts
		}
		if len(rest) > 1 {
			third, thirdPts = &rest[1].id, rest[1].pts
		}
	}

	seasonID := league.SeasonID

	if err := s.awardRepo.CreateAward(ctx, seasonID, leagueID, "champion", championID, championPoints); err != nil {
		return err
	}
	if err := s.awardRepo.CreateAward(ctx, seasonID, leagueID, "top_scorer", topScorer.UserID, int(topScorer.GoalsFor)); err != nil {
		return err
	}
	// Дополнительные трофеи витрины — не критичны, ошибки только логируем.
	if runnerUp != nil {
		if err := s.awardRepo.CreateAward(ctx, seasonID, leagueID, "runner_up", *runnerUp, runnerPts); err != nil {
			logger.FromContext(ctx).Error("award runner_up", "league_id", leagueID, "err", err)
		}
	}
	if third != nil {
		if err := s.awardRepo.CreateAward(ctx, seasonID, leagueID, "third_place", *third, thirdPts); err != nil {
			logger.FromContext(ctx).Error("award third_place", "league_id", leagueID, "err", err)
		}
	}
	if err := s.awardRepo.CreateAward(ctx, seasonID, leagueID, "best_defense", bestDefense.UserID, int(bestDefense.GoalsAgainst)); err != nil {
		logger.FromContext(ctx).Error("award best_defense", "league_id", leagueID, "err", err)
	}

	if err := s.achievRepo.Award(ctx, championID, "league_champion", &leagueID); err != nil {
		logger.FromContext(ctx).Error("award league_champion achievement", "user_id", championID, "league_id", leagueID, "err", err)
	}

	s.tournamentTrophies(ctx, seasonID, leagueID, members)
	s.titleAchievements(ctx, championID)

	return nil
}

// tournamentTrophies — дополнительные трофеи по итогам матчей турнира:
// 💯 непобеждённый, 🧤 золотая перчатка, ⚡ лучшая разница, 💥 самый крупный
// разгром, 🔥 самая длинная победная серия. Ошибки не валят финализацию.
func (s *AwardService) tournamentTrophies(ctx context.Context, seasonID, leagueID int64, members []*models.LeagueMember) {
	give := func(awardType string, userID int64, value int) {
		if err := s.awardRepo.CreateAward(ctx, seasonID, leagueID, awardType, userID, value); err != nil {
			logger.FromContext(ctx).Error("award "+awardType, "league_id", leagueID, "err", err)
		}
	}

	// Лучшая разница мячей — из таблицы.
	bestDiff := members[0]
	for _, m := range members[1:] {
		if m.GoalsFor-m.GoalsAgainst > bestDiff.GoalsFor-bestDiff.GoalsAgainst {
			bestDiff = m
		}
	}
	give("best_diff", bestDiff.UserID, int(bestDiff.GoalsFor-bestDiff.GoalsAgainst))

	// Непобеждённый: ни одного поражения при 3+ победах (при нескольких — больше очков).
	var unbeaten *models.LeagueMember
	for _, m := range members {
		if m.Losses == 0 && m.Wins >= 3 && (unbeaten == nil || m.Points > unbeaten.Points) {
			unbeaten = m
		}
	}
	if unbeaten != nil {
		give("unbeaten", unbeaten.UserID, int(unbeaten.Points))
	}

	// Пер-матчевые: сухие матчи, крупнейшая победа, победная серия.
	if s.matchRepo == nil {
		return
	}
	matches, err := s.matchRepo.GetAllForLeague(ctx, leagueID)
	if err != nil {
		logger.FromContext(ctx).Error("trophies: league matches", "league_id", leagueID, "err", err)
		return
	}
	clean := map[int64]int{}   // «сухие» матчи
	bigWin := map[int64]int{}  // максимальная разница в одной победе
	curRun := map[int64]int{}  // текущая победная серия
	bestRun := map[int64]int{} // лучшая победная серия
	for _, mt := range matches {
		if mt.Status != "confirmed" || mt.HomeGoals == nil || mt.AwayGoals == nil {
			continue
		}
		hg, ag := int(*mt.HomeGoals), int(*mt.AwayGoals)
		h, a := mt.HomeUserID, mt.AwayUserID
		if ag == 0 {
			clean[h]++
		}
		if hg == 0 {
			clean[a]++
		}
		switch {
		case hg > ag:
			if hg-ag > bigWin[h] {
				bigWin[h] = hg - ag
			}
			curRun[h]++
			curRun[a] = 0
		case ag > hg:
			if ag-hg > bigWin[a] {
				bigWin[a] = ag - hg
			}
			curRun[a]++
			curRun[h] = 0
		default:
			curRun[h], curRun[a] = 0, 0
		}
		for _, id := range []int64{h, a} {
			if curRun[id] > bestRun[id] {
				bestRun[id] = curRun[id]
			}
		}
	}
	pick := func(m map[int64]int, min int) (int64, int) {
		var uid int64
		best := min - 1
		for id, v := range m {
			if v > best {
				best, uid = v, id
			}
		}
		return uid, best
	}
	if uid, v := pick(clean, 2); uid != 0 {
		give("golden_glove", uid, v)
	}
	if uid, v := pick(bigWin, 5); uid != 0 { // от +5: в eFootball +3 не редкость
		give("biggest_win", uid, v)
	}
	if uid, v := pick(bestRun, 3); uid != 0 {
		give("win_streak", uid, v)
	}
}

// titleAchievements — «двукратный/трёхкратный чемпион/легенда» по числу титулов.
func (s *AwardService) titleAchievements(ctx context.Context, championID int64) {
	all, err := s.awardRepo.GetByUser(ctx, championID)
	if err != nil {
		logger.FromContext(ctx).Error("title achievements", "user_id", championID, "err", err)
		return
	}
	titles := 0
	for _, a := range all {
		if a.AwardType == "champion" {
			titles++
		}
	}
	award := func(code string) {
		if err := s.achievRepo.Award(ctx, championID, code, nil); err != nil {
			logger.FromContext(ctx).Error("award "+code, "user_id", championID, "err", err)
		}
	}
	if titles >= 2 {
		award("champ_2")
	}
	if titles >= 3 {
		award("champ_3")
	}
	if titles >= 5 {
		award("champ_5")
	}
}
