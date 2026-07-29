package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	DatabaseURL    string
	FrontendOrigin string
}

func Load() Config {
	// Local development can use backend/.env. Existing process environment
	// variables keep precedence for production and container deployments.
	_ = godotenv.Load()

	return Config{
		Port:           valueOrDefault("APP_PORT", "8080"),
		DatabaseURL:    valueOrDefault("DATABASE_URL", "postgres://hubby:hubby@localhost:5432/hubby?sslmode=disable"),
		FrontendOrigin: valueOrDefault("FRONTEND_ORIGIN", "http://localhost:5173"),
	}
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
