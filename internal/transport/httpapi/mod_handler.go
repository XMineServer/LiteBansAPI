package httpapi

import (
	"net/http"
	"xmine/litebans-api/internal/domain"
	"xmine/litebans-api/internal/service"
)

type ModHandler struct {
	svc *service.PunishmentService
}

func NewModHandler(svc *service.PunishmentService) *ModHandler {
	return &ModHandler{svc: svc}
}

// List handles GET /mod/punishments/list: the full list across all players/moderators.
// Access is already restricted to holders of the moderator permission by RequireModPermission.
func (h *ModHandler) List(w http.ResponseWriter, r *http.Request) {
	var types []domain.PunishmentType
	if typeParam := r.URL.Query().Get("type"); typeParam != "" {
		t, ok := domain.ParsePunishmentTypeSingular(typeParam)
		if !ok {
			writeError(w, domain.NewInvalidParameter("type must be one of: ban, mute, warning, kick"))
			return
		}
		types = []domain.PunishmentType{t}
	}

	playerUUID, err := ParseUUIDParam(r, "playerUuid")
	if err != nil {
		writeError(w, err)
		return
	}
	moderatorUUID, err := ParseUUIDParam(r, "moderatorUuid")
	if err != nil {
		writeError(w, err)
		return
	}
	active, err := ParseBoolParam(r, "active")
	if err != nil {
		writeError(w, err)
		return
	}
	silent, err := ParseBoolParam(r, "silent")
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
		Active:        active,
		Silent:        silent,
		PlayerUUID:    playerUUID,
		ModeratorUUID: moderatorUUID,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
