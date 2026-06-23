package apiApp

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
	"xmine/litebans-api/internal/repository"
	"xmine/litebans-api/internal/service"
	"xmine/litebans-api/internal/transport/httpapi"
)

type ApiApp struct {
	cfg    config.Config
	db     *sql.DB
	server *http.Server
}

// New wires up the database connection, repositories, services, router, and HTTP server.
func New(ctx context.Context, cfg config.Config) (*ApiApp, error) {
	conn, err := db.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}

	punishmentRepo := repository.NewPunishmentRepository(conn, cfg.TablePrefix)
	historyRepo := repository.NewHistoryRepository(conn, cfg.TablePrefix)

	playerSvc := service.NewPlayerService(historyRepo, cfg.ConsoleAliases)
	idObfuscator := service.NewIDObfuscator(cfg.ObfuscateIDs, cfg.ObfuscationSecret)
	punishmentSvc := service.NewPunishmentService(
		punishmentRepo, historyRepo, playerSvc, idObfuscator,
		cfg.IncludeInactiveDefault, cfg.IncludeSilentDefault,
		cfg.DefaultPageSize, cfg.MaxPageSize,
	)

	jwtValidator, err := auth.NewValidator(cfg.JWTPublicKeyPath, cfg.JWTIssuer)
	if err != nil {
		return nil, err
	}
	authorityClient := auth.NewAuthorityClient(cfg.AuthorityAPIURL, cfg.InternalToken, cfg.AuthorityCacheTTL)
	authComp := httpapi.NewAuth(jwtValidator, authorityClient, cfg.ModPermission)

	mux := httpapi.NewRouter(punishmentSvc, playerSvc, authComp, authorityClient, cfg.PublicTypes, cfg.ModPermission)

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: mux,
	}

	return &ApiApp{cfg: cfg, db: conn, server: server}, nil
}

func (a *ApiApp) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		slog.Info("Starting HTTP API server", slog.String("addr", a.cfg.HTTPAddr))
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		slog.Info("Shutting down HTTP API server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			slog.Error("Error during server shutdown", slog.Any("error", err))
		}
		a.db.Close()
		return <-errCh
	case err := <-errCh:
		a.db.Close()
		return err
	}
}
