package ui

// highlightView applies syntax highlighting to the textarea view.
// Uses HighlightSQLPreserveANSI to preserve existing ANSI codes (cursor, etc.)
func (m Model) highlightView(view string) string {
	return HighlightSQLPreserveANSI(view)
}
