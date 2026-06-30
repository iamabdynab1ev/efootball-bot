package api

import (
	"strings"
	"testing"
)

func newTestClient(topics ...string) *sseClient {
	set := map[string]struct{}{}
	for _, t := range topics {
		set[t] = struct{}{}
	}
	return &sseClient{ch: make(chan string, 8), topics: set}
}

func recv(c *sseClient) string {
	select {
	case m := <-c.ch:
		return m
	default:
		return ""
	}
}

func TestEncodeEventFormat(t *testing.T) {
	msg := encodeEvent("audit", map[string]any{"id": 7})
	if !strings.HasPrefix(msg, "event: audit\n") {
		t.Fatalf("нет именованного события: %q", msg)
	}
	if !strings.Contains(msg, `data: {"id":7}`) || !strings.HasSuffix(msg, "\n\n") {
		t.Fatalf("неверное обрамление SSE: %q", msg)
	}
}

func TestHubTopicRouting(t *testing.T) {
	admin := newTestClient(topicAdmins, topicUser(1))
	user2 := newTestClient(topicUser(2))
	pub := newTestClient(topicPublic)
	for _, c := range []*sseClient{admin, user2, pub} {
		hub.add(c)
		defer hub.remove(c)
	}

	// Событие в admins получает только админ.
	publishEvent(topicAdmins, "audit", map[string]any{"x": 1})
	if recv(admin) == "" {
		t.Fatal("админ не получил admins-событие")
	}
	if recv(user2) != "" || recv(pub) != "" {
		t.Fatal("admins-событие утекло не туда")
	}

	// Личное событие user:2 — только второму.
	publishEvent(topicUser(2), "notification", map[string]any{"x": 2})
	if recv(user2) == "" {
		t.Fatal("user2 не получил личное событие")
	}
	if recv(admin) != "" || recv(pub) != "" {
		t.Fatal("личное событие утекло не туда")
	}

	// match_update публичен — получает только публичный подписчик.
	PublishMatchUpdate(42, 100)
	got := recv(pub)
	if got == "" || !strings.Contains(got, `"league_id":42`) || !strings.Contains(got, `"match_id":100`) {
		t.Fatalf("публичный подписчик не получил match_update: %q", got)
	}
	if recv(admin) != "" || recv(user2) != "" {
		t.Fatal("match_update утёк в личные/админские топики")
	}
}

func TestHubRemoveStopsDelivery(t *testing.T) {
	c := newTestClient(topicAdmins)
	hub.add(c)
	hub.remove(c)
	publishEvent(topicAdmins, "audit", map[string]any{"x": 1})
	if recv(c) != "" {
		t.Fatal("удалённый клиент всё ещё получает события")
	}
}
