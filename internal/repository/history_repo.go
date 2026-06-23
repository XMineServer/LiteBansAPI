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

// HistoryFilter selects records by the uuid's role: as the punished player, or as the issuing moderator.
type HistoryFilter struct {
	UUID     string
	ByPlayer bool // true: WHERE uuid = ?; false: WHERE banned_by_uuid = ?
	Before   *int64
	After    *int64
}

func unionSubquery(prefix string, t domain.PunishmentType, f HistoryFilter) (string, []any) {
	table := tableName(prefix, t)
	column := "uuid"
	if !f.ByPlayer {
		column = "banned_by_uuid"
	}
	where := []string{fmt.Sprintf("%s = ?", column)}
	args := []any{f.UUID}
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

// List returns a page of the merged, time-descending history across all 4 punishment tables for the given uuid.
func (r *HistoryRepository) List(ctx context.Context, f HistoryFilter, pageSize int) ([]PunishmentRow, int64, error) {
	var unionParts []string
	var args []any
	for _, t := range allPunishmentTypes {
		q, a := unionSubquery(r.tablePrefix, t, f)
		unionParts = append(unionParts, q)
		args = append(args, a...)
	}
	union := strings.Join(unionParts, " UNION ALL ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS merged", union)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count history: %w", err)
	}

	listQuery := fmt.Sprintf("SELECT * FROM (%s) AS merged ORDER BY time DESC LIMIT ?", union)
	listArgs := append(append([]any{}, args...), pageSize)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list history: %w", err)
	}
	defer rows.Close()

	var items []PunishmentRow
	for rows.Next() {
		var typeStr string
		var row PunishmentRow
		if err := rows.Scan(
			&typeStr, &row.ID, &row.PlayerUUID, &row.Reason, &row.ModeratorUUID, &row.ModeratorName, &row.Time,
			&row.RemovedByUUID, &row.RemovedByName, &row.Until, &row.Removed,
			&row.RemovedByDate, &row.RemovedByReason, &row.Active, &row.Silent,
			&row.ServerOrigin, &row.ServerScope, &row.IPBan, &row.Acknowledged,
		); err != nil {
			return nil, 0, fmt.Errorf("scan history row: %w", err)
		}
		row.Type = domain.PunishmentType(typeStr)
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate history: %w", err)
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
