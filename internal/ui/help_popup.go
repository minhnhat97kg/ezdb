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
	popupBox := PopupStyle.
		Width(popupWidth).
		MaxHeight(m.height - 4).
		Padding(1, 2).
		Background(PopupBg()).
		Render(content.String())

	return overlay.Composite(popupBox, main, overlay.Center, overlay.Center, 0, 0)
}
