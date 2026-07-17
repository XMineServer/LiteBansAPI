package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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
	// Before/After filter on the time (issuedAt) column: Before => time < *Before, After => time > *After.
	Before *int64
	After  *int64
}

type PunishmentRepository struct {
	db          *sql.DB
	tablePrefix string
}

func NewPunishmentRepository(db *sql.DB, tablePrefix string) *PunishmentRepository {
	return &PunishmentRepository{db: db, tablePrefix: tablePrefix}
}

type scanner interface {
	Scan(...any) error
}

func scanBanRow(scan scanner) (BanRow, error) {
	var row BanRow
	err := scan.Scan(
		&row.ID, &row.UUID, &row.IP, &row.Reason,
		&row.BannedByUUID, &row.BannedByName, &row.Time, &row.Until,
		&row.Template, &row.ServerScope, &row.ServerOrigin, &row.Silent,
		&row.IpBan, &row.IpBanWildcard, &row.Active,
		&row.RemovedByUUID, &row.RemovedByName, &row.RemovedByReason,
		&row.RemovedByDate,
	)
	return row, err
}

func banColums() string {
	var columns = []string{
		"id", "uuid", "ip", "reason",
		"banned_by_uuid", "banned_by_name", "time", "until",
		"template", "server_scope", "server_origin", bitColumn("silent"),
		bitColumn("ipban"), bitColumn("ipban_wildcard"), bitColumn("active"),
		"removed_by_uuid", "removed_by_name", "removed_by_reason",
		"removed_by_date",
	}
	return strings.Join(columns, ", ")
}

func scanMuteRow(scan scanner) (MuteRow, error) {
	var row MuteRow
	err := scan.Scan(
		&row.ID, &row.UUID, &row.IP, &row.Reason,
		&row.BannedByUUID, &row.BannedByName, &row.Time, &row.Until,
		&row.Template, &row.ServerScope, &row.ServerOrigin, &row.Silent,
		&row.IpBan, &row.IpBanWildcard, &row.Active,
		&row.RemovedByUUID, &row.RemovedByName, &row.RemovedByReason,
		&row.RemovedByDate,
	)
	return row, err
}

func muteColums() string {
	var columns = []string{
		"id", "uuid", "ip", "reason",
		"banned_by_uuid", "banned_by_name", "time", "until",
		"template", "server_scope", "server_origin", bitColumn("silent"),
		bitColumn("ipban"), bitColumn("ipban_wildcard"), bitColumn("active"),
		"removed_by_uuid", "removed_by_name", "removed_by_reason",
		"removed_by_date",
	}
	return strings.Join(columns, ", ")
}

func scanWarningRow(scan scanner) (WarningRow, error) {
	var row WarningRow
	err := scan.Scan(
		&row.ID, &row.UUID, &row.IP, &row.Reason,
		&row.BannedByUUID, &row.BannedByName, &row.Time, &row.Until,
		&row.Template, &row.ServerScope, &row.ServerOrigin, &row.Silent,
		&row.IpBan, &row.IpBanWildcard, &row.Active,
		&row.RemovedByUUID, &row.RemovedByName, &row.RemovedByReason,
		&row.RemovedByDate, &row.Warned,
	)
	return row, err
}

func warningColums() string {
	var columns = []string{
		"id", "uuid", "ip", "reason",
		"banned_by_uuid", "banned_by_name", "time", "until",
		"template", "server_scope", "server_origin", bitColumn("silent"),
		bitColumn("ipban"), bitColumn("ipban_wildcard"), bitColumn("active"),
		"removed_by_uuid", "removed_by_name", "removed_by_reason",
		"removed_by_date", bitColumn("warned"),
	}
	return strings.Join(columns, ", ")
}

func scanKickRow(scan scanner) (KickRow, error) {
	var row KickRow
	err := scan.Scan(
		&row.ID, &row.UUID, &row.IP, &row.Reason,
		&row.BannedByUUID, &row.BannedByName, &row.Time, &row.Until,
		&row.Template, &row.ServerScope, &row.ServerOrigin, &row.Silent,
		&row.IpBan, &row.IpBanWildcard, &row.Active,
	)
	return row, err
}

func kickColums() string {
	var columns = []string{
		"id", "uuid", "ip", "reason",
		"banned_by_uuid", "banned_by_name", "time", "until",
		"template", "server_scope", "server_origin", bitColumn("silent"),
		bitColumn("ipban"), bitColumn("ipban_wildcard"), bitColumn("active"),
	}
	return strings.Join(columns, ", ")
}

func scanUnifiedRow(scan scanner) (UnifiedRow, error) {
	var row UnifiedRow
	err := scan.Scan(
		&row.typ,
		&row.ID, &row.UUID, &row.IP, &row.Reason,
		&row.BannedByUUID, &row.BannedByName, &row.Time, &row.Until,
		&row.Template, &row.ServerScope, &row.ServerOrigin, &row.Silent,
		&row.IpBan, &row.IpBanWildcard, &row.Active,
		&row.RemovedByUUID, &row.RemovedByName, &row.RemovedByReason,
		&row.RemovedByDate, &row.Warned,
	)
	return row, err
}

// bitColumn casts a MySQL BIT(1) column to an unsigned integer. The go-sql-driver/mysql driver
// returns BIT values as a raw single byte (0x00/0x01), which sql.NullBool.Scan can't parse
// (it expects the ASCII string "0"/"1", not a literal byte) — casting server-side avoids that.
func bitColumn(name string) string {
	return fmt.Sprintf("%s + 0 AS %s", name, name)
}

// unifiedSourceColumns returns the column list for one branch of the UnifiedList UNION ALL,
// substituting NULL for columns the branch's table doesn't physically have (kicks lack
// removed_by_*, and only warnings have "warned").
func unifiedSourceColumns(hasRemoved, hasWarned bool) string {
	cols := []string{
		"id", "uuid", "ip", "reason",
		"banned_by_uuid", "banned_by_name", "time", "until",
		"template", "server_scope", "server_origin", bitColumn("silent"),
		bitColumn("ipban"), bitColumn("ipban_wildcard"), bitColumn("active"),
	}
	if hasRemoved {
		cols = append(cols, "removed_by_uuid", "removed_by_name", "removed_by_reason", "removed_by_date")
	} else {
		cols = append(cols, "NULL", "NULL", "NULL", "NULL")
	}
	if hasWarned {
		cols = append(cols, bitColumn("warned"))
	} else {
		cols = append(cols, "NULL")
	}
	return strings.Join(cols, ", ")
}

// buildVisibilityWhere builds the WHERE fragment + args for list/count queries: excludes rows
// without a linked player and applies the active/silent/time-range filters.
func buildVisibilityWhere(f PunishmentFilter, now int64) (string, []any) {
	clauses := []string{"uuid IS NOT NULL", fmt.Sprintf("uuid <> '%s'", OfflineUUIDMarker)}
	var args []any

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
	if f.PlayerUUID != nil {
		clauses = append(clauses, "uuid = ?")
		args = append(args, *f.PlayerUUID)
	}
	if f.ModeratorUUID != nil {
		clauses = append(clauses, "banned_by_uuid = ?")
		args = append(args, *f.ModeratorUUID)
	}
	if f.Before != nil {
		clauses = append(clauses, "time < ?")
		args = append(args, *f.Before)
	}
	if f.After != nil {
		clauses = append(clauses, "time > ?")
		args = append(args, *f.After)
	}
	return strings.Join(clauses, " AND "), args
}

func (r *PunishmentRepository) BanList(ctx context.Context, f PunishmentFilter, page int, pageSize int, now int64) ([]BanRow, int64, error) {
	where, args := buildVisibilityWhere(f, now)
	table := r.tablePrefix + "bans"

	countQuery := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s", table, where)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s ORDER BY time DESC LIMIT ? OFFSET ?",
		banColums(), table, where,
	)
	listArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []BanRow
	for rows.Next() {
		row, err := scanBanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, err
}

func (r *PunishmentRepository) WarningList(ctx context.Context, f PunishmentFilter, page int, pageSize int, now int64) ([]WarningRow, int64, error) {
	where, args := buildVisibilityWhere(f, now)
	table := r.tablePrefix + "warnings"

	countQuery := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s", table, where)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s ORDER BY time DESC LIMIT ? OFFSET ?",
		warningColums(), table, where,
	)
	listArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []WarningRow
	for rows.Next() {
		row, err := scanWarningRow(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, err
}

func (r *PunishmentRepository) MuteList(ctx context.Context, f PunishmentFilter, page int, pageSize int, now int64) ([]MuteRow, int64, error) {
	where, args := buildVisibilityWhere(f, now)
	table := r.tablePrefix + "mutes"

	countQuery := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s", table, where)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s ORDER BY time DESC LIMIT ? OFFSET ?",
		muteColums(), table, where,
	)
	listArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []MuteRow
	for rows.Next() {
		row, err := scanMuteRow(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, err
}

func (r *PunishmentRepository) KickList(ctx context.Context, f PunishmentFilter, page int, pageSize int, now int64) ([]KickRow, int64, error) {
	where, args := buildVisibilityWhere(f, now)
	table := r.tablePrefix + "kicks"

	countQuery := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s", table, where)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s ORDER BY time DESC LIMIT ? OFFSET ?",
		kickColums(), table, where,
	)
	listArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []KickRow
	for rows.Next() {
		row, err := scanKickRow(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, err
}

// unifiedSource describes one table participating in the UnifiedList UNION ALL.
type unifiedSource struct {
	typ                   string
	tableSuffix           string
	hasRemoved, hasWarned bool
}

var unifiedSources = []unifiedSource{
	{typ: "ban", tableSuffix: "bans", hasRemoved: true, hasWarned: false},
	{typ: "mute", tableSuffix: "mutes", hasRemoved: true, hasWarned: false},
	{typ: "warning", tableSuffix: "warnings", hasRemoved: true, hasWarned: true},
	{typ: "kick", tableSuffix: "kicks", hasRemoved: false, hasWarned: false},
}

func (r *PunishmentRepository) UnifiedList(ctx context.Context, f PunishmentFilter, page int, pageSize int, now int64) ([]UnifiedRow, int64, error) {
	where, args := buildVisibilityWhere(f, now)

	parts := make([]string, 0, len(unifiedSources))
	for _, s := range unifiedSources {
		parts = append(parts, fmt.Sprintf(
			"SELECT '%s' AS type, %s FROM %s",
			s.typ, unifiedSourceColumns(s.hasRemoved, s.hasWarned), r.tablePrefix+s.tableSuffix,
		))
	}
	union := strings.Join(parts, " UNION ALL ")

	countQuery := fmt.Sprintf("SELECT count(*) FROM (%s) AS merged WHERE %s", union, where)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := fmt.Sprintf(
		"SELECT * FROM (%s) AS merged WHERE %s ORDER BY time DESC LIMIT ? OFFSET ?",
		union, where,
	)
	listArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []UnifiedRow
	for rows.Next() {
		row, err := scanUnifiedRow(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, err
}

func (r *PunishmentRepository) GetBanByID(ctx context.Context, id int64) (*BanRow, error) {
	table := r.tablePrefix + "bans"
	query := fmt.Sprintf(
		`SELECT %s FROM %s WHERE id = ?`,
		banColums(), table,
	)
	row := r.db.QueryRowContext(ctx, query, id)
	result, err := scanBanRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

func (r *PunishmentRepository) GetMuteByID(ctx context.Context, id int64) (*MuteRow, error) {
	table := r.tablePrefix + "mutes"
	query := fmt.Sprintf(
		`SELECT %s FROM %s WHERE id = ?`,
		muteColums(), table,
	)
	row := r.db.QueryRowContext(ctx, query, id)
	result, err := scanMuteRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

func (r *PunishmentRepository) GetWarningByID(ctx context.Context, id int64) (*WarningRow, error) {
	table := r.tablePrefix + "warnings"
	query := fmt.Sprintf(
		`SELECT %s FROM %s WHERE id = ?`,
		warningColums(), table,
	)
	row := r.db.QueryRowContext(ctx, query, id)
	result, err := scanWarningRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

func (r *PunishmentRepository) GetKickByID(ctx context.Context, id int64) (*KickRow, error) {
	table := r.tablePrefix + "kicks"
	query := fmt.Sprintf(
		`SELECT %s FROM %s WHERE id = ?`,
		kickColums(), table,
	)
	row := r.db.QueryRowContext(ctx, query, id)
	result, err := scanKickRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

func (r *PunishmentRepository) CountBan(ctx context.Context, f PunishmentFilter, now int64) (int64, error) {
	return r.countPunishment(ctx, f, now, r.tablePrefix+"bans")
}

func (r *PunishmentRepository) CountKick(ctx context.Context, f PunishmentFilter, now int64) (int64, error) {
	return r.countPunishment(ctx, f, now, r.tablePrefix+"kicks")
}

func (r *PunishmentRepository) CountMute(ctx context.Context, f PunishmentFilter, now int64) (int64, error) {
	return r.countPunishment(ctx, f, now, r.tablePrefix+"mutes")
}

func (r *PunishmentRepository) CountWarning(ctx context.Context, f PunishmentFilter, now int64) (int64, error) {
	return r.countPunishment(ctx, f, now, r.tablePrefix+"warnings")
}

func (r *PunishmentRepository) countPunishment(ctx context.Context, f PunishmentFilter, now int64, table string) (int64, error) {
	where, args := buildVisibilityWhere(f, now)
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, where)
	var total int64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}
