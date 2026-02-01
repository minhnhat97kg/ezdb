package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nhath/ezdb/internal/ui/highlight"
	"github.com/nhath/ezdb/internal/ui/icons"
	"github.com/nhath/ezdb/internal/ui/styles"
)

func (m Model) updateHistoryViewport() Model {
	// Calculate dynamic editor height
	editorContent := m.editor.Value()
	lineCount := strings.Count(editorContent, "\n") + 1
	minHeight := 3
	maxHeight := m.height / 2
	if maxHeight < minHeight {
		maxHeight = minHeight
	}
	editorHeight := lineCount
	if editorHeight < minHeight {
		editorHeight = minHeight
	}
	if editorHeight > maxHeight {
		editorHeight = maxHeight
	}
	m.editor.SetHeight(editorHeight)

	// Status bar
	statusBar := m.renderStatusBar()
	// Help
	helpText := m.renderHelp()
	// Input area
	inputWidth := m.width - 4
	inputView := styles.InputStyle.Width(inputWidth).Render(m.highlightView(m.editor.View()))
	// Suggestions (only in insert mode)
	chromeHeight := lipgloss.Height(statusBar) + lipgloss.Height(helpText)
	availableHeight := m.height - chromeHeight
	if availableHeight < 0 {
		availableHeight = 0
	}

	historyHeight := availableHeight - lipgloss.Height(inputView)
	if historyHeight < 0 {
		historyHeight = 0
	}

	m.viewport.Width = m.width
	m.viewport.Height = historyHeight
	m.viewport.SetContent(m.renderHistoryContent(historyHeight))
	return m
}

// renderHistoryContent generates the string for the viewport
func (m Model) renderHistoryContent(minHeight int) string {
	if len(m.history) == 0 {
		return ""
	}

	var sections []string
	for i := range m.history {
		sections = append(sections, strings.TrimRight(m.renderHistoryItem(i), "\n"))
	}
	// Join with newline separator for margin between cards
	content := strings.Join(sections, "\n\n")

	// Add a bit more padding at the top of the entire list for the first item
	content = lipgloss.NewStyle().MarginTop(1).Render(content)

	h := lipgloss.Height(content)
	if h < minHeight && h > 0 {
		return strings.Repeat("\n", minHeight-h) + content
	}

	return content
}

// renderHistoryItem renders a single history entry
func (m Model) renderHistoryItem(i int) string {
	if i < 0 || i >= len(m.history) {
		return ""
	}
	entry := m.history[i]
	isSelected := m.mode == VisualMode && i == m.selected
	isExpanded := m.expandedID == entry.ID

	// No wrapper style needed - header handles its own styling
	_ = isSelected // Used below for header styling

	// Content construction
	var content strings.Builder

	// Build header section (query + metadata) with subtle background
	var headerContent strings.Builder

	// Query Line with syntax highlighting
	if entry.Status != "info" {
		indicator := " " + icons.IconCollapsed + " "
		if isExpanded {
			indicator = " " + icons.IconExpanded + " "
		}
		headerContent.WriteString(indicator)
	}

	queryText := entry.QueryPreview(m.width - 14) // Adjusted for margins
	if isExpanded {
		queryText = entry.Query
	}

	// SQL syntax highlighting (background stripped, foreground only)
	if entry.Status == "info" {
		headerContent.WriteString(queryText)
	} else {
		headerContent.WriteString(highlight.SQL(queryText))
	}

	// [EXPANDED] indicator
	if isExpanded {
		headerContent.WriteString(" [EXPANDED]")
	}
	headerContent.WriteString("\n")

	// Meta Line - plain text for consistent background
	statusIcon := icons.IconSuccess
	if entry.Status == "error" {
		statusIcon = icons.IconError
	} else if entry.Status == "info" {
		statusIcon = icons.IconInfo
	}

	var metaInfo string
	if entry.Status == "info" {
		metaInfo = fmt.Sprintf("  %s %s", statusIcon, entry.ExecutedAt.Format("15:04:05"))
	} else {
		metaInfo = fmt.Sprintf("  %s %dms | %d rows | %s", statusIcon, entry.DurationMs, entry.RowCount, entry.ExecutedAt.Format("15:04:05"))
	}
	headerContent.WriteString(metaInfo)

	// Apply full-width background to entire header section
	// Using cardBg for better contrast
	headerBg := styles.CardBg()

	headerStyle := lipgloss.NewStyle().
		Background(headerBg).
		Foreground(styles.TextPrimary()). // Nord4 text
		Width(m.width).                   // Full viewport width
		Padding(1, 1)

	// Add left accent border for selected items
	if isSelected {
		headerStyle = headerStyle.
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(styles.AccentColor()). // Global accent
			PaddingLeft(1)
	}

	content.WriteString(headerStyle.Render(headerContent.String()))
	content.WriteString("\n")

	// Details
	if entry.ErrorMessage != "" {
		if isSelected {
			content.WriteString(styles.ErrorStyle.Render("  " + entry.ErrorMessage))
			content.WriteString("\n")
		} else {
			content.WriteString(styles.ErrorGrayStyle.Render("  " + entry.ErrorMessage))
			content.WriteString("\n")
		}
	}

	if isExpanded && entry.RowCount > 0 {
		// Render the expanded table component
		tableContentView := m.expandedTable.View()
		previewStyle := lipgloss.NewStyle().
			Padding(1, 4)

		if isSelected {
			previewStyle = previewStyle.
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(styles.AccentColor()).
				PaddingLeft(3) // Adjusted for BorderLeft (4-1=3)
		}

		content.WriteString(previewStyle.Render(tableContentView))
		content.WriteString("\n")
	} else if isExpanded && entry.Preview != "" {
		// Fallback for non-tabular preview (e.g., affected rows message)
		previewLines := strings.Split(entry.Preview, "\n")
		var previewBody strings.Builder
		for _, line := range previewLines {
			previewBody.WriteString(line + "\n")
		}

		previewStyle := lipgloss.NewStyle().
			Foreground(styles.TextFaint()).
			Padding(1, 4)

		if isSelected {
			previewStyle = previewStyle.
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(styles.AccentColor()).
				PaddingLeft(3)
		}

		content.WriteString(previewStyle.Render(previewBody.String()))
		content.WriteString("\n")
	}

	// Add spacing between history items for visual separation
	// Add margin between cards
	content.WriteString("\n\n")

	return content.String()
}

// ensureSelectionVisible updates the viewport to keep the selected item in view
func (m Model) ensureSelectionVisible() Model {
	if len(m.history) == 0 {
		return m
	}

	var sections []string
	for i := range m.history {
		sections = append(sections, strings.TrimRight(m.renderHistoryItem(i), "\n"))
	}

	// Calculate base heights including margins
	top := 1 // Account for the MarginTop(1) added in renderHistoryContent
	for i := 0; i < m.selected; i++ {
		// lipgloss.Height(sections[i]) includes the item's Margin(1, 1).
		// Margin(1, 1) means 1 top, 1 bottom. Total 2 lines of vertical margin.
		top += lipgloss.Height(sections[i]) + 1 // +1 for JoinVertical newline
	}

	itemHeight := lipgloss.Height(sections[m.selected])
	bottom := top + itemHeight

	// Calculate total content height
	content := lipgloss.NewStyle().MarginTop(1).Render(lipgloss.JoinVertical(lipgloss.Left, sections...))
	totalHeight := lipgloss.Height(content)

	vHeight := m.viewport.Height
	if totalHeight < vHeight {
		padding := vHeight - totalHeight
		top += padding
		bottom += padding
	}

	// Viewport window
	vTop := m.viewport.YOffset
	vBottom := vTop + vHeight // Visible bottom

	if top < vTop {
		m.viewport.SetYOffset(top)
	} else if bottom > vBottom {
		m.viewport.SetYOffset(bottom - vHeight)
	}

	return m
}
