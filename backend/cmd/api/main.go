package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hubby-id/hubby/backend/internal/config"
	"github.com/hubby-id/hubby/backend/internal/database"
	"github.com/hubby-id/hubby/backend/internal/httpapi"
	"github.com/hubby-id/hubby/backend/internal/telegrambot"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	authConfigured, err := httpapi.BootstrapAuth(ctx, pool, cfg.AuthEmail, cfg.AuthInitialPassword)
	if err != nil {
		logger.Error("authentication configuration failed", "error", err)
		os.Exit(1)
	}
	if !authConfigured {
		logger.Warn("no application password is configured; set AUTH_INITIAL_PASSWORD before first login")
	}

	if cfg.TelegramBotToken != "" {
		bot, err := telegrambot.New(pool, logger, telegrambot.Config{
			Token:        cfg.TelegramBotToken,
			PairingCode:  cfg.TelegramPairingCode,
			LocalUserID:  cfg.TelegramLocalUserID,
			TimezoneName: cfg.TelegramTimezone,
		})
		if err != nil {
			logger.Error("Telegram bot configuration failed", "error", err)
			os.Exit(1)
		}
		go func() {
			if err := bot.Run(ctx); err != nil {
				logger.Error("Telegram bot stopped", "error", err)
			}
		}()
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(pool, logger, cfg.FrontendOrigin, cfg.TMDBAPIToken),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("hubby finance API started", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
