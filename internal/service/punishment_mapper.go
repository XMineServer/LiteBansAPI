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

func nullStringValue(ns sql.NullString) string {
	if !ns.Valid {
		return ""
	}
	return ns.String
}

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// toDomain maps any punishment row (BanRow/MuteRow/WarningRow/KickRow/UnifiedRow) to the API
// domain shape. Removal and acknowledgement info aren't part of the shared PunishmentBase (kicks
// have no removed_by_* columns, only warnings have "warned"), so those are pulled out per concrete
// row type below.
func (s *PunishmentService) toDomain(ctx context.Context, row repository.Punishment, now int64) (domain.Punishment, error) {
	base := row.Base()
	t := domain.PunishmentType(row.Type())

	moderator, err := s.players.ResolveModerator(ctx, base.BannedByUUID, base.BannedByName)
	if err != nil {
		return domain.Punishment{}, err
	}

	reason := ""
	if base.Reason.Valid {
		reason = CleanReason(base.Reason.String)
	}

	var id any = base.ID
	if s.idObfuscator.Enabled() {
		id = s.idObfuscator.Encode(t, base.ID)
	}

	item := domain.Punishment{
		ID:           id,
		Type:         t,
		PlayerUUID:   nullStringValue(base.UUID),
		Reason:       reason,
		Moderator:    moderator,
		IssuedAt:     base.Time,
		ServerOrigin: nullIfEmpty(base.ServerOrigin),
	}

	if t == domain.TypeKick {
		return item, nil
	}

	permanent := base.Until <= 0
	item.Permanent = ptr(permanent)
	if !permanent {
		item.ExpiresAt = ptr(base.Until)
	}

	item.Active = ptr(base.Active.Valid && base.Active.Bool)
	item.Silent = ptr(base.Silent.Valid && base.Silent.Bool)
	item.IPBan = ptr(base.IpBan.Valid && base.IpBan.Bool)
	item.ServerScope = nullIfEmpty(base.ServerScope)

	var removedUUID, removedName, removedReason sql.NullString
	var removedDate sql.NullTime
	var warned sql.NullBool
	switch r := row.(type) {
	case *repository.BanRow:
		removedUUID, removedName, removedReason, removedDate = r.RemovedByUUID, r.RemovedByName, r.RemovedByReason, r.RemovedByDate
	case *repository.MuteRow:
		removedUUID, removedName, removedReason, removedDate = r.RemovedByUUID, r.RemovedByName, r.RemovedByReason, r.RemovedByDate
	case *repository.WarningRow:
		removedUUID, removedName, removedReason, removedDate = r.RemovedByUUID, r.RemovedByName, r.RemovedByReason, r.RemovedByDate
		warned = r.Warned
	case *repository.UnifiedRow:
		removedUUID, removedName, removedReason, removedDate = r.RemovedByUUID, r.RemovedByName, r.RemovedByReason, r.RemovedByDate
		warned = r.Warned
	}

	explicitlyRemoved := removedDate.Valid
	expired := !explicitlyRemoved && !permanent && base.Until > 0 && base.Until <= now
	item.Expired = ptr(expired)

	if explicitlyRemoved {
		removedBy, err := s.players.ResolveModerator(ctx, nullStringValue(removedUUID), removedName)
		if err != nil {
			return domain.Punishment{}, err
		}
		var removedReasonPtr *string
		if removedReason.Valid {
			removedReasonPtr = ptr(CleanReason(removedReason.String))
		}
		item.Removed = &domain.Removed{
			By:                   removedBy,
			At:                   removedDate.Time.UnixMilli(),
			Reason:               removedReasonPtr,
			ExpiredAutomatically: false,
		}
	}

	if t == domain.TypeWarning {
		item.Acknowledged = ptr(warned.Valid && warned.Bool)
	}

	return item, nil
}
