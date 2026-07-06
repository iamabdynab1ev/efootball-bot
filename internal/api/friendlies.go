package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"efootball-bot/internal/logger"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"

	"github.com/go-chi/chi/v5"
)

// Товарищеские матчи: вызов → принятие → игра в eFootball → счёт → подтверждение
// соперником → пересчёт ELO. Без турнира, влияет на рейтинг профиля.

func (s *Server) SetFriendlyRepo(fr repository.FriendlyRepository) { s.friendlyRepo = fr }

// ExpireStaleFriendlies — периодическая очистка зависших матчей (вызов без
// ответа, несыгранный матч, неподтверждённый счёт). Освобождает пару для
// нового вызова и предупреждает обоих участников.
func (s *Server) ExpireStaleFriendlies(ctx context.Context) error {
	if s.friendlyRepo == nil {
		return nil
	}
	expired, err := s.friendlyRepo.ExpireStale(ctx)
	if err != nil {
		return err
	}
	for _, ref := range expired {
		s.notify(ctx, []int64{ref.ChallengerID, ref.OpponentID}, models.NotifFriendly,
			"⌛ Товарищеский матч истёк",
			"Матч не был завершён вовремя и закрыт — можно бросить вызов заново", "/friendlies")
	}
	return nil
}

func (s *Server) friendlyByParticipant(w http.ResponseWriter, r *http.Request) (*models.Friendly, int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "bad id", http.StatusBadRequest)
		return nil, 0, false
	}
	f, err := s.friendlyRepo.Get(r.Context(), id)
	if err != nil {
		jsonError(w, "матч не найден", http.StatusNotFound)
		return nil, 0, false
	}
	uid := currentUserID(r)
	if uid != f.ChallengerID && uid != f.OpponentID {
		jsonError(w, "это не ваш матч", http.StatusForbidden)
		return nil, 0, false
	}
	return f, uid, true
}

// handleCreateFriendly — POST /api/friendlies {opponent_id}: вызов на матч.
func (s *Server) handleCreateFriendly(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OpponentID int64 `json:"opponent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OpponentID <= 0 {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	uid := currentUserID(r)
	if req.OpponentID == uid {
		jsonError(w, "нельзя вызвать самого себя", http.StatusBadRequest)
		return
	}
	f, err := s.friendlyRepo.Create(r.Context(), uid, req.OpponentID)
	if err != nil {
		if errors.Is(err, repository.ErrFriendlyActiveExists) {
			jsonError(w, err.Error(), http.StatusConflict)
			return
		}
		jsonErrorLog(w, r, "db error", http.StatusInternalServerError, err)
		return
	}
	s.notify(r.Context(), []int64{f.OpponentID}, models.NotifFriendly,
		"⚔️ Вызов на товарищеский матч",
		f.ChallengerName+" вызывает вас на матч в eFootball", "/friendlies")
	if s.webPush != nil {
		go s.webPush.Notify([]int64{f.OpponentID}, "⚔️ Вызов на матч", f.ChallengerName+" ждёт вас в eFootball", "/friendlies")
	}
	jsonOK(w, f)
}

// handleListFriendlies — GET /api/friendlies: мои вызовы и матчи.
func (s *Server) handleListFriendlies(w http.ResponseWriter, r *http.Request) {
	list, err := s.friendlyRepo.ListForUser(r.Context(), currentUserID(r), 50)
	if err != nil {
		jsonErrorLog(w, r, "db error", http.StatusInternalServerError, err)
		return
	}
	if list == nil {
		list = []*models.Friendly{}
	}
	jsonOK(w, map[string]any{"friendlies": list})
}

// handleFriendlyRespond — POST /api/friendlies/{id}/accept|decline|cancel.
func (s *Server) handleFriendlyRespond(w http.ResponseWriter, r *http.Request, action string) {
	f, uid, ok := s.friendlyByParticipant(w, r)
	if !ok {
		return
	}
	switch action {
	case "accept", "decline":
		if uid != f.OpponentID {
			jsonError(w, "принять или отклонить может только приглашённый", http.StatusForbidden)
			return
		}
		to := map[string]string{"accept": "accepted", "decline": "declined"}[action]
		changed, err := s.friendlyRepo.SetStatus(r.Context(), f.ID, "pending", to)
		if err != nil || !changed {
			jsonError(w, "вызов уже неактуален", http.StatusConflict)
			return
		}
		if action == "accept" {
			s.notify(r.Context(), []int64{f.ChallengerID}, models.NotifFriendly,
				"✅ Вызов принят", f.OpponentName+" принял вызов — играйте и внесите счёт", "/friendlies")
			if s.webPush != nil {
				go s.webPush.Notify([]int64{f.ChallengerID}, "✅ Вызов принят", f.OpponentName+" готов играть", "/friendlies")
			}
		} else {
			s.notify(r.Context(), []int64{f.ChallengerID}, models.NotifFriendly,
				"Вызов отклонён", f.OpponentName+" отклонил товарищеский матч", "/friendlies")
		}
	case "cancel":
		if uid != f.ChallengerID {
			jsonError(w, "отменить может только автор вызова", http.StatusForbidden)
			return
		}
		if changed, err := s.friendlyRepo.SetStatus(r.Context(), f.ID, "pending", "cancelled"); err != nil || !changed {
			jsonError(w, "отменить можно только неотвеченный вызов", http.StatusConflict)
			return
		}
	}
	jsonOK(w, map[string]any{"ok": true})
}

// handleFriendlyScore — POST /api/friendlies/{id}/score {my_goals, opp_goals}.
func (s *Server) handleFriendlyScore(w http.ResponseWriter, r *http.Request) {
	f, uid, ok := s.friendlyByParticipant(w, r)
	if !ok {
		return
	}
	var req struct {
		MyGoals  int16 `json:"my_goals"`
		OppGoals int16 `json:"opp_goals"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil ||
		req.MyGoals < 0 || req.MyGoals > 50 || req.OppGoals < 0 || req.OppGoals > 50 {
		jsonError(w, "неверный счёт", http.StatusBadRequest)
		return
	}
	// Нормализуем к паре challenger/opponent.
	cg, og := req.MyGoals, req.OppGoals
	if uid == f.OpponentID {
		cg, og = req.OppGoals, req.MyGoals
	}
	changed, err := s.friendlyRepo.ClaimScore(r.Context(), f.ID, uid, cg, og)
	if err != nil || !changed {
		jsonError(w, "счёт можно внести только после принятия вызова", http.StatusConflict)
		return
	}
	other := f.ChallengerID
	if uid == f.ChallengerID {
		other = f.OpponentID
	}
	s.notify(r.Context(), []int64{other}, models.NotifFriendly,
		"⚽ Счёт товарищеского матча",
		"Внесён счёт — подтвердите результат", "/friendlies")
	if s.webPush != nil {
		go s.webPush.Notify([]int64{other}, "⚽ Подтвердите счёт", "Соперник внёс результат товарищеского матча", "/friendlies")
	}
	jsonOK(w, map[string]any{"ok": true})
}

// handleFriendlyConfirm — POST /api/friendlies/{id}/confirm: подтверждает
// сторона, НЕ вносившая счёт; после — пересчёт ELO обоих.
func (s *Server) handleFriendlyConfirm(w http.ResponseWriter, r *http.Request) {
	f, uid, ok := s.friendlyByParticipant(w, r)
	if !ok {
		return
	}
	if f.Status != "score_claimed" || f.ClaimedBy == nil || *f.ClaimedBy == uid {
		jsonError(w, "подтвердить должен соперник, после внесения счёта", http.StatusConflict)
		return
	}
	changed, err := s.friendlyRepo.Confirm(r.Context(), f.ID)
	if err != nil || !changed {
		jsonError(w, "результат уже подтверждён", http.StatusConflict)
		return
	}

	// ELO: challenger = «домашний».
	if f.ChallengerGoals != nil && f.OpponentGoals != nil {
		home, err1 := s.userRepo.GetByID(r.Context(), f.ChallengerID)
		away, err2 := s.userRepo.GetByID(r.Context(), f.OpponentID)
		if err1 == nil && err2 == nil && home != nil && away != nil {
			s.applyEloUpdate(r.Context(), home, away, *f.ChallengerGoals, *f.OpponentGoals)
		} else {
			logger.FromContext(r.Context()).Error("friendly elo users", "id", f.ID, "err1", err1, "err2", err2)
		}
	}

	both := []int64{f.ChallengerID, f.OpponentID}
	s.notify(r.Context(), both, models.NotifFriendly,
		"🤝 Товарищеский матч завершён",
		"Результат подтверждён — рейтинг обновлён", "/friendlies")
	jsonOK(w, map[string]any{"ok": true})
}

// handleFriendlyRejectScore — POST /api/friendlies/{id}/reject-score: спорный
// счёт возвращается в accepted — вносите заново.
func (s *Server) handleFriendlyRejectScore(w http.ResponseWriter, r *http.Request) {
	f, uid, ok := s.friendlyByParticipant(w, r)
	if !ok {
		return
	}
	if f.Status != "score_claimed" || f.ClaimedBy == nil || *f.ClaimedBy == uid {
		jsonError(w, "оспорить может только соперник внёсшего счёт", http.StatusConflict)
		return
	}
	if changed, err := s.friendlyRepo.SetStatus(r.Context(), f.ID, "score_claimed", "accepted"); err != nil || !changed {
		jsonError(w, "статус уже изменился", http.StatusConflict)
		return
	}
	other := *f.ClaimedBy
	s.notify(r.Context(), []int64{other}, models.NotifFriendly,
		"❌ Счёт оспорен", "Соперник не согласен со счётом — внесите заново", "/friendlies")
	jsonOK(w, map[string]any{"ok": true})
}
