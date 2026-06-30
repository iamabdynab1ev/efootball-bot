package api

import (
	"net/http"
	"sync"
)

// presenceTracker — кто сейчас онлайн. Пользователь онлайн, пока держит хотя бы
// одно аутентифицированное SSE-соединение (несколько вкладок/устройств считаются
// по refcount). Состояние in-memory: деплой одно-инстансовый (Render,
// WEB_CONCURRENCY=1), поэтому распределённое хранилище не нужно; при горизонтальном
// масштабировании сюда подключится общий стор (Redis pub/sub).
type presenceTracker struct {
	mu    sync.Mutex
	conns map[int64]int
}

var presence = &presenceTracker{conns: map[int64]int{}}

// connect регистрирует соединение и возвращает true, если пользователь только
// что стал онлайн (0 → 1) — повод разослать presence-дельту.
func (p *presenceTracker) connect(userID int64) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.conns[userID]++
	return p.conns[userID] == 1
}

// disconnect снимает соединение и возвращает true, если пользователь стал офлайн
// (1 → 0).
func (p *presenceTracker) disconnect(userID int64) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conns[userID] <= 1 {
		delete(p.conns, userID)
		return true
	}
	p.conns[userID]--
	return false
}

func (p *presenceTracker) online() []int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	ids := make([]int64, 0, len(p.conns))
	for id := range p.conns {
		ids = append(ids, id)
	}
	return ids
}

// trackPresence вызывается из handleSSE для аутентифицированного клиента: метит
// онлайн при входе и возвращает функцию снятия (для defer), рассылая дельты в
// public-топик. Для анонимов (userID == 0) — no-op.
func trackPresence(userID int64) func() {
	if userID == 0 {
		return func() {}
	}
	if presence.connect(userID) {
		publishEvent(topicPublic, "presence", map[string]any{"user_id": userID, "online": true})
	}
	return func() {
		if presence.disconnect(userID) {
			publishEvent(topicPublic, "presence", map[string]any{"user_id": userID, "online": false})
		}
	}
}

// handleOnline — GET /api/online — текущий снимок онлайн-пользователей (id),
// чтобы фронт получил стартовое состояние, а дальше слушал presence-дельты.
func (s *Server) handleOnline(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]any{"online": presence.online()})
}
