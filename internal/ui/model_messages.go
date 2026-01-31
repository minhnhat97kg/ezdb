// internal/ui/model_messages.go
// Consolidated message types for Bubble Tea Update cycle
package ui

import (
	"github.com/nhath/ezdb/internal/config"
	"github.com/nhath/ezdb/internal/db"
	"github.com/nhath/ezdb/internal/history"
)

// DebounceMsg triggers the actual autocomplete lookup after delay
type DebounceMsg struct {
	ID int
}

// QueryResultMsg sent when query execution completes
type QueryResultMsg struct {
	Result     *db.QueryResult
	Entry      *history.HistoryEntry
	AllEntries []*history.HistoryEntry // For multi-statement execution
	Err        error
}

// HistoryLoadedMsg sent when history loads from SQLite
type HistoryLoadedMsg struct {
	Entries []history.HistoryEntry
	Err     error
}

// RerunResultMsg sent when re-running a query from history
type RerunResultMsg struct {
	Entry  *history.HistoryEntry
	Result *db.QueryResult
	Err    error
}

// PagerFinishedMsg indicates external pager finished
type PagerFinishedMsg struct {
	Err error
}

// ProfileConnectedMsg is sent when profile connection completes
type ProfileConnectedMsg struct {
	Driver db.Driver
	Err    error
}

// ClipboardCopiedMsg is sent when clipboard copy completes
type ClipboardCopiedMsg struct {
	Text string
	Err  error
}

// ExportTableCompleteMsg is sent when table export completes
type ExportTableCompleteMsg struct {
	Filename string
	Rows     int
	Err      error
}

// ImportTableCompleteMsg is sent when table import completes
type ImportTableCompleteMsg struct {
	Rows int
	Err  error
}

// ExportCompleteMsg is sent when export is complete
type ExportCompleteMsg struct {
	Path string
	Err  error
}

// ThemeSelectedMsg is sent when a theme is selected
type ThemeSelectedMsg struct {
	ThemeName string
	Theme     config.Theme
}
