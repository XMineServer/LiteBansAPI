package httpapi

import (
	"net/http"
	"xmine/litebans-api/internal/domain"
	"xmine/litebans-api/internal/service"
)

type PlayersHandler struct {
	punishments *service.PunishmentService
	players     *service.PlayerService
}

func NewPlayersHandler(punishments *service.PunishmentService, players *service.PlayerService) *PlayersHandler {
	return &PlayersHandler{punishments: punishments, players: players}
}

func (h *PlayersHandler) history(w http.ResponseWriter, r *http.Request, byPlayer bool) {
	uuid, err := NormalizeUUID(r.PathValue("uuid"))
	if err != nil {
		writeError(w, err)
		return
	}
	before, err := ParseInt64Param(r, "before")
	if err != nil {
		writeError(w, err)
		return
	}
	after, err := ParseInt64Param(r, "after")
	if err != nil {
		writeError(w, err)
		return
	}
	pageSize, err := ParseIntParam(r, "pageSize")
	if err != nil {
		writeError(w, err)
		return
	}

	result, err := h.punishments.History(r.Context(), uuid, byPlayer, before, after, pageSize)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *PlayersHandler) History(w http.ResponseWriter, r *http.Request) {
	h.history(w, r, true)
}

func (h *PlayersHandler) IssuedHistory(w http.ResponseWriter, r *http.Request) {
	h.history(w, r, false)
}

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
