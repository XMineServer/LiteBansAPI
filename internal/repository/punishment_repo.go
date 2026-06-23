package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"xmine/litebans-api/internal/domain"
)

// PunishmentFilter describes the filters applied to a list/count query.
// A nil pointer means "no filter on this dimension".
type PunishmentFilter struct {
	// ActiveFilter: true => only rows currently active (active=1 AND not expired by time);
	// false => only rows not currently active (removed/expired); nil => no filter.
	ActiveFilter *bool
	// SilentFilter: nil => no filter, otherwise filter on the silent column.
	SilentFilter  *bool
	PlayerUUID    *string
	ModeratorUUID *string
}

type PunishmentRepository struct {
	db          *sql.DB
	tablePrefix string
}

func NewPunishmentRepository(db *sql.DB, tablePrefix string) *PunishmentRepository {
	return &PunishmentRepository{db: db, tablePrefix: tablePrefix}
}

// selectColumns returns the canonical 18-column SELECT list for a punishment type,
// substituting SQL NULL/0 literals for columns the type's table doesn't have.
func selectColumns(t domain.PunishmentType) string {
	cols := []string{
		"id", "uuid", "reason", "banned_by_uuid", "banned_by_name", "time",
	}
	if hasDurationColumns(t) {
		cols = append(cols,
			"removed_by_uuid", "removed_by_name", "until", "removed",
			"removed_by_date", "removed_by_reason", "active", "silent",
		)
	} else {
		cols = append(cols,
			"NULL", "NULL", "NULL", "NULL",
			"NULL", "NULL", "NULL", "NULL",
		)
	}
	cols = append(cols, "server_origin")
	if t != domain.TypeKick {
		cols = append(cols, "server_scope")
	} else {
		cols = append(cols, "NULL")
	}
	if hasIPBanColumn(t) {
		cols = append(cols, "ipban")
	} else {
		cols = append(cols, "NULL")
	}
	if hasAcknowledgedColumn(t) {
		cols = append(cols, "warned")
	} else {
		cols = append(cols, "NULL")
	}
	return strings.Join(cols, ", ")
}

func scanRow(t domain.PunishmentType, scanner interface{ Scan(...any) error }) (PunishmentRow, error) {
	var row PunishmentRow
	row.Type = t
	err := scanner.Scan(
		&row.ID, &row.PlayerUUID, &row.Reason, &row.ModeratorUUID, &row.ModeratorName, &row.Time,
		&row.RemovedByUUID, &row.RemovedByName, &row.Until, &row.Removed,
		&row.RemovedByDate, &row.RemovedByReason, &row.Active, &row.Silent,
		&row.ServerOrigin, &row.ServerScope, &row.IPBan, &row.Acknowledged,
	)
	return row, err
}

// buildVisibilityWhere builds the WHERE fragment + args for list/count queries: excludes rows
// without a linked player and applies the active/silent filters per the resolved deployment settings.
func buildVisibilityWhere(t domain.PunishmentType, f PunishmentFilter, now int64) (string, []any) {
	clauses := []string{"uuid IS NOT NULL", fmt.Sprintf("uuid <> '%s'", OfflineUUIDMarker)}
	var args []any

	if hasDurationColumns(t) {
		if f.ActiveFilter != nil {
			if *f.ActiveFilter {
				clauses = append(clauses, "(active = 1 AND (until <= 0 OR until > ?))")
				args = append(args, now)
			} else {
				clauses = append(clauses, "NOT (active = 1 AND (until <= 0 OR until > ?))")
				args = append(args, now)
			}
		}
		if f.SilentFilter != nil {
			clauses = append(clauses, "silent = ?")
			args = append(args, *f.SilentFilter)
		}
	}
	if f.PlayerUUID != nil {
		clauses = append(clauses, "uuid = ?")
		args = append(args, *f.PlayerUUID)
	}
	if f.ModeratorUUID != nil {
		clauses = append(clauses, "banned_by_uuid = ?")
		args = append(args, *f.ModeratorUUID)
	}
	return strings.Join(clauses, " AND "), args
}

func (r *PunishmentRepository) List(ctx context.Context, t domain.PunishmentType, f PunishmentFilter, page, pageSize int, now int64) ([]PunishmentRow, int64, error) {
	where, args := buildVisibilityWhere(t, f, now)
	table := tableName(r.tablePrefix, t)

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, where)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count %s: %w", table, err)
	}

	listQuery := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s ORDER BY time DESC LIMIT ? OFFSET ?",
		selectColumns(t), table, where,
	)
	listArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list %s: %w", table, err)
	}
	defer rows.Close()

	var items []PunishmentRow
	for rows.Next() {
		row, err := scanRow(t, rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan %s: %w", table, err)
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate %s: %w", table, err)
	}
	return items, total, nil
}

func (r *PunishmentRepository) GetByID(ctx context.Context, t domain.PunishmentType, id int64) (*PunishmentRow, error) {
	table := tableName(r.tablePrefix, t)
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ?", selectColumns(t), table)
	row := r.db.QueryRowContext(ctx, query, id)
	result, err := scanRow(t, row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get %s by id: %w", table, err)
	}
	return &result, nil
}

func (r *PunishmentRepository) Count(ctx context.Context, t domain.PunishmentType, f PunishmentFilter, now int64) (int64, error) {
	where, args := buildVisibilityWhere(t, f, now)
	table := tableName(r.tablePrefix, t)
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, where)
	var total int64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count %s: %w", table, err)
	}
	return total, nil
}
