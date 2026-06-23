package repository

import (
	"fmt"
	"xmine/litebans-api/internal/domain"
)

// OfflineUUIDMarker is the special UUID value LiteBans uses for records without a linked account.
const OfflineUUIDMarker = "#offline#"

// tableName returns the fully-qualified table name for a punishment type under the configured prefix.
func tableName(prefix string, t domain.PunishmentType) string {
	switch t {
	case domain.TypeBan:
		return prefix + "bans"
	case domain.TypeMute:
		return prefix + "mutes"
	case domain.TypeWarning:
		return prefix + "warnings"
	case domain.TypeKick:
		return prefix + "kicks"
	default:
		panic(fmt.Sprintf("unknown punishment type %q", t))
	}
}

func historyTableName(prefix string) string {
	return prefix + "history"
}

// hasDurationColumns reports whether a punishment type's table has until/active/removed/silent columns.
func hasDurationColumns(t domain.PunishmentType) bool {
	return t != domain.TypeKick
}

// hasIPBanColumn reports whether a punishment type's table has an ipban column.
func hasIPBanColumn(t domain.PunishmentType) bool {
	return t == domain.TypeBan
}

// hasAcknowledgedColumn reports whether a punishment type's table has a "warned" (acknowledged) column.
func hasAcknowledgedColumn(t domain.PunishmentType) bool {
	return t == domain.TypeWarning
}
