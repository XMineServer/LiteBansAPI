package repository

import (
	"database/sql"
)

type PunishmentBase struct {
	ID            int64
	UUID          sql.NullString
	IP            sql.NullString
	Reason        sql.NullString
	BannedByUUID  string
	BannedByName  sql.NullString
	Time          int64
	Until         int64
	Template      int32
	ServerScope   sql.NullString
	ServerOrigin  sql.NullString
	Silent        sql.NullBool
	IpBan         sql.NullBool
	IpBanWildcard sql.NullBool
	Active        sql.NullBool
}

type Punishment interface {
	Base() *PunishmentBase
	Type() string
}

type BanRow struct {
	PunishmentBase
	RemovedByUUID   sql.NullString
	RemovedByName   sql.NullString
	RemovedByReason sql.NullString
	RemovedByDate   sql.NullTime
}

func (r *BanRow) Base() *PunishmentBase {
	return &r.PunishmentBase
}

func (r *BanRow) Type() string {
	return "ban"
}

type WarningRow struct {
	PunishmentBase
	RemovedByUUID   sql.NullString
	RemovedByName   sql.NullString
	RemovedByReason sql.NullString
	RemovedByDate   sql.NullTime
	Warned          sql.NullBool
}

func (r *WarningRow) Base() *PunishmentBase {
	return &r.PunishmentBase
}

func (r *WarningRow) Type() string {
	return "warning"
}

type MuteRow struct {
	PunishmentBase
	RemovedByUUID   sql.NullString
	RemovedByName   sql.NullString
	RemovedByReason sql.NullString
	RemovedByDate   sql.NullTime
}

func (r *MuteRow) Base() *PunishmentBase {
	return &r.PunishmentBase
}

func (r *MuteRow) Type() string {
	return "mute"
}

type KickRow struct {
	PunishmentBase
}

func (r *KickRow) Base() *PunishmentBase {
	return &r.PunishmentBase
}

func (r *KickRow) Type() string {
	return "kick"
}

type UnifiedRow struct {
	PunishmentBase
	typ             string
	RemovedByUUID   sql.NullString
	RemovedByName   sql.NullString
	RemovedByReason sql.NullString
	RemovedByDate   sql.NullTime
	Warned          sql.NullBool
}

func (r *UnifiedRow) Base() *PunishmentBase {
	return &r.PunishmentBase
}

func (r *UnifiedRow) Type() string {
	return r.typ
}
