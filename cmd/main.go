package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"xmine/litebans-api/internal/apiApp"
	"xmine/litebans-api/internal/config"
	"xmine/litebans-api/internal/logger"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("env: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	logger.SetupLogger(cfg.LogFormat, cfg.LogLevel)
	slog.Info("Loaded config", slog.Any("config", cfg))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := apiApp.New(ctx, cfg)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	if err := app.Run(ctx); err != nil {
		log.Fatalf("run app: %v", err)
	}
}
