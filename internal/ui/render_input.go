package ui

import "github.com/nhath/ezdb/internal/ui/highlight"

// highlightView applies syntax highlighting to the textarea view.
// Uses highlight.SQLPreserveANSI to preserve existing ANSI codes (cursor, etc.)
func (m Model) highlightView(view string) string {
	return highlight.SQLPreserveANSI(view)
}
