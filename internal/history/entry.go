// internal/history/entry.go
package history

import "time"

// HistoryEntry represents a single query execution in history
type HistoryEntry struct {
	ID           int64
	ProfileName  string
	Query        string
	ExecutedAt   time.Time
	DurationMs   int64
	RowCount     int
	Status       string `json:"status"` // "success", "error"
	ErrorMessage string `json:"error_message,omitempty"`
	Preview      string `json:"preview,omitempty"` // First 3 rows
}

// QueryPreview returns a truncated version of the query
func (e *HistoryEntry) QueryPreview(maxLen int) string {
	q := e.Query
	if len(q) > maxLen {
		return q[:maxLen-3] + "..."
	}
	return q
}
