package httpapi

import (
	"net/http"
	"slices"
	"xmine/litebans-api/internal/auth"
	"xmine/litebans-api/internal/domain"
	"xmine/litebans-api/internal/service"
)

type PunishmentsHandler struct {
	svc           *service.PunishmentService
	publicTypes   []string
	authority     *auth.AuthorityClient
	modPermission string
}

func NewPunishmentsHandler(svc *service.PunishmentService, publicTypes []string, authority *auth.AuthorityClient, modPermission string) *PunishmentsHandler {
	return &PunishmentsHandler{svc: svc, publicTypes: publicTypes, authority: authority, modPermission: modPermission}
}

func (h *PunishmentsHandler) isPublicType(t domain.PunishmentType) bool {
	return slices.Contains(h.publicTypes, string(t))
}

// List handles GET /public/punishments: publicly visible punishments, restricted to
// the PUBLIC_TYPES whitelist (defaults to "ban" only).
func (h *PunishmentsHandler) List(w http.ResponseWriter, r *http.Request) {
	typeParam := r.URL.Query().Get("type")
	if typeParam == "" {
		typeParam = h.publicTypes[0]
	}
	t := domain.PunishmentType(typeParam)
	if !h.isPublicType(t) {
		writeError(w, domain.NewInvalidParameter("type must be one of the publicly exposed punishment types"))
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

// GetByID handles GET /punishments/{type}/{id}, the contextual endpoint shared by all
// access levels. Authorization happens after load (load-then-authorize), since deciding
// "own vs. someone else's" requires the record's playerUuid: public types (PUBLIC_TYPES,
// e.g. "ban") are served to anyone; non-public types are served only to the owning player
// or a moderator, and otherwise 404 (not 403) to avoid revealing the record's existence.
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

	if h.isPublicType(t) {
		writeJSON(w, http.StatusOK, result)
		return
	}

	uuid, ok := PlayerUUIDFromContext(r.Context())
	if !ok {
		writeError(w, domain.NewNotFound("punishment not found"))
		return
	}
	if uuid == result.PlayerUUID {
		writeJSON(w, http.StatusOK, result)
		return
	}
	hasPermission, err := h.authority.HasPermission(r.Context(), uuid, h.modPermission)
	if err != nil || !hasPermission {
		writeError(w, domain.NewNotFound("punishment not found"))
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
