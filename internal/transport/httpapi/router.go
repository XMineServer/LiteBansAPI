package httpapi

import (
	"net/http"
	"xmine/litebans-api/internal/auth"
	"xmine/litebans-api/internal/service"
)

// NewRouter registers all v1 routes on a fresh http.ServeMux, grouped by access level:
// /public (anonymous), /player (JWT), /mod (JWT + permission), plus the contextual
// /punishments/{type}/{id} which authorizes after loading the record.
func NewRouter(
	punishmentSvc *service.PunishmentService,
	playerSvc *service.PlayerService,
	authComp *Auth,
	authorityClient *auth.AuthorityClient,
	publicTypes []string,
	modPermission string,
) *http.ServeMux {
	mux := http.NewServeMux()

	publicPunishments := NewPunishmentsHandler(punishmentSvc, publicTypes, authorityClient, modPermission)
	players := NewPlayersHandler(playerSvc)
	player := NewPlayerHandler(punishmentSvc)
	mod := NewModHandler(punishmentSvc)

	// /public/* — no JWT required.
	mux.HandleFunc("GET /api/v1/public/punishments", publicPunishments.List)
	mux.HandleFunc("GET /api/v1/public/punishments/stats", publicPunishments.Stats)
	mux.HandleFunc("GET /api/v1/public/lookup", players.Lookup)

	// /player/* — valid JWT required.
	mux.HandleFunc("GET /api/v1/player/punishments/me", authComp.RequireJWT(player.Me))

	// /mod/* — JWT + moderator permission required.
	mux.HandleFunc("GET /api/v1/mod/punishments/list", authComp.RequireModPermission(mod.List))

	// Contextual: JWT optional, authorization happens after loading the record.
	mux.HandleFunc("GET /api/v1/punishments/{type}/{id}", authComp.OptionalJWT(publicPunishments.GetByID))

	return mux
}
