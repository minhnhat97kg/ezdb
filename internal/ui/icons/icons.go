package icons

const (
	// Database Icons (Nerd Font)
	IconPostgres = ""
	IconMySQL    = ""
	IconSQLite   = "󰆼"
	IconGeneric  = "󰆼"

	// Utility Icons
	IconLock      = "󰌾"
	IconSuccess   = "✓"
	IconError     = "⚠"
	IconSelect    = "▸"
	IconBullet    = "•"
	IconSeparator = "  •  "
	IconArrow     = "👉"
)

func GetDatabaseIcon(dbType string) string {
	switch dbType {
	case "postgres", "postgresql":
		return IconPostgres
	case "mysql":
		return IconMySQL
	case "sqlite":
		return IconSQLite
	default:
		return IconGeneric
	}
}
