package httpapi

import (
	"net/http"
	"xmine/litebans-api/internal/service"
)

// NewRouter registers all v1 routes on a fresh http.ServeMux.
func NewRouter(punishmentSvc *service.PunishmentService, playerSvc *service.PlayerService) *http.ServeMux {
	mux := http.NewServeMux()

	punishments := NewPunishmentsHandler(punishmentSvc)
	players := NewPlayersHandler(punishmentSvc, playerSvc)

	mux.HandleFunc("GET /api/v1/punishments/stats", punishments.Stats)
	mux.HandleFunc("GET /api/v1/punishments/{type}", punishments.List)
	mux.HandleFunc("GET /api/v1/punishments/{type}/{id}", punishments.GetByID)

	mux.HandleFunc("GET /api/v1/players/lookup", players.Lookup)
	mux.HandleFunc("GET /api/v1/players/{uuid}/history", players.History)
	mux.HandleFunc("GET /api/v1/players/{uuid}/issued-history", players.IssuedHistory)

	return mux
}
