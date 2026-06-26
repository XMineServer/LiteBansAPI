package httpapi

import (
	"context"
	"slices"
	"xmine/litebans-api/internal/api"
	"xmine/litebans-api/internal/auth"
	"xmine/litebans-api/internal/domain"
	"xmine/litebans-api/internal/service"
)

// Server implements api.StrictServerInterface, dispatching to the service layer.
type Server struct {
	punishmentSvc *service.PunishmentService
	playerSvc     *service.PlayerService
	authority     *auth.AuthorityClient
	publicTypes   []string
	modPermission string
}

func NewServer(
	punishmentSvc *service.PunishmentService,
	playerSvc *service.PlayerService,
	authority *auth.AuthorityClient,
	publicTypes []string,
	modPermission string,
) *Server {
	return &Server{
		punishmentSvc: punishmentSvc,
		playerSvc:     playerSvc,
		authority:     authority,
		publicTypes:   publicTypes,
		modPermission: modPermission,
	}
}

func (s *Server) isPublicType(t domain.PunishmentType) bool {
	return slices.Contains(s.publicTypes, string(t))
}

// GetPublicPunishments handles GET /public/punishments: publicly visible punishments,
// restricted to the PUBLIC_TYPES whitelist (defaults to "ban" only).
func (s *Server) GetPublicPunishments(ctx context.Context, request api.GetPublicPunishmentsRequestObject) (api.GetPublicPunishmentsResponseObject, error) {
	p := request.Params

	typeParam := s.publicTypes[0]
	if p.Type != nil && string(*p.Type) != "" {
		typeParam = string(*p.Type)
	}
	t := domain.PunishmentType(typeParam)
	if !s.isPublicType(t) {
		return nil, domain.NewInvalidParameter("type must be one of the publicly exposed punishment types")
	}

	playerUUID, err := normalizeOptionalUUID(p.PlayerUuid)
	if err != nil {
		return nil, err
	}
	moderatorUUID, err := normalizeOptionalUUID(p.ModeratorUuid)
	if err != nil {
		return nil, err
	}

	result, err := s.punishmentSvc.List(ctx, t, service.ListParams{
		Page:          intPtrFrom32(p.Page),
		PageSize:      intPtrFrom32(p.PageSize),
		Active:        p.Active,
		Silent:        p.Silent,
		PlayerUUID:    playerUUID,
		ModeratorUUID: moderatorUUID,
	})
	if err != nil {
		return nil, err
	}
	return api.GetPublicPunishments200JSONResponse(toAPIPunishmentList(result)), nil
}

// GetPublicPunishmentsStats handles GET /public/punishments/stats.
func (s *Server) GetPublicPunishmentsStats(ctx context.Context, request api.GetPublicPunishmentsStatsRequestObject) (api.GetPublicPunishmentsStatsResponseObject, error) {
	result, err := s.punishmentSvc.Stats(ctx)
	if err != nil {
		return nil, err
	}
	return api.GetPublicPunishmentsStats200JSONResponse(toAPIStats(result)), nil
}

// GetPublicLookup handles GET /public/lookup: uuid<->name resolution. Not sensitive, stays public.
func (s *Server) GetPublicLookup(ctx context.Context, request api.GetPublicLookupRequestObject) (api.GetPublicLookupResponseObject, error) {
	p := request.Params

	var name, uuidParam, moderatorName, moderatorUUID string
	if p.Name != nil {
		name = *p.Name
	}
	if p.Uuid != nil {
		uuidParam = *p.Uuid
	}
	if p.ModeratorName != nil {
		moderatorName = *p.ModeratorName
	}
	if p.ModeratorUuid != nil {
		moderatorUUID = *p.ModeratorUuid
	}

	provided := 0
	for _, v := range []string{name, uuidParam, moderatorName, moderatorUUID} {
		if v != "" {
			provided++
		}
	}
	if provided == 0 {
		return nil, domain.NewInvalidParameter("at least one of name, uuid, moderatorName, moderatorUuid is required")
	}
	if provided > 1 {
		return nil, domain.NewInvalidParameter("specify exactly one of name, uuid, moderatorName, moderatorUuid")
	}

	var result domain.Player
	var found bool
	var err error

	switch {
	case name != "":
		if err = ValidatePlayerName(name); err != nil {
			return nil, err
		}
		result, found, err = s.playerSvc.ResolvePlayerByName(ctx, name)
	case moderatorName != "":
		if err = ValidatePlayerName(moderatorName); err != nil {
			return nil, err
		}
		result, found, err = s.playerSvc.ResolvePlayerByName(ctx, moderatorName)
	case uuidParam != "":
		var normalized string
		normalized, err = NormalizeUUID(uuidParam)
		if err != nil {
			return nil, err
		}
		result, found, err = s.playerSvc.ResolvePlayerByUUID(ctx, normalized)
	case moderatorUUID != "":
		var normalized string
		normalized, err = NormalizeUUID(moderatorUUID)
		if err != nil {
			return nil, err
		}
		result, found, err = s.playerSvc.ResolvePlayerByUUID(ctx, normalized)
	}

	if err != nil {
		return nil, domain.NewServiceUnavailable("failed to resolve player", err)
	}
	if !found {
		return nil, domain.NewNotFound("player not found")
	}
	return api.GetPublicLookup200JSONResponse(toAPIPlayer(result)), nil
}

// GetPlayerPunishmentsMe handles GET /player/punishments/me: the caller's own punishments
// across all types. The player uuid always comes from the validated JWT (set by the auth
// middleware); any playerUuid query parameter is ignored to prevent a player from viewing
// someone else's records.
func (s *Server) GetPlayerPunishmentsMe(ctx context.Context, request api.GetPlayerPunishmentsMeRequestObject) (api.GetPlayerPunishmentsMeResponseObject, error) {
	uuid, ok := PlayerUUIDFromContext(ctx)
	if !ok {
		return nil, domain.NewUnauthorized("missing or invalid token")
	}

	p := request.Params

	var types []domain.PunishmentType
	if p.Type != nil && string(*p.Type) != "" {
		t, ok := domain.ParsePunishmentTypeSingular(string(*p.Type))
		if !ok {
			return nil, domain.NewInvalidParameter("type must be one of: ban, mute, warning, kick")
		}
		types = []domain.PunishmentType{t}
	}

	moderatorUUID, err := normalizeOptionalUUID(p.ModeratorUuid)
	if err != nil {
		return nil, err
	}

	result, err := s.punishmentSvc.ListUnified(ctx, service.UnifiedListParams{
		Types:         types,
		Page:          intPtrFrom32(p.Page),
		PageSize:      intPtrFrom32(p.PageSize),
		PlayerUUID:    &uuid,
		ModeratorUUID: moderatorUUID,
		Before:        p.Before,
		After:         p.After,
	})
	if err != nil {
		return nil, err
	}
	return api.GetPlayerPunishmentsMe200JSONResponse(toAPIPunishmentList(result)), nil
}

// GetModPunishmentsList handles GET /mod/punishments/list: the full list across all
// players/moderators. Access is already restricted to holders of the moderator
// permission by the auth middleware.
func (s *Server) GetModPunishmentsList(ctx context.Context, request api.GetModPunishmentsListRequestObject) (api.GetModPunishmentsListResponseObject, error) {
	p := request.Params

	var types []domain.PunishmentType
	if p.Type != nil && string(*p.Type) != "" {
		t, ok := domain.ParsePunishmentTypeSingular(string(*p.Type))
		if !ok {
			return nil, domain.NewInvalidParameter("type must be one of: ban, mute, warning, kick")
		}
		types = []domain.PunishmentType{t}
	}

	playerUUID, err := normalizeOptionalUUID(p.PlayerUuid)
	if err != nil {
		return nil, err
	}
	moderatorUUID, err := normalizeOptionalUUID(p.ModeratorUuid)
	if err != nil {
		return nil, err
	}

	result, err := s.punishmentSvc.ListUnified(ctx, service.UnifiedListParams{
		Types:         types,
		Page:          intPtrFrom32(p.Page),
		PageSize:      intPtrFrom32(p.PageSize),
		Active:        p.Active,
		Silent:        p.Silent,
		PlayerUUID:    playerUUID,
		ModeratorUUID: moderatorUUID,
	})
	if err != nil {
		return nil, err
	}
	return api.GetModPunishmentsList200JSONResponse(toAPIPunishmentList(result)), nil
}

// GetPunishmentByID handles GET /punishments/{type}/{id}, the contextual endpoint shared by all
// access levels. Authorization happens after load (load-then-authorize), since deciding
// "own vs. someone else's" requires the record's playerUuid: public types (PUBLIC_TYPES,
// e.g. "ban") are served to anyone; non-public types are served only to the owning player
// or a moderator, and otherwise 404 (not 403) to avoid revealing the record's existence.
func (s *Server) GetPunishmentByID(ctx context.Context, request api.GetPunishmentByIDRequestObject) (api.GetPunishmentByIDResponseObject, error) {
	t, ok := domain.ParsePunishmentType(string(request.Type))
	if !ok {
		return nil, domain.NewInvalidType("type must be one of: bans, mutes, warnings, kicks")
	}

	result, err := s.punishmentSvc.GetByID(ctx, t, request.Id)
	if err != nil {
		return nil, err
	}

	if s.isPublicType(t) {
		return api.GetPunishmentByID200JSONResponse(toAPIPunishment(result)), nil
	}

	uuid, ok := PlayerUUIDFromContext(ctx)
	if !ok {
		return nil, domain.NewNotFound("punishment not found")
	}
	if uuid == result.PlayerUUID {
		return api.GetPunishmentByID200JSONResponse(toAPIPunishment(result)), nil
	}
	hasPermission, err := s.authority.HasPermission(ctx, uuid, s.modPermission)
	if err != nil || !hasPermission {
		return nil, domain.NewNotFound("punishment not found")
	}
	return api.GetPunishmentByID200JSONResponse(toAPIPunishment(result)), nil
}

func intPtrFrom32(p *int32) *int {
	if p == nil {
		return nil
	}
	v := int(*p)
	return &v
}

func normalizeOptionalUUID(p *string) (*string, error) {
	if p == nil || *p == "" {
		return nil, nil
	}
	normalized, err := NormalizeUUID(*p)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}
