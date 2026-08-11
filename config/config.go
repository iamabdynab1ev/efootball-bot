package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Env      string // "production" | ""
	Telegram TelegramConfig
	Postgres PostgresConfig
	Admin    AdminConfig
	API      APIConfig
}

type TelegramConfig struct {
	// Enabled — рубильник всего Telegram (бот, уведомления, привязка).
	// TELEGRAM_ENABLED=0 выключает канал: проект работает на web push +
	// WhatsApp, код Telegram остаётся на месте.
	Enabled     bool
	BotToken    string
	BotUsername string
	GroupID     int64
}

type PostgresConfig struct {
	DSN string
}

type AdminConfig struct {
	TelegramID int64
	Username   string
	Password   string
}

type APIConfig struct {
	Port           string
	JWTSecret      string
	GoogleClientID string
	FrontendURL    string
	VAPIDPublic    string
	VAPIDPrivate   string
	VAPIDSubject   string
}

func Load() *Config {
	_ = godotenv.Load()

	// TELEGRAM_ENABLED=0 — канал выключен, BOT_TOKEN тогда не обязателен.
	tgEnabled := os.Getenv("TELEGRAM_ENABLED") != "0"
	botToken := os.Getenv("BOT_TOKEN")
	if tgEnabled && botToken == "" {
		log.Fatal("❌ Обязательная переменная окружения не задана: BOT_TOKEN (или выключите Telegram: TELEGRAM_ENABLED=0)")
	}
	dsn := mustEnv("POSTGRES_DSN")

	var groupID int64
	if v := os.Getenv("GROUP_CHAT_ID"); v != "" {
		if g, err := strconv.ParseInt(v, 10, 64); err == nil {
			groupID = g
		} else {
			log.Printf("warning: invalid GROUP_CHAT_ID: %v", err)
		}
	}

	var adminID int64
	if v := os.Getenv("ADMIN_TELEGRAM_ID"); v != "" {
		if a, err := strconv.ParseInt(v, 10, 64); err == nil {
			adminID = a
		} else {
			log.Printf("warning: invalid ADMIN_TELEGRAM_ID: %v", err)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		log.Fatal("❌ JWT_SECRET не задан")
	}
	// HS256: ключ короче 32 байт уязвим к брутфорсу. В production — жёстко.
	if len(jwtSecret) < 32 {
		if os.Getenv("APP_ENV") == "production" {
			log.Fatal("❌ JWT_SECRET слишком короткий: минимум 32 символа в production")
		}
		log.Printf("⚠️  JWT_SECRET короче 32 символов — небезопасно для production")
	}

	return &Config{
		Env: os.Getenv("APP_ENV"),
		Telegram: TelegramConfig{
			Enabled:     tgEnabled,
			BotToken:    botToken,
			BotUsername: os.Getenv("BOT_USERNAME"),
			GroupID:     groupID,
		},
		Postgres: PostgresConfig{
			DSN: dsn,
		},
		Admin: AdminConfig{
			TelegramID: adminID,
			// TrimSpace: при копипасте в дашборд Render в значение часто попадает
			// невидимый хвостовой пробел/перенос строки — иначе bcrypt не сходится
			// и вход админа падает с «неверный логин/пароль».
			Username: strings.TrimSpace(os.Getenv("ADMIN_USERNAME")),
			Password: strings.TrimSpace(os.Getenv("ADMIN_PASSWORD")),
		},
		API: APIConfig{
			Port:           port,
			JWTSecret:      jwtSecret,
			GoogleClientID: strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
			FrontendURL:    strings.TrimSpace(os.Getenv("FRONTEND_URL")),
			VAPIDPublic:    strings.TrimSpace(os.Getenv("VAPID_PUBLIC_KEY")),
			VAPIDPrivate:   strings.TrimSpace(os.Getenv("VAPID_PRIVATE_KEY")),
			VAPIDSubject:   cmpOr(strings.TrimSpace(os.Getenv("VAPID_SUBJECT")), "mailto:admin@efootleague.app"),
		},
	}
}

// cmpOr возвращает a, если не пусто, иначе b.
func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func mustEnv(key string) string {
	// TrimSpace защищает от случайного переноса строки/пробела в значении env
	// (например, при копировании DSN в дашборд) — иначе URL не парсится.
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		log.Fatalf("❌ Обязательная переменная окружения не задана: %s", key)
	}
	return v
}
