package ui

import (
	"github.com/nhath/ezdb/internal/history"
	"github.com/nhath/ezdb/internal/ui/components/historylist"
)

// HistoryItemAdapter wraps HistoryEntry to implement historylist.Item
type HistoryItemAdapter struct {
	entry history.HistoryEntry
}

// NewHistoryItemAdapter creates a new adapter
func NewHistoryItemAdapter(entry history.HistoryEntry) HistoryItemAdapter {
	return HistoryItemAdapter{entry: entry}
}

// Implement historylist.Item interface
func (a HistoryItemAdapter) ID() int64                      { return a.entry.ID }
func (a HistoryItemAdapter) Query() string                  { return a.entry.Query }
func (a HistoryItemAdapter) QueryPreview(maxLen int) string { return a.entry.QueryPreview(maxLen) }
func (a HistoryItemAdapter) Status() string                 { return a.entry.Status }
func (a HistoryItemAdapter) ErrorMessage() string           { return a.entry.ErrorMessage }
func (a HistoryItemAdapter) Preview() string                { return a.entry.Preview }
func (a HistoryItemAdapter) DurationMs() int64              { return a.entry.DurationMs }
func (a HistoryItemAdapter) RowCount() int                  { return a.entry.RowCount }
func (a HistoryItemAdapter) ExecutedAtFormatted() string {
	return a.entry.ExecutedAt.Format("15:04:05")
}

// Entry returns the underlying HistoryEntry
func (a HistoryItemAdapter) Entry() history.HistoryEntry { return a.entry }

// ConvertToItems converts a slice of HistoryEntry to historylist.Item slice
func ConvertToItems(entries []history.HistoryEntry) []historylist.Item {
	items := make([]historylist.Item, len(entries))
	for i, e := range entries {
		items[i] = NewHistoryItemAdapter(e)
	}
	return items
}
