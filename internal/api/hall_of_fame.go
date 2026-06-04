package api

import "net/http"

func (s *Server) handleHallOfFame(w http.ResponseWriter, r *http.Request) {
	if s.awardRepo == nil {
		jsonOK(w, map[string]any{"seasons": []any{}})
		return
	}
	awards, err := s.awardRepo.GetAll(r.Context())
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	// Group by season → league
	type leagueKey struct {
		seasonID int64
		leagueID int64
	}
	type leagueGroup struct {
		SeasonID   int64            `json:"season_id"`
		SeasonName string           `json:"season_name"`
		LeagueID   int64            `json:"league_id"`
		LeagueName string           `json:"league_name"`
		Awards     []map[string]any `json:"awards"`
	}

	groupMap := map[leagueKey]*leagueGroup{}
	var order []leagueKey

	for _, a := range awards {
		lid := int64(0)
		if a.LeagueID != nil {
			lid = *a.LeagueID
		}
		key := leagueKey{a.SeasonID, lid}
		if _, ok := groupMap[key]; !ok {
			groupMap[key] = &leagueGroup{
				SeasonID:   a.SeasonID,
				SeasonName: a.SeasonName,
				LeagueID:   lid,
				LeagueName: a.LeagueName,
			}
			order = append(order, key)
		}
		awardMap := map[string]any{
			"award_type":   a.AwardType,
			"user_id":      a.UserID,
			"display_name": a.UserDisplayName,
			"value":        a.Value,
			"created_at":   a.CreatedAt,
		}
		groupMap[key].Awards = append(groupMap[key].Awards, awardMap)
	}

	// Group further by season
	type seasonGroup struct {
		SeasonID   int64          `json:"season_id"`
		SeasonName string         `json:"season_name"`
		Leagues    []*leagueGroup `json:"leagues"`
	}
	seasonMap := map[int64]*seasonGroup{}
	var seasonOrder []int64
	for _, key := range order {
		if _, ok := seasonMap[key.seasonID]; !ok {
			g := groupMap[key]
			seasonMap[key.seasonID] = &seasonGroup{
				SeasonID:   key.seasonID,
				SeasonName: g.SeasonName,
			}
			seasonOrder = append(seasonOrder, key.seasonID)
		}
		seasonMap[key.seasonID].Leagues = append(seasonMap[key.seasonID].Leagues, groupMap[key])
	}

	result := make([]*seasonGroup, 0, len(seasonOrder))
	for _, sid := range seasonOrder {
		result = append(result, seasonMap[sid])
	}
	jsonOK(w, map[string]any{"seasons": result})
}
