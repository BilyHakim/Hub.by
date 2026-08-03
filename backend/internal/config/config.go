package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	DatabaseURL         string
	FrontendOrigin      string
	AuthEmail           string
	AuthInitialPassword string
	OMDbAPIKey          string
	TelegramBotToken    string
	TelegramPairingCode string
	TelegramLocalUserID int64
	TelegramTimezone    string
}

func Load() Config {
	// Local development can use backend/.env. Existing process environment
	// variables keep precedence for production and container deployments.
	_ = godotenv.Load()

	return Config{
		Port:                valueOrDefault("APP_PORT", "8080"),
		DatabaseURL:         valueOrDefault("DATABASE_URL", "postgres://hubby:hubby@localhost:5432/hubby?sslmode=disable"),
		FrontendOrigin:      valueOrDefault("FRONTEND_ORIGIN", "http://localhost:5173"),
		AuthEmail:           valueOrDefault("AUTH_EMAIL", "bily@hubby.local"),
		AuthInitialPassword: os.Getenv("AUTH_INITIAL_PASSWORD"),
		OMDbAPIKey:          os.Getenv("OMDB_API_KEY"),
		TelegramBotToken:    os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramPairingCode: os.Getenv("TELEGRAM_PAIRING_CODE"),
		TelegramLocalUserID: int64OrDefault("TELEGRAM_LOCAL_USER_ID", 1),
		TelegramTimezone:    valueOrDefault("TELEGRAM_TIMEZONE", "Asia/Jakarta"),
	}
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func int64OrDefault(key string, fallback int64) int64 {
	value, err := strconv.ParseInt(os.Getenv(key), 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
