package api

import (
	"context"
	"efootball-bot/internal/logger"
	"efootball-bot/internal/models"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// deleteUserAndRecalc удаляет пользователя и пересчитывает таблицы затронутых
// лиг + сбрасывает кэши.
func (s *Server) deleteUserAndRecalc(ctx context.Context, userID int64) error {
	leagues, err := s.userRepo.DeleteUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, lid := range leagues {
		if err := s.leagueRepo.RecalculateTable(ctx, lid); err != nil {
			logger.FromContext(ctx).Error("recalc table after user delete", "league_id", lid, "error", err)
		}
		InvalidateStandings(lid)
	}
	globalUserCache.Invalidate(userID)
	InvalidatePlayers()
	InvalidateLeagues()
	return nil
}

// handleDeleteMe — DELETE /api/me. Игрок удаляет свой аккаунт полностью.
func (s *Server) handleDeleteMe(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	if err := s.deleteUserAndRecalc(r.Context(), uid); err != nil {
		jsonErrorLog(w, r, "failed to delete account", http.StatusInternalServerError, err)
		return
	}
	jsonOK(w, map[string]string{"status": "deleted"})
}

// handleAdminDeleteUser — DELETE /api/admin/users/{uid}. Админ удаляет игрока.
func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	uid, err := strconv.ParseInt(chi.URLParam(r, "uid"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return
	}
	// Супер-админа удалить нельзя (защита от блокировки управления).
	if role, _ := s.adminRepo.GetAdminRoleByUserID(r.Context(), uid); role == "super_admin" {
		jsonError(w, "cannot delete super admin", http.StatusForbidden)
		return
	}
	if err := s.deleteUserAndRecalc(r.Context(), uid); err != nil {
		jsonErrorLog(w, r, "failed to delete user", http.StatusInternalServerError, err)
		return
	}
	s.audit(r, &models.AuditEntry{
		Action:     models.AuditUserDelete,
		EntityType: "user",
		EntityID:   &uid,
		TargetID:   &uid,
	})
	jsonOK(w, map[string]string{"status": "deleted"})
}
