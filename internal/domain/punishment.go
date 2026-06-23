package domain

// PunishmentType identifies one of the 4 punishment kinds.
type PunishmentType string

const (
	TypeBan     PunishmentType = "ban"
	TypeMute    PunishmentType = "mute"
	TypeWarning PunishmentType = "warning"
	TypeKick    PunishmentType = "kick"
)

// ParsePunishmentType maps a URL path segment (plural, e.g. "bans") to a PunishmentType.
func ParsePunishmentType(pathSegment string) (PunishmentType, bool) {
	switch pathSegment {
	case "bans":
		return TypeBan, true
	case "mutes":
		return TypeMute, true
	case "warnings":
		return TypeWarning, true
	case "kicks":
		return TypeKick, true
	default:
		return "", false
	}
}

// Moderator describes who issued or removed a punishment.
type Moderator struct {
	UUID      *string `json:"uuid"`
	Name      *string `json:"name"`
	IsConsole bool    `json:"isConsole"`
}

// Player describes a resolved player identity, as returned by the lookup endpoint.
type Player struct {
	UUID        *string `json:"uuid"`
	Name        *string `json:"name"`
	IsConsole   bool    `json:"isConsole"`
	OfflineMode bool    `json:"offlineMode"`
}

// Removed describes how a punishment was closed early/explicitly.
type Removed struct {
	By                   Moderator `json:"by"`
	At                   int64     `json:"at"`
	Reason               *string   `json:"reason"`
	ExpiredAutomatically bool      `json:"expiredAutomatically"`
}

// Punishment is the unified representation of a Ban/Mute/Warning/Kick record.
type Punishment struct {
	ID           any            `json:"id"`
	Type         PunishmentType `json:"type"`
	PlayerUUID   string         `json:"playerUuid"`
	Reason       string         `json:"reason"`
	Moderator    Moderator      `json:"moderator"`
	IssuedAt     int64          `json:"issuedAt"`
	ExpiresAt    *int64         `json:"expiresAt,omitempty"`
	Permanent    *bool          `json:"permanent,omitempty"`
	Active       *bool          `json:"active,omitempty"`
	Expired      *bool          `json:"expired,omitempty"`
	IPBan        *bool          `json:"ipBan,omitempty"`
	Silent       *bool          `json:"silent,omitempty"`
	ServerOrigin *string        `json:"serverOrigin"`
	ServerScope  *string        `json:"serverScope,omitempty"`
	Removed      *Removed       `json:"removed,omitempty"`
	Acknowledged *bool          `json:"acknowledged,omitempty"`
}

// PunishmentList is the paginated list response for offset-paginated endpoints.
type PunishmentList struct {
	Items      []Punishment `json:"items"`
	Page       int          `json:"page"`
	PageSize   int          `json:"pageSize"`
	TotalItems int64        `json:"totalItems"`
	TotalPages int          `json:"totalPages"`
}

// HistoryCursors describes the cursor bounds of a returned history page.
type HistoryCursors struct {
	Before *int64 `json:"before"`
	After  *int64 `json:"after"`
}

// HistoryPage is the cursor-paginated response for player history endpoints.
type HistoryPage struct {
	Items      []Punishment   `json:"items"`
	TotalItems int64          `json:"totalItems"`
	Cursors    HistoryCursors `json:"cursors"`
}

// Stats are the aggregate counters for the dashboard endpoint.
type Stats struct {
	Bans     int64 `json:"bans"`
	Mutes    int64 `json:"mutes"`
	Warnings int64 `json:"warnings"`
	Kicks    int64 `json:"kicks"`
}
