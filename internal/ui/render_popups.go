package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nhath/ezdb/internal/ui/styles"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

// --- Results popup orchestration ---

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

	if m.showExportPopup {
		resultsView = m.renderExportPopup(resultsView)
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

		// Helper to get key string
		k := func(keys []string, def string) string {
			if len(keys) > 0 {
				return keys[0]
			}
			return def
		}

		shortcutsStr := fmt.Sprintf("%s/%s:page • %s/%s:scroll • %s:filter • %s:actions • %s:export • %s:close • %s:help",
			k(m.config.Keys.NextPage, "n"), k(m.config.Keys.PrevPage, "b"),
			k(m.config.Keys.ScrollLeft, "h"), k(m.config.Keys.ScrollRight, "l"),
			k(m.config.Keys.Filter, "/"),
			k(m.config.Keys.RowAction, "enter"),
			k(m.config.Keys.Export, "ctrl+e"),
			k(m.config.Keys.Exit, "q"),
			k(m.config.Keys.Help, "?"))

		shortcuts := lipgloss.NewStyle().Faint(true).Render(shortcutsStr)
		content.WriteString(shortcuts)
	}

	// Popup sizing
	popupWidth := m.width - 10
	if popupWidth < 60 {
		popupWidth = 60
	}
	if popupWidth > m.width {
		popupWidth = m.width - 4
	}
	popupHeight := m.height - 6
	if popupHeight < 15 {
		popupHeight = 15
	}

	// Table handles its own horizontal scrolling via h/l keys
	popupBox := styles.PopupStyle.
		Width(popupWidth).
		Height(popupHeight).
		Background(styles.PopupBg()).
		Render(content.String())

	// Use bubbletea-overlay to composite popup over main content
	return overlay.Composite(popupBox, main, overlay.Center, overlay.Center, 0, 0)
}

func (m Model) renderActionPopup(main string) string {
	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.AccentColor()).Render("Row Actions"))
	content.WriteString("\n\n")
	content.WriteString("• Edit Row\n")
	content.WriteString("• Delete Row\n")
	content.WriteString("• Copy Row JSON\n")
	content.WriteString("• Copy Row CSV\n")
	content.WriteString("\n(Press q or Esc to close)")

	popupBox := styles.PopupStyle.
		Width(40).
		Background(styles.PopupBg()).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.BorderColor()).
		Padding(1).
		Render(content.String())

	return overlay.Composite(popupBox, main, overlay.Center, overlay.Center, 0, 0)
}

func (m Model) renderRowActionPopup(main string) string {
	var content strings.Builder
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.AccentColor()).
		Render("Row Actions")
	content.WriteString(header + "\n\n")

	// Show available actions
	content.WriteString("1 - Select this row\n")
	content.WriteString("2 - View Full Row\n")
	content.WriteString("3 - Copy as JSON\n")
	content.WriteString("4 - Copy as CSV\n")
	content.WriteString("\nPress 1-4, q to close")

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
		Background(styles.PopupBg()).
		Foreground(styles.TextPrimary()).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.HighlightColor()).
		Padding(1).
		Render(content.String())

	return overlay.Composite(popupBox, main, overlay.Center, overlay.Center, 0, 0)
}

func (m Model) renderConfirmPopup(main string) string {
	var content strings.Builder

	header := styles.WarningStyle.Render(" CONFIRM DESTRUCTIVE ACTION ")
	content.WriteString(header + "\n\n")
	content.WriteString("Strict Mode is active. Do you really want to execute this query?\n\n")

	// Query Preview
	q := m.pendingQuery
	if len(q) > 400 {
		q = q[:397] + "..."
	}
	content.WriteString(lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(styles.TextFaint()).
		Padding(1).
		Foreground(styles.TextPrimary()).
		Render(q))

	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.SuccessColor()).Render("(y) Yes, execute") + "  " +
		lipgloss.NewStyle().Bold(true).Foreground(styles.ErrorColor()).Render("(n/Esc) No, cancel"))

	// Box styling with background
	popupBox := styles.PopupStyle.
		Width(min(80, m.width-4)).
		Background(styles.PopupBg()). // Dark background for popup
		Render(content.String())

	// Use bubbletea-overlay to composite popup over main content
	return overlay.Composite(popupBox, main, overlay.Center, overlay.Center, 0, 0)
}

func (m Model) renderExportPopup(main string) string {
	var content strings.Builder

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.AccentColor()).
		Render("Export Results")
	content.WriteString(header + "\n\n")

	content.WriteString("Enter filename (or path):\n\n")
	content.WriteString(m.exportInput.View())
	content.WriteString("\n\n")

	hint := lipgloss.NewStyle().Faint(true).Render("Enter: Export | Esc: Cancel")
	content.WriteString(hint)

	popupBox := lipgloss.NewStyle().
		Width(50).
		Background(styles.PopupBg()).
		Foreground(styles.TextPrimary()).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.SuccessColor()).
		Padding(1).
		Render(content.String())

	return overlay.Composite(popupBox, main, overlay.Center, overlay.Center, 0, 0)
}

// --- Help popup ---

func (m Model) getHelpContext() HelpContext {
	if m.schemaBrowser.IsVisible() {
		return HelpContextSchema
	}
	if m.showPopup {
		return HelpContextPopup
	}
	if m.mode == InsertMode {
		return HelpContextInsert
	}
	return HelpContextVisual
}

func (m Model) renderHelpPopup(main string) string {
	var content strings.Builder

	keys := m.config.Keys
	ctx := m.getHelpContext()

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.AccentColor()).
		MarginBottom(1)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.HighlightColor()).
		MarginTop(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(styles.TextPrimary()).
		Background(styles.CardBg()).
		Padding(0, 1).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(styles.TextSecondary())

	rowStyle := lipgloss.NewStyle().
		MarginLeft(1)

	footerStyle := lipgloss.NewStyle().
		Faint(true).
		MarginTop(1)

	// Helper to render a key binding row
	renderRow := func(key, desc string) string {
		return rowStyle.Render(keyStyle.Render(key) + " " + descStyle.Render(desc))
	}

	// Context title
	var contextName string
	switch ctx {
	case HelpContextInsert:
		contextName = "Insert Mode"
	case HelpContextPopup:
		contextName = "Results View"
	case HelpContextSchema:
		contextName = "Schema Browser"
	default:
		contextName = "Visual Mode"
	}

	content.WriteString(titleStyle.Render("Shortcuts - " + contextName))
	content.WriteString("\n")

	// Helper to get first key or fallback
	key := func(bindings []string, fallback string) string {
		if len(bindings) > 0 {
			return bindings[0]
		}
		return fallback
	}

	// Helper to join first keys with separator
	keyPair := func(a, b []string) string {
		return key(a, "?") + "/" + key(b, "?")
	}

	// Context-specific shortcuts
	switch ctx {
	case HelpContextInsert:
		content.WriteString(sectionStyle.Render("Query"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.Execute, "ctrl+d"), "Execute query"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.Explain, "X"), "Explain query"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.Autocomplete, "ctrl+space"), "Autocomplete"))
		content.WriteString("\n")

		content.WriteString(sectionStyle.Render("Edit"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.Undo, "ctrl+z"), "Undo"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.Redo, "ctrl+y"), "Redo"))
		content.WriteString("\n")

		content.WriteString(sectionStyle.Render("Navigation"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.Exit, "esc"), "Exit to Visual mode"))
		content.WriteString("\n")

	case HelpContextPopup:
		content.WriteString(sectionStyle.Render("Navigation"))
		content.WriteString("\n")
		content.WriteString(renderRow(keyPair(keys.MoveUp, keys.MoveDown), "Navigate rows"))
		content.WriteString("\n")
		content.WriteString(renderRow(keyPair(keys.ScrollLeft, keys.ScrollRight), "Scroll columns"))
		content.WriteString("\n")
		content.WriteString(renderRow(keyPair(keys.NextPage, keys.PrevPage), "Page up/down"))
		content.WriteString("\n")

		content.WriteString(sectionStyle.Render("Actions"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.RowAction, "enter"), "Row actions"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.Filter, "/"), "Filter results"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.Export, "ctrl+e"), "Export to file"))
		content.WriteString("\n")

		content.WriteString(sectionStyle.Render("Exit"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.Exit, "esc"), "Close popup"))
		content.WriteString("\n")

	case HelpContextSchema:
		content.WriteString(sectionStyle.Render("Navigation"))
		content.WriteString("\n")
		content.WriteString(renderRow(keyPair(keys.MoveUp, keys.MoveDown), "Navigate tables"))
		content.WriteString("\n")
		content.WriteString(renderRow(keyPair(keys.ScrollRight, keys.ScrollLeft), "Switch tabs"))
		content.WriteString("\n")

		content.WriteString(sectionStyle.Render("Actions"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.ToggleExpand, "enter"), "View columns"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.ToggleTheme, "t"), "Query templates"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.Export, "e"), "Export table"))
		content.WriteString("\n")

		content.WriteString(sectionStyle.Render("Exit"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.ToggleSchema, "tab"), "Close browser"))
		content.WriteString("\n")

	default: // Visual mode
		content.WriteString(sectionStyle.Render("Navigation"))
		content.WriteString("\n")
		content.WriteString(renderRow(keyPair(keys.MoveUp, keys.MoveDown), "Navigate history"))
		content.WriteString("\n")
		content.WriteString(renderRow(keyPair(keys.GoTop, keys.GoBottom), "Jump to top/bottom"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.ToggleExpand, "enter"), "Expand/collapse"))
		content.WriteString("\n")

		content.WriteString(sectionStyle.Render("Actions"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.InsertMode, "i"), "Enter Insert mode"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.Rerun, "r"), "Rerun query"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.Edit, "e"), "Edit query"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.Copy, "y"), "Copy query"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.Delete, "x"), "Delete entry"))
		content.WriteString("\n")

		content.WriteString(sectionStyle.Render("Panels"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.ToggleSchema, "tab"), "Schema browser"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.ToggleTheme, "t"), "Theme selector"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.ShowProfiles, "P"), "Switch profile"))
		content.WriteString("\n")
		content.WriteString(renderRow(key(keys.ToggleStrict, "m"), "Toggle strict mode"))
		content.WriteString("\n")
	}

	// Always show quit
	content.WriteString(sectionStyle.Render("General"))
	content.WriteString("\n")
	content.WriteString(renderRow(key(keys.Help, "?"), "Toggle this help"))
	content.WriteString("\n")
	content.WriteString(renderRow(key(keys.Quit, "ctrl+c"), "Quit"))
	content.WriteString("\n")

	content.WriteString(footerStyle.Render("Press " + key(keys.Help, "?") + " or " + key(keys.Exit, "esc") + " to close"))

	// Style popup
	popupWidth := 42
	popupBox := styles.PopupStyle.
		Width(popupWidth).
		MaxHeight(m.height-4).
		Padding(1, 2).
		Background(styles.PopupBg()).
		Render(content.String())

	return overlay.Composite(popupBox, main, overlay.Center, overlay.Center, 0, 0)
}

// --- Template popup ---

func (m Model) renderTemplatePopup(main string) string {
	var content strings.Builder

	// Title
	title := lipgloss.NewStyle().Bold(true).Foreground(styles.AccentColor()).Render(
		fmt.Sprintf("Quick Queries for: %s", m.templateTable))
	content.WriteString(title)
	content.WriteString("\n\n")

	// List templates
	for i, t := range m.config.QueryTemplates {
		style := lipgloss.NewStyle().Foreground(styles.TextSecondary())
		prefix := "  "
		if i == m.templateIdx {
			style = lipgloss.NewStyle().Foreground(styles.SuccessColor()).Bold(true)
			prefix = " "
		}
		// Show template with replaced table name for preview
		preview := strings.ReplaceAll(t.Query, "<table>", m.templateTable)
		if len(preview) > 50 {
			preview = preview[:47] + "..."
		}
		content.WriteString(fmt.Sprintf("%s%s\n", prefix, style.Render(t.Name)))
		content.WriteString(fmt.Sprintf("    %s\n\n", lipgloss.NewStyle().Faint(true).Render(preview)))
	}

	content.WriteString(lipgloss.NewStyle().Faint(true).Render("Enter: execute • i: insert into editor • Esc: cancel"))

	// Style popup
	popupWidth := 60
	if popupWidth > m.width-10 {
		popupWidth = m.width - 10
	}
	popupBox := styles.PopupStyle.
		Width(popupWidth).
		MaxHeight(m.height - 4).
		Background(styles.PopupBg()).
		Render(content.String())

	return overlay.Composite(popupBox, main, overlay.Center, overlay.Center, 0, 0)
}

// --- Import popup ---

func (m Model) renderImportPopup(main string) string {
	var content strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(styles.AccentColor()).Render(
		fmt.Sprintf("Import into: %s", m.importTable))
	content.WriteString(title)
	content.WriteString("\n\n")
	content.WriteString(m.importInput.View())
	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Faint(true).Render("Enter: import • Esc: cancel"))

	popupWidth := 60
	popupBox := styles.PopupStyle.
		Width(popupWidth).
		MaxHeight(10).
		Background(styles.PopupBg()).
		Render(content.String())

	return overlay.Composite(popupBox, main, overlay.Center, overlay.Center, 0, 0)
}
