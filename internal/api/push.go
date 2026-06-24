package api

import (
	"context"
	"efootball-bot/internal/logger"
	"efootball-bot/internal/repository"
	"encoding/json"
	"net/http"
	"strconv"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// itoa16 форматирует int16 (счёт матча) в строку.
func itoa16(v int16) string { return strconv.Itoa(int(v)) }

// WebPushNotifier рассылает web-push уведомления подписанным браузерам.
type WebPushNotifier struct {
	repo    repository.PushRepository
	public  string
	private string
	subject string
}

func NewWebPushNotifier(repo repository.PushRepository, public, private, subject string) *WebPushNotifier {
	return &WebPushNotifier{repo: repo, public: public, private: private, subject: subject}
}

func (n *WebPushNotifier) enabled() bool {
	return n != nil && n.public != "" && n.private != "" && n.repo != nil
}

// Enabled сообщает, настроен ли web push (есть VAPID-ключи).
func (n *WebPushNotifier) Enabled() bool { return n.enabled() }

type pushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

// Notify шлёт уведомление всем устройствам указанных пользователей.
// Безопасно вызывать в горутине — ошибки логируются, мёртвые подписки удаляются.
func (n *WebPushNotifier) Notify(userIDs []int64, title, body, url string) {
	if !n.enabled() || len(userIDs) == 0 {
		return
	}
	ctx := context.Background()
	subs, err := n.repo.GetByUserIDs(ctx, userIDs)
	if err != nil {
		logger.FromContext(ctx).Warn("push: get subs failed", "error", err)
		return
	}
	if len(subs) == 0 {
		return
	}
	payload, _ := json.Marshal(pushPayload{Title: title, Body: body, URL: url})

	for _, s := range subs {
		sub := &webpush.Subscription{
			Endpoint: s.Endpoint,
			Keys:     webpush.Keys{P256dh: s.P256dh, Auth: s.Auth},
		}
		resp, err := webpush.SendNotification(payload, sub, &webpush.Options{
			Subscriber:      n.subject,
			VAPIDPublicKey:  n.public,
			VAPIDPrivateKey: n.private,
			TTL:             86400,
		})
		if err != nil {
			logger.FromContext(ctx).Warn("push send failed", "error", err)
			continue
		}
		// 404/410 — подписка мертва, удаляем
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
			_ = n.repo.DeleteByEndpoint(ctx, s.Endpoint)
		}
		resp.Body.Close()
	}
}

// ── HTTP handlers ─────────────────────────────────────────────────────────────

// handleVapidPublic отдаёт публичный VAPID-ключ фронтенду (для подписки).
func (s *Server) handleVapidPublic(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]string{"key": s.cfg.API.VAPIDPublic})
}

type subscribeRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func (s *Server) handlePushSubscribe(w http.ResponseWriter, r *http.Request) {
	if s.pushRepo == nil {
		jsonError(w, "push not configured", http.StatusServiceUnavailable)
		return
	}
	var req subscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Endpoint == "" || req.Keys.P256dh == "" || req.Keys.Auth == "" {
		jsonError(w, "invalid subscription", http.StatusBadRequest)
		return
	}
	err := s.pushRepo.Save(r.Context(), repository.PushSubscription{
		UserID:   currentUserID(r),
		Endpoint: req.Endpoint,
		P256dh:   req.Keys.P256dh,
		Auth:     req.Keys.Auth,
	})
	if err != nil {
		jsonError(w, "failed to save", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]bool{"ok": true})
}

type unsubscribeRequest struct {
	Endpoint string `json:"endpoint"`
}

func (s *Server) handlePushUnsubscribe(w http.ResponseWriter, r *http.Request) {
	if s.pushRepo == nil {
		jsonOK(w, map[string]bool{"ok": true})
		return
	}
	var req unsubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Endpoint == "" {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	_ = s.pushRepo.DeleteByEndpoint(r.Context(), req.Endpoint)
	jsonOK(w, map[string]bool{"ok": true})
}

// handlePushTest шлёт тестовое уведомление текущему пользователю.
func (s *Server) handlePushTest(w http.ResponseWriter, r *http.Request) {
	if s.webPush == nil || !s.webPush.Enabled() {
		jsonError(w, "push not configured on server", http.StatusServiceUnavailable)
		return
	}
	// Синхронно, чтобы вернуть реальный результат пользователю.
	s.webPush.Notify([]int64{currentUserID(r)}, "eFootLeague",
		"🔔 Уведомления работают! Вы будете получать важные события.", "/")
	jsonOK(w, map[string]bool{"ok": true})
}
