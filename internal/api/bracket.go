package api

import (
	"context"
	"efootball-bot/internal/logger"
	"efootball-bot/internal/models"
	"efootball-bot/internal/service"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// handleBracket — GET /api/leagues/{id}/bracket
func (s *Server) handleBracket(w http.ResponseWriter, r *http.Request) {
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}

	slots, err := s.bracketRepo.GetAllSlots(r.Context(), leagueID)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	// Group by stage keeping order
	type stageGroup struct {
		Stage string           `json:"stage"`
		Label string           `json:"label"`
		Slots []map[string]any `json:"slots"`
	}

	stageMap := map[string]*stageGroup{}
	order := []string{}

	for _, slot := range slots {
		if _, exists := stageMap[slot.Stage]; !exists {
			label := models.StageLabel[slot.Stage]
			if label == "" {
				label = slot.Stage
			}
			stageMap[slot.Stage] = &stageGroup{Stage: slot.Stage, Label: label}
			order = append(order, slot.Stage)
		}
		stageMap[slot.Stage].Slots = append(stageMap[slot.Stage].Slots, bracketSlotDTO(slot))
	}

	result := make([]stageGroup, 0, len(order))
	for _, stage := range order {
		result = append(result, *stageMap[stage])
	}

	jsonOK(w, map[string]any{"stages": result})
}

// isGroupFormat reports whether the league's rounds_type uses a group stage
// followed by a knockout playoff (hybrid/groups/groups_playoff).
func isGroupFormat(roundsType string) bool {
	return roundsType == "hybrid" || roundsType == "groups" || roundsType == "groups_playoff"
}

// groupAdvanceRange computes the overall valid [min,max] range for the
// number of teams advancing per group, as the intersection of each group's
// individual service.AdvanceRange (group sizes may differ by at most 1).
// Один запрос GetMembers вместо запроса на каждую группу.
func (s *Server) groupAdvanceRange(ctx context.Context, leagueID int64) (groups []map[string]any, min, max int, err error) {
	members, err := s.leagueRepo.GetMembers(ctx, leagueID)
	if err != nil {
		return nil, 0, 0, err
	}
	sizes := map[string]int{}
	var order []string
	for _, m := range members {
		if m.GroupName == "" {
			continue
		}
		if _, seen := sizes[m.GroupName]; !seen {
			order = append(order, m.GroupName)
		}
		sizes[m.GroupName]++
	}
	sort.Strings(order)
	for _, g := range order {
		size := sizes[g]
		groups = append(groups, map[string]any{"name": g, "size": size})
		gMin, gMax := service.AdvanceRange(size)
		if min == 0 || gMin > min {
			min = gMin
		}
		if max == 0 || gMax < max {
			max = gMax
		}
	}
	if max < min {
		max = min
	}
	return groups, min, max, nil
}

// playoffBracketOptions возвращает варианты «сколько выходит из каждой группы»,
// дающие РОВНУЮ сетку (степень двойки участников ≤ 32). Если чистое advance×групп
// не степень двойки — добираем лучшими командами следующего места (как на Евро:
// топ-2 из групп + лучшие третьи), поле runners_up. Так админ выбирает из
// понятных вариантов (1/16, 1/8, 1/4…) без сеток с кучей bye.
func playoffBracketOptions(groupSizes []int) []map[string]any {
	opts := []map[string]any{}
	numGroups := len(groupSizes)
	if numGroups < 1 {
		return opts
	}
	minSize := groupSizes[0]
	for _, sz := range groupSizes {
		if sz < minSize {
			minSize = sz
		}
	}
	for advance := 1; advance <= minSize; advance++ {
		total := advance * numGroups
		if total > 32 {
			break // стадии сетки идут максимум до r32 (32 команды)
		}
		if total >= 2 && total&(total-1) == 0 {
			opts = append(opts, map[string]any{
				"advance":    advance,
				"runners_up": 0,
				"qualifiers": total,
				"stage":      models.FirstStageForSize(total),
			})
			continue
		}
		// Добор до степени двойки лучшими командами места (advance+1).
		// Кандидатов — по одному из каждой группы, где такое место существует.
		candidates := 0
		for _, sz := range groupSizes {
			if sz > advance {
				candidates++
			}
		}
		p2 := models.NextPowerOf2(total)
		extra := p2 - total
		if extra >= 1 && extra < numGroups && extra <= candidates && p2 >= 2 && p2 <= 32 {
			opts = append(opts, map[string]any{
				"advance":    advance,
				"runners_up": extra,
				"qualifiers": p2,
				"stage":      models.FirstStageForSize(p2),
			})
		}
	}
	return opts
}

// handleAdminPlayoffOptions — GET /api/admin/leagues/{id}/playoff-options
// Returns the group sizes, the valid range of "advance per group" values and
// the clean power-of-2 bracket options for the playoff-generation dropdown.
// For non-group formats returns an empty groups list.
func (s *Server) handleAdminPlayoffOptions(w http.ResponseWriter, r *http.Request) {
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}

	league, err := s.leagueRepo.GetByID(r.Context(), leagueID)
	if err != nil || league == nil {
		jsonError(w, "league not found", http.StatusNotFound)
		return
	}

	if !isGroupFormat(league.RoundsType) {
		jsonOK(w, map[string]any{"groups": []map[string]any{}})
		return
	}

	groups, min, max, err := s.groupAdvanceRange(r.Context(), leagueID)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	def := min
	if members, mErr := s.leagueRepo.GetMembers(r.Context(), leagueID); mErr == nil {
		cfg := service.Calculate(len(members), league.RoundsType)
		def = cfg.GroupAdvance
		if def < min {
			def = min
		}
		if def > max {
			def = max
		}
	}

	// Ровные варианты сетки (степень двойки), при необходимости — с добором
	// лучших команд следующего места (как на Евро).
	sizes := make([]int, 0, len(groups))
	for _, g := range groups {
		if sz, ok := g["size"].(int); ok {
			sizes = append(sizes, sz)
		}
	}
	options := playoffBracketOptions(sizes)
	// Дефолт — наибольшая ровная сетка (макс. участников), если она есть.
	if len(options) > 0 {
		if a, ok := options[len(options)-1]["advance"].(int); ok {
			def = a
		}
	}

	jsonOK(w, map[string]any{
		"groups":          groups,
		"advance_min":     min,
		"advance_max":     max,
		"advance_default": def,
		"options":         options,
	})
}

// handleAdminPlayoff — POST /api/admin/leagues/{id}/playoff
func (s *Server) handleAdminPlayoff(w http.ResponseWriter, r *http.Request) {
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}

	// Проверяем что все матчи основного этапа подтверждены
	remaining, err := s.matchRepo.CountUnconfirmedLeagueMatches(r.Context(), leagueID)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	if remaining > 0 {
		jsonError(w, fmt.Sprintf("не все матчи сыграны: осталось %d", remaining), http.StatusBadRequest)
		return
	}

	league, err := s.leagueRepo.GetByID(r.Context(), leagueID)
	if err != nil || league == nil {
		jsonError(w, "league not found", http.StatusNotFound)
		return
	}

	if isGroupFormat(league.RoundsType) {
		var body struct {
			GroupAdvance  int  `json:"group_advance"`
			BestRunnersUp int  `json:"best_runners_up"`
			RandomDraw    bool `json:"random_draw"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		groupAdvance := body.GroupAdvance
		bestRunnersUp := body.BestRunnersUp
		if groupAdvance > 0 {
			// Валидация против реальных размеров групп: из группы нельзя вывести
			// больше игроков, чем в ней есть; добор — максимум по одному кандидату
			// с места (advance+1) из каждой группы, но не все сразу.
			groups, _, _, rErr := s.groupAdvanceRange(r.Context(), leagueID)
			if rErr != nil {
				jsonError(w, "db error", http.StatusInternalServerError)
				return
			}
			minSize, candidates := 0, 0
			for _, g := range groups {
				if sz, ok := g["size"].(int); ok {
					if minSize == 0 || sz < minSize {
						minSize = sz
					}
					if sz > groupAdvance {
						candidates++
					}
				}
			}
			if groupAdvance < 1 || groupAdvance > minSize {
				jsonError(w, fmt.Sprintf("group_advance must be between 1 and %d", minSize), http.StatusBadRequest)
				return
			}
			if bestRunnersUp < 0 || bestRunnersUp > candidates || bestRunnersUp >= len(groups) {
				jsonError(w, "invalid best_runners_up", http.StatusBadRequest)
				return
			}
			if total := groupAdvance*len(groups) + bestRunnersUp; total < 2 || total > 32 {
				jsonError(w, "qualifiers must be between 2 and 32", http.StatusBadRequest)
				return
			}
		} else {
			members, mErr := s.leagueRepo.GetMembers(r.Context(), leagueID)
			if mErr != nil {
				jsonError(w, "db error", http.StatusInternalServerError)
				return
			}
			cfg := service.Calculate(len(members), league.RoundsType)
			groupAdvance, bestRunnersUp = cfg.GroupAdvance, cfg.BestRunnersUp
		}

		err = s.groupStageSvc.GeneratePlayoffFromGroups(
			r.Context(), leagueID,
			groupAdvance, bestRunnersUp, body.RandomDraw,
			s.bracketRepo,
		)
		if err == nil {
			// Сохраняем выбранное число проходящих — таблицы групп подсвечивают
			// зону выхода в плей-офф («Топ-N → плей-офф»).
			numGroups := int(league.NumGroups)
			if groups, _, _, gErr := s.groupAdvanceRange(r.Context(), leagueID); gErr == nil && len(groups) > 0 {
				numGroups = len(groups)
			}
			if cfgErr := s.leagueRepo.SetLeagueGroupConfig(r.Context(), leagueID, numGroups, groupAdvance); cfgErr != nil {
				logger.FromContext(r.Context()).Error("save group config after playoff", "league_id", leagueID, "error", cfgErr)
			}
		}
	} else if league.RoundsType == "double_elim" {
		var body struct {
			TopK int `json:"top_k"`
		}
		if dErr := json.NewDecoder(r.Body).Decode(&body); dErr != nil || body.TopK <= 0 {
			body.TopK = 8
		}
		if s.deSvc == nil {
			jsonError(w, "double elimination not available", http.StatusInternalServerError)
			return
		}
		err = s.deSvc.Generate(r.Context(), leagueID, body.TopK)
	} else {
		var body struct {
			TopK int `json:"top_k"`
		}
		if dErr := json.NewDecoder(r.Body).Decode(&body); dErr != nil || body.TopK <= 0 {
			body.TopK = 8
		}
		err = s.playoffSvc.GeneratePlayoff(r.Context(), leagueID, body.TopK)
	}
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Новость в группу: пары плей-офф.
	s.newsPlayoffDraw(r.Context(), leagueID)
	jsonOK(w, map[string]string{"status": "playoff_generated"})
}

// handleLeagueDeadlines — GET /api/leagues/{id}/deadlines (публичный).
// Актуальные (необработанные) дедлайны туров и стадий — фронт рисует по ним
// обратный отсчёт на главной и на странице лиги.
func (s *Server) handleLeagueDeadlines(w http.ResponseWriter, r *http.Request) {
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	if s.deadlineRepo == nil {
		jsonOK(w, map[string]any{"deadlines": []any{}})
		return
	}
	all, err := s.deadlineRepo.GetDeadlines(r.Context(), leagueID)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	out := make([]map[string]any, 0, len(all))
	for _, d := range all {
		if d.ProcessedAt != nil {
			continue // прошедшие и закрытые автоматикой не показываем
		}
		out = append(out, map[string]any{
			"round":    d.Round,
			"stage":    d.Stage,
			"deadline": d.Deadline.UTC().Format(time.RFC3339),
		})
	}
	jsonOK(w, map[string]any{"deadlines": out})
}

func bracketSlotDTO(s *models.BracketSlot) map[string]any {
	m := map[string]any{
		"slot":        s.Slot,
		"stage":       s.Stage,
		"home_name":   s.HomeName,
		"away_name":   s.AwayName,
		"home_club":   s.HomeClub,
		"away_club":   s.AwayClub,
		"winner_club": s.WinnerClub,
	}
	if s.HomeUserID != nil {
		m["home_user_id"] = *s.HomeUserID
	}
	if s.AwayUserID != nil {
		m["away_user_id"] = *s.AwayUserID
	}
	if s.MatchID != nil {
		m["match_id"] = *s.MatchID
	}
	if s.WinnerUserID != nil {
		m["winner_user_id"] = *s.WinnerUserID
		m["winner_name"] = s.WinnerName
	}
	if s.HomeGoals != nil {
		m["home_goals"] = *s.HomeGoals
	}
	if s.AwayGoals != nil {
		m["away_goals"] = *s.AwayGoals
	}
	if s.MatchStatus != "" {
		m["match_status"] = s.MatchStatus
	}
	return m
}
