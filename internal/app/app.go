package app

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"time"
	"xmine/litebans-api/internal/auth"
	"xmine/litebans-api/internal/config"
	"xmine/litebans-api/internal/db"
	"xmine/litebans-api/internal/httpapi"
	"xmine/litebans-api/internal/middleware"
	"xmine/litebans-api/internal/repository"
	"xmine/litebans-api/internal/service"
)

type App struct {
	cfg    config.Config
	db     *sql.DB
	server *http.Server
	log    *slog.Logger
}

// New wires up the database connection, repositories, services, router, and HTTP server.
func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*App, error) {
	conn, err := db.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}

	punishmentRepo := repository.NewPunishmentRepository(conn, cfg.TablePrefix)
	historyRepo := repository.NewHistoryRepository(conn, cfg.TablePrefix)

	playerSvc := service.NewPlayerService(historyRepo, cfg.ConsoleAliases)
	idObfuscator := service.NewIDObfuscator(cfg.ObfuscateIDs, cfg.ObfuscationSecret)
	punishmentSvc := service.NewPunishmentService(
		punishmentRepo, playerSvc, idObfuscator,
		cfg.IncludeInactiveDefault, cfg.IncludeSilentDefault,
		cfg.DefaultPageSize, cfg.MaxPageSize,
	)

	jwtValidator, err := auth.NewValidator(cfg.JWTPublicKeyPath, cfg.JWTIssuer)
	if err != nil {
		return nil, err
	}
	authorityClient := auth.NewAuthorityClient(cfg.AuthorityAPIURL, cfg.InternalToken, cfg.AuthorityCacheTTL)
	authComp := middleware.NewAuth(jwtValidator, authorityClient, cfg.ModPermission)

	mux := httpapi.NewRouter(punishmentSvc, playerSvc, authComp, authorityClient, cfg.ModPermission, log)

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: mux,
	}

	return &App{cfg: cfg, db: conn, server: server, log: log}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		a.log.Info("Starting HTTP API server", slog.String("addr", a.cfg.HTTPAddr))
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		a.log.Info("Shutting down HTTP API server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			a.log.Error("Error during server shutdown", slog.Any("error", err))
		}
		a.db.Close()
		return <-errCh
	case err := <-errCh:
		a.db.Close()
		return err
	}
}
