package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"xmine/litebans-api/internal/domain"
)

type HistoryRepository struct {
	db          *sql.DB
	tablePrefix string
}

func NewHistoryRepository(db *sql.DB, tablePrefix string) *HistoryRepository {
	return &HistoryRepository{db: db, tablePrefix: tablePrefix}
}

var allPunishmentTypes = []domain.PunishmentType{domain.TypeBan, domain.TypeMute, domain.TypeWarning, domain.TypeKick}

// UnifiedFilter selects punishment records across multiple types for the union endpoints
// (/player/punishments/me and /mod/punishments/list when type is unspecified).
type UnifiedFilter struct {
	Types         []domain.PunishmentType
	PlayerUUID    *string
	ModeratorUUID *string
	ActiveFilter  *bool
	SilentFilter  *bool
	Before        *int64
	After         *int64
}

func unifiedSubquery(prefix string, t domain.PunishmentType, f UnifiedFilter, now int64) (string, []any) {
	table := tableName(prefix, t)
	var where []string
	var args []any

	where = append(where, "uuid IS NOT NULL", fmt.Sprintf("uuid <> '%s'", OfflineUUIDMarker))

	if hasDurationColumns(t) {
		if f.ActiveFilter != nil {
			if *f.ActiveFilter {
				where = append(where, "(active = 1 AND (until <= 0 OR until > ?))")
				args = append(args, now)
			} else {
				where = append(where, "NOT (active = 1 AND (until <= 0 OR until > ?))")
				args = append(args, now)
			}
		}
		if f.SilentFilter != nil {
			where = append(where, "silent = ?")
			args = append(args, *f.SilentFilter)
		}
	}
	if f.PlayerUUID != nil {
		where = append(where, "uuid = ?")
		args = append(args, *f.PlayerUUID)
	}
	if f.ModeratorUUID != nil {
		where = append(where, "banned_by_uuid = ?")
		args = append(args, *f.ModeratorUUID)
	}
	if f.Before != nil {
		where = append(where, "time < ?")
		args = append(args, *f.Before)
	}
	if f.After != nil {
		where = append(where, "time > ?")
		args = append(args, *f.After)
	}

	query := fmt.Sprintf(
		"SELECT '%s' AS punishment_type, %s FROM %s WHERE %s",
		t, selectColumns(t), table, strings.Join(where, " AND "),
	)
	return query, args
}

// ListOffset returns an offset-paginated page of the merged, time-descending punishment
// records across f.Types (or all 4 types if empty), used by the /player and /mod union endpoints.
func (r *HistoryRepository) ListOffset(ctx context.Context, f UnifiedFilter, page, pageSize int, now int64) ([]PunishmentRow, int64, error) {
	types := f.Types
	if len(types) == 0 {
		types = allPunishmentTypes
	}

	var unionParts []string
	var args []any
	for _, t := range types {
		q, a := unifiedSubquery(r.tablePrefix, t, f, now)
		unionParts = append(unionParts, q)
		args = append(args, a...)
	}
	union := strings.Join(unionParts, " UNION ALL ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS merged", union)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count unified: %w", err)
	}

	listQuery := fmt.Sprintf("SELECT * FROM (%s) AS merged ORDER BY time DESC LIMIT ? OFFSET ?", union)
	listArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list unified: %w", err)
	}
	defer rows.Close()

	var items []PunishmentRow
	for rows.Next() {
		var typeStr string
		var row PunishmentRow
		if err := rows.Scan(
			&typeStr, &row.ID, &row.PlayerUUID, &row.Reason, &row.ModeratorUUID, &row.ModeratorName, &row.Time,
			&row.RemovedByUUID, &row.RemovedByName, &row.Until,
			&row.RemovedByDate, &row.RemovedByReason, &row.Active, &row.Silent,
			&row.ServerOrigin, &row.ServerScope, &row.IPBan, &row.Acknowledged,
		); err != nil {
			return nil, 0, fmt.Errorf("scan unified row: %w", err)
		}
		row.Type = domain.PunishmentType(typeStr)
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate unified: %w", err)
	}
	return items, total, nil
}

// LatestNameByUUID returns the most recently recorded display name for a uuid, or "" if unknown.
func (r *HistoryRepository) LatestNameByUUID(ctx context.Context, uuid string) (string, error) {
	table := historyTableName(r.tablePrefix)
	query := fmt.Sprintf("SELECT name FROM %s WHERE uuid = ? ORDER BY date DESC LIMIT 1", table)
	var name string
	err := r.db.QueryRowContext(ctx, query, uuid).Scan(&name)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("latest name by uuid: %w", err)
	}
	return name, nil
}

// LatestUUIDByName returns the uuid most recently associated with the given name (case-insensitive), or "" if unknown.
func (r *HistoryRepository) LatestUUIDByName(ctx context.Context, name string) (string, error) {
	table := historyTableName(r.tablePrefix)
	query := fmt.Sprintf("SELECT uuid FROM %s WHERE name = ? ORDER BY date DESC LIMIT 1", table)
	var uuid string
	err := r.db.QueryRowContext(ctx, query, name).Scan(&uuid)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("latest uuid by name: %w", err)
	}
	return uuid, nil
}
