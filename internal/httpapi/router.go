package httpapi

import (
	"log/slog"
	"net/http"
	"xmine/litebans-api/api"
	"xmine/litebans-api/internal/auth"
	"xmine/litebans-api/internal/middleware"
	"xmine/litebans-api/internal/service"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewRouter registers all v1 routes on a fresh http.ServeMux. Authorization is
// enforced by an AuthPolicy: explicit per-operation rules win, anything not
// listed falls back to Default (RequireJWT — this service does not yet accept
// calls from other services, so there's no internal-token default here).
// /health and /metrics are registered directly on the top-level mux and are
// not wrapped by middleware.Observability — they're not part of the API
// contract and don't need request logging or HTTP metrics of their own.
func NewRouter(
	punishmentSvc *service.PunishmentService,
	playerSvc *service.PlayerService,
	authComp *middleware.Auth,
	authorityClient *auth.AuthorityClient,
	modPermission string,
	log *slog.Logger,
) http.Handler {
	srv := NewServer(punishmentSvc, playerSvc, authorityClient, modPermission)

	policy := middleware.AuthPolicy{
		Default: []middleware.AuthStep{authComp.RequireJWT()},
		Rules: []middleware.AuthRule{
			{Operations: []string{"GetPublicPunishments", "GetPublicPunishmentsStats", "GetPublicLookup", "PostPublicLookupBatch", "GetPublicPunishmentByID"}, Chain: []middleware.AuthStep{middleware.AllowAnonymous}},
			{Operations: []string{
				"GetPlayerPunishmentsMe", "GetPlayerPunishmentsMeBan", "GetPlayerPunishmentsMeMute",
				"GetPlayerPunishmentsMeWarning", "GetPlayerPunishmentsMeKick", "GetPlayerPunishmentByID",
			}, Chain: []middleware.AuthStep{authComp.RequireJWT()}},
			{Operations: []string{
				"GetModPunishments", "GetModPunishmentsBan", "GetModPunishmentsMute",
				"GetModPunishmentsWarning", "GetModPunishmentsKick", "GetModPunishmentByID",
			}, Chain: []middleware.AuthStep{authComp.RequireJWT(), authComp.RequireModPermission()}},
		},
	}

	strictHandler := api.NewStrictHandlerWithOptions(srv, []api.StrictMiddlewareFunc{middleware.Authorize(policy)}, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  writeError,
		ResponseErrorHandlerFunc: writeError,
	})

	apiMux := http.NewServeMux()
	api.HandlerWithOptions(strictHandler, api.StdHTTPServerOptions{BaseRouter: apiMux})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.Handle("/", middleware.Observability(log)(apiMux))

	return mux
}
