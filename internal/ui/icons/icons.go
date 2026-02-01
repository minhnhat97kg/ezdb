package icons

// Nerd Font icons for consistent UI
// Reference: https://www.nerdfonts.com/cheat-sheet
const (
	// Database Icons
	IconPostgres = "" // nf-dev-postgresql
	IconMySQL    = "" // nf-dev-mysql
	IconSQLite   = "󰆼" // nf-md-database
	IconGeneric  = "󰆼" // nf-md-database

	// Status Icons
	IconSuccess = "" // nf-fa-check
	IconError   = "" // nf-fa-remove
	IconWarning = "⚠" // nf-fa-exclamation_triangle
	IconInfo    = "" // nf-fa-info

	// Navigation Icons
	IconSelect    = "▶" // nf-fa-chevron_right
	IconExpanded  = "▼" // nf-fa-chevron_down
	IconCollapsed = "▶" // nf-fa-chevron_right
	IconArrowUp   = "↑" // nf-cod-arrow_up
	IconArrowDown = "↓" // nf-cod-arrow_down
	IconPointer     = "❯" // nf-cod-triangle_right
	IconPointerFill = "►" // nf-fa-hand_o_right
	IconVertNav     = "󰁼" // nf-md-arrow_up_down

	// Action Icons
	IconEdit    = "✎" // nf-fa-edit
	IconDelete  = "🗑" // nf-fa-trash
	IconAdd     = "+" // nf-fa-plus
	IconSave    = "💾" // nf-fa-floppy_o
	IconExport  = "󰈔" // nf-md-file_export
	IconImport  = "󰈠" // nf-md-file_import
	IconRefresh = "↻" // nf-fa-refresh
	IconSearch  = "🔍" // nf-fa-search
	IconFilter  = "" // nf-fa-filter
	IconSort    = "" // nf-oct-sort_asc
	IconCopy    = "" // nf-fa-copy
	IconExecute = "▶" // nf-fa-play
	IconStop    = "■" // nf-fa-stop
	IconCancel  = "✕" // nf-fa-ban

	// UI Elements
	IconBullet    = "•" // nf-fa-circle (small)
	IconSeparator = "  •  "
	IconDivider   = "│"
	IconConnect   = "" // nf-cod-link
	IconLock      = "🔒" // nf-fa-lock
	IconUnlock    = "🔓" // nf-fa-unlock
	IconKey       = "🔑" // nf-fa-key
	IconSSH       = "󰢹" // nf-md-ssh

	// Schema Browser Icons
	IconTable    = "" // nf-fa-table
	IconColumn   = "" // nf-fa-columns
	IconIndex    = "󰛉" // nf-md-alpha_i_box
	IconPKey     = "🔑" // nf-fa-key
	IconFKey     = "󰌷" // nf-md-link_variant
	IconView     = "" // nf-fa-eye
	IconFunction = "󰊕" // nf-md-function
	IconTrigger  = "󰗗" // nf-md-flash
	IconSchema   = "󱔗" // nf-md-folder_table

	// Suggestion Type Icons
	IconKeyword = "󰌏" // nf-md-keyboard
	IconTypeT   = "󰾹" // nf-md-alpha_t_box (Table)
	IconTypeC   = "󰾺" // nf-md-alpha_c_box (Column)
	IconTypeF   = "󰊕" // nf-md-function

	// Query Status
	IconRunning = "⠋" // nf-fa-spinner
	IconPending = "" // nf-fa-clock_o
	IconQuery   = "󰘳" // nf-md-code_braces

	// Mode Icons
	IconInsertMode = "I" // nf-fa-i_cursor
	IconVisualMode = "V" // nf-fa-eye
	IconNormalMode = "N" // nf-cod-terminal

	// Misc
	IconHelp       = "" // nf-fa-question_circle
	IconSettings   = "⚙" // nf-fa-cog
	IconHistory    = "" // nf-fa-history
	IconProfile    = "👤" // nf-fa-user
	IconConnection = "" // nf-fa-plug
	IconRows       = "󰕭" // nf-md-table_row
	IconTime       = "" // nf-fa-clock_o
	IconDuration   = "󱎫" // nf-md-timer_outline
)

// GetDatabaseIcon returns the appropriate icon for a database type
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

// GetStatusIcon returns the appropriate icon for a status
func GetStatusIcon(status string) string {
	switch status {
	case "success", "ok":
		return IconSuccess
	case "error", "fail", "failed":
		return IconError
	case "warning", "warn":
		return IconWarning
	case "info":
		return IconInfo
	case "running", "pending":
		return IconRunning
	default:
		return IconSuccess
	}
}

// GetSuggestionTypeIcon returns icon for autocomplete suggestion type
func GetSuggestionTypeIcon(sugType string) string {
	switch sugType {
	case "keyword", "K":
		return IconKeyword
	case "table", "T":
		return IconTable
	case "column", "C":
		return IconColumn
	case "function", "F":
		return IconFunction
	default:
		return IconBullet
	}
}
