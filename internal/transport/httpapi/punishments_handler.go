package httpapi

import (
	"net/http"
	"xmine/litebans-api/internal/domain"
	"xmine/litebans-api/internal/service"
)

type PunishmentsHandler struct {
	svc *service.PunishmentService
}

func NewPunishmentsHandler(svc *service.PunishmentService) *PunishmentsHandler {
	return &PunishmentsHandler{svc: svc}
}

func (h *PunishmentsHandler) List(w http.ResponseWriter, r *http.Request) {
	t, ok := domain.ParsePunishmentType(r.PathValue("type"))
	if !ok {
		writeError(w, domain.NewInvalidType("type must be one of: bans, mutes, warnings, kicks"))
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

	result, err := h.svc.List(r.Context(), t, service.ListParams{
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

func (h *PunishmentsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	t, ok := domain.ParsePunishmentType(r.PathValue("type"))
	if !ok {
		writeError(w, domain.NewInvalidType("type must be one of: bans, mutes, warnings, kicks"))
		return
	}

	result, err := h.svc.GetByID(r.Context(), t, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *PunishmentsHandler) Stats(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Stats(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
