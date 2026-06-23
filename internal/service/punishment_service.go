package service

import (
	"context"
	"time"
	"xmine/litebans-api/internal/domain"
	"xmine/litebans-api/internal/repository"
)

type PunishmentService struct {
	repo            *repository.PunishmentRepository
	historyRepo     *repository.HistoryRepository
	players         *PlayerService
	idObfuscator    *IDObfuscator
	includeInactive bool
	includeSilent   bool
	defaultPageSize int
	maxPageSize     int
}

func NewPunishmentService(
	repo *repository.PunishmentRepository,
	historyRepo *repository.HistoryRepository,
	players *PlayerService,
	idObfuscator *IDObfuscator,
	includeInactive, includeSilent bool,
	defaultPageSize, maxPageSize int,
) *PunishmentService {
	return &PunishmentService{
		repo:            repo,
		historyRepo:     historyRepo,
		players:         players,
		idObfuscator:    idObfuscator,
		includeInactive: includeInactive,
		includeSilent:   includeSilent,
		defaultPageSize: defaultPageSize,
		maxPageSize:     maxPageSize,
	}
}

// ListParams carries the resolved query parameters for GET /punishments/{type}.
type ListParams struct {
	Page          *int
	PageSize      *int
	Active        *bool
	Silent        *bool
	PlayerUUID    *string
	ModeratorUUID *string
}

func (s *PunishmentService) resolveVisibilityFilter(t domain.PunishmentType, p ListParams) repository.PunishmentFilter {
	f := repository.PunishmentFilter{
		PlayerUUID:    p.PlayerUUID,
		ModeratorUUID: p.ModeratorUUID,
	}
	if t == domain.TypeKick {
		return f
	}

	if p.Active != nil {
		f.ActiveFilter = p.Active
	} else if !s.includeInactive {
		activeOnly := true
		f.ActiveFilter = &activeOnly
	}

	if p.Silent != nil {
		f.SilentFilter = p.Silent
	} else if !s.includeSilent {
		notSilent := false
		f.SilentFilter = &notSilent
	}
	return f
}

func (s *PunishmentService) List(ctx context.Context, t domain.PunishmentType, p ListParams) (domain.PunishmentList, error) {
	page, pageSize, err := ResolveOffsetPage(p.Page, p.PageSize, s.defaultPageSize, s.maxPageSize)
	if err != nil {
		return domain.PunishmentList{}, err
	}

	filter := s.resolveVisibilityFilter(t, p)
	now := time.Now().UnixMilli()

	rows, total, err := s.repo.List(ctx, t, filter, page, pageSize, now)
	if err != nil {
		return domain.PunishmentList{}, domain.NewServiceUnavailable("failed to query punishments", err)
	}

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

func (s *PunishmentService) GetByID(ctx context.Context, t domain.PunishmentType, idToken string) (domain.Punishment, error) {
	id, err := s.resolveID(t, idToken)
	if err != nil {
		return domain.Punishment{}, domain.NewInvalidParameter("invalid punishment id")
	}

	row, err := s.repo.GetByID(ctx, t, id)
	if err != nil {
		return domain.Punishment{}, domain.NewServiceUnavailable("failed to query punishment", err)
	}
	if row == nil {
		return domain.Punishment{}, domain.NewNotFound("punishment not found")
	}

	now := time.Now().UnixMilli()
	item, err := s.toDomain(ctx, *row, now)
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

func (s *PunishmentService) Stats(ctx context.Context) (domain.Stats, error) {
	now := time.Now().UnixMilli()
	var stats domain.Stats
	var err error
	if stats.Bans, err = s.repo.Count(ctx, domain.TypeBan, s.resolveVisibilityFilter(domain.TypeBan, ListParams{}), now); err != nil {
		return domain.Stats{}, domain.NewServiceUnavailable("failed to count bans", err)
	}
	if stats.Mutes, err = s.repo.Count(ctx, domain.TypeMute, s.resolveVisibilityFilter(domain.TypeMute, ListParams{}), now); err != nil {
		return domain.Stats{}, domain.NewServiceUnavailable("failed to count mutes", err)
	}
	if stats.Warnings, err = s.repo.Count(ctx, domain.TypeWarning, s.resolveVisibilityFilter(domain.TypeWarning, ListParams{}), now); err != nil {
		return domain.Stats{}, domain.NewServiceUnavailable("failed to count warnings", err)
	}
	if stats.Kicks, err = s.repo.Count(ctx, domain.TypeKick, s.resolveVisibilityFilter(domain.TypeKick, ListParams{}), now); err != nil {
		return domain.Stats{}, domain.NewServiceUnavailable("failed to count kicks", err)
	}
	return stats, nil
}

// History returns a page of merged punishment history for a player (received or issued), per TOR 4.3/4.4.
func (s *PunishmentService) History(ctx context.Context, uuid string, byPlayer bool, before, after *int64, pageSizeParam *int) (domain.HistoryPage, error) {
	pageSize, err := ResolveHistoryPageSize(pageSizeParam, s.defaultPageSize, s.maxPageSize)
	if err != nil {
		return domain.HistoryPage{}, err
	}

	filter := repository.HistoryFilter{UUID: uuid, ByPlayer: byPlayer, Before: before, After: after}
	rows, total, err := s.historyRepo.List(ctx, filter, pageSize)
	if err != nil {
		return domain.HistoryPage{}, domain.NewServiceUnavailable("failed to query history", err)
	}

	now := time.Now().UnixMilli()
	items := make([]domain.Punishment, 0, len(rows))
	for _, row := range rows {
		item, err := s.toDomain(ctx, row, now)
		if err != nil {
			return domain.HistoryPage{}, domain.NewServiceUnavailable("failed to resolve punishment", err)
		}
		items = append(items, item)
	}

	cursors := domain.HistoryCursors{}
	if len(items) > 0 {
		newest := items[0].IssuedAt
		oldest := items[len(items)-1].IssuedAt
		cursors.After = &newest
		cursors.Before = &oldest
	}

	return domain.HistoryPage{Items: items, TotalItems: total, Cursors: cursors}, nil
}
