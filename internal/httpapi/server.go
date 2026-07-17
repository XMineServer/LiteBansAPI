package httpapi

import (
	"context"
	"xmine/litebans-api/api"
	"xmine/litebans-api/internal/auth"
	"xmine/litebans-api/internal/domain"
	"xmine/litebans-api/internal/middleware"
	"xmine/litebans-api/internal/service"
)

// Server implements api.StrictServerInterface, dispatching to the service layer.
type Server struct {
	punishmentSvc *service.PunishmentService
	playerSvc     *service.PlayerService
	authority     *auth.AuthorityClient
	modPermission string
}

func NewServer(
	punishmentSvc *service.PunishmentService,
	playerSvc *service.PlayerService,
	authority *auth.AuthorityClient,
	modPermission string,
) *Server {
	return &Server{
		punishmentSvc: punishmentSvc,
		playerSvc:     playerSvc,
		authority:     authority,
		modPermission: modPermission,
	}
}

// GetPublicPunishments handles GET /public/punishments: always active, non-silent bans.
func (s *Server) GetPublicPunishments(ctx context.Context, request api.GetPublicPunishmentsRequestObject) (api.GetPublicPunishmentsResponseObject, error) {
	p := request.Params

	playerUUID, err := normalizeOptionalUUID(p.PlayerUuid)
	if err != nil {
		return nil, err
	}
	moderatorUUID, err := normalizeOptionalUUID(p.ModeratorUuid)
	if err != nil {
		return nil, err
	}

	result, err := s.punishmentSvc.ListBans(ctx, service.ListParams{
		Page:          intPtrFrom32(p.Page),
		PageSize:      intPtrFrom32(p.PageSize),
		Active:        ptr(true),
		Silent:        ptr(false),
		PlayerUUID:    playerUUID,
		ModeratorUUID: moderatorUUID,
		Before:        p.Before,
		After:         p.After,
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

// maxLookupBatchSize caps PostPublicLookupBatch requests to keep the underlying IN (...) query
// (and the request body) reasonably sized.
const maxLookupBatchSize = 200

// PostPublicLookupBatch handles POST /public/lookup/batch: uuid->name resolution for many
// players in one query, to avoid N+1 lookups on the client side. Unresolvable uuids are simply
// omitted from the response rather than causing an error.
func (s *Server) PostPublicLookupBatch(ctx context.Context, request api.PostPublicLookupBatchRequestObject) (api.PostPublicLookupBatchResponseObject, error) {
	if request.Body == nil || len(request.Body.Uuids) == 0 {
		return nil, domain.NewInvalidParameter("uuids must not be empty")
	}
	if len(request.Body.Uuids) > maxLookupBatchSize {
		return nil, domain.NewInvalidParameter("uuids must not contain more than 200 items")
	}

	seen := make(map[string]struct{}, len(request.Body.Uuids))
	uuids := make([]string, 0, len(request.Body.Uuids))
	for _, raw := range request.Body.Uuids {
		normalized, err := NormalizeUUID(raw)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		uuids = append(uuids, normalized)
	}

	names, err := s.playerSvc.ResolveNames(ctx, uuids)
	if err != nil {
		return nil, domain.NewServiceUnavailable("failed to resolve players", err)
	}
	return api.PostPublicLookupBatch200JSONResponse{Players: names}, nil
}

// GetPublicPunishmentByID handles GET /public/punishments/{type}/{id}: only active bans are
// visible; anything else (wrong type, inactive/removed ban) 404s like a missing record.
func (s *Server) GetPublicPunishmentByID(ctx context.Context, request api.GetPublicPunishmentByIDRequestObject) (api.GetPublicPunishmentByIDResponseObject, error) {
	t, ok := domain.ParsePunishmentTypeSingular(string(request.Type))
	if !ok {
		return nil, domain.NewInvalidType("type must be one of: ban, mute, warning, kick")
	}

	result, err := s.punishmentSvc.GetByID(ctx, t, request.Id)
	if err != nil {
		return nil, err
	}
	if t != domain.TypeBan || result.Active == nil || !*result.Active {
		return nil, domain.NewNotFound("punishment not found")
	}
	return api.GetPublicPunishmentByID200JSONResponse(toAPIPunishment(result)), nil
}

// GetPlayerPunishmentsMe handles GET /player/punishments/me: the caller's own punishments
// across all types. The player uuid always comes from the validated JWT (set by the auth
// middleware); any playerUuid query parameter is ignored to prevent a player from viewing
// someone else's records.
func (s *Server) GetPlayerPunishmentsMe(ctx context.Context, request api.GetPlayerPunishmentsMeRequestObject) (api.GetPlayerPunishmentsMeResponseObject, error) {
	uuid, ok := middleware.PlayerUUIDFromContext(ctx)
	if !ok {
		return nil, domain.NewUnauthorized("missing or invalid token")
	}

	p := request.Params
	moderatorUUID, err := normalizeOptionalUUID(p.ModeratorUuid)
	if err != nil {
		return nil, err
	}

	result, err := s.punishmentSvc.ListAll(ctx, service.ListParams{
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

func (s *Server) GetPlayerPunishmentsMeBan(ctx context.Context, request api.GetPlayerPunishmentsMeBanRequestObject) (api.GetPlayerPunishmentsMeBanResponseObject, error) {
	result, err := s.listPlayerMeByType(ctx, s.punishmentSvc.ListBans, request.Params.ModeratorUuid, request.Params.Before, request.Params.After, request.Params.Page, request.Params.PageSize)
	if err != nil {
		return nil, err
	}
	return api.GetPlayerPunishmentsMeBan200JSONResponse(toAPIPunishmentList(result)), nil
}

func (s *Server) GetPlayerPunishmentsMeMute(ctx context.Context, request api.GetPlayerPunishmentsMeMuteRequestObject) (api.GetPlayerPunishmentsMeMuteResponseObject, error) {
	result, err := s.listPlayerMeByType(ctx, s.punishmentSvc.ListMutes, request.Params.ModeratorUuid, request.Params.Before, request.Params.After, request.Params.Page, request.Params.PageSize)
	if err != nil {
		return nil, err
	}
	return api.GetPlayerPunishmentsMeMute200JSONResponse(toAPIPunishmentList(result)), nil
}

func (s *Server) GetPlayerPunishmentsMeWarning(ctx context.Context, request api.GetPlayerPunishmentsMeWarningRequestObject) (api.GetPlayerPunishmentsMeWarningResponseObject, error) {
	result, err := s.listPlayerMeByType(ctx, s.punishmentSvc.ListWarnings, request.Params.ModeratorUuid, request.Params.Before, request.Params.After, request.Params.Page, request.Params.PageSize)
	if err != nil {
		return nil, err
	}
	return api.GetPlayerPunishmentsMeWarning200JSONResponse(toAPIPunishmentList(result)), nil
}

func (s *Server) GetPlayerPunishmentsMeKick(ctx context.Context, request api.GetPlayerPunishmentsMeKickRequestObject) (api.GetPlayerPunishmentsMeKickResponseObject, error) {
	result, err := s.listPlayerMeByType(ctx, s.punishmentSvc.ListKicks, request.Params.ModeratorUuid, request.Params.Before, request.Params.After, request.Params.Page, request.Params.PageSize)
	if err != nil {
		return nil, err
	}
	return api.GetPlayerPunishmentsMeKick200JSONResponse(toAPIPunishmentList(result)), nil
}

// listPlayerMeByType is the shared implementation behind the 4 GetPlayerPunishmentsMe{Type}
// handlers: same params, same "caller's own uuid" restriction, differing only in which
// per-type service method they delegate to.
func (s *Server) listPlayerMeByType(
	ctx context.Context,
	list func(context.Context, service.ListParams) (domain.PunishmentList, error),
	moderatorUuid *string, before, after *int64, page, pageSize *int32,
) (domain.PunishmentList, error) {
	uuid, ok := middleware.PlayerUUIDFromContext(ctx)
	if !ok {
		return domain.PunishmentList{}, domain.NewUnauthorized("missing or invalid token")
	}
	moderatorUUID, err := normalizeOptionalUUID(moderatorUuid)
	if err != nil {
		return domain.PunishmentList{}, err
	}
	return list(ctx, service.ListParams{
		Page:          intPtrFrom32(page),
		PageSize:      intPtrFrom32(pageSize),
		PlayerUUID:    &uuid,
		ModeratorUUID: moderatorUUID,
		Before:        before,
		After:         after,
	})
}

// GetPlayerPunishmentByID handles GET /player/punishments/{type}/{id}: the caller's own
// punishments of any type, plus any active ban regardless of owner.
func (s *Server) GetPlayerPunishmentByID(ctx context.Context, request api.GetPlayerPunishmentByIDRequestObject) (api.GetPlayerPunishmentByIDResponseObject, error) {
	t, ok := domain.ParsePunishmentTypeSingular(string(request.Type))
	if !ok {
		return nil, domain.NewInvalidType("type must be one of: ban, mute, warning, kick")
	}

	result, err := s.punishmentSvc.GetByID(ctx, t, request.Id)
	if err != nil {
		return nil, err
	}

	uuid, ok := middleware.PlayerUUIDFromContext(ctx)
	if ok && uuid == result.PlayerUUID {
		return api.GetPlayerPunishmentByID200JSONResponse(toAPIPunishment(result)), nil
	}
	if t == domain.TypeBan && result.Active != nil && *result.Active {
		return api.GetPlayerPunishmentByID200JSONResponse(toAPIPunishment(result)), nil
	}
	return nil, domain.NewNotFound("punishment not found")
}

// GetModPunishments handles GET /mod/punishments: the full list across all players/moderators
// and all 4 types. Access is already restricted to holders of the moderator permission by the
// auth middleware.
func (s *Server) GetModPunishments(ctx context.Context, request api.GetModPunishmentsRequestObject) (api.GetModPunishmentsResponseObject, error) {
	p := request.Params
	result, err := s.listModByType(ctx, s.punishmentSvc.ListAll, p.PlayerUuid, p.ModeratorUuid, p.Active, p.Silent, p.Before, p.After, p.Page, p.PageSize)
	if err != nil {
		return nil, err
	}
	return api.GetModPunishments200JSONResponse(toAPIPunishmentList(result)), nil
}

func (s *Server) GetModPunishmentsBan(ctx context.Context, request api.GetModPunishmentsBanRequestObject) (api.GetModPunishmentsBanResponseObject, error) {
	p := request.Params
	result, err := s.listModByType(ctx, s.punishmentSvc.ListBans, p.PlayerUuid, p.ModeratorUuid, p.Active, p.Silent, p.Before, p.After, p.Page, p.PageSize)
	if err != nil {
		return nil, err
	}
	return api.GetModPunishmentsBan200JSONResponse(toAPIPunishmentList(result)), nil
}

func (s *Server) GetModPunishmentsMute(ctx context.Context, request api.GetModPunishmentsMuteRequestObject) (api.GetModPunishmentsMuteResponseObject, error) {
	p := request.Params
	result, err := s.listModByType(ctx, s.punishmentSvc.ListMutes, p.PlayerUuid, p.ModeratorUuid, p.Active, p.Silent, p.Before, p.After, p.Page, p.PageSize)
	if err != nil {
		return nil, err
	}
	return api.GetModPunishmentsMute200JSONResponse(toAPIPunishmentList(result)), nil
}

func (s *Server) GetModPunishmentsWarning(ctx context.Context, request api.GetModPunishmentsWarningRequestObject) (api.GetModPunishmentsWarningResponseObject, error) {
	p := request.Params
	result, err := s.listModByType(ctx, s.punishmentSvc.ListWarnings, p.PlayerUuid, p.ModeratorUuid, p.Active, p.Silent, p.Before, p.After, p.Page, p.PageSize)
	if err != nil {
		return nil, err
	}
	return api.GetModPunishmentsWarning200JSONResponse(toAPIPunishmentList(result)), nil
}

func (s *Server) GetModPunishmentsKick(ctx context.Context, request api.GetModPunishmentsKickRequestObject) (api.GetModPunishmentsKickResponseObject, error) {
	p := request.Params
	result, err := s.listModByType(ctx, s.punishmentSvc.ListKicks, p.PlayerUuid, p.ModeratorUuid, p.Active, p.Silent, p.Before, p.After, p.Page, p.PageSize)
	if err != nil {
		return nil, err
	}
	return api.GetModPunishmentsKick200JSONResponse(toAPIPunishmentList(result)), nil
}

// listModByType is the shared implementation behind GetModPunishments and the 4
// GetModPunishments{Type} handlers: same params, no ownership restriction, differing only in
// which service method they delegate to.
func (s *Server) listModByType(
	ctx context.Context,
	list func(context.Context, service.ListParams) (domain.PunishmentList, error),
	playerUuid, moderatorUuid *string, active, silent *bool, before, after *int64, page, pageSize *int32,
) (domain.PunishmentList, error) {
	playerUUID, err := normalizeOptionalUUID(playerUuid)
	if err != nil {
		return domain.PunishmentList{}, err
	}
	moderatorUUID, err := normalizeOptionalUUID(moderatorUuid)
	if err != nil {
		return domain.PunishmentList{}, err
	}
	return list(ctx, service.ListParams{
		Page:          intPtrFrom32(page),
		PageSize:      intPtrFrom32(pageSize),
		Active:        active,
		Silent:        silent,
		PlayerUUID:    playerUUID,
		ModeratorUUID: moderatorUUID,
		Before:        before,
		After:         after,
	})
}

// GetModPunishmentByID handles GET /mod/punishments/{type}/{id}: any punishment, any owner,
// regardless of visibility. Access is already restricted to holders of the moderator
// permission by the auth middleware.
func (s *Server) GetModPunishmentByID(ctx context.Context, request api.GetModPunishmentByIDRequestObject) (api.GetModPunishmentByIDResponseObject, error) {
	t, ok := domain.ParsePunishmentTypeSingular(string(request.Type))
	if !ok {
		return nil, domain.NewInvalidType("type must be one of: ban, mute, warning, kick")
	}

	result, err := s.punishmentSvc.GetByID(ctx, t, request.Id)
	if err != nil {
		return nil, err
	}
	return api.GetModPunishmentByID200JSONResponse(toAPIPunishment(result)), nil
}

func ptr[T any](v T) *T {
	return &v
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
