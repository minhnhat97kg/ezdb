// internal/ui/messages.go
package ui

import (
	"github.com/nhath/ezdb/internal/db"
	"github.com/nhath/ezdb/internal/history"
)

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
// PagerFinishedMsg indicates external pager finished
type PagerFinishedMsg struct {
	Err error
}

// ProfileConnectedMsg is sent when profile connection completes
type ProfileConnectedMsg struct {
	Driver db.Driver
	Err    error
}
