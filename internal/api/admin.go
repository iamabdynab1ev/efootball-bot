package api

import (
	"context"
	"efootball-bot/internal/logger"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"efootball-bot/internal/service"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleAdminListLeagues(w http.ResponseWriter, r *http.Request) {
	leagues, err := s.leagueRepo.GetAllLeagues(r.Context())
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	result := make([]map[string]any, 0, len(leagues))
	for _, l := range leagues {
		result = append(result, leagueDTO(l))
	}
	jsonOK(w, result)
}

func (s *Server) handleAdminCreateLeague(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name          string  `json:"name"`
		Deadline      *string `json:"registration_deadline"`
		RoundsType    string  `json:"rounds_type"`
		NumGroups     int16   `json:"num_groups"`
		GroupAdvance  int16   `json:"group_advance"`
		BestRunnersUp int16   `json:"best_runners_up"`
		BestOf        int16   `json:"best_of"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		jsonError(w, "name required", http.StatusBadRequest)
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if len(body.Name) > 100 {
		jsonError(w, "name must be 1-100 characters", http.StatusBadRequest)
		return
	}
	var deadline *time.Time
	if body.Deadline != nil && *body.Deadline != "" {
		t, err := time.Parse(time.RFC3339, *body.Deadline)
		if err != nil {
			// попробуем без секунд
			t, err = time.Parse("2006-01-02T15:04", *body.Deadline)
		}
		if err == nil {
			deadline = &t
		}
	}
	season, err := s.leagueRepo.GetOrCreateActiveSeason(r.Context())
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	league, err := s.leagueRepo.CreateLeague(r.Context(), season.ID, body.Name, deadline, body.RoundsType, body.NumGroups, body.GroupAdvance, body.BestRunnersUp)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	// Best-of-X (для матчей на выбывание): нечётное значение ≥ 3 включает серии.
	if body.BestOf >= 3 && body.BestOf%2 == 1 {
		if err := s.leagueRepo.SetLeagueBestOf(r.Context(), league.ID, int(body.BestOf)); err != nil {
			logger.FromContext(r.Context()).Error("SetLeagueBestOf failed", "league_id", league.ID, "error", err)
		} else {
			league.BestOf = body.BestOf
		}
	}
	s.audit(r, &models.AuditEntry{
		Action:     models.AuditLeagueCreate,
		EntityType: "league",
		EntityID:   &league.ID,
		LeagueID:   &league.ID,
		Metadata:   map[string]any{"name": league.Name, "rounds_type": body.RoundsType},
	})
	jsonOK(w, leagueDTO(league))
}

func (s *Server) handleAdminUpdateLeague(w http.ResponseWriter, r *http.Request) {
	if ok, _ := s.adminRepo.IsSuperAdminByUserID(r.Context(), currentUserID(r)); !ok {
		jsonError(w, "super admin required", http.StatusForbidden)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	var body struct {
		Name     string  `json:"name"`
		Deadline *string `json:"registration_deadline"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		jsonError(w, "name required", http.StatusBadRequest)
		return
	}
	var deadline *time.Time
	if body.Deadline != nil && *body.Deadline != "" {
		t, err := time.Parse(time.RFC3339, *body.Deadline)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04", *body.Deadline)
		}
		if err == nil {
			deadline = &t
		}
	}
	if err := s.leagueRepo.UpdateLeague(r.Context(), id, name, deadline); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	league, err := s.leagueRepo.GetByID(r.Context(), id)
	if err != nil || league == nil {
		jsonError(w, "league not found", http.StatusNotFound)
		return
	}
	InvalidateLeagues()
	s.audit(r, &models.AuditEntry{
		Action:     models.AuditLeagueUpdate,
		EntityType: "league",
		EntityID:   &league.ID,
		LeagueID:   &league.ID,
		Metadata:   map[string]any{"name": league.Name},
	})
	jsonOK(w, leagueDTO(league))
}

func (s *Server) handleAdminDeleteLeague(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := s.leagueRepo.ArchiveLeague(r.Context(), id); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, &models.AuditEntry{
		Action:     models.AuditLeagueDelete,
		EntityType: "league",
		EntityID:   &id,
		LeagueID:   &id,
		Metadata:   map[string]any{"mode": "archive"},
	})
	jsonOK(w, map[string]string{"status": "archived"})
}

// handleAdminPurgeLeague — полное удаление из БД (только архивированные, только super_admin).
func (s *Server) handleAdminPurgeLeague(w http.ResponseWriter, r *http.Request) {
	if ok, _ := s.adminRepo.IsSuperAdminByUserID(r.Context(), currentUserID(r)); !ok {
		jsonError(w, "super admin required", http.StatusForbidden)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	league, err := s.leagueRepo.GetByID(r.Context(), id)
	if err != nil || league == nil {
		jsonError(w, "league not found", http.StatusNotFound)
		return
	}
	if string(league.Status) != "archived" {
		jsonError(w, "only archived leagues can be permanently deleted", http.StatusBadRequest)
		return
	}
	if err := s.leagueRepo.DeleteLeague(r.Context(), id); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "deleted"})
}

func (s *Server) handleAdminMembers(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	pending, err := s.leagueRepo.GetPendingMembers(r.Context(), id)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	all, err := s.leagueRepo.GetMembers(r.Context(), id)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	pendingList := make([]map[string]any, 0, len(pending))
	for _, m := range pending {
		pendingList = append(pendingList, memberDTO(m))
	}
	allList := make([]map[string]any, 0, len(all))
	for _, m := range all {
		allList = append(allList, memberDTO(m))
	}
	jsonOK(w, map[string]any{"pending": pendingList, "approved": allList})
}

func (s *Server) handleAdminApprove(w http.ResponseWriter, r *http.Request) {
	leagueID, err1 := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	userID, err2 := strconv.ParseInt(chi.URLParam(r, "uid"), 10, 64)
	if err1 != nil || err2 != nil {
		jsonError(w, "bad params", http.StatusBadRequest)
		return
	}
	league, err := s.leagueRepo.GetByID(r.Context(), leagueID)
	if err != nil || league == nil {
		jsonError(w, "league not found", http.StatusNotFound)
		return
	}
	if league.Status != models.LeagueRegistration {
		jsonError(w, "approvals only allowed in registration status", http.StatusBadRequest)
		return
	}
	if err := s.leagueRepo.ApproveMember(r.Context(), leagueID, userID); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	// Уведомляем игрока что его заявка одобрена
	if league, err := s.leagueRepo.GetByID(r.Context(), leagueID); err == nil && league != nil {
		if user, err := s.userRepo.GetByID(r.Context(), userID); err == nil && user != nil {
			s.notifier.MemberApproved(league.Name, user.TelegramID)
		}
		if s.webPush != nil {
			go s.webPush.Notify([]int64{userID}, "✅ Заявка одобрена",
				"Вас приняли в лигу «"+league.Name+"»", "/leagues")
		}
	}
	s.audit(r, &models.AuditEntry{
		Action:     models.AuditMemberApprove,
		EntityType: "league",
		EntityID:   &leagueID,
		LeagueID:   &leagueID,
		TargetID:   &userID,
	})
	jsonOK(w, map[string]string{"status": "approved"})
}

func (s *Server) handleAdminReject(w http.ResponseWriter, r *http.Request) {
	leagueID, err1 := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	userID, err2 := strconv.ParseInt(chi.URLParam(r, "uid"), 10, 64)
	if err1 != nil || err2 != nil {
		jsonError(w, "bad params", http.StatusBadRequest)
		return
	}
	if err := s.leagueRepo.RejectMember(r.Context(), leagueID, userID); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, &models.AuditEntry{
		Action:     models.AuditMemberReject,
		EntityType: "league",
		EntityID:   &leagueID,
		LeagueID:   &leagueID,
		TargetID:   &userID,
	})
	jsonOK(w, map[string]string{"status": "rejected"})
}

// handleAdminOpenRegistration переводит лигу из draft → registration.
func (s *Server) handleAdminOpenRegistration(w http.ResponseWriter, r *http.Request) {
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
	if league.Status != models.LeagueDraft {
		jsonError(w, "league must be in draft status", http.StatusBadRequest)
		return
	}
	if err := s.leagueRepo.SetLeagueStatus(r.Context(), leagueID, string(models.LeagueRegistration)); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	InvalidateLeagues()
	jsonOK(w, map[string]string{"status": string(models.LeagueRegistration)})
}

func (s *Server) handleAdminDraw(w http.ResponseWriter, r *http.Request) {
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
	if league.Status != models.LeagueRegistration {
		jsonError(w, "draw is only allowed in registration status", http.StatusBadRequest)
		return
	}
	// Считаем реальное число одобренных игроков
	members, err := s.leagueRepo.GetMembers(r.Context(), leagueID)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	n := len(members)
	if n < 2 {
		jsonError(w, "need at least 2 approved players", http.StatusBadRequest)
		return
	}

	// Полностью автоматический расчёт параметров из числа игроков
	cfg := service.Calculate(n, league.RoundsType)

	var genErr error
	switch league.RoundsType {
	case "hybrid", "groups", "groups_playoff":
		genErr = s.groupStageSvc.GenerateGroupStage(r.Context(), leagueID, cfg.NumGroups, cfg.GroupAdvance)
	case "cup":
		genErr = s.cupSvc.GenerateCup(r.Context(), leagueID)
	case "swiss":
		genErr = s.swissSvc.GenerateFirstRound(r.Context(), leagueID)
	case "nations_league":
		genErr = s.nationsLeagueSvc.GenerateNationsLeague(r.Context(), leagueID, cfg.NumDivisions)
	case "double_elim":
		// Жеребьёвка для двойной элиминации = генерация сетки из всех одобренных
		// участников (посев по таблице/рейтингу). Требуется степень двойки.
		if s.deSvc == nil {
			genErr = fmt.Errorf("double elimination not available")
		} else {
			genErr = s.deSvc.Generate(r.Context(), leagueID, n)
		}
	default: // "single" | "double"
		double := league.RoundsType == "double"
		genErr = s.schedSvc.GenerateSchedule(r.Context(), leagueID, double)
	}
	// Сохраняем num_groups и group_advance в leagues для корректного отображения сетки
	if cfg.NumGroups > 0 {
		if err := s.leagueRepo.SetLeagueGroupConfig(r.Context(), leagueID, cfg.NumGroups, cfg.GroupAdvance); err != nil {
			logger.FromContext(r.Context()).Error("SetLeagueGroupConfig failed",
				"league_id", leagueID, "error", err)
		}
	}
	if genErr != nil {
		jsonError(w, genErr.Error(), http.StatusBadRequest)
		return
	}
	if err := s.leagueRepo.SetLeagueStatus(r.Context(), leagueID, string(models.LeagueActive)); err != nil {
		logger.FromContext(r.Context()).Error("SetLeagueStatus failed",
			"league_id", leagueID, "error", err)
	}
	logger.DrawGenerated(r.Context(), leagueID, 0, currentUserID(r))
	InvalidateLeagues()

	// Уведомляем всех участников о жеребьёвке.
	// context.Background() — HTTP-контекст закрывается до завершения горутины.
	leagueName := league.Name
	logger.Go("draw-notify", func() {
		members, err := s.leagueRepo.GetMembers(context.Background(), leagueID)
		if err != nil {
			return
		}
		tgIDs := make([]int64, 0, len(members))
		userIDs := make([]int64, 0, len(members))
		for _, m := range members {
			if m.User != nil && m.User.TelegramID != 0 {
				tgIDs = append(tgIDs, m.User.TelegramID)
			}
			userIDs = append(userIDs, m.UserID)
		}
		s.notifier.DrawGenerated(leagueName, tgIDs)
		if s.webPush != nil {
			s.webPush.Notify(userIDs, "🎲 Расписание готово",
				"В лиге «"+leagueName+"» составлено расписание — проверьте свои матчи", "/leagues")
		}
	})

	s.audit(r, &models.AuditEntry{
		Action:     models.AuditLeagueDraw,
		EntityType: "league",
		EntityID:   &leagueID,
		LeagueID:   &leagueID,
	})
	jsonOK(w, map[string]string{"status": "schedule_generated"})
}

func (s *Server) handleAdminResolve(w http.ResponseWriter, r *http.Request) {
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
	m, err := s.matchSvc.AdminResolve(r.Context(), matchID, body.HomeGoals, body.AwayGoals, currentUserID(r), body.Note)
	if err != nil {
		logger.FromContext(r.Context()).Error("admin resolve failed",
			"match_id", matchID, "admin_id", currentUserID(r), "error", err)
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	logger.AdminResolve(r.Context(), matchID, currentUserID(r), body.HomeGoals, body.AwayGoals)

	homeUser, err := s.userRepo.GetByID(r.Context(), m.HomeUserID)
	if err != nil {
		logger.FromContext(r.Context()).Error("GetByID homeUser", "user_id", m.HomeUserID, "error", err)
	}
	awayUser, err := s.userRepo.GetByID(r.Context(), m.AwayUserID)
	if err != nil {
		logger.FromContext(r.Context()).Error("GetByID awayUser", "user_id", m.AwayUserID, "error", err)
	}
	if homeUser != nil && awayUser != nil {
		s.applyEloUpdate(r.Context(), homeUser, awayUser, body.HomeGoals, body.AwayGoals)
		s.notifier.AdminResolved(
			homeUser.DisplayName, awayUser.DisplayName,
			body.HomeGoals, body.AwayGoals,
			homeUser.TelegramID, awayUser.TelegramID,
		)
	}

	InvalidateStandings(m.LeagueID)
	InvalidatePlayers()
	PublishMatchUpdate(m.LeagueID, m.ID)

	s.audit(r, &models.AuditEntry{
		Action:     models.AuditAdminResolve,
		EntityType: "match",
		EntityID:   &m.ID,
		LeagueID:   &m.LeagueID,
		Metadata: map[string]any{
			"home_goals": body.HomeGoals,
			"away_goals": body.AwayGoals,
			"note":       body.Note,
		},
	})
	jsonOK(w, map[string]string{"status": "resolved"})
}

func (s *Server) handleAdminDisputed(w http.ResponseWriter, r *http.Request) {
	matches, err := s.matchRepo.GetAllDisputed(r.Context())
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	result := make([]map[string]any, 0, len(matches))
	for _, m := range matches {
		result = append(result, matchDTO(m))
	}
	jsonOK(w, result)
}

func (s *Server) handleAdminNextRound(w http.ResponseWriter, r *http.Request) {
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
	if league.RoundsType != "swiss" {
		jsonError(w, "only swiss format supports next-round", http.StatusBadRequest)
		return
	}
	// Считаем реальное число участников для Swiss
	swissMembers, _ := s.leagueRepo.GetMembers(r.Context(), leagueID)
	maxRounds := service.NumRoundsForSwiss(len(swissMembers))
	if err := s.swissSvc.GenerateNextRound(r.Context(), leagueID, maxRounds); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, map[string]string{"status": "next_round_generated"})
}

func (s *Server) handleAdminFinalFour(w http.ResponseWriter, r *http.Request) {
	leagueID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := s.nationsLeagueSvc.GenerateFinalFour(r.Context(), leagueID); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, map[string]string{"status": "final_four_generated"})
}

func (s *Server) handleAdminListAdmins(w http.ResponseWriter, r *http.Request) {
	admins, err := s.adminRepo.List(r.Context())
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	result := make([]map[string]any, 0, len(admins))
	for _, admin := range admins {
		row := map[string]any{
			"id":      admin.ID,
			"user_id": admin.UserID,
			"role":    admin.Role,
		}
		if admin.User != nil {
			row["telegram_id"] = admin.User.TelegramID
			row["display_name"] = admin.User.DisplayName
			if admin.User.Username != nil {
				row["username"] = *admin.User.Username
			}
		}
		result = append(result, row)
	}
	jsonOK(w, result)
}

func (s *Server) handleAdminAdd(w http.ResponseWriter, r *http.Request) {
	if ok, _ := s.adminRepo.IsSuperAdminByUserID(r.Context(), currentUserID(r)); !ok {
		jsonError(w, "super admin required", http.StatusForbidden)
		return
	}

	var body struct {
		TelegramID int64  `json:"telegram_id"`
		UserID     int64  `json:"user_id"`
		Role       string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}

	role := models.AdminRole(body.Role)
	if role == "" {
		role = models.RoleAdmin
	}
	if role != models.RoleAdmin && role != models.RoleSuperAdmin {
		jsonError(w, "bad role", http.StatusBadRequest)
		return
	}

	userID := body.UserID
	if userID == 0 {
		if body.TelegramID == 0 {
			jsonError(w, "user_id or telegram_id required", http.StatusBadRequest)
			return
		}
		user, err := s.userRepo.GetByTelegramID(r.Context(), body.TelegramID)
		if err != nil || user == nil {
			jsonError(w, "user not found", http.StatusNotFound)
			return
		}
		userID = user.ID
	}

	if err := s.adminRepo.Add(r.Context(), userID, role); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	InvalidateAdminStatus(userID)
	globalUserCache.Invalidate(userID)
	jsonOK(w, map[string]string{"status": "added"})
}

func (s *Server) handleAdminRemove(w http.ResponseWriter, r *http.Request) {
	if ok, _ := s.adminRepo.IsSuperAdminByUserID(r.Context(), currentUserID(r)); !ok {
		jsonError(w, "super admin required", http.StatusForbidden)
		return
	}

	userID, err := strconv.ParseInt(chi.URLParam(r, "uid"), 10, 64)
	if err != nil {
		jsonError(w, "bad user id", http.StatusBadRequest)
		return
	}
	if userID == currentUserID(r) {
		jsonError(w, "cannot remove yourself", http.StatusBadRequest)
		return
	}
	if err := s.adminRepo.Remove(r.Context(), userID); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	InvalidateAdminStatus(userID)
	globalUserCache.Invalidate(userID)
	jsonOK(w, map[string]string{"status": "removed"})
}

func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 500 {
		limit = l
	}
	users, err := s.adminRepo.GetUsersWithRoles(r.Context(), limit)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	if users == nil {
		users = []*repository.UserWithRole{}
	}
	jsonOK(w, users)
}

func (s *Server) handleAdminResetRatings(w http.ResponseWriter, r *http.Request) {
	if ok, _ := s.adminRepo.IsSuperAdminByUserID(r.Context(), currentUserID(r)); !ok {
		jsonError(w, "super admin required", http.StatusForbidden)
		return
	}
	if err := s.userRepo.ResetAllRatings(r.Context()); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "ratings_reset"})
}

// ── Round Deadlines ───────────────────────────────────────────────────────────

func (s *Server) handleAdminGetDeadlines(w http.ResponseWriter, r *http.Request) {
	if s.deadlineRepo == nil {
		jsonOK(w, []any{})
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	deadlines, err := s.deadlineRepo.GetDeadlines(r.Context(), id)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	result := make([]map[string]any, 0, len(deadlines))
	for _, d := range deadlines {
		result = append(result, map[string]any{
			"id":                d.ID,
			"league_id":         d.LeagueID,
			"round":             d.Round,
			"deadline":          d.Deadline.UTC().Format(time.RFC3339),
			"reminder_24h_sent": d.Reminder24hSent,
			"reminder_1h_sent":  d.Reminder1hSent,
		})
	}
	jsonOK(w, result)
}

func (s *Server) handleAdminSetDeadline(w http.ResponseWriter, r *http.Request) {
	if s.deadlineRepo == nil {
		jsonError(w, "not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	var body struct {
		Round    int    `json:"round"`
		Deadline string `json:"deadline"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Round <= 0 || body.Deadline == "" {
		jsonError(w, "invalid body: round and deadline required", http.StatusBadRequest)
		return
	}
	deadline, err := time.Parse(time.RFC3339, body.Deadline)
	if err != nil {
		jsonError(w, "invalid deadline format (use RFC3339)", http.StatusBadRequest)
		return
	}
	if err := s.deadlineRepo.SetDeadline(r.Context(), id, body.Round, deadline); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "ok"})
}

func (s *Server) handleAdminDeleteDeadline(w http.ResponseWriter, r *http.Request) {
	if s.deadlineRepo == nil {
		jsonError(w, "not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	round, err := strconv.Atoi(chi.URLParam(r, "round"))
	if err != nil {
		jsonError(w, "bad round", http.StatusBadRequest)
		return
	}
	if err := s.deadlineRepo.DeleteDeadline(r.Context(), id, round); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "deleted"})
}

// ── Finalize League ───────────────────────────────────────────────────────────

func (s *Server) handleAdminFinalize(w http.ResponseWriter, r *http.Request) {
	if s.awardSvc == nil {
		jsonError(w, "not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := s.awardSvc.FinalizeLeague(r.Context(), id); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, map[string]string{"status": "finalized"})
}

// handleAdminBroadcast рассылает произвольное сообщение администратора всем
// игрокам: web push (подписанным) + Telegram (привязанным).
func (s *Server) handleAdminBroadcast(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	body.Text = strings.TrimSpace(body.Text)
	if len(body.Text) < 1 || len(body.Text) > 1000 {
		jsonError(w, "text must be 1-1000 characters", http.StatusBadRequest)
		return
	}
	title := strings.TrimSpace(body.Title)
	if title == "" {
		title = "📢 eFootLeague"
	}

	// Web push всем подписанным
	pushed := 0
	if s.webPush != nil {
		pushed = s.webPush.Broadcast(title, body.Text, "/")
	}

	// Telegram всем привязанным
	tg := 0
	if s.notifier != nil {
		if ids, err := s.userRepo.GetAllTelegramIDs(r.Context()); err == nil {
			s.notifier.BroadcastCustom(body.Text, ids)
			tg = len(ids)
		}
	}

	s.audit(r, &models.AuditEntry{
		Action:   models.AuditBroadcast,
		Metadata: map[string]any{"pushed": pushed, "telegram": tg, "title": title},
	})
	jsonOK(w, map[string]any{"pushed": pushed, "telegram": tg})
}

// handleAdminNotifyUser шлёт уведомление одному конкретному игроку
// (web push + Telegram, если привязан).
func (s *Server) handleAdminNotifyUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID int64  `json:"user_id"`
		Title  string `json:"title"`
		Text   string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == 0 {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	body.Text = strings.TrimSpace(body.Text)
	if len(body.Text) < 1 || len(body.Text) > 1000 {
		jsonError(w, "text must be 1-1000 characters", http.StatusBadRequest)
		return
	}
	title := strings.TrimSpace(body.Title)
	if title == "" {
		title = "📢 eFootLeague"
	}

	user, err := s.userRepo.GetByID(r.Context(), body.UserID)
	if err != nil || user == nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	pushed := 0
	if s.webPush != nil {
		// Notify не возвращает счётчик; считаем «отправлено», если есть подписки.
		s.webPush.Notify([]int64{body.UserID}, title, body.Text, "/")
		pushed = 1
	}
	tg := 0
	if s.notifier != nil && user.TelegramID != 0 {
		s.notifier.BroadcastCustom(body.Text, []int64{user.TelegramID})
		tg = 1
	}

	jsonOK(w, map[string]any{"pushed": pushed, "telegram": tg, "name": user.DisplayName})
}
