package api

import (
	"efootball-bot/internal/cardgen"
	"efootball-bot/internal/models"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// handleGetPlayer returns detailed profile for a single player.
func (s *Server) handleGetPlayer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	user, err := s.userRepo.GetByID(r.Context(), id)
	if err != nil || user == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	result := userDTO(user)

	stats, err := s.userRepo.GetGlobalStats(r.Context(), id)
	if err == nil && stats != nil {
		result["total_matches"] = stats.TotalMatches
		result["total_wins"] = stats.TotalWins
		result["total_draws"] = stats.TotalDraws
		result["total_losses"] = stats.TotalLosses
		result["total_goals_for"] = stats.TotalGoalsFor
		result["total_goals_against"] = stats.TotalGoalsAgainst
		if stats.TotalMatches > 0 {
			result["win_rate"] = float64(stats.TotalWins) / float64(stats.TotalMatches) * 100
		} else {
			result["win_rate"] = 0.0
		}
	}

	if s.achievRepo != nil {
		achievements, err := s.achievRepo.GetUserAchievements(r.Context(), id)
		if err == nil {
			achievList := make([]map[string]any, 0, len(achievements))
			for _, ua := range achievements {
				if ua.Achievement == nil {
					continue
				}
				item := map[string]any{
					"code":      ua.Achievement.Code,
					"icon":      ua.Achievement.Icon,
					"name_ru":   ua.Achievement.NameRu,
					"name_uz":   ua.Achievement.NameUz,
					"name_tg":   ua.Achievement.NameTg,
					"earned_at": ua.EarnedAt,
				}
				if ua.LeagueID != nil {
					item["league_id"] = *ua.LeagueID
				}
				achievList = append(achievList, item)
			}
			result["achievements"] = achievList
		}
	}

	jsonOK(w, result)
}

// handlePlayerCard generates and returns a PNG player card.
func (s *Server) handlePlayerCard(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	user, err := s.userRepo.GetByID(r.Context(), id)
	if err != nil || user == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	data := cardgen.CardData{
		DisplayName: user.DisplayName,
		Rank:        user.Rank,
		Rating:      user.Rating,
		TeamPower:   user.TeamPower,
	}
	if user.FavoriteClub != nil {
		data.FavoriteClub = *user.FavoriteClub
	}

	stats, err := s.userRepo.GetGlobalStats(r.Context(), id)
	if err == nil && stats != nil {
		data.Wins = stats.TotalWins
		data.Draws = stats.TotalDraws
		data.Losses = stats.TotalLosses
		data.TotalGoals = stats.TotalGoalsFor
		if stats.TotalMatches > 0 {
			data.WinRate = float64(stats.TotalWins) / float64(stats.TotalMatches) * 100
		}
	}

	if s.achievRepo != nil {
		achievements, err := s.achievRepo.GetUserAchievements(r.Context(), id)
		if err == nil {
			for _, ua := range achievements {
				if ua.Achievement != nil {
					data.Achievements = append(data.Achievements, ua.Achievement.Icon)
				}
				if len(data.Achievements) >= 5 {
					break
				}
			}
		}
	}

	png, err := cardgen.GenerateCard(data)
	if err != nil {
		jsonError(w, "card generation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=600")
	w.Header().Set("Content-Length", strconv.Itoa(len(png)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

// handleHeadToHead — личные встречи (H2H) между текущим игроком и игроком {id}.
func (s *Server) handleHeadToHead(w http.ResponseWriter, r *http.Request) {
	opponentID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	me := currentUserID(r)
	if opponentID == me {
		jsonError(w, "cannot compare with yourself", http.StatusBadRequest)
		return
	}

	matches, err := s.matchRepo.GetConfirmedBetweenUsers(r.Context(), me, opponentID, 50)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	var myWins, oppWins, draws, myGoals, oppGoals int
	recent := make([]map[string]any, 0, len(matches))
	for _, m := range matches {
		if m.HomeGoals == nil || m.AwayGoals == nil {
			continue
		}
		hg, ag := int(*m.HomeGoals), int(*m.AwayGoals)
		// приводим к перспективе текущего игрока
		var myG, oppG int
		if m.HomeUserID == me {
			myG, oppG = hg, ag
		} else {
			myG, oppG = ag, hg
		}
		myGoals += myG
		oppGoals += oppG
		switch {
		case myG > oppG:
			myWins++
		case myG < oppG:
			oppWins++
		default:
			draws++
		}
		if len(recent) < 10 {
			res := "D"
			if myG > oppG {
				res = "W"
			} else if myG < oppG {
				res = "L"
			}
			recent = append(recent, map[string]any{
				"my_goals":  myG,
				"opp_goals": oppG,
				"result":    res,
				"played_at": m.PlayedAt,
			})
		}
	}

	jsonOK(w, map[string]any{
		"played":    myWins + oppWins + draws,
		"my_wins":   myWins,
		"opp_wins":  oppWins,
		"draws":     draws,
		"my_goals":  myGoals,
		"opp_goals": oppGoals,
		"recent":    recent,
	})
}

// handleRatingHistory — GET /api/players/{id}/rating-history — точки ELO для
// графика динамики на профиле (публично, как и сам профиль).
func (s *Server) handleRatingHistory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	points, err := s.userRepo.GetRatingHistory(r.Context(), id, 60)
	if err != nil {
		jsonErrorLog(w, r, "db error", http.StatusInternalServerError, err)
		return
	}
	if points == nil {
		points = []models.RatingPoint{}
	}
	jsonOK(w, map[string]any{"points": points})
}
