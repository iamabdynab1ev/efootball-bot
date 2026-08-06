package api

import (
	"fmt"
	"efootball-bot/internal/i18n"
	"efootball-bot/internal/models"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// handleAdminSetScore — POST /api/admin/matches/{id}/score — ручная
// установка/изменение счёта ЛЮБОГО матча (в т.ч. подтверждённого). Админ имеет
// приоритет; таблица пересчитывается целиком (без дрейфа).
func (s *Server) handleAdminSetScore(w http.ResponseWriter, r *http.Request) {
	matchID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	var body struct {
		HomeGoals int16  `json:"home_goals"`
		AwayGoals int16  `json:"away_goals"`
		Note      string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.HomeGoals < 0 || body.HomeGoals > 50 || body.AwayGoals < 0 || body.AwayGoals > 50 {
		jsonError(w, "invalid score: goals must be 0-50", http.StatusBadRequest)
		return
	}
	// Ничья в одиночном матче плей-офф недопустима — победитель обязан пройти
	// дальше по сетке (иначе она зависнет). Best-of-X серии ничью переигрывают.
	if mm, mErr := s.matchRepo.GetByID(r.Context(), matchID); mErr == nil && mm != nil &&
		models.IsKnockoutStage(mm.Stage) && mm.BestOf <= 1 && body.HomeGoals == body.AwayGoals {
		jsonError(w, "draw_knockout", http.StatusBadRequest)
		return
	}

	m, err := s.matchSvc.AdminSetScore(r.Context(), matchID, body.HomeGoals, body.AwayGoals)
	if err != nil {
		jsonErrorLog(w, r, err.Error(), http.StatusBadRequest, err)
		return
	}

	InvalidateStandings(m.LeagueID)
	InvalidatePlayers()
	PublishMatchUpdate(m.LeagueID, m.ID)

	// Личные Telegram-уведомления игрокам + рассылка итога в группы
	// (Telegram/WhatsApp) — как при обычном подтверждении счёта.
	homeUser, _ := s.userRepo.GetByID(r.Context(), m.HomeUserID)
	awayUser, _ := s.userRepo.GetByID(r.Context(), m.AwayUserID)
	if homeUser != nil && awayUser != nil {
		s.notifier.AdminResolved(
			homeUser.DisplayName, awayUser.DisplayName,
			body.HomeGoals, body.AwayGoals,
			homeUser.TelegramID, awayUser.TelegramID,
		)
	}

	scoreLine := itoa16(body.HomeGoals) + ":" + itoa16(body.AwayGoals)
	s.notifyT(r.Context(), []int64{m.HomeUserID, m.AwayUserID}, models.NotifAdminResolve, leagueLink(m.LeagueID),
		func(lang string) (string, string) {
			return i18n.T(lang, "match.adminscore.title"), fmt.Sprintf(i18n.T(lang, "match.adminscore.body"), scoreLine)
		})
	s.audit(r, &models.AuditEntry{
		Action:     models.AuditAdminResolve,
		EntityType: "match",
		EntityID:   &m.ID,
		LeagueID:   &m.LeagueID,
		Metadata: map[string]any{
			"home_goals": body.HomeGoals,
			"away_goals": body.AwayGoals,
			"note":       body.Note,
			"mode":       "set",
		},
	})
	jsonOK(w, map[string]any{"status": "score_set", "match": matchDTO(m)})
}

// handleAdminCancelScore — POST /api/admin/matches/{id}/cancel — отмена
// результата матча (возврат в scheduled, очистка счёта, пересчёт таблицы).
func (s *Server) handleAdminCancelScore(w http.ResponseWriter, r *http.Request) {
	matchID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}

	m, err := s.matchSvc.AdminCancelScore(r.Context(), matchID)
	if err != nil {
		jsonErrorLog(w, r, err.Error(), http.StatusBadRequest, err)
		return
	}

	InvalidateStandings(m.LeagueID)
	InvalidatePlayers()
	PublishMatchUpdate(m.LeagueID, m.ID)

	s.notifyT(r.Context(), []int64{m.HomeUserID, m.AwayUserID}, models.NotifAdminResolve, leagueLink(m.LeagueID),
		func(lang string) (string, string) {
			return i18n.T(lang, "match.admincancel.title"), i18n.T(lang, "match.admincancel.body")
		})
	s.audit(r, &models.AuditEntry{
		Action:     models.AuditAdminResolve,
		EntityType: "match",
		EntityID:   &m.ID,
		LeagueID:   &m.LeagueID,
		Metadata:   map[string]any{"mode": "cancel"},
	})
	jsonOK(w, map[string]any{"status": "score_cancelled", "match": matchDTO(m)})
}
