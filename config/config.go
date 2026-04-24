package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Telegram TelegramConfig
	Postgres PostgresConfig
	Admin    AdminConfig
}

type TelegramConfig struct {
	BotToken string
	GroupID  int64
}

type PostgresConfig struct {
	DSN string
}

type AdminConfig struct {
	TelegramID int64 // главный администратор (ты)
}

func Load() *Config {
	_ = godotenv.Load()

	// BOT_TOKEN and POSTGRES_DSN are required
	botToken := mustEnv("BOT_TOKEN")
	dsn := mustEnv("POSTGRES_DSN")

	// GROUP_CHAT_ID and ADMIN_TELEGRAM_ID are optional (default 0)
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

	return &Config{
		Telegram: TelegramConfig{
			BotToken: botToken,
			GroupID:  groupID,
		},
		Postgres: PostgresConfig{
			DSN: dsn,
		},
		Admin: AdminConfig{
			TelegramID: adminID,
		},
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("❌ Обязательная переменная окружения не задана: %s", key)
	}
	return v
}
