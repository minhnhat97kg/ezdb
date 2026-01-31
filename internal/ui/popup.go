package ui

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/rmhubbert/bubbletea-overlay"

	"github.com/nhath/ezdb/internal/db"
)

func (m Model) renderPopupOverlay(main string) string {
	if m.confirming {
		return m.renderConfirmPopup(main)
	}

	// Layer the popups: results -> action menu -> row action
	resultsView := main
	if m.popupEntry != nil && m.popupResult != nil {
		resultsView = m.renderResultsPopup(main)
	}

	if m.showActionPopup {
		resultsView = m.renderActionPopup(resultsView)
	}

	if m.showRowActionPopup {
		resultsView = m.renderRowActionPopup(resultsView)
	}

	return resultsView
}

func (m Model) renderResultsPopup(main string) string {
	var content strings.Builder

	// Header
	q := m.popupEntry.Query
	if len(q) > 100 {
		q = q[:97] + "..."
	}
	content.WriteString(fmt.Sprintf("Query: %s\n", q))
	content.WriteString(fmt.Sprintf("Execution Time: %dms | Rows: %d\n\n",
		m.popupEntry.DurationMs, m.popupResult.RowCount))

	// Table
	if len(m.popupResult.Columns) > 0 {
		content.WriteString(m.popupTable.View())
	} else {
		content.WriteString("(No results)")
	}

	// Show keyboard shortcuts below table
	if m.tableFilterActive {
		content.WriteString("\n\n")
		content.WriteString(m.tableFilterInput.View())
	} else {
		content.WriteString("\n\n")
		shortcuts := lipgloss.NewStyle().Faint(true).Render(
			"n/b:page • h/l:scroll • /:filter • enter:actions • e:export • q:close")
		content.WriteString(shortcuts)
	}

	// Popup width constraint
	// Use 100% width but rely on table being smaller (width-30)
	maxPopupWidth := m.width

	popupBox := PopupStyle.
		MaxWidth(maxPopupWidth).
		MaxHeight(m.height - 4).
		Background(lipgloss.Color("#1a1b26")).
		Render(content.String())

	// Use bubbletea-overlay to composite popup over main content
	return overlay.Composite(popupBox, main, overlay.Center, overlay.Center, 0, 0)
}

func (m Model) renderActionPopup(main string) string {
	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD")).Render("Row Actions"))
	content.WriteString("\n\n")
	content.WriteString("• Edit Row\n")
	content.WriteString("• Delete Row\n")
	content.WriteString("• Copy Row JSON\n")
	content.WriteString("• Copy Row CSV\n")
	content.WriteString("\n(Press q or Esc to close)")

	popupBox := PopupStyle.
		Width(40).
		Background(lipgloss.Color("#1a1b26")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#bd93f9")).
		Padding(1).
		Render(content.String())

	return overlay.Composite(popupBox, main, overlay.Center, overlay.Center, 0, 0)
}

func (m Model) renderRowActionPopup(main string) string {
	highlightedRow := m.popupTable.HighlightedRow()

	var content strings.Builder
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#8BE9FD")).
		Render("Row Actions")
	content.WriteString(header + "\n\n")

	// Show available actions
	content.WriteString("1 - View Full Row\n")
	content.WriteString("2 - Copy as JSON\n")
	content.WriteString("3 - Copy as CSV\n")
	content.WriteString("4 - Select this row\n")

	if highlightedRow.Data != nil && m.popupResult != nil {
		content.WriteString("\nPreview:\n")

		// Show first 2 fields only
		count := 0
		for _, col := range m.popupResult.Columns {
			if count >= 2 {
				break
			}
			if val, ok := highlightedRow.Data[col]; ok {
				valStr := fmt.Sprintf("%v", val)
				if len(valStr) > 25 {
					valStr = valStr[:22] + "..."
				}
				if len(col) > 12 {
					col = col[:9] + "..."
				}
				content.WriteString(fmt.Sprintf("%s: %s\n", col, valStr))
				count++
			}
		}
	}

	content.WriteString("\nPress 1-3, q to close")

	// Calculate max content width
	// Total rendered width = content width + 2 (borders) + 2 (padding) = content + 4
	// So: content width = terminal width - 4 - safety margin
	maxContentWidth := m.width - 8 // Border(2) + Padding(2) + Safety margin(4)
	if maxContentWidth > 35 {
		maxContentWidth = 35 // Cap at 35 for readability
	}
	if maxContentWidth < 20 {
		maxContentWidth = 20 // Minimum viable width
	}

	popupBox := lipgloss.NewStyle().
		Width(maxContentWidth).
		Background(lipgloss.Color("#1a1b26")).
		Foreground(lipgloss.Color("#D8DEE9")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#ff79c6")).
		Padding(1).
		Render(content.String())

	return overlay.Composite(popupBox, main, overlay.Center, overlay.Center, 0, 0)
}

func (m Model) renderConfirmPopup(main string) string {
	var content strings.Builder

	header := WarningStyle.Render(" CONFIRM DESTRUCTIVE ACTION ")
	content.WriteString(header + "\n\n")
	content.WriteString("Strict Mode is active. Do you really want to execute this query?\n\n")

	// Query Preview
	q := m.pendingQuery
	if len(q) > 400 {
		q = q[:397] + "..."
	}
	content.WriteString(lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(textFaint).
		Padding(1).
		Foreground(textPrimary).
		Render(q))

	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Bold(true).Foreground(successColor).Render("(y) Yes, execute") + "  " +
		lipgloss.NewStyle().Bold(true).Foreground(errorColor).Render("(n/Esc) No, cancel"))

	// Box styling with background
	popupBox := PopupStyle.
		Width(min(80, m.width-4)).
		Background(lipgloss.Color("#1a1b26")). // Dark background for popup
		Render(content.String())

	// Use bubbletea-overlay to composite popup over main content
	return overlay.Composite(popupBox, main, overlay.Center, overlay.Center, 0, 0)
}

// openPager opens the result in an external pager
func (m Model) openPager(result *db.QueryResult) tea.Cmd {
	if m.config.Pager == "" || result == nil || len(result.Rows) == 0 {
		return nil
	}

	// Create temp file
	f, err := os.CreateTemp("", "ezdb-*.csv")
	if err != nil {
		return func() tea.Msg { return PagerFinishedMsg{Err: err} }
	}
	defer f.Close()

	// Write CSV
	w := csv.NewWriter(f)
	w.Comma = ','

	if err := w.Write(result.Columns); err != nil {
		return func() tea.Msg { return PagerFinishedMsg{Err: err} }
	}
	for _, row := range result.Rows {
		if err := w.Write(row); err != nil {
			return func() tea.Msg { return PagerFinishedMsg{Err: err} }
		}
	}
	w.Flush()

	// Command
	parts := strings.Fields(m.config.Pager)
	cmdName := parts[0]
	args := append(parts[1:], f.Name())

	c := exec.Command(cmdName, args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		os.Remove(f.Name()) // Cleanup
		return PagerFinishedMsg{Err: err}
	})
}
