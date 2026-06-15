package api

import (
	"efootball-bot/internal/models"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// handleDoubleElimBracket — GET /api/leagues/{id}/double-elim
// Возвращает граф двойной элиминации, сгруппированный по сетке (winners/losers/
// grand) и раунду — готовый к рендеру тремя секциями.
func (s *Server) handleDoubleElimBracket(w http.ResponseWriter, r *http.Request) {
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	if s.deRepo == nil {
		jsonOK(w, map[string]any{"brackets": []any{}})
		return
	}

	nodes, err := s.deRepo.GetDoubleElimNodes(r.Context(), leagueID)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	type roundGroup struct {
		Round int              `json:"round"`
		Nodes []map[string]any `json:"nodes"`
	}
	type bracketGroup struct {
		Bracket string       `json:"bracket"` // de_w | de_l | de_gf
		Rounds  []roundGroup `json:"rounds"`
	}

	// Сохраняем порядок появления (репозиторий уже сортирует bracket→round→ord).
	bracketOrder := []string{}
	bracketIdx := map[string]int{}
	roundIdx := map[string]int{} // ключ "bracket#round" → позиция в Rounds
	var brackets []bracketGroup

	for _, n := range nodes {
		bi, ok := bracketIdx[n.Bracket]
		if !ok {
			bi = len(brackets)
			bracketIdx[n.Bracket] = bi
			bracketOrder = append(bracketOrder, n.Bracket)
			brackets = append(brackets, bracketGroup{Bracket: n.Bracket})
		}
		rk := n.Bracket + "#" + strconv.Itoa(n.Round)
		ri, ok := roundIdx[rk]
		if !ok {
			ri = len(brackets[bi].Rounds)
			roundIdx[rk] = ri
			brackets[bi].Rounds = append(brackets[bi].Rounds, roundGroup{Round: n.Round})
		}
		brackets[bi].Rounds[ri].Nodes = append(brackets[bi].Rounds[ri].Nodes, deNodeDTO(n))
	}
	_ = bracketOrder

	jsonOK(w, map[string]any{"brackets": brackets})
}

func deNodeDTO(n *models.DENode) map[string]any {
	m := map[string]any{
		"node_key":  n.NodeKey,
		"bracket":   n.Bracket,
		"round":     n.Round,
		"ord":       n.Ord,
		"is_reset":  n.IsReset,
		"home_name": n.HomeName,
		"away_name": n.AwayName,
		"home_club": n.HomeClub,
		"away_club": n.AwayClub,
		"status":    n.MatchStatus,
	}
	if n.HomeUserID != nil {
		m["home_user_id"] = *n.HomeUserID
	}
	if n.AwayUserID != nil {
		m["away_user_id"] = *n.AwayUserID
	}
	if n.WinnerUserID != nil {
		m["winner_user_id"] = *n.WinnerUserID
		m["winner_name"] = n.WinnerName
	}
	if n.HomeGoals != nil {
		m["home_goals"] = *n.HomeGoals
	}
	if n.AwayGoals != nil {
		m["away_goals"] = *n.AwayGoals
	}
	return m
}
