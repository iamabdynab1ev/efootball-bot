package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
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
	"efootball-bot/internal/groupcast"
	"efootball-bot/internal/i18n"
	"efootball-bot/internal/logger"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"efootball-bot/internal/service"
	"efootball-bot/internal/storage"
	"efootball-bot/internal/wa"
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
	dbUnavailable := false // БД лежит (квота Neon/сеть) — стартуем в режиме самовосстановления
	if err := goose.Up(dbGoose, migrationsPath); err != nil {
		if isDBConnError(err) {
			// БД недоступна (исчерпана квота Neon, сеть, спящий compute) — это
			// НЕ битая схема. Поднимаемся и ждём: фоновая петля применит
			// миграции, как только база оживёт (например, при сбросе квоты
			// 1-го числа) — прод самовосстановится без ручного деплоя.
			log.Printf("⚠️ БД недоступна, старт в режиме ожидания (миграции применятся автоматически): %v", err)
			dbUnavailable = true
		} else if cfg.Env == "production" {
			// Несогласованная схема опаснее простоя — падаем сразу,
			// чтобы не отдавать запросы против неполной/битой БД.
			log.Fatalf("❌ Миграции (production): %v", err)
		} else {
			log.Printf("⚠️ Миграции: %v", err)
		}
	}
	dbGoose.Close()

	// ── БД пул ────────────────────────────────────────────────────────
	poolCfg, err := pgxpool.ParseConfig(cfg.Postgres.DSN)
	if err != nil {
		log.Fatalf("❌ pgxpool config: %v", err)
	}
	// Neon Free: compute-часы ограничены, база должна ЗАСЫПАТЬ в тишине.
	// MinConns=0 — без постоянных соединений; редкий health-check не будит
	// уснувший compute; первый запрос после сна просто чуть дольше (~1с).
	poolCfg.MaxConns = 25
	poolCfg.MinConns = 0
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.MaxConnIdleTime = 4 * time.Minute
	poolCfg.HealthCheckPeriod = 10 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		log.Fatalf("❌ pgxpool: %v", err)
	}
	defer pool.Close()

	if !dbUnavailable {
		if err := pool.Ping(context.Background()); err != nil {
			log.Fatalf("❌ БД недоступна: %v", err)
		}
		log.Println("✅ БД соединение установлено")
	} else {
		log.Println("⏳ БД недоступна — сервис поднят, ждём оживления (миграции применятся автоматически)")
	}

	// ── Репозитории ───────────────────────────────────────────────────
	userRepo := repository.NewUserRepository(pool)
	leagueRepo := repository.NewLeagueRepository(pool)
	matchRepo := repository.NewMatchRepository(pool)
	adminRepo := repository.NewAdminRepository(pool)
	bracketRepo := repository.NewBracketRepository(pool)
	deRepo := repository.NewDoubleElimRepository(pool)

	// ── Сидер супер-администратора ────────────────────────────────────
	seedSuperAdmin(context.Background(), adminRepo, cfg)

	// Самовосстановление: если БД лежала на старте (квота Neon, сеть) —
	// пробуем миграции фоном, пока не оживёт. Как только Neon проснётся
	// (например, при сбросе квоты 1-го числа), прод возвращается в строй
	// сам: миграции + синк админ-кредов, без ручного деплоя.
	if dbUnavailable {
		go func() {
			for {
				time.Sleep(3 * time.Minute)
				db, oErr := sql.Open("pgx", cfg.Postgres.DSN)
				if oErr != nil {
					continue
				}
				mErr := goose.Up(db, migrationsPath)
				db.Close()
				if mErr == nil {
					seedSuperAdmin(context.Background(), adminRepo, cfg)
					log.Println("✅ БД ожила — миграции применены, сервис в строю")
					return
				}
			}
		}()
	}

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
	predRepo := repository.NewPredictionRepository(pool)
	awardRepo := repository.NewAwardRepository(pool)
	notifyGroupRepo := repository.NewNotifyGroupRepository(pool)
	statsRepo := repository.NewStatsRepository(pool)
	pushRepo := repository.NewPushRepository(pool)
	settingsRepo := repository.NewSettingsRepository(pool)
	friendlyRepo := repository.NewFriendlyRepository(pool)

	achievSvc := service.NewAchievementService(achievRepo, matchRepo)
	matchSvc.SetAchievementService(achievSvc)
	awardSvc := service.NewAwardService(awardRepo, leagueRepo, achievRepo, matchRepo)

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
	apiServer.SetDBPinger(pool) // /readyz пингует БД для uptime-мониторинга

	// Аудит действий: асинхронная запись + живая лента админам через SSE.
	auditRepo := repository.NewAuditRepository(pool)
	auditSvc := service.NewAuditService(auditRepo, apiServer.PublishAudit)
	apiServer.SetAudit(auditSvc)

	// Уведомления: персист + живая доставка в личный SSE-топик.
	notifRepo := repository.NewNotificationRepository(pool)
	notifSvc := service.NewNotificationService(notifRepo, apiServer.PublishNotification)
	// Язык получателя для локализованных уведомлений (users.language).
	notifSvc.SetLangResolver(func(ctx context.Context, uid int64) string {
		if u, err := userRepo.GetByID(ctx, uid); err == nil && u != nil && u.Language != "" {
			return u.Language
		}
		return "ru"
	})
	apiServer.SetNotifications(notifSvc)
	// Трофеи и достижения сообщают о себе владельцу (celebration на клиенте).
	awardSvc.SetNotifications(notifSvc)
	achievSvc.SetNotifications(notifSvc)

	// Чат турнира: персист + адресная доставка участникам через SSE.
	chatRepo := repository.NewChatRepository(pool)
	chatSvc := service.NewChatService(chatRepo, leagueRepo, apiServer.PublishChat)
	chatSvc.SetMentionHandler(apiServer.NotifyChatMention)  // @упоминание → колокольчик + пуш
	chatSvc.SetDirectHandler(apiServer.NotifyDirectMessage) // ЛС → уведомление собеседнику
	apiServer.SetChat(chatSvc)

	// Медиа в чате (голосовые/фото) через Cloudflare R2. Если R2_* не заданы —
	// фича просто отключена (r2 == nil).
	if r2, err := storage.NewR2FromEnv(); err != nil {
		log.Printf("R2 init: %v (голосовые отключены)", err)
	} else if r2 != nil {
		apiServer.SetMedia(r2)
		log.Println("R2 media storage подключён — голосовые доступны")
	}
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
	var bot *tgbotapi.BotAPI
	if !cfg.Telegram.Enabled {
		log.Println("📴 Telegram выключен (TELEGRAM_ENABLED=0) — работаем на web push + WhatsApp")
	} else {
		var err error
		bot, err = tgbotapi.NewBotAPI(cfg.Telegram.BotToken)
		if err != nil {
			// Вне production живём без Telegram: HTTP API и веб-кабинет работают,
			// бот просто отключён (локальная разработка без реального токена).
			if cfg.Env == "production" {
				log.Fatalf("❌ Telegram: %v", err)
			}
			log.Printf("⚠️ Telegram недоступен (%v) — запускаемся без бота (dev)", err)
			bot = nil
		}
	}
	if bot != nil {
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
	}

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
	apiServer.SetNotifyGroupRepo(notifyGroupRepo)
	apiServer.SetAwardService(awardSvc)
	apiServer.SetStatsRepo(statsRepo)
	apiServer.SetDoubleElim(deRepo, deSvc)

	webPush := api.NewWebPushNotifier(pushRepo, cfg.API.VAPIDPublic, cfg.API.VAPIDPrivate, cfg.API.VAPIDSubject)
	apiServer.SetPush(pushRepo, webPush)
	apiServer.SetSettingsRepo(settingsRepo)
	apiServer.SetFriendlyRepo(friendlyRepo)
	if cfg.API.VAPIDPublic != "" {
		log.Println("🔔 Web Push включён")
	}

	// ── Групповые уведомления (Telegram-группа + WhatsApp) ───────────
	groupHub := groupcast.NewHub()
	tgGroupSink = groupcast.NewTelegramGroup(bot, settingsRepo, cfg.Telegram.GroupID)
	notifyGroupSink = notifyGroupRepo
	adminTelegramID = cfg.Admin.TelegramID
	groupHub.Add(tgGroupSink)
	telegramNotifier.SetGroups(groupHub)
	apiServer.SetTGGroup(tgGroupSink)
	apiServer.SetGroupHub(groupHub)
	if tgGroupSink.ChatID() != 0 {
		log.Printf("👥 Telegram-группа подключена: %d", tgGroupSink.ChatID())
	}
	if waClient, err := wa.NewFromEnv(cfg.Postgres.DSN, settingsRepo); err != nil {
		log.Printf("⚠️ WhatsApp не запустился: %v", err)
	} else if waClient != nil {
		groupHub.Add(waClient)
		apiServer.SetWhatsApp(waClient)
		log.Println("💬 WhatsApp-канал включён (WA_ENABLED=1)")
	}

	reminderSvc := service.NewReminderService(deadlineRepo, matchRepo, leagueRepo, userRepo, telegramNotifier)
	reminderSvc.SetGroups(groupHub)
	reminderSvc.SetNotifications(notifSvc)

	// Исполнение дедлайнов: по истечении срока автоматика закрывает несыгранные
	// матчи (тур — тех. ничья 0:0, плей-офф — тех. победа сида, заявленный счёт —
	// авто-подтверждение). Прогон на старте — Render мог проспать дедлайн.
	deadlineSvc := service.NewDeadlineService(deadlineRepo, matchRepo, leagueRepo, userRepo, matchSvc)
	deadlineSvc.SetNotifications(notifSvc)
	deadlineSvc.SetGroups(groupHub)
	deadlineSvc.SetEloApplier(apiServer.ApplyEloByIDs)
	matchSvc.SetChampionNews(apiServer.NewsChampion)

	// Сезоны: закрытие с церемонией, номинации, итоговый пост в группу.
	seasonSvc := service.NewSeasonService(leagueRepo, awardRepo, userRepo)
	seasonSvc.SetNotifications(notifSvc)
	seasonSvc.SetGroups(groupHub)
	seasonSvc.SetPredictions(predRepo)
	apiServer.SetSeasonService(seasonSvc)
	apiServer.SetPredictionRepo(predRepo)

	// Прогнозы: очки начисляются при подтверждении матча (любой путь —
	// ручной, админский, авто-дедлайн); авторы точных прогнозов получают
	// поздравление на своём языке.
	matchSvc.SetPredictionScorer(func(ctx context.Context, m *models.Match) {
		exact, err := predRepo.ScoreMatch(ctx, m.ID, *m.HomeGoals, *m.AwayGoals)
		if err != nil {
			log.Printf("prediction scoring (match %d): %v", m.ID, err)
			return
		}
		if len(exact) > 0 && notifSvc != nil {
			link := fmt.Sprintf("/leagues/details?id=%d&tab=predict", m.LeagueID)
			notifSvc.NotifyT(ctx, exact, "system", link, func(lang string) (string, string) {
				return i18n.T(lang, "predict.exact.title"),
					fmt.Sprintf(i18n.T(lang, "predict.exact.body"), *m.HomeGoals, *m.AwayGoals)
			})
		}
	})
	go func() {
		if err := deadlineSvc.EnforceDue(context.Background()); err != nil {
			log.Printf("startup deadline enforce: %v", err)
		}
	}()

	// ── Периодические задачи ─────────────────────────────────────────
	go func() {
		cacheTicker := time.NewTicker(5 * time.Minute)
		rankTicker := time.NewTicker(time.Hour) // было 5м — жгло Neon-квоту впустую
		reminderTicker := time.NewTicker(5 * time.Minute)
		friendlyTicker := time.NewTicker(time.Hour)
		defer cacheTicker.Stop()
		defer rankTicker.Stop()
		defer reminderTicker.Stop()
		defer friendlyTicker.Stop()
		// idle — никто не заходил в приложение: не будим уснувшую БД (Neon
		// Free тарифицирует compute-часы). Всё наверстается при первом визите:
		// прогон дедлайнов идёт и на старте, и на первом тике после активности.
		idle := func() bool { return time.Since(apiServer.LastActivityAt()) > 20*time.Minute }
		for {
			select {
			case <-cacheTicker.C:
				api.CleanupAllCaches()
			case <-rankTicker.C:
				if idle() {
					continue
				}
				ctx := context.Background()
				if err := userRepo.RecalculateAllRanks(ctx); err != nil {
					log.Printf("periodic RecalculateAllRanks: %v", err)
				}
			case <-reminderTicker.C:
				if idle() {
					continue
				}
				ctx := context.Background()
				if err := deadlineSvc.EnforceDue(ctx); err != nil {
					log.Printf("periodic deadline enforce: %v", err)
				}
				if err := reminderSvc.CheckAndSend(ctx); err != nil {
					log.Printf("periodic reminder check: %v", err)
				}
			case <-friendlyTicker.C:
				if idle() {
					continue
				}
				if err := apiServer.ExpireStaleFriendlies(context.Background()); err != nil {
					log.Printf("periodic friendly expire: %v", err)
				}
			}
		}
	}()

	var wg sync.WaitGroup
	var jobs chan tgbotapi.Update
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Обработка Telegram-апдейтов — только при живом боте (в dev без токена
	// весь веб-кабинет работает, а бот просто выключен).
	if bot != nil {
		h := handlers.New(bot, userRepo, leagueRepo, matchRepo, matchSvc, schedSvc, groupStageSvc, adminRepo, eloSvc, cfg.Admin.TelegramID, cfg.Telegram.GroupID)
		h.SetAchievementRepo(achievRepo)
		ah := handlers.NewAdminHandlers(bot, userRepo, leagueRepo, adminRepo, cfg.Admin.TelegramID)

		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60

		const numWorkers = 150 // было 50 — больше параллельных обработчиков
		updates := bot.GetUpdatesChan(u)
		jobs = make(chan tgbotapi.Update, 1000) // было 200 — больше буфер очереди

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
	}

	<-quit
	log.Println("🛑 Завершение работы...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)

	if bot != nil {
		bot.StopReceivingUpdates()
		close(jobs)
	}
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

	// Групповые чаты: бот молчит, кроме команд подключения уведомлений —
	// иначе любая реплика в группе спамила бы «направляем на сайт».
	if msg.Chat.IsGroup() || msg.Chat.IsSuperGroup() {
		cmd := strings.ToLower(strings.TrimSuffix(strings.Fields(text + " ")[0], "@"+bot.Self.UserName))
		switch cmd {
		case "/connect", "/подключить":
			handleGroupConnect(ctx, bot, msg, userRepo, true)
		case "/disconnect", "/отключить":
			handleGroupConnect(ctx, bot, msg, userRepo, false)
		}
		return
	}

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

// tgGroupSink — канал «Telegram-группа» (задаётся в main), команды /connect
// и /disconnect в группе подключают/отключают её к уведомлениям турнира.
var tgGroupSink *groupcast.TelegramGroup

// notifyGroupSink — реестр подключённых групп (мульти-группы). /connect в группе
// добавляет её сюда, /disconnect — убирает.
var notifyGroupSink repository.NotifyGroupRepository

// adminTelegramID — Telegram супер-админа из конфига (для проверки прав).
var adminTelegramID int64

func handleGroupConnect(ctx context.Context, bot *tgbotapi.BotAPI, msg *tgbotapi.Message, _ repository.UserRepository, connect bool) {
	if tgGroupSink == nil {
		return
	}
	// Право подключать группу — только у супер-админа (ADMIN_TELEGRAM_ID).
	if adminTelegramID == 0 || msg.From == nil || msg.From.ID != adminTelegramID {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "⛔ Подключать уведомления может только администратор турнира.")
		_, _ = bot.Send(reply)
		return
	}
	chatIDStr := strconv.FormatInt(msg.Chat.ID, 10)
	var replyText string
	if connect {
		if err := tgGroupSink.SetChatID(ctx, msg.Chat.ID); err != nil {
			replyText = "❌ Не удалось сохранить настройку, попробуйте ещё раз."
		} else {
			// Добавляем в реестр мульти-групп (в админке можно будет привязать
			// к конкретной лиге). Дубли не плодятся — Upsert по channel+chat_id.
			if notifyGroupSink != nil {
				if _, err := notifyGroupSink.Upsert(ctx, "telegram", chatIDStr, msg.Chat.Title); err != nil {
					log.Printf("notify group upsert: %v", err)
				}
			}
			replyText = "✅ Группа подключена!\nСюда будут приходить: результаты матчей, жеребьёвки, напоминания о дедлайнах и объявления.\n\nВ админке (Интеграции) можно привязать эту группу к конкретной лиге."
		}
	} else {
		if err := tgGroupSink.SetChatID(ctx, 0); err != nil {
			replyText = "❌ Не удалось сохранить настройку, попробуйте ещё раз."
		} else {
			if notifyGroupSink != nil {
				if err := notifyGroupSink.DeleteByChat(ctx, "telegram", chatIDStr); err != nil {
					log.Printf("notify group delete: %v", err)
				}
			}
			replyText = "🔕 Группа отключена от уведомлений турнира."
		}
	}
	_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, replyText))
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
		// Идемпотентность: код мог быть уже использован при двойной обработке
		// апдейта (например, два инстанса бота) или повторном клике. Если этот
		// Telegram уже привязан — не пугаем пользователя ложной ошибкой.
		if existing, _ := userRepo.GetByTelegramID(ctx, msg.From.ID); existing != nil {
			return
		}
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

// isDBConnError отличает «база недоступна» (квота Neon, сеть, спящий compute)
// от битой схемы: при недоступности стартуем в режиме самовосстановления,
// при ошибке схемы в production — падаем (fail-fast).
func isDBConnError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range []string{
		"failed to connect", "dial", "quota", "connection refused",
		"network is unreachable", "no such host", "timeout", "connection reset",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func seedSuperAdmin(ctx context.Context, adminRepo repository.AdminRepository, cfg *config.Config) {
	if cfg.Admin.Username == "" || cfg.Admin.Password == "" {
		log.Println("⚠️  ADMIN_USERNAME/ADMIN_PASSWORD не заданы — супер-администратор не создан")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.Admin.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("❌ bcrypt: %v", err)
		return
	}
	exists, err := adminRepo.SuperAdminExists(ctx)
	if err != nil {
		log.Printf("⚠️  Проверка супер-администратора: %v", err)
		return
	}
	if exists {
		// Креды управляются через env (Render): смена ADMIN_USERNAME/PASSWORD
		// должна работать без ручных манипуляций с БД — синхронизируем хеш.
		if err := adminRepo.SyncSuperAdminCredential(ctx, cfg.Admin.Username, string(hash)); err != nil {
			log.Printf("⚠️  Синхронизация пароля супер-администратора: %v", err)
			return
		}
		// Длина логина в логе помогает заметить скрытый пробел/перенос в env.
		log.Printf("✅ Креды супер-администратора синхронизированы из env: логин=%q (%d симв.), пароль=%d симв.",
			cfg.Admin.Username, len(cfg.Admin.Username), len(cfg.Admin.Password))
		return
	}
	if err := adminRepo.SeedSuperAdmin(ctx, cfg.Admin.Username, string(hash), "Супер-Администратор"); err != nil {
		log.Printf("❌ Seed super admin: %v", err)
		return
	}
	log.Printf("✅ Супер-администратор создан: %s", cfg.Admin.Username)
}
