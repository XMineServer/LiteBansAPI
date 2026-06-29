package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"xmine/litebans-api/internal/app"
	"xmine/litebans-api/internal/config"
	"xmine/litebans-api/internal/logging"
)

const serviceName = "litebans-api"

func main() {
	cfg, err := config.Load()
	if err != nil {
		logging.New(serviceName, slog.LevelInfo, logging.LogFormatJSON).
			Error("failed to load config", "err", err)
		os.Exit(1)
	}

	log := logging.New(serviceName, cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(log)
	log.Info("config loaded", "http_addr", cfg.HTTPAddr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application, err := app.New(ctx, cfg, log)
	if err != nil {
		log.Error("failed to initialize service", "err", err)
		os.Exit(1)
	}

	if err := application.Run(ctx); err != nil {
		log.Error("service exited with error", "err", err)
		os.Exit(1)
	}
}
