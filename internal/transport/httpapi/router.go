package httpapi

import (
	"net/http"
	"xmine/litebans-api/api"
	"xmine/litebans-api/internal/auth"
	"xmine/litebans-api/internal/service"
)

// NewRouter registers all v1 routes on a fresh http.ServeMux. Authorization
// (JWT/permission requirements per operation) is applied by Auth.StrictMiddleware,
// dispatched by operationId since the generated strict server has a single handler
// signature for every operation.
func NewRouter(
	punishmentSvc *service.PunishmentService,
	playerSvc *service.PlayerService,
	authComp *Auth,
	authorityClient *auth.AuthorityClient,
	publicTypes []string,
	modPermission string,
) http.Handler {
	srv := NewServer(punishmentSvc, playerSvc, authorityClient, publicTypes, modPermission)

	strictHandler := api.NewStrictHandlerWithOptions(srv, []api.StrictMiddlewareFunc{authComp.StrictMiddleware}, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  writeError,
		ResponseErrorHandlerFunc: writeError,
	})

	return api.HandlerWithOptions(strictHandler, api.StdHTTPServerOptions{BaseRouter: http.NewServeMux()})
}
