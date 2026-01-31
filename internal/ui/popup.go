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
	if m.showActionPopup {
		// Render action popup on top of the results popup
		// First render the results popup as the "main" background for the action popup
		resultsPopup := m.renderResultsPopup(main)
		return m.renderActionPopup(resultsPopup)
	}
	if m.popupEntry == nil || m.popupResult == nil {
		return main
	}
	return m.renderResultsPopup(main)
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

	content.WriteString("\n\n(Press q or Esc to close, 'a' for actions)")

	// Box styling with background
	popupBox := PopupStyle.
		Width(min(120, m.width-4)).
		MaxHeight(m.height - 4).
		Background(lipgloss.Color("#1a1b26")). // Dark background for popup
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
