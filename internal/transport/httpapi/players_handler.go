package httpapi

import (
	"net/http"
	"xmine/litebans-api/internal/domain"
	"xmine/litebans-api/internal/service"
)

type PlayersHandler struct {
	players *service.PlayerService
}

func NewPlayersHandler(players *service.PlayerService) *PlayersHandler {
	return &PlayersHandler{players: players}
}

// Lookup handles GET /public/lookup: uuid<->name resolution. Not sensitive, stays public.
func (h *PlayersHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	name := q.Get("name")
	uuid := q.Get("uuid")
	moderatorName := q.Get("moderatorName")
	moderatorUUID := q.Get("moderatorUuid")

	provided := 0
	for _, v := range []string{name, uuid, moderatorName, moderatorUUID} {
		if v != "" {
			provided++
		}
	}
	if provided == 0 {
		writeError(w, domain.NewInvalidParameter("at least one of name, uuid, moderatorName, moderatorUuid is required"))
		return
	}
	if provided > 1 {
		writeError(w, domain.NewInvalidParameter("specify exactly one of name, uuid, moderatorName, moderatorUuid"))
		return
	}

	var result domain.Player
	var found bool
	var err error

	switch {
	case name != "":
		if err = ValidatePlayerName(name); err != nil {
			writeError(w, err)
			return
		}
		result, found, err = h.players.ResolvePlayerByName(r.Context(), name)
	case moderatorName != "":
		if err = ValidatePlayerName(moderatorName); err != nil {
			writeError(w, err)
			return
		}
		result, found, err = h.players.ResolvePlayerByName(r.Context(), moderatorName)
	case uuid != "":
		var normalized string
		normalized, err = NormalizeUUID(uuid)
		if err != nil {
			writeError(w, err)
			return
		}
		result, found, err = h.players.ResolvePlayerByUUID(r.Context(), normalized)
	case moderatorUUID != "":
		var normalized string
		normalized, err = NormalizeUUID(moderatorUUID)
		if err != nil {
			writeError(w, err)
			return
		}
		result, found, err = h.players.ResolvePlayerByUUID(r.Context(), normalized)
	}

	if err != nil {
		writeError(w, domain.NewServiceUnavailable("failed to resolve player", err))
		return
	}
	if !found {
		writeError(w, domain.NewNotFound("player not found"))
		return
	}
	writeJSON(w, http.StatusOK, result)
}
