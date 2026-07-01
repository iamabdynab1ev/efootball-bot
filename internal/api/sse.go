package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ── SSE hub ───────────────────────────────────────────────────────────────────
//
// Один общий канал реального времени для всего проекта. Клиент держит одно
// SSE-соединение (/api/events) и получает события по топикам, на которые он
// подписан исходя из своей личности:
//
//   public        — публичные события (обновления матчей/таблиц), все клиенты;
//   user:{id}     — личные события (нотификации, presence), только сам юзер;
//   admins        — админские живые ленты (аудит, presence-обзор), только админы;
//   room:{id}     — комнаты чата (групповой чат), только участники.
//
// Доставка адресная через индекс topic→clients (без обхода всех соединений),
// что держит нагрузку линейной по числу получателей, а не по числу клиентов.

const (
	topicPublic = "public"
	topicAdmins = "admins"
)

func topicUser(userID int64) string { return "user:" + strconv.FormatInt(userID, 10) }
func topicRoom(roomID int64) string { return "room:" + strconv.FormatInt(roomID, 10) }

type sseClient struct {
	ch     chan string
	userID int64 // 0 = анонимный
	topics map[string]struct{}
}

type sseHub struct {
	mu      sync.RWMutex
	byTopic map[string]map[*sseClient]struct{}
}

var hub = &sseHub{byTopic: map[string]map[*sseClient]struct{}{}}

func (h *sseHub) add(c *sseClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for t := range c.topics {
		m := h.byTopic[t]
		if m == nil {
			m = map[*sseClient]struct{}{}
			h.byTopic[t] = m
		}
		m[c] = struct{}{}
	}
}

func (h *sseHub) remove(c *sseClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for t := range c.topics {
		if m := h.byTopic[t]; m != nil {
			delete(m, c)
			if len(m) == 0 {
				delete(h.byTopic, t)
			}
		}
	}
}

// publish рассылает уже сериализованное SSE-сообщение подписчикам топика.
// Медленным клиентам (полный буфер) сообщение пропускается — целостность
// обеспечивается персистентностью и дозагрузкой при реконнекте, а не живым
// каналом, поэтому один отстающий клиент не блокирует остальных.
func (h *sseHub) publish(topic, msg string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.byTopic[topic] {
		select {
		case c.ch <- msg:
		default:
		}
	}
}

// hasSubscribers сообщает, слушает ли кто-то топик — позволяет не сериализовать
// событие впустую (напр. живая лента аудита без открытой админ-вкладки).
func (h *sseHub) hasSubscribers(topic string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.byTopic[topic]) > 0
}

// encodeEvent формирует именованное SSE-событие: фронт слушает его через
// addEventListener(type), что позволяет мультиплексировать разные фичи
// (match_update, audit, notification, presence, chat) по одному соединению.
func encodeEvent(eventType string, data any) string {
	payload, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, payload)
}

// publishEvent — общий помощник: сериализует один раз и рассылает в топик.
func publishEvent(topic, eventType string, data any) {
	if !hub.hasSubscribers(topic) {
		return
	}
	if msg := encodeEvent(eventType, data); msg != "" {
		hub.publish(topic, msg)
	}
}

// PublishMatchUpdate broadcasts a match-update event to all connected clients.
// Результаты матчей и таблицы публичны, поэтому событие идёт в топик public;
// matchID позволяет фронтенду подсветить именно изменившийся матч (живое табло).
func PublishMatchUpdate(leagueID, matchID int64) {
	publishEvent(topicPublic, "match_update", map[string]int64{
		"league_id": leagueID,
		"match_id":  matchID,
	})
}

// ── handler ───────────────────────────────────────────────────────────────────

// handleSSE — GET /api/events[?token=JWT] — subscribes the caller to events.
// Токен передаётся query-параметром, т.к. EventSource не умеет слать заголовки;
// по нему клиент подписывается на личный и (для админов) админский топики.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	topics := map[string]struct{}{topicPublic: {}}
	userID := s.userIDFromToken(r.URL.Query().Get("token"))
	if userID != 0 {
		topics[topicUser(userID)] = struct{}{}
		if s.isAdminCached(r, userID) {
			topics[topicAdmins] = struct{}{}
		}
	}

	client := &sseClient{ch: make(chan string, 16), userID: userID, topics: topics}
	hub.add(client)
	defer hub.remove(client)

	// Онлайн-статус: метим пользователя онлайн на время жизни соединения.
	untrack := trackPresence(userID)
	defer untrack()

	// «Был(а) в сети»: фиксируем активность при входе и при выходе (для тех, кто
	// сейчас офлайн — покажем время последнего соединения).
	if userID != 0 && s.userRepo != nil {
		_ = s.userRepo.TouchLastSeen(r.Context(), userID)
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = s.userRepo.TouchLastSeen(ctx, userID)
		}()
	}

	// Initial heartbeat
	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	// Пинг каждые 25с держит соединение живым через прокси (idle-timeout) и
	// позволяет рантайму заметить мёртвого клиента (ошибка записи → выход).
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()

	for {
		select {
		case msg := <-client.ch:
			if _, err := fmt.Fprint(w, msg); err != nil {
				return
			}
			flusher.Flush()
		case <-ping.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// userIDFromToken валидирует JWT (как authMiddleware) и возвращает user_id, либо
// 0 для анонимного/невалидного — SSE для анонимов остаётся публичным потоком.
func (s *Server) userIDFromToken(tokenStr string) int64 {
	if tokenStr == "" {
		return 0
	}
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.cfg.API.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return 0
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0
	}
	idFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0
	}
	return int64(idFloat)
}

// isAdminCached — admin-проверка через тот же 5-мин кэш, что и adminMiddleware.
func (s *Server) isAdminCached(r *http.Request, userID int64) bool {
	if isAdmin, cached := adminStatusCache.get(userID); cached {
		return isAdmin
	}
	isAdmin, err := s.adminRepo.IsAdminByUserID(r.Context(), userID)
	if err != nil {
		return false
	}
	adminStatusCache.set(userID, isAdmin)
	return isAdmin
}
