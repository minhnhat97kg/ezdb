package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

// HelpContext represents the current UI context for help display
type HelpContext int

const (
	HelpContextVisual HelpContext = iota
	HelpContextInsert
	HelpContextPopup
	HelpContextSchema
)

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
		Foreground(AccentColor()).
		MarginBottom(1)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(HighlightColor()).
		MarginTop(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(TextPrimary()).
		Background(CardBg()).
		Padding(0, 1).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(TextSecondary())

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

	// Context-specific shortcuts
	switch ctx {
	case HelpContextInsert:
		content.WriteString(sectionStyle.Render("Query"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.Execute[0], "Execute query"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.Explain[0], "Explain query"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.Autocomplete[0], "Autocomplete"))
		content.WriteString("\n")

		content.WriteString(sectionStyle.Render("Edit"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.Undo[0], "Undo"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.Redo[0], "Redo"))
		content.WriteString("\n")

		content.WriteString(sectionStyle.Render("Navigation"))
		content.WriteString("\n")
		content.WriteString(renderRow("esc", "Exit to Visual mode"))
		content.WriteString("\n")

	case HelpContextPopup:
		content.WriteString(sectionStyle.Render("Navigation"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.MoveUp[0]+"/"+keys.MoveDown[0], "Navigate rows"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.ScrollLeft[0]+"/"+keys.ScrollRight[0], "Scroll columns"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.NextPage[0]+"/"+keys.PrevPage[0], "Page up/down"))
		content.WriteString("\n")

		content.WriteString(sectionStyle.Render("Actions"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.RowAction[0], "Row actions"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.Filter[0], "Filter results"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.Export[0], "Export to file"))
		content.WriteString("\n")

		content.WriteString(sectionStyle.Render("Exit"))
		content.WriteString("\n")
		content.WriteString(renderRow("esc", "Close popup"))
		content.WriteString("\n")

	case HelpContextSchema:
		content.WriteString(sectionStyle.Render("Navigation"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.MoveUp[0]+"/"+keys.MoveDown[0], "Navigate tables"))
		content.WriteString("\n")
		content.WriteString(renderRow("l/h", "Switch tabs"))
		content.WriteString("\n")

		content.WriteString(sectionStyle.Render("Actions"))
		content.WriteString("\n")
		content.WriteString(renderRow("enter", "View columns"))
		content.WriteString("\n")
		content.WriteString(renderRow("t", "Query templates"))
		content.WriteString("\n")
		content.WriteString(renderRow("e", "Export table"))
		content.WriteString("\n")
		content.WriteString(renderRow("o", "Import into table"))
		content.WriteString("\n")

		content.WriteString(sectionStyle.Render("Exit"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.ToggleSchema[0], "Close browser"))
		content.WriteString("\n")

	default: // Visual mode
		content.WriteString(sectionStyle.Render("Navigation"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.MoveUp[0]+"/"+keys.MoveDown[0], "Navigate history"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.GoTop[0]+"/"+keys.GoBottom[0], "Jump to top/bottom"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.ToggleExpand[0], "Expand/collapse"))
		content.WriteString("\n")

		content.WriteString(sectionStyle.Render("Actions"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.InsertMode[0], "Enter Insert mode"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.Rerun[0], "Rerun query"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.Edit[0], "Edit query"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.Copy[0], "Copy query"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.Delete[0], "Delete entry"))
		content.WriteString("\n")

		content.WriteString(sectionStyle.Render("Panels"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.ToggleSchema[0], "Schema browser"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.ToggleTheme[0], "Theme selector"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.ShowProfiles[0], "Switch profile"))
		content.WriteString("\n")
		content.WriteString(renderRow(keys.ToggleStrict[0], "Toggle strict mode"))
		content.WriteString("\n")
	}

	// Always show quit
	content.WriteString(sectionStyle.Render("General"))
	content.WriteString("\n")
	content.WriteString(renderRow(keys.Help[0], "Toggle this help"))
	content.WriteString("\n")
	content.WriteString(renderRow(keys.Quit[0], "Quit"))
	content.WriteString("\n")

	content.WriteString(footerStyle.Render("Press ? or Esc to close"))

	// Style popup
	popupWidth := 42
	popupBox := PopupStyle.
		Width(popupWidth).
		MaxHeight(m.height - 4).
		Padding(1, 2).
		Background(PopupBg()).
		Render(content.String())

	return overlay.Composite(popupBox, main, overlay.Center, overlay.Center, 0, 0)
}
