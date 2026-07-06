package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strconv"

	"efootball-bot/internal/i18n"
	"efootball-bot/internal/logger"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"

	"github.com/go-chi/chi/v5"
)

// Товарищеские матчи: вызов → принятие → игра в eFootball → счёт → подтверждение
// соперником → пересчёт ELO. Без турнира, влияет на рейтинг профиля.

func (s *Server) SetFriendlyRepo(fr repository.FriendlyRepository) { s.friendlyRepo = fr }

// friendlyNotify — уведомление участнику товарищеского матча на его языке по
// всем каналам: колокольчик+SSE всегда; web-push — только если вкладка не на
// переднем плане (иначе внутри приложения уже есть звук и тост); Telegram —
// только если приложение вообще не открыто (человек «не в приложении»).
func (s *Server) friendlyNotify(ctx context.Context, userID int64, notifType, pushKind, key string, args ...any) {
	lang := i18n.LangRu
	var tgID int64
	if u, err := s.userRepo.GetByID(ctx, userID); err == nil && u != nil {
		lang, tgID = u.Language, u.TelegramID
	}
	title := i18n.T(lang, key+".title")
	body := i18n.T(lang, key+".body")
	if len(args) > 0 {
		body = fmt.Sprintf(body, args...)
	}
	s.notify(ctx, []int64{userID}, notifType, title, body, "/friendlies")
	if s.webPush != nil && !isAppVisible(userID) {
		go s.webPush.NotifyKind([]int64{userID}, pushKind, title, body, "/friendlies")
	}
	if s.notifier != nil && tgID != 0 && !presence.isOnline(userID) {
		go s.notifier.send(tgID, "<b>"+html.EscapeString(title)+"</b>\n"+html.EscapeString(body))
	}
}

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
		s.friendlyNotify(ctx, ref.ChallengerID, models.NotifFriendly, "system", "friendly.expired")
		s.friendlyNotify(ctx, ref.OpponentID, models.NotifFriendly, "system", "friendly.expired")
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
	s.friendlyNotify(r.Context(), f.OpponentID, models.NotifFriendly, "challenge", "friendly.challenge", f.ChallengerName)
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
			s.friendlyNotify(r.Context(), f.ChallengerID, models.NotifFriendly, "challenge", "friendly.accepted", f.OpponentName)
		} else {
			s.friendlyNotify(r.Context(), f.ChallengerID, models.NotifFriendly, "system", "friendly.declined", f.OpponentName)
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
	s.friendlyNotify(r.Context(), other, models.NotifFriendlyResult, "result", "friendly.score")
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

	s.friendlyNotify(r.Context(), f.ChallengerID, models.NotifFriendlyResult, "result", "friendly.confirmed")
	s.friendlyNotify(r.Context(), f.OpponentID, models.NotifFriendlyResult, "result", "friendly.confirmed")
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
	s.friendlyNotify(r.Context(), *f.ClaimedBy, models.NotifFriendlyResult, "result", "friendly.disputed")
	jsonOK(w, map[string]any{"ok": true})
}
