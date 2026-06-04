package service

import (
	"context"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"sort"
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

	// Group members by (points, goal_diff, goals_for) — those need H2H tiebreak
	type key struct {
		pts, gd, gf int16
	}
	groups := map[key][]*models.LeagueMember{}
	for _, m := range members {
		k := key{m.Points, m.GoalDiff(), m.GoalsFor}
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

		// Calculate H2H points and GD for the subgroup
		type h2hStats struct {
			pts int
			gd  int
		}
		stats := map[int64]*h2hStats{}
		for _, uid := range userIDs {
			stats[uid] = &h2hStats{}
		}
		for _, m := range h2hMatches {
			if m.HomeGoals == nil || m.AwayGoals == nil {
				continue
			}
			hg, ag := int(*m.HomeGoals), int(*m.AwayGoals)
			stats[m.HomeUserID].gd += hg - ag
			stats[m.AwayUserID].gd += ag - hg
			if hg > ag {
				stats[m.HomeUserID].pts += 3
			} else if hg == ag {
				stats[m.HomeUserID].pts++
				stats[m.AwayUserID].pts++
			} else {
				stats[m.AwayUserID].pts += 3
			}
		}

		// Sort group by H2H pts desc, then H2H gd desc
		sort.SliceStable(group, func(i, j int) bool {
			si := stats[group[i].UserID]
			sj := stats[group[j].UserID]
			if si.pts != sj.pts {
				return si.pts > sj.pts
			}
			return si.gd > sj.gd
		})

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
