package service

import (
	"context"
	"database/sql"
	"strconv"
	"xmine/litebans-api/internal/domain"
	"xmine/litebans-api/internal/repository"
)

func ptr[T any](v T) *T {
	return &v
}

func nullIfEmpty(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func (s *PunishmentService) toDomain(ctx context.Context, row repository.PunishmentRow, now int64) (domain.Punishment, error) {
	moderator, err := s.players.ResolveModerator(ctx, row.ModeratorUUID, row.ModeratorName)
	if err != nil {
		return domain.Punishment{}, err
	}

	reason := ""
	if row.Reason.Valid {
		reason = CleanReason(row.Reason.String)
	}

	var id any = row.ID
	if s.idObfuscator.Enabled() {
		id = s.idObfuscator.Encode(row.Type, row.ID)
	}

	item := domain.Punishment{
		ID:           id,
		Type:         row.Type,
		PlayerUUID:   row.PlayerUUID,
		Reason:       reason,
		Moderator:    moderator,
		IssuedAt:     row.Time,
		ServerOrigin: nullIfEmpty(row.ServerOrigin),
	}

	if row.Type == domain.TypeKick {
		return item, nil
	}

	permanent := !row.Until.Valid || row.Until.Int64 <= 0
	item.Permanent = ptr(permanent)
	if !permanent {
		item.ExpiresAt = ptr(row.Until.Int64)
	}

	activeFlag := row.Active.Valid && row.Active.Bool
	item.Active = ptr(activeFlag)

	explicitlyRemoved := row.RemovedByDate.Valid
	expired := !explicitlyRemoved && !permanent && row.Until.Valid && row.Until.Int64 <= now
	item.Expired = ptr(expired)

	item.Silent = ptr(row.Silent.Valid && row.Silent.Bool)
	item.IPBan = ptr(row.IPBan.Valid && row.IPBan.Bool)
	item.ServerScope = nullIfEmpty(row.ServerScope)

	if explicitlyRemoved {
		removedBy, err := s.players.ResolveModerator(ctx, row.RemovedByUUID, row.RemovedByName)
		if err != nil {
			return domain.Punishment{}, err
		}
		var removedReason *string
		if row.RemovedByReason.Valid {
			removedReason = ptr(CleanReason(row.RemovedByReason.String))
		}
		item.Removed = &domain.Removed{
			By:                   removedBy,
			At:                   row.RemovedByDate.Int64,
			Reason:               removedReason,
			ExpiredAutomatically: false,
		}
	}

	if row.Type == domain.TypeWarning {
		item.Acknowledged = ptr(row.Acknowledged.Valid && row.Acknowledged.Bool)
	}

	return item, nil
}
