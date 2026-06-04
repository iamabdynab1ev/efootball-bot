package api

import (
	"fmt"
	"net/http"
	"sync"
)

// ── SSE hub ───────────────────────────────────────────────────────────────────

type sseClient struct {
	ch       chan string
	leagueID int64
}

var (
	sseMu      sync.Mutex
	sseClients = map[*sseClient]struct{}{}
)

// PublishMatchUpdate broadcasts a match-update event to all SSE subscribers
// that are watching the given league.
func PublishMatchUpdate(leagueID int64) {
	msg := fmt.Sprintf("data: {\"league_id\":%d}\n\n", leagueID)
	sseMu.Lock()
	defer sseMu.Unlock()
	for c := range sseClients {
		if c.leagueID == 0 || c.leagueID == leagueID {
			select {
			case c.ch <- msg:
			default:
			}
		}
	}
}

// handleSSE — GET /api/events — subscribes the caller to server-sent events.
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

	client := &sseClient{ch: make(chan string, 8)}
	sseMu.Lock()
	sseClients[client] = struct{}{}
	sseMu.Unlock()

	defer func() {
		sseMu.Lock()
		delete(sseClients, client)
		sseMu.Unlock()
	}()

	// Initial heartbeat
	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case msg := <-client.ch:
			fmt.Fprint(w, msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
