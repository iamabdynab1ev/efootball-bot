package api

import (
	"efootball-bot/internal/models"
	"sync"
	"time"
)

// ── Generic TTL cache ─────────────────────────────────────────────────────────

type ttlEntry[V any] struct {
	value  V
	expiry time.Time
}

type ttlCache[K comparable, V any] struct {
	mu  sync.RWMutex
	m   map[K]ttlEntry[V]
	ttl time.Duration
}

func newTTLCache[K comparable, V any](ttl time.Duration) *ttlCache[K, V] {
	return &ttlCache[K, V]{m: map[K]ttlEntry[V]{}, ttl: ttl}
}

func (c *ttlCache[K, V]) get(key K) (V, bool) {
	c.mu.RLock()
	e, ok := c.m[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expiry) {
		var zero V
		return zero, false
	}
	return e.value, true
}

func (c *ttlCache[K, V]) set(key K, val V) {
	c.mu.Lock()
	c.m[key] = ttlEntry[V]{value: val, expiry: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

func (c *ttlCache[K, V]) Invalidate(key K) {
	c.mu.Lock()
	delete(c.m, key)
	c.mu.Unlock()
}

func (c *ttlCache[K, V]) flush() {
	c.mu.Lock()
	c.m = map[K]ttlEntry[V]{}
	c.mu.Unlock()
}

// ── User cache (key = userID int64) ──────────────────────────────────────────

type userCache struct{ *ttlCache[int64, *models.User] }

func newUserCache(ttl time.Duration) *userCache {
	return &userCache{newTTLCache[int64, *models.User](ttl)}
}

func (c *userCache) set(u *models.User) { c.ttlCache.set(u.ID, u) }

// ── Admin-status cache (key = userID, value = isAdmin bool) ──────────────────

type adminCache struct{ *ttlCache[int64, bool] }

func newAdminCache(ttl time.Duration) *adminCache {
	return &adminCache{newTTLCache[int64, bool](ttl)}
}

// ── Package-level cache instances ─────────────────────────────────────────────

var (
	globalUserCache  = newUserCache(60 * time.Second)
	adminStatusCache = newAdminCache(5 * time.Minute)
	leaguesCache     = newTTLCache[string, any](30 * time.Second)
	standingsCache   = newTTLCache[int64, any](30 * time.Second)
	playersCache     = newTTLCache[int, any](60 * time.Second)
)

// ── Invalidation helpers ──────────────────────────────────────────────────────

func InvalidateLeagues() {
	leaguesCache.flush()
}

func InvalidateStandings(leagueID int64) {
	standingsCache.Invalidate(leagueID)
}

func InvalidatePlayers() {
	playersCache.flush()
}

func InvalidateAdminStatus(userID int64) {
	adminStatusCache.Invalidate(userID)
}

// CleanupAllCaches drops all cache entries (called periodically from main).
func CleanupAllCaches() {
	globalUserCache.flush()
	adminStatusCache.flush()
	leaguesCache.flush()
	standingsCache.flush()
	playersCache.flush()
}
