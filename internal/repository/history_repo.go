package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type HistoryRepository struct {
	db          *sql.DB
	tablePrefix string
}

func NewHistoryRepository(db *sql.DB, tablePrefix string) *HistoryRepository {
	return &HistoryRepository{db: db, tablePrefix: tablePrefix}
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

// LatestNamesByUUIDs resolves the most recently recorded display name for each of the given
// uuids in a single query, used to avoid N+1 name lookups when rendering a list of records.
// uuids not found in history are simply absent from the returned map.
func (r *HistoryRepository) LatestNamesByUUIDs(ctx context.Context, uuids []string) (map[string]string, error) {
	result := make(map[string]string, len(uuids))
	if len(uuids) == 0 {
		return result, nil
	}

	table := historyTableName(r.tablePrefix)
	placeholders := make([]string, len(uuids))
	args := make([]any, len(uuids))
	for i, uuid := range uuids {
		placeholders[i] = "?"
		args[i] = uuid
	}
	query := fmt.Sprintf(
		`SELECT uuid, name FROM (
			SELECT uuid, name, ROW_NUMBER() OVER (PARTITION BY uuid ORDER BY date DESC) AS rn
			FROM %s WHERE uuid IN (%s)
		) ranked WHERE rn = 1`,
		table, strings.Join(placeholders, ", "),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("latest names by uuids: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var uuid, name string
		if err := rows.Scan(&uuid, &name); err != nil {
			return nil, fmt.Errorf("scan latest names by uuids: %w", err)
		}
		result[uuid] = name
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate latest names by uuids: %w", err)
	}
	return result, nil
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
