package repository

import (
	"database/sql"
	"xmine/litebans-api/internal/domain"
)

// PunishmentRow is the raw row shape shared by bans/mutes/warnings/kicks.
// Columns that don't exist for a given punishment type are populated as NULL by the SELECT.
type PunishmentRow struct {
	Type            domain.PunishmentType
	ID              int64
	PlayerUUID      string
	Reason          sql.NullString
	ModeratorUUID   sql.NullString
	ModeratorName   sql.NullString
	RemovedByUUID   sql.NullString
	RemovedByName   sql.NullString
	Time            int64
	Until           sql.NullInt64
	Removed         sql.NullBool
	RemovedByDate   sql.NullInt64
	RemovedByReason sql.NullString
	Active          sql.NullBool
	ServerOrigin    sql.NullString
	ServerScope     sql.NullString
	Silent          sql.NullBool
	IPBan           sql.NullBool
	Acknowledged    sql.NullBool
}

// HistoryItem is one row out of the UNION ALL across all 4 punishment tables.
type HistoryItem struct {
	PunishmentRow
}
