package api

import (
	"efootball-bot/config"
	"efootball-bot/internal/repository"
	"efootball-bot/internal/service"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	cfg        *config.Config
	staticFS   fs.FS
	userRepo   repository.UserRepository
	leagueRepo repository.LeagueRepository
	matchRepo  repository.MatchRepository
	adminRepo  repository.AdminRepository
	matchSvc   *service.MatchService
	schedSvc   *service.ScheduleService
	eloSvc     *service.EloService
}

func NewServer(
	cfg *config.Config,
	staticFS fs.FS,
	userRepo repository.UserRepository,
	leagueRepo repository.LeagueRepository,
	matchRepo repository.MatchRepository,
	adminRepo repository.AdminRepository,
	matchSvc *service.MatchService,
	schedSvc *service.ScheduleService,
	eloSvc *service.EloService,
) *Server {
	return &Server{
		cfg:        cfg,
		staticFS:   staticFS,
		userRepo:   userRepo,
		leagueRepo: leagueRepo,
		matchRepo:  matchRepo,
		adminRepo:  adminRepo,
		matchSvc:   matchSvc,
		schedSvc:   schedSvc,
		eloSvc:     eloSvc,
	}
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{s.cfg.API.FrontendURL, "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// Auth
	r.Post("/auth/google", s.handleGoogleAuth)
	r.Post("/auth/admin-login", s.handleAdminLogin)

	// Public
	r.Get("/api/leagues", s.handleListLeagues)
	r.Get("/api/leagues/{id}", s.handleGetLeague)
	r.Get("/api/leagues/{id}/standings", s.handleStandings)
	r.Get("/api/leagues/{id}/schedule", s.handleSchedule)
	r.Get("/api/players", s.handlePlayers)
	r.Get("/api/top-scorers", s.handleTopScorers)

	// Protected
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)

		r.Get("/api/me", s.handleMe)
		r.Put("/api/me", s.handleUpdateMe)
		r.Post("/api/me/link-telegram", s.handleGenerateLinkCode)
		r.Get("/api/me/leagues", s.handleMyLeagues)
		r.Get("/api/me/history", s.handleMyHistory)

		r.Get("/api/leagues/{id}/my-matches", s.handleMyMatches)
		r.Post("/api/leagues/{id}/join", s.handleJoinLeague)

		r.Get("/api/matches/{id}", s.handleGetMatch)
		r.Post("/api/matches/{id}/result", s.handleSubmitResult)
		r.Post("/api/matches/{id}/confirm", s.handleConfirmMatch)
		r.Post("/api/matches/{id}/dispute", s.handleDisputeMatch)
	})

	// Admin
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Use(s.adminMiddleware)

		r.Get("/api/admin/leagues", s.handleAdminListLeagues)
		r.Post("/api/admin/leagues", s.handleAdminCreateLeague)
		r.Delete("/api/admin/leagues/{id}", s.handleAdminDeleteLeague)
		r.Get("/api/admin/leagues/{id}/members", s.handleAdminMembers)
		r.Post("/api/admin/leagues/{id}/members/{uid}/approve", s.handleAdminApprove)
		r.Post("/api/admin/leagues/{id}/members/{uid}/reject", s.handleAdminReject)
		r.Post("/api/admin/leagues/{id}/draw", s.handleAdminDraw)
		r.Post("/api/admin/matches/{id}/resolve", s.handleAdminResolve)
		r.Get("/api/admin/disputed", s.handleAdminDisputed)
		r.Get("/api/admin/users", s.handleAdminUsers)
		r.Get("/api/admin/admins", s.handleAdminListAdmins)
		r.Post("/api/admin/admins", s.handleAdminAdd)
		r.Delete("/api/admin/admins/{uid}", s.handleAdminRemove)
		r.Post("/api/admin/ratings/reset", s.handleAdminResetRatings)
	})

	// SPA static file serving (catch-all)
	if s.staticFS != nil {
		spa := s.spaHandler()
		r.Get("/", spa)
		r.Handle("/*", spa)
	}

	return r
}

// spaHandler serves the embedded Next.js static export.
// Falls back to index.html for any path not found (client-side routing).
func (s *Server) spaHandler() http.HandlerFunc {
	fileServer := http.FileServer(http.FS(s.staticFS))

	serveHTML := func(w http.ResponseWriter, name string) {
		data, err := fs.ReadFile(s.staticFS, name)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")

		// Root → index.html
		if path == "" {
			serveHTML(w, "index.html")
			return
		}

		// Static assets (_next/, images, etc.) — serve via FileServer
		if f, err := s.staticFS.Open(path); err == nil {
			stat, statErr := f.Stat()
			f.Close()
			if statErr == nil && !stat.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Next.js export: /login → login.html, /leagues/details → leagues/details.html
		if !strings.Contains(path, ".") {
			if _, err := s.staticFS.Open(path + ".html"); err == nil {
				serveHTML(w, path+".html")
				return
			}
		}

		// SPA fallback
		serveHTML(w, "index.html")
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
