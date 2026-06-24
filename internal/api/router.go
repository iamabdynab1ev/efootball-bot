package api

import (
	"efootball-bot/config"
	"efootball-bot/internal/data"
	"efootball-bot/internal/logger"
	"efootball-bot/internal/repository"
	"efootball-bot/internal/service"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	cfg              *config.Config
	staticFS         fs.FS
	userRepo         repository.UserRepository
	leagueRepo       repository.LeagueRepository
	matchRepo        repository.MatchRepository
	adminRepo        repository.AdminRepository
	bracketRepo      repository.BracketRepository
	achievRepo       repository.AchievementRepository
	deadlineRepo     repository.DeadlineRepository
	awardRepo        repository.AwardRepository
	statsRepo        repository.StatsRepository
	matchSvc         *service.MatchService
	schedSvc         *service.ScheduleService
	eloSvc           *service.EloService
	playoffSvc       *service.PlayoffService
	groupStageSvc    *service.GroupStageService
	cupSvc           *service.CupService
	swissSvc         *service.SwissService
	nationsLeagueSvc *service.NationsLeagueService
	awardSvc         *service.AwardService
	deRepo           repository.DoubleElimRepository
	deSvc            *service.DoubleElimService
	notifier         *TelegramNotifier
	pushRepo         repository.PushRepository
	webPush          *WebPushNotifier
}

func (s *Server) SetPush(pr repository.PushRepository, wp *WebPushNotifier) {
	s.pushRepo, s.webPush = pr, wp
}

func (s *Server) SetNotifier(n *TelegramNotifier)                          { s.notifier = n }
func (s *Server) SetGroupStageService(gs *service.GroupStageService)       { s.groupStageSvc = gs }
func (s *Server) SetCupService(cs *service.CupService)                     { s.cupSvc = cs }
func (s *Server) SetSwissService(ss *service.SwissService)                 { s.swissSvc = ss }
func (s *Server) SetNationsLeagueService(ns *service.NationsLeagueService) { s.nationsLeagueSvc = ns }
func (s *Server) SetAchievementRepo(a repository.AchievementRepository)    { s.achievRepo = a }
func (s *Server) SetDeadlineRepo(d repository.DeadlineRepository)          { s.deadlineRepo = d }
func (s *Server) SetAwardRepo(a repository.AwardRepository)                { s.awardRepo = a }
func (s *Server) SetAwardService(a *service.AwardService)                  { s.awardSvc = a }
func (s *Server) SetStatsRepo(sr repository.StatsRepository)               { s.statsRepo = sr }
func (s *Server) SetDoubleElim(dr repository.DoubleElimRepository, ds *service.DoubleElimService) {
	s.deRepo, s.deSvc = dr, ds
}

func NewServer(
	cfg *config.Config,
	staticFS fs.FS,
	userRepo repository.UserRepository,
	leagueRepo repository.LeagueRepository,
	matchRepo repository.MatchRepository,
	adminRepo repository.AdminRepository,
	bracketRepo repository.BracketRepository,
	matchSvc *service.MatchService,
	schedSvc *service.ScheduleService,
	eloSvc *service.EloService,
	playoffSvc *service.PlayoffService,
) *Server {
	return &Server{
		cfg:         cfg,
		staticFS:    staticFS,
		userRepo:    userRepo,
		leagueRepo:  leagueRepo,
		matchRepo:   matchRepo,
		adminRepo:   adminRepo,
		bracketRepo: bracketRepo,
		matchSvc:    matchSvc,
		schedSvc:    schedSvc,
		eloSvc:      eloSvc,
		playoffSvc:  playoffSvc,
	}
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Use(logger.RequestIDMiddleware) // добавляет X-Request-Id к каждому запросу
	r.Use(logger.HTTPLogger)          // структурированный лог, пропускает статику
	r.Use(middleware.Recoverer)
	// gzip для HTML/JS/CSS/JSON/SVG; text/event-stream не в списке — SSE не трогает
	r.Use(middleware.Compress(5))
	r.Use(maxBodyMiddleware(1 << 20)) // 1 MiB лимит тела запроса (SSE — GET без тела, не затронут)
	r.Use(securityHeadersMiddleware)

	// CORS — localhost:3000 только в не-продакшне
	origins := []string{s.cfg.API.FrontendURL}
	if s.cfg.Env != "production" && s.cfg.Env != "" {
		origins = append(origins, "http://localhost:3000")
	}
	if len(origins) == 0 || origins[0] == "" {
		origins = []string{"http://localhost:3000"}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	rl := func(max int, d time.Duration) func(http.Handler) http.Handler {
		return rateLimitMiddleware(max, d)
	}

	// Auth
	r.Post("/auth/google", rl(20, time.Minute)(http.HandlerFunc(s.handleGoogleAuth)).ServeHTTP)
	r.Post("/auth/admin-login", rl(10, time.Minute)(http.HandlerFunc(s.handleAdminLogin)).ServeHTTP)

	// Dev endpoints — только вне продакшна
	if s.cfg.Env != "production" {
		r.Post("/auth/dev-login", s.handleDevLogin)
		r.Get("/auth/dev-users", s.handleDevUsers)
	}

	// Public
	r.Get("/api/clubs", func(w http.ResponseWriter, r *http.Request) {
		// Клубы никогда не меняются — агрессивное кэширование на клиенте
		w.Header().Set("Cache-Control", "public, max-age=86400") // 24 часа
		jsonOK(w, data.Clubs)
	})
	r.Get("/api/leagues", s.handleListLeagues)
	r.Get("/api/leagues/{id}", s.handleGetLeague)
	r.Get("/api/leagues/{id}/standings", s.handleStandings)
	r.Get("/api/leagues/{id}/schedule", s.handleSchedule)
	r.Get("/api/players", s.handlePlayers)
	r.Get("/api/players/{id}", s.handleGetPlayer)
	r.Get("/api/players/{id}/card.png", s.handlePlayerCard)
	r.Get("/api/push/vapid-public", s.handleVapidPublic)
	r.Get("/api/top-scorers", s.handleTopScorers)
	r.Get("/api/hall-of-fame", s.handleHallOfFame)
	r.Get("/api/stats/win-rate", s.handleStatWinRate)
	r.Get("/api/stats/streaks", s.handleStatStreaks)
	r.Get("/api/stats/avg-goals", s.handleStatAvgGoals)
	r.Get("/api/stats/team-power", s.handleStatTeamPower)
	r.Get("/api/stats/activity", s.handleStatActivity)
	r.Get("/api/leagues/{id}/bracket", s.handleBracket)
	r.Get("/api/leagues/{id}/double-elim", s.handleDoubleElimBracket)
	r.Get("/api/leagues/{id}/progress", s.handleLeagueProgress)
	r.Get("/api/leagues/{id}/groups", s.handleLeagueGroupsList)
	r.Get("/api/leagues/{id}/groups/{group}/standings", s.handleGroupStandings)
	r.Get("/api/leagues/{id}/groups/{group}/schedule", s.handleGroupSchedule)
	r.Get("/api/events", s.handleSSE)

	// Protected
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)

		r.Get("/api/me", s.handleMe)
		r.Patch("/api/me", s.handleUpdateMe)
		r.Delete("/api/me", s.handleDeleteMe)
		r.Post("/api/me/link-telegram", rl(5, time.Minute)(http.HandlerFunc(s.handleGenerateLinkCode)).ServeHTTP)
		r.Get("/api/me/leagues", s.handleMyLeagues)
		r.Get("/api/me/history", s.handleMyHistory)
		r.Get("/api/players/{id}/h2h", s.handleHeadToHead)
		r.Post("/api/me/push/subscribe", s.handlePushSubscribe)
		r.Post("/api/me/push/unsubscribe", s.handlePushUnsubscribe)
		r.Post("/api/me/push/test", rl(5, time.Minute)(http.HandlerFunc(s.handlePushTest)).ServeHTTP)

		r.Get("/api/leagues/{id}/my-matches", s.handleMyMatches)
		r.Post("/api/leagues/{id}/join", rl(10, time.Minute)(http.HandlerFunc(s.handleJoinLeague)).ServeHTTP)

		r.Get("/api/matches/{id}", s.handleGetMatch)
		r.Post("/api/matches/{id}/result", rl(10, time.Minute)(http.HandlerFunc(s.handleSubmitResult)).ServeHTTP)
		r.Post("/api/matches/{id}/confirm", s.handleConfirmMatch)
		r.Post("/api/matches/{id}/dispute", s.handleDisputeMatch)
	})

	// Admin
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Use(s.adminMiddleware)

		r.Get("/api/admin/leagues", s.handleAdminListLeagues)
		r.Post("/api/admin/leagues", s.handleAdminCreateLeague)
		r.Patch("/api/admin/leagues/{id}", s.handleAdminUpdateLeague)
		r.Delete("/api/admin/leagues/{id}", s.handleAdminDeleteLeague)
		r.Delete("/api/admin/leagues/{id}/purge", s.handleAdminPurgeLeague)
		r.Get("/api/admin/leagues/{id}/members", s.handleAdminMembers)
		r.Post("/api/admin/leagues/{id}/members/{uid}/approve", s.handleAdminApprove)
		r.Post("/api/admin/leagues/{id}/members/{uid}/reject", s.handleAdminReject)
		r.Post("/api/admin/leagues/{id}/open", s.handleAdminOpenRegistration)
		r.Post("/api/admin/leagues/{id}/draw", s.handleAdminDraw)
		r.Get("/api/admin/leagues/{id}/playoff-options", s.handleAdminPlayoffOptions)
		r.Post("/api/admin/leagues/{id}/playoff", s.handleAdminPlayoff)
		r.Post("/api/admin/leagues/{id}/next-round", s.handleAdminNextRound) // Swiss: следующий тур
		r.Post("/api/admin/leagues/{id}/final-four", s.handleAdminFinalFour) // Nations League: Финал четырёх
		r.Post("/api/admin/matches/{id}/resolve", s.handleAdminResolve)
		r.Get("/api/admin/disputed", s.handleAdminDisputed)
		r.Get("/api/admin/leagues/{id}/deadlines", s.handleAdminGetDeadlines)
		r.Post("/api/admin/leagues/{id}/deadlines", s.handleAdminSetDeadline)
		r.Delete("/api/admin/leagues/{id}/deadlines/{round}", s.handleAdminDeleteDeadline)
		r.Post("/api/admin/leagues/{id}/finalize", s.handleAdminFinalize)
		r.Get("/api/admin/users", s.handleAdminUsers)
		r.Delete("/api/admin/users/{uid}", s.handleAdminDeleteUser)
		r.Get("/api/admin/admins", s.handleAdminListAdmins)
		r.Post("/api/admin/admins", s.handleAdminAdd)
		r.Delete("/api/admin/admins/{uid}", s.handleAdminRemove)
		r.Post("/api/admin/ratings/reset", s.handleAdminResetRatings)
		r.Delete("/api/admin/ratings", s.handleAdminResetRatings) // REST-совместимый алиас
		r.Post("/api/admin/broadcast", rl(10, time.Minute)(http.HandlerFunc(s.handleAdminBroadcast)).ServeHTTP)
		r.Post("/api/admin/notify", rl(30, time.Minute)(http.HandlerFunc(s.handleAdminNotifyUser)).ServeHTTP)
	})

	// SPA static file serving (catch-all)
	if s.staticFS != nil {
		spa := s.spaHandler()
		r.Get("/", spa)
		r.Handle("/*", spa)
	}

	return r
}

// ── Rate limiter ──────────────────────────────────────────────────────────────

type rateBucket struct {
	count   int
	resetAt time.Time
}

var (
	rateMu      sync.Mutex
	rateBuckets = map[string]*rateBucket{}
)

// rateLimitMiddleware ограничивает количество запросов per IP (max за interval).
func rateLimitMiddleware(max int, interval time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if idx := strings.LastIndex(ip, ":"); idx != -1 {
				ip = ip[:idx]
			}

			rateMu.Lock()
			now := time.Now()
			// Очищаем устаревшие записи чтобы map не росла бесконечно.
			if len(rateBuckets) > 500 {
				for k, v := range rateBuckets {
					if now.After(v.resetAt) {
						delete(rateBuckets, k)
					}
				}
			}
			b, ok := rateBuckets[ip]
			if !ok || now.After(b.resetAt) {
				b = &rateBucket{count: 0, resetAt: now.Add(interval)}
				rateBuckets[ip] = b
			}
			b.count++
			exceeded := b.count > max
			rateMu.Unlock()

			if exceeded {
				jsonError(w, "too many requests, try again later", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// maxBodyMiddleware ограничивает размер тела запроса — защита от OOM на больших
// payload. Превышение лимита превращает чтение тела в ошибку (обрабатывается
// json.Decode → 400). SSE и прочие GET без тела не затрагиваются.
func maxBodyMiddleware(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, n)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ── SPA ───────────────────────────────────────────────────────────────────────

// spaHandler serves the embedded Next.js static export.
func (s *Server) spaHandler() http.HandlerFunc {
	fileServer := http.FileServer(http.FS(s.staticFS))

	serveHTML := func(w http.ResponseWriter, name string) {
		data, err := fs.ReadFile(s.staticFS, name)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// HTML всегда ревалидируется — деплой подхватывается сразу
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(data)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")

		if path == "" {
			serveHTML(w, "index.html")
			return
		}

		if f, err := s.staticFS.Open(path); err == nil {
			stat, statErr := f.Stat()
			f.Close()
			if statErr == nil && !stat.IsDir() {
				if strings.HasPrefix(path, "_next/static/") {
					// Имена содержат content-hash — можно кэшировать навсегда
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					w.Header().Set("Cache-Control", "public, max-age=3600")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		if !strings.Contains(path, ".") {
			if _, err := s.staticFS.Open(path + ".html"); err == nil {
				serveHTML(w, path+".html")
				return
			}
		}

		serveHTML(w, "index.html")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// securityHeadersMiddleware добавляет стандартные security headers.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		// HSTS с preload — Lighthouse требует preload директиву
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		// COOP — изоляция окна от попапов (совместимо с Google OAuth)
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin-allow-popups")
		// COEP — не ставим, ломает Google Sign-In iframe
		// CSP: unsafe-inline нужен для Next.js inline scripts; strict-dynamic усиливает allowlist
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' https://accounts.google.com; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: blob: https://r2.thesportsdb.com https://www.thesportsdb.com https://crests.football-data.org; "+
				"font-src 'self' data:; "+
				"connect-src 'self' https://accounts.google.com https://oauth2.googleapis.com; "+
				"frame-src https://accounts.google.com; "+
				"frame-ancestors 'self'; "+
				"base-uri 'self'; "+
				"form-action 'self' https://accounts.google.com;")
		next.ServeHTTP(w, r)
	})
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// errorCode возвращает стабильный машиночитаемый код по HTTP-статусу.
// Клиент может ветвиться по code, не разбирая текст error (который может быть
// локализован/изменён).
func errorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusTooManyRequests:
		return "rate_limited"
	default:
		if status >= 500 {
			return "server_error"
		}
		return "error"
	}
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	// error — человекочитаемое сообщение (как раньше), code — стабильный машинный.
	json.NewEncoder(w).Encode(map[string]string{"error": msg, "code": errorCode(code)})
}

// jsonErrorLog — как jsonError но дополнительно логирует реальную ошибку сервера.
func jsonErrorLog(w http.ResponseWriter, r *http.Request, msg string, code int, err error) {
	if code >= 500 && err != nil {
		logger.FromContext(r.Context()).Error("server error",
			"status", code,
			"message", msg,
			"error", err.Error(),
			"method", r.Method,
			"path", r.URL.Path,
		)
	}
	jsonError(w, msg, code)
}
