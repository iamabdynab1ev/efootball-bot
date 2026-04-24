package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"path/filepath"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"efootball-bot/config"
	"efootball-bot/internal/bot/handlers"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"efootball-bot/internal/service"
)

func main() {
	cfg := config.Load()

	// ── Миграции ───────────────────────────────────────────────────
	dbGoose, err := sql.Open("pgx", cfg.Postgres.DSN)
	if err != nil {
		log.Fatalf("❌ Миграции: %v", err)
	}
	goose.SetDialect("postgres")

	exe, _ := os.Executable()
	migrationsPath := "./migrations"
	if exe != "" {
		tryPath := filepath.Join(filepath.Dir(exe), "migrations")
		if _, err := os.Stat(tryPath); err == nil {
			migrationsPath = tryPath
		}
	}
	if err := goose.Up(dbGoose, migrationsPath); err != nil {
		log.Printf("⚠️ Миграции: %v", err)
	}
	dbGoose.Close()

	// ── БД пул ─────────────────────────────────────────────────────
	poolConfig, err := pgxpool.ParseConfig(cfg.Postgres.DSN)
	if err != nil {
		log.Fatalf("❌ pgxpool config: %v", err)
	}

	poolConfig.MaxConns = 20
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		log.Fatalf("❌ pgxpool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("❌ БД недоступна: %v", err)
	}
	log.Println("✅ БД соединение установлено")

	// ── Репозитории ────────────────────────────────────────────────
	userRepo := repository.NewUserRepository(pool)
	leagueRepo := repository.NewLeagueRepository(pool)
	matchRepo := repository.NewMatchRepository(pool)
	adminRepo := repository.NewAdminRepository(pool)

	// ── Сервисы ────────────────────────────────────────────────────
	matchSvc := service.NewMatchService(matchRepo, leagueRepo)
	schedSvc := service.NewScheduleService(matchRepo, leagueRepo)
	eloSvc := service.NewEloService(userRepo)

	// ── Bot ────────────────────────────────────────────────────────
	bot, err := tgbotapi.NewBotAPI(cfg.Telegram.BotToken)
	if err != nil {
		log.Fatalf("❌ Telegram билан боғланишда хатолик: %v", err)
	}

	_, _ = bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})

	adminID := cfg.Admin.TelegramID
	groupID := cfg.Telegram.GroupID

	h := handlers.New(bot, userRepo, leagueRepo, matchRepo, matchSvc, schedSvc, adminRepo, eloSvc, adminID, groupID)
	ah := handlers.NewAdminHandlers(bot, userRepo, leagueRepo, adminRepo, adminID)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	const numWorkers = 50
	updates := bot.GetUpdatesChan(u)
	jobs := make(chan tgbotapi.Update, 200)

	var wg sync.WaitGroup
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for update := range jobs {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				processUpdate(ctx, update, h, ah)
				cancel()
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
	log.Println("🛑 Сигнал остановки получен. Ждем завершения процессов...")
	bot.StopReceivingUpdates()
	close(jobs)
	wg.Wait()
	log.Println("✅ Все воркеры завершены, бот остановлен.")
}

func processUpdate(ctx context.Context, update tgbotapi.Update, h *handlers.Handler, ah *handlers.AdminHandlers) {
	if update.CallbackQuery != nil {
		cb := update.CallbackQuery
		if !ah.HandleCallback(ctx, cb) {
			h.HandleCallback(ctx, cb)
		}
		return
	}

	if update.Message == nil {
		return
	}

	msg := update.Message
	text := strings.TrimSpace(msg.Text)

	switch {
	case text == "/start":
		h.HandleStart(ctx, msg)

	case strings.HasPrefix(text, "/join "):
		leagueName := strings.TrimSpace(strings.TrimPrefix(text, "/join "))
		h.HandleJoin(ctx, msg, leagueName)

	case strings.HasPrefix(text, "/result "):
		parts := strings.Fields(strings.TrimPrefix(text, "/result "))
		if len(parts) == 2 {
			matchID, err1 := strconv.ParseInt(parts[0], 10, 64)
			score := strings.Split(parts[1], ":")
			if err1 == nil && len(score) == 2 {
				hg, err2 := strconv.ParseInt(score[0], 10, 16)
				ag, err3 := strconv.ParseInt(score[1], 10, 16)
				if err2 == nil && err3 == nil {
					h.HandleResult(ctx, msg, matchID, int16(hg), int16(ag))
				} else {
					h.Send(msg.Chat.ID, "❗ Ошибка в счёте. Пример: 3:1")
				}
			} else {
				h.Send(msg.Chat.ID, "❗ Неверный формат ID матча.")
			}
		} else {
			h.Send(msg.Chat.ID, "❗ Формат: `/result <matchID> 3:1`")
		}

	case strings.HasPrefix(text, "/confirm_"):
		id, err := strconv.ParseInt(strings.TrimPrefix(text, "/confirm_"), 10, 64)
		if err == nil {
			h.HandleConfirm(ctx, msg, id)
		}

	case strings.HasPrefix(text, "/dispute_"):
		id, err := strconv.ParseInt(strings.TrimPrefix(text, "/dispute_"), 10, 64)
		if err == nil {
			h.HandleDispute(ctx, msg, id)
		}

	case strings.HasPrefix(text, "/admin resolve "):
		parts := strings.Fields(strings.TrimPrefix(text, "/admin resolve "))
		if len(parts) == 2 {
			matchID, err1 := strconv.ParseInt(parts[0], 10, 64)
			score := strings.Split(parts[1], ":")
			if err1 == nil && len(score) == 2 {
				hg, err2 := strconv.ParseInt(score[0], 10, 16)
				ag, err3 := strconv.ParseInt(score[1], 10, 16)
				if err2 == nil && err3 == nil {
					h.AdminResolve(ctx, msg, matchID, int16(hg), int16(ag))
				}
			}
		}

	case strings.HasPrefix(text, "/admin add "):
		parts := strings.Fields(strings.TrimPrefix(text, "/admin add "))
		if len(parts) >= 1 {
			targetID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				role := models.RoleAdmin
				if len(parts) >= 2 && parts[1] == "super" {
					role = models.RoleSuperAdmin
				}
				ah.AddAdmin(ctx, msg.Chat.ID, msg.From.ID, targetID, role)
			}
		}

	case strings.HasPrefix(text, "/admin remove "):
		targetStr := strings.TrimSpace(strings.TrimPrefix(text, "/admin remove "))
		targetID, err := strconv.ParseInt(targetStr, 10, 64)
		if err == nil {
			ah.RemoveAdmin(ctx, msg.Chat.ID, msg.From.ID, targetID)
		}

	default:
		h.HandleMessage(ctx, msg)
	}
}


