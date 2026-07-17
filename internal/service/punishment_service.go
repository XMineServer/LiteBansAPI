package service

import (
	"context"
	"time"
	"xmine/litebans-api/internal/domain"
	"xmine/litebans-api/internal/repository"
)

type PunishmentService struct {
	repo            *repository.PunishmentRepository
	players         *PlayerService
	idObfuscator    *IDObfuscator
	includeInactive bool
	includeSilent   bool
	defaultPageSize int
	maxPageSize     int
}

func NewPunishmentService(
	repo *repository.PunishmentRepository,
	players *PlayerService,
	idObfuscator *IDObfuscator,
	includeInactive, includeSilent bool,
	defaultPageSize, maxPageSize int,
) *PunishmentService {
	return &PunishmentService{
		repo:            repo,
		players:         players,
		idObfuscator:    idObfuscator,
		includeInactive: includeInactive,
		includeSilent:   includeSilent,
		defaultPageSize: defaultPageSize,
		maxPageSize:     maxPageSize,
	}
}

// ListParams carries the resolved query parameters shared by all list endpoints.
type ListParams struct {
	Page          *int
	PageSize      *int
	Active        *bool
	Silent        *bool
	PlayerUUID    *string
	ModeratorUUID *string
	Before        *int64
	After         *int64
}

// resolveFilter turns the request-level params into a repository filter, falling back to the
// deployment-wide visibility defaults (2.4 TOR) when the caller didn't specify active/silent.
func (s *PunishmentService) resolveFilter(p ListParams) repository.PunishmentFilter {
	f := repository.PunishmentFilter{
		PlayerUUID:    p.PlayerUUID,
		ModeratorUUID: p.ModeratorUUID,
		Before:        p.Before,
		After:         p.After,
	}
	if p.Active != nil {
		f.ActiveFilter = p.Active
	} else if !s.includeInactive {
		f.ActiveFilter = ptr(true)
	}
	if p.Silent != nil {
		f.SilentFilter = p.Silent
	} else if !s.includeSilent {
		f.SilentFilter = ptr(false)
	}
	return f
}

func banRowsToPunishments(rows []repository.BanRow) []repository.Punishment {
	out := make([]repository.Punishment, len(rows))
	for i := range rows {
		out[i] = &rows[i]
	}
	return out
}

func muteRowsToPunishments(rows []repository.MuteRow) []repository.Punishment {
	out := make([]repository.Punishment, len(rows))
	for i := range rows {
		out[i] = &rows[i]
	}
	return out
}

func warningRowsToPunishments(rows []repository.WarningRow) []repository.Punishment {
	out := make([]repository.Punishment, len(rows))
	for i := range rows {
		out[i] = &rows[i]
	}
	return out
}

func kickRowsToPunishments(rows []repository.KickRow) []repository.Punishment {
	out := make([]repository.Punishment, len(rows))
	for i := range rows {
		out[i] = &rows[i]
	}
	return out
}

func unifiedRowsToPunishments(rows []repository.UnifiedRow) []repository.Punishment {
	out := make([]repository.Punishment, len(rows))
	for i := range rows {
		out[i] = &rows[i]
	}
	return out
}

func (s *PunishmentService) toList(ctx context.Context, rows []repository.Punishment, total int64, page, pageSize int, now int64) (domain.PunishmentList, error) {
	items := make([]domain.Punishment, 0, len(rows))
	for _, row := range rows {
		item, err := s.toDomain(ctx, row, now)
		if err != nil {
			return domain.PunishmentList{}, domain.NewServiceUnavailable("failed to resolve punishment", err)
		}
		items = append(items, item)
	}
	return domain.PunishmentList{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: TotalPages(total, pageSize),
	}, nil
}

func (s *PunishmentService) ListBans(ctx context.Context, p ListParams) (domain.PunishmentList, error) {
	page, pageSize, err := ResolveOffsetPage(p.Page, p.PageSize, s.defaultPageSize, s.maxPageSize)
	if err != nil {
		return domain.PunishmentList{}, err
	}
	now := time.Now().UnixMilli()
	rows, total, err := s.repo.BanList(ctx, s.resolveFilter(p), page, pageSize, now)
	if err != nil {
		return domain.PunishmentList{}, domain.NewServiceUnavailable("failed to query punishments", err)
	}
	return s.toList(ctx, banRowsToPunishments(rows), total, page, pageSize, now)
}

func (s *PunishmentService) ListMutes(ctx context.Context, p ListParams) (domain.PunishmentList, error) {
	page, pageSize, err := ResolveOffsetPage(p.Page, p.PageSize, s.defaultPageSize, s.maxPageSize)
	if err != nil {
		return domain.PunishmentList{}, err
	}
	now := time.Now().UnixMilli()
	rows, total, err := s.repo.MuteList(ctx, s.resolveFilter(p), page, pageSize, now)
	if err != nil {
		return domain.PunishmentList{}, domain.NewServiceUnavailable("failed to query punishments", err)
	}
	return s.toList(ctx, muteRowsToPunishments(rows), total, page, pageSize, now)
}

func (s *PunishmentService) ListWarnings(ctx context.Context, p ListParams) (domain.PunishmentList, error) {
	page, pageSize, err := ResolveOffsetPage(p.Page, p.PageSize, s.defaultPageSize, s.maxPageSize)
	if err != nil {
		return domain.PunishmentList{}, err
	}
	now := time.Now().UnixMilli()
	rows, total, err := s.repo.WarningList(ctx, s.resolveFilter(p), page, pageSize, now)
	if err != nil {
		return domain.PunishmentList{}, domain.NewServiceUnavailable("failed to query punishments", err)
	}
	return s.toList(ctx, warningRowsToPunishments(rows), total, page, pageSize, now)
}

func (s *PunishmentService) ListKicks(ctx context.Context, p ListParams) (domain.PunishmentList, error) {
	page, pageSize, err := ResolveOffsetPage(p.Page, p.PageSize, s.defaultPageSize, s.maxPageSize)
	if err != nil {
		return domain.PunishmentList{}, err
	}
	now := time.Now().UnixMilli()
	rows, total, err := s.repo.KickList(ctx, s.resolveFilter(p), page, pageSize, now)
	if err != nil {
		return domain.PunishmentList{}, domain.NewServiceUnavailable("failed to query punishments", err)
	}
	return s.toList(ctx, kickRowsToPunishments(rows), total, page, pageSize, now)
}

// ListAll returns an offset-paginated, merged listing across all 4 punishment types, used by
// the /mod/punishments and /player/punishments/me endpoints.
func (s *PunishmentService) ListAll(ctx context.Context, p ListParams) (domain.PunishmentList, error) {
	page, pageSize, err := ResolveOffsetPage(p.Page, p.PageSize, s.defaultPageSize, s.maxPageSize)
	if err != nil {
		return domain.PunishmentList{}, err
	}
	now := time.Now().UnixMilli()
	rows, total, err := s.repo.UnifiedList(ctx, s.resolveFilter(p), page, pageSize, now)
	if err != nil {
		return domain.PunishmentList{}, domain.NewServiceUnavailable("failed to query punishments", err)
	}
	return s.toList(ctx, unifiedRowsToPunishments(rows), total, page, pageSize, now)
}

// GetByID resolves a single punishment of the given type by its (possibly obfuscated) id token.
func (s *PunishmentService) GetByID(ctx context.Context, t domain.PunishmentType, idToken string) (domain.Punishment, error) {
	id, err := s.resolveID(t, idToken)
	if err != nil {
		return domain.Punishment{}, domain.NewInvalidParameter("invalid punishment id")
	}

	var row repository.Punishment
	switch t {
	case domain.TypeBan:
		r, err := s.repo.GetBanByID(ctx, id)
		if err != nil {
			return domain.Punishment{}, domain.NewServiceUnavailable("failed to query punishment", err)
		}
		if r == nil {
			return domain.Punishment{}, domain.NewNotFound("punishment not found")
		}
		row = r
	case domain.TypeMute:
		r, err := s.repo.GetMuteByID(ctx, id)
		if err != nil {
			return domain.Punishment{}, domain.NewServiceUnavailable("failed to query punishment", err)
		}
		if r == nil {
			return domain.Punishment{}, domain.NewNotFound("punishment not found")
		}
		row = r
	case domain.TypeWarning:
		r, err := s.repo.GetWarningByID(ctx, id)
		if err != nil {
			return domain.Punishment{}, domain.NewServiceUnavailable("failed to query punishment", err)
		}
		if r == nil {
			return domain.Punishment{}, domain.NewNotFound("punishment not found")
		}
		row = r
	case domain.TypeKick:
		r, err := s.repo.GetKickByID(ctx, id)
		if err != nil {
			return domain.Punishment{}, domain.NewServiceUnavailable("failed to query punishment", err)
		}
		if r == nil {
			return domain.Punishment{}, domain.NewNotFound("punishment not found")
		}
		row = r
	default:
		return domain.Punishment{}, domain.NewInvalidParameter("type must be one of: ban, mute, warning, kick")
	}

	now := time.Now().UnixMilli()
	item, err := s.toDomain(ctx, row, now)
	if err != nil {
		return domain.Punishment{}, domain.NewServiceUnavailable("failed to resolve punishment", err)
	}
	return item, nil
}

func (s *PunishmentService) resolveID(t domain.PunishmentType, idToken string) (int64, error) {
	if s.idObfuscator.Enabled() {
		return s.idObfuscator.Decode(t, idToken)
	}
	return parseInt64(idToken)
}

// Stats returns global counters for the dashboard endpoint, using the deployment-wide
// visibility defaults (no per-request overrides apply here).
func (s *PunishmentService) Stats(ctx context.Context) (domain.Stats, error) {
	now := time.Now().UnixMilli()
	filter := s.resolveFilter(ListParams{})

	var stats domain.Stats
	var err error
	if stats.Bans, err = s.repo.CountBan(ctx, filter, now); err != nil {
		return domain.Stats{}, domain.NewServiceUnavailable("failed to count bans", err)
	}
	if stats.Mutes, err = s.repo.CountMute(ctx, filter, now); err != nil {
		return domain.Stats{}, domain.NewServiceUnavailable("failed to count mutes", err)
	}
	if stats.Warnings, err = s.repo.CountWarning(ctx, filter, now); err != nil {
		return domain.Stats{}, domain.NewServiceUnavailable("failed to count warnings", err)
	}
	if stats.Kicks, err = s.repo.CountKick(ctx, filter, now); err != nil {
		return domain.Stats{}, domain.NewServiceUnavailable("failed to count kicks", err)
	}
	return stats, nil
}
