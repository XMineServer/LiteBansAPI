package httpapi

import (
	"net/http"
	"xmine/litebans-api/internal/domain"
	"xmine/litebans-api/internal/service"
)

type PlayerHandler struct {
	svc *service.PunishmentService
}

func NewPlayerHandler(svc *service.PunishmentService) *PlayerHandler {
	return &PlayerHandler{svc: svc}
}

// Me handles GET /player/punishments/me: the caller's own punishments across all types.
// The player uuid always comes from the validated JWT (set by RequireJWT); any playerUuid
// query parameter is ignored to prevent a player from viewing someone else's records.
func (h *PlayerHandler) Me(w http.ResponseWriter, r *http.Request) {
	uuid, ok := PlayerUUIDFromContext(r.Context())
	if !ok {
		writeError(w, domain.NewUnauthorized("missing or invalid token"))
		return
	}

	var types []domain.PunishmentType
	if typeParam := r.URL.Query().Get("type"); typeParam != "" {
		t, ok := domain.ParsePunishmentTypeSingular(typeParam)
		if !ok {
			writeError(w, domain.NewInvalidParameter("type must be one of: ban, mute, warning, kick"))
			return
		}
		types = []domain.PunishmentType{t}
	}

	moderatorUUID, err := ParseUUIDParam(r, "moderatorUuid")
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
	page, err := ParseIntParam(r, "page")
	if err != nil {
		writeError(w, err)
		return
	}
	pageSize, err := ParseIntParam(r, "pageSize")
	if err != nil {
		writeError(w, err)
		return
	}

	result, err := h.svc.ListUnified(r.Context(), service.UnifiedListParams{
		Types:         types,
		Page:          page,
		PageSize:      pageSize,
		PlayerUUID:    &uuid,
		ModeratorUUID: moderatorUUID,
		Before:        before,
		After:         after,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
