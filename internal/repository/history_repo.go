package repository

import (
	"context"
	"database/sql"
	"fmt"
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
