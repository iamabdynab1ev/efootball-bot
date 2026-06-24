package config

import (
	"log"
	"os"
	"strconv"

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

	botToken := mustEnv("BOT_TOKEN")
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

	jwtSecret := os.Getenv("JWT_SECRET")
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
			BotToken:    botToken,
			BotUsername: os.Getenv("BOT_USERNAME"),
			GroupID:     groupID,
		},
		Postgres: PostgresConfig{
			DSN: dsn,
		},
		Admin: AdminConfig{
			TelegramID: adminID,
			Username:   os.Getenv("ADMIN_USERNAME"),
			Password:   os.Getenv("ADMIN_PASSWORD"),
		},
		API: APIConfig{
			Port:           port,
			JWTSecret:      jwtSecret,
			GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"),
			FrontendURL:    os.Getenv("FRONTEND_URL"),
			VAPIDPublic:    os.Getenv("VAPID_PUBLIC_KEY"),
			VAPIDPrivate:   os.Getenv("VAPID_PRIVATE_KEY"),
			VAPIDSubject:   cmpOr(os.Getenv("VAPID_SUBJECT"), "mailto:admin@efootleague.app"),
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
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("❌ Обязательная переменная окружения не задана: %s", key)
	}
	return v
}
