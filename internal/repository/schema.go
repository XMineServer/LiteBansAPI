package repository

// OfflineUUIDMarker is the special UUID value LiteBans uses for records without a linked account.
const OfflineUUIDMarker = "#offline#"

func historyTableName(prefix string) string {
	return prefix + "history"
}
