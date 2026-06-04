package logger

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5/middleware"
)

type contextKey string

const logKey contextKey = "logger"

// L is the global logger.
var L *slog.Logger

// Init initializes the global logger.
func Init(env string) {
	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	L = slog.New(handler)
}

// FromContext returns the logger stored in ctx, or the global logger.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(logKey).(*slog.Logger); ok && l != nil {
		return l
	}
	if L != nil {
		return L
	}
	return slog.Default()
}

// RequestIDMiddleware adds a request ID to each request's context.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return middleware.RequestID(next)
}

// HTTPLogger logs each HTTP request.
func HTTPLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip logging for static files
		if len(r.URL.Path) > 6 && r.URL.Path[:7] == "/_next/" {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ── Audit helpers ─────────────────────────────────────────────────────────────

func DrawGenerated(ctx context.Context, leagueID, round int64, adminID int64) {
	FromContext(ctx).Info("draw generated",
		"league_id", leagueID,
		"round", round,
		"admin_id", adminID,
	)
}

func AdminResolve(ctx context.Context, matchID, adminID int64, homeGoals, awayGoals int16) {
	FromContext(ctx).Info("admin resolved match",
		"match_id", matchID,
		"admin_id", adminID,
		"home_goals", homeGoals,
		"away_goals", awayGoals,
	)
}

func MatchConfirmed(ctx context.Context, matchID, leagueID, homeUserID, awayUserID int64, homeGoals, awayGoals int16) {
	FromContext(ctx).Info("match confirmed",
		"match_id", matchID,
		"league_id", leagueID,
		"home_user_id", homeUserID,
		"away_user_id", awayUserID,
		"home_goals", homeGoals,
		"away_goals", awayGoals,
	)
}
