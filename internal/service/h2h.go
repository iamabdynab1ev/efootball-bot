package service

import (
	"context"
	"efootball-bot/internal/engine"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
)

// RecalculatePositionsH2H refines positions for teams with equal points/GD/GF using H2H results.
func RecalculatePositionsH2H(ctx context.Context, leagueID int64, leagueRepo repository.LeagueRepository, matchRepo repository.MatchRepository) error {
	members, err := leagueRepo.GetMembers(ctx, leagueID)
	if err != nil {
		return err
	}
	if len(members) == 0 {
		return nil
	}

	// Group members by (group, points, goal_diff, goals_for) — равные по очкам
	// внутри ОДНОЙ группы решаются очным зачётом. group обязателен в ключе,
	// иначе игроки из разных групп смешиваются и ломают позиции внутри групп.
	type key struct {
		group       string
		pts, gd, gf int16
	}
	groups := map[key][]*models.LeagueMember{}
	for _, m := range members {
		k := key{m.GroupName, m.Points, m.GoalDiff(), m.GoalsFor}
		groups[k] = append(groups[k], m)
	}

	// For each tied group with >1 member, sort by H2H then update positions
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		userIDs := make([]int64, len(group))
		for i, m := range group {
			userIDs[i] = m.UserID
		}
		h2hMatches, err := matchRepo.GetConfirmedMatchesBetween(ctx, leagueID, userIDs, "")
		if err != nil {
			continue
		}

		// Очный зачёт считаем чистой функцией engine.RankH2H (покрыта тестами).
		results := make([]engine.H2HResult, 0, len(h2hMatches))
		for _, m := range h2hMatches {
			if m.HomeGoals == nil || m.AwayGoals == nil {
				continue
			}
			results = append(results, engine.H2HResult{
				HomeID:    m.HomeUserID,
				AwayID:    m.AwayUserID,
				HomeGoals: int(*m.HomeGoals),
				AwayGoals: int(*m.AwayGoals),
			})
		}
		ranked := engine.RankH2H(userIDs, results)

		// Переупорядочиваем group по результату ранжирования.
		byID := make(map[int64]*models.LeagueMember, len(group))
		for _, m := range group {
			byID[m.UserID] = m
		}
		for i, uid := range ranked {
			group[i] = byID[uid]
		}

		// Assign positions within the tied group (preserving the base position of the group's lowest member)
		if group[0].Position == nil {
			continue
		}
		basePos := *group[0].Position
		// Find the minimum position in the group
		for _, m := range group {
			if m.Position != nil && *m.Position < basePos {
				basePos = *m.Position
			}
		}
		for i, m := range group {
			pos := basePos + int16(i)
			if err := leagueRepo.SetMemberPosition(ctx, m.ID, pos); err != nil {
				return err
			}
		}
	}
	return nil
}
