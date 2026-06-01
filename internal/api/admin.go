package api

import (
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"encoding/json"
	"net/http"
	"strconv"

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
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		jsonError(w, "name required", http.StatusBadRequest)
		return
	}
	season, err := s.leagueRepo.GetOrCreateActiveSeason(r.Context())
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	league, err := s.leagueRepo.CreateLeague(r.Context(), season.ID, body.Name)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
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
	jsonOK(w, map[string]string{"status": "archived"})
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
	if err := s.leagueRepo.ApproveMember(r.Context(), leagueID, userID); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
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
	jsonOK(w, map[string]string{"status": "rejected"})
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
	double := league.RoundsType == "double"
	if err := s.schedSvc.GenerateSchedule(r.Context(), leagueID, double); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = s.leagueRepo.SetLeagueStatus(r.Context(), leagueID, string(models.LeagueActive))
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
	if err := s.matchRepo.AdminResolve(r.Context(), matchID, body.HomeGoals, body.AwayGoals, currentUserID(r), body.Note); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	m, _ := s.matchRepo.GetByID(r.Context(), matchID)
	if m != nil {
		_ = s.leagueRepo.ApplyMatchResultStats(r.Context(), m.LeagueID, m.HomeUserID, m.AwayUserID, body.HomeGoals, body.AwayGoals)
		_ = s.leagueRepo.RecalculateTable(r.Context(), m.LeagueID)
		homeUser, _ := s.userRepo.GetByID(r.Context(), m.HomeUserID)
		awayUser, _ := s.userRepo.GetByID(r.Context(), m.AwayUserID)
		if homeUser != nil && awayUser != nil {
			newHome, newAway := s.eloSvc.Calculate(
				r.Context(),
				homeUser.ID, awayUser.ID,
				homeUser.Rating, awayUser.Rating,
				homeUser.TeamPower, awayUser.TeamPower,
				body.HomeGoals, body.AwayGoals,
			)
			_ = s.userRepo.UpdateRating(r.Context(), homeUser.ID, newHome)
			_ = s.userRepo.UpdateRating(r.Context(), awayUser.ID, newAway)
			_ = s.userRepo.RecalculateAllRanks(r.Context())
		}
	}
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
	jsonOK(w, map[string]string{"status": "removed"})
}

func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.adminRepo.GetUsersWithRoles(r.Context(), 500)
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
