package main

import (
	"context"
	"database/sql"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"golang.org/x/crypto/bcrypt"

	"efootball-bot/config"
	"efootball-bot/internal/api"
	"efootball-bot/internal/bot/handlers"
	"efootball-bot/internal/logger"
	"efootball-bot/internal/repository"
	"efootball-bot/internal/service"
)

func main() {
	cfg := config.Load()
	websiteURL = cfg.API.FrontendURL // куда бот направляет игроков

	// Инициализируем структурированный логгер сразу после загрузки конфига
	logger.Init(cfg.Env)
	logger.L.Info("starting",
		"env", cfg.Env,
		"port", cfg.API.Port,
		"google_auth", cfg.API.GoogleClientID != "",
		"telegram_bot", cfg.Telegram.BotUsername,
	)

	// ── Миграции ──────────────────────────────────────────────────────
	dbGoose, err := sql.Open("pgx", cfg.Postgres.DSN)
	if err != nil {
		log.Fatalf("❌ Миграции: %v", err)
	}
	goose.SetDialect("postgres")

	migrationsPath := "./migrations"
	if exe, _ := os.Executable(); exe != "" {
		if p := filepath.Join(filepath.Dir(exe), "migrations"); fileExists(p) {
			migrationsPath = p
		}
	}
	if err := goose.Up(dbGoose, migrationsPath); err != nil {
		// В production несогласованная схема опаснее простоя — падаем сразу,
		// чтобы не отдавать запросы против неполной/битой БД.
		if cfg.Env == "production" {
			log.Fatalf("❌ Миграции (production): %v", err)
		}
		log.Printf("⚠️ Миграции: %v", err)
	}
	dbGoose.Close()

	// ── БД пул ────────────────────────────────────────────────────────
	poolCfg, err := pgxpool.ParseConfig(cfg.Postgres.DSN)
	if err != nil {
		log.Fatalf("❌ pgxpool config: %v", err)
	}
	poolCfg.MaxConns = 100 // было 20 — хватит при 10k users
	poolCfg.MinConns = 10  // было 5
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.MaxConnIdleTime = 10 * time.Minute // было 30 — освобождаем быстрее
	poolCfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		log.Fatalf("❌ pgxpool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("❌ БД недоступна: %v", err)
	}
	log.Println("✅ БД соединение установлено")

	// ── Репозитории ───────────────────────────────────────────────────
	userRepo := repository.NewUserRepository(pool)
	leagueRepo := repository.NewLeagueRepository(pool)
	matchRepo := repository.NewMatchRepository(pool)
	adminRepo := repository.NewAdminRepository(pool)
	bracketRepo := repository.NewBracketRepository(pool)
	deRepo := repository.NewDoubleElimRepository(pool)

	// ── Сидер супер-администратора ────────────────────────────────────
	seedSuperAdmin(context.Background(), adminRepo, cfg)

	// ── Сервисы ───────────────────────────────────────────────────────
	matchSvc := service.NewMatchService(matchRepo, leagueRepo)
	schedSvc := service.NewScheduleService(matchRepo, leagueRepo)
	eloSvc := service.NewEloService(userRepo)
	playoffSvc := service.NewPlayoffService(leagueRepo, bracketRepo)
	deSvc := service.NewDoubleElimService(leagueRepo, deRepo)
	groupStageSvc := service.NewGroupStageService(matchRepo, leagueRepo)
	cupSvc := service.NewCupService(matchRepo, leagueRepo, bracketRepo)
	swissSvc := service.NewSwissService(matchRepo, leagueRepo, bracketRepo)
	nationsLeagueSvc := service.NewNationsLeagueService(matchRepo, leagueRepo, bracketRepo)

	achievRepo := repository.NewAchievementRepository(pool)
	deadlineRepo := repository.NewDeadlineRepository(pool)
	awardRepo := repository.NewAwardRepository(pool)
	statsRepo := repository.NewStatsRepository(pool)
	pushRepo := repository.NewPushRepository(pool)
	settingsRepo := repository.NewSettingsRepository(pool)

	achievSvc := service.NewAchievementService(achievRepo, matchRepo)
	matchSvc.SetAchievementService(achievSvc)
	awardSvc := service.NewAwardService(awardRepo, leagueRepo, achievRepo)

	matchSvc.SetPlayoffService(playoffSvc)
	matchSvc.SetDoubleElimService(deSvc)
	matchSvc.SetAwardService(awardSvc)
	service.OnLeaguesChanged = api.InvalidateLeagues

	// ── HTTP API ──────────────────────────────────────────────────────
	var uiFS fs.FS
	if sub, err := fs.Sub(embeddedUI, "ui"); err == nil {
		uiFS = sub
	}
	apiServer := api.NewServer(cfg, uiFS, userRepo, leagueRepo, matchRepo, adminRepo, bracketRepo, matchSvc, schedSvc, eloSvc, playoffSvc)
	httpServer := &http.Server{
		Addr:    ":" + cfg.API.Port,
		Handler: apiServer.Handler(),
		// Таймаузы против Slowloris и зависших соединений.
		// WriteTimeout НЕ задаём: /api/events (SSE) держит ответ открытым.
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       20 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MiB
	}
	go func() {
		log.Printf("🌐 HTTP API запущен на порту %s", cfg.API.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ HTTP сервер: %v", err)
		}
	}()

	// ── Telegram Bot ──────────────────────────────────────────────────
	bot, err := tgbotapi.NewBotAPI(cfg.Telegram.BotToken)
	if err != nil {
		log.Fatalf("❌ Telegram: %v", err)
	}
	// Берём реальный @username бота из Telegram — это надёжнее, чем переменная
	// BOT_USERNAME (её легко забыть задать). Нужен для deep-link «Открыть бота»
	// (https://t.me/<username>?start=link_CODE). cfg — указатель, общий с API,
	// поэтому значение сразу станет доступно в handleGenerateLinkCode.
	if bot.Self.UserName != "" {
		cfg.Telegram.BotUsername = bot.Self.UserName
		log.Printf("🤖 Telegram бот: @%s", bot.Self.UserName)
	}
	// Таймаут чуть больше long-poll (u.Timeout=60s): зависший вызов Telegram
	// API не блокирует горутину уведомлений/воркера навсегда.
	bot.Client = &http.Client{Timeout: 75 * time.Second}
	_, _ = bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})

	telegramNotifier := api.NewTelegramNotifier(bot)
	apiServer.SetNotifier(telegramNotifier)
	matchSvc.SetNotifier(telegramNotifier)
	apiServer.SetGroupStageService(groupStageSvc)
	apiServer.SetCupService(cupSvc)
	apiServer.SetSwissService(swissSvc)
	apiServer.SetNationsLeagueService(nationsLeagueSvc)
	apiServer.SetAchievementRepo(achievRepo)
	apiServer.SetDeadlineRepo(deadlineRepo)
	apiServer.SetAwardRepo(awardRepo)
	apiServer.SetAwardService(awardSvc)
	apiServer.SetStatsRepo(statsRepo)
	apiServer.SetDoubleElim(deRepo, deSvc)

	webPush := api.NewWebPushNotifier(pushRepo, cfg.API.VAPIDPublic, cfg.API.VAPIDPrivate, cfg.API.VAPIDSubject)
	apiServer.SetPush(pushRepo, webPush)
	apiServer.SetSettingsRepo(settingsRepo)
	if cfg.API.VAPIDPublic != "" {
		log.Println("🔔 Web Push включён")
	}

	reminderSvc := service.NewReminderService(deadlineRepo, matchRepo, leagueRepo, userRepo, telegramNotifier)

	// ── Периодические задачи ─────────────────────────────────────────
	go func() {
		cacheTicker := time.NewTicker(5 * time.Minute)
		rankTicker := time.NewTicker(5 * time.Minute)
		reminderTicker := time.NewTicker(30 * time.Minute)
		defer cacheTicker.Stop()
		defer rankTicker.Stop()
		defer reminderTicker.Stop()
		for {
			select {
			case <-cacheTicker.C:
				api.CleanupAllCaches()
			case <-rankTicker.C:
				ctx := context.Background()
				if err := userRepo.RecalculateAllRanks(ctx); err != nil {
					log.Printf("periodic RecalculateAllRanks: %v", err)
				}
			case <-reminderTicker.C:
				ctx := context.Background()
				if err := reminderSvc.CheckAndSend(ctx); err != nil {
					log.Printf("periodic reminder check: %v", err)
				}
			}
		}
	}()

	h := handlers.New(bot, userRepo, leagueRepo, matchRepo, matchSvc, schedSvc, groupStageSvc, adminRepo, eloSvc, cfg.Admin.TelegramID, cfg.Telegram.GroupID)
	h.SetAchievementRepo(achievRepo)
	ah := handlers.NewAdminHandlers(bot, userRepo, leagueRepo, adminRepo, cfg.Admin.TelegramID)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	const numWorkers = 150 // было 50 — больше параллельных обработчиков
	updates := bot.GetUpdatesChan(u)
	jobs := make(chan tgbotapi.Update, 1000) // было 200 — больше буфер очереди

	var wg sync.WaitGroup
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for update := range jobs {
				processUpdateSafely(update, h, ah, userRepo, bot)
			}
		}()
	}

	go func() {
		for update := range updates {
			select {
			case jobs <- update:
			default:
				log.Println("⚠️ Очередь переполнена, апдейт пропущен")
			}
		}
	}()

	<-quit
	log.Println("🛑 Завершение работы...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)

	bot.StopReceivingUpdates()
	close(jobs)
	wg.Wait()
	log.Println("✅ Все воркеры завершены.")
}

// processUpdateSafely runs processUpdate with its own timeout and recovers from
// panics so that one bad update can't permanently kill a worker goroutine.
func processUpdateSafely(
	update tgbotapi.Update,
	h *handlers.Handler,
	ah *handlers.AdminHandlers,
	userRepo repository.UserRepository,
	bot *tgbotapi.BotAPI,
) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️ panic while processing update: %v\n%s", r, debug.Stack())
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	processUpdate(ctx, update, h, ah, userRepo, bot)
}

func processUpdate(
	ctx context.Context,
	update tgbotapi.Update,
	h *handlers.Handler,
	ah *handlers.AdminHandlers,
	userRepo repository.UserRepository,
	bot *tgbotapi.BotAPI,
) {
	// Режим «только уведомления»: игровые действия перенесены на сайт.
	// Инлайн-кнопки отключены — гасим «часики» и ничего не делаем.
	if update.CallbackQuery != nil {
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Действия теперь на сайте"))
		return
	}

	if update.Message == nil {
		return
	}

	msg := update.Message
	text := strings.TrimSpace(msg.Text)

	switch {
	// Привязка аккаунта (код выдаётся на сайте) — единственное действие бота.
	case strings.HasPrefix(text, "/start link_"):
		handleLinkTelegram(ctx, bot, msg, userRepo, strings.TrimPrefix(text, "/start link_"))

	case strings.HasPrefix(text, "/link "):
		handleLinkTelegram(ctx, bot, msg, userRepo, strings.TrimSpace(strings.TrimPrefix(text, "/link ")))

	default:
		// Всё остальное — направляем на сайт.
		sendWebsiteNotice(bot, msg.Chat.ID)
	}
}

// websiteURL — адрес веб-приложения (из FRONTEND_URL), куда бот направляет
// пользователей. Заполняется в main().
var websiteURL string

// sendWebsiteNotice объясняет, что турнир ведётся на сайте, а бот — только для
// уведомлений и привязки аккаунта.
func sendWebsiteNotice(bot *tgbotapi.BotAPI, chatID int64) {
	site := websiteURL
	if site == "" {
		site = "сайте турнира"
	}
	text := "🏆 Турнир проходит на сайте!\n\n" +
		"Регистрация, вступление в лигу и ввод результатов — здесь:\n" + site + "\n\n" +
		"Этот бот присылает только уведомления (жеребьёвка, результаты матчей и т.п.).\n" +
		"Чтобы получать их — на сайте: Профиль → «Привязать Telegram» → отправьте код сюда."
	m := tgbotapi.NewMessage(chatID, text)
	_, _ = bot.Send(m)
}

func handleLinkTelegram(ctx context.Context, bot *tgbotapi.BotAPI, msg *tgbotapi.Message, userRepo repository.UserRepository, code string) {
	if len(code) != 6 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❗ Код должен состоять из 6 цифр. Получите код на сайте в разделе Профиль.")
		_, _ = bot.Send(reply)
		return
	}

	var username *string
	if msg.From.UserName != "" {
		u := msg.From.UserName
		username = &u
	}

	user, err := userRepo.LinkTelegramByCode(ctx, code, msg.From.ID, username)
	if err != nil || user == nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Код неверный или истёк. Сгенерируйте новый код на сайте.")
		_, _ = bot.Send(reply)
		return
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID,
		"✅ Ваш Telegram успешно привязан к аккаунту на сайте!\n\nТеперь вы будете получать уведомления о матчах прямо здесь.")
	_, _ = bot.Send(reply)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func seedSuperAdmin(ctx context.Context, adminRepo repository.AdminRepository, cfg *config.Config) {
	if cfg.Admin.Username == "" || cfg.Admin.Password == "" {
		log.Println("⚠️  ADMIN_USERNAME/ADMIN_PASSWORD не заданы — супер-администратор не создан")
		return
	}
	exists, err := adminRepo.SuperAdminExists(ctx)
	if err != nil {
		log.Printf("⚠️  Проверка супер-администратора: %v", err)
		return
	}
	if exists {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.Admin.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("❌ bcrypt: %v", err)
		return
	}
	if err := adminRepo.SeedSuperAdmin(ctx, cfg.Admin.Username, string(hash), "Супер-Администратор"); err != nil {
		log.Printf("❌ Seed super admin: %v", err)
		return
	}
	log.Printf("✅ Супер-администратор создан: %s", cfg.Admin.Username)
}
