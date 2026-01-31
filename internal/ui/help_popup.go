package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

func (m Model) renderHelpPopup(main string) string {
	var content strings.Builder

	// Title
	title := lipgloss.NewStyle().Bold(true).Foreground(AccentColor()).Render("⌨️  Keyboard Shortcuts")
	content.WriteString(title)
	content.WriteString("\n\n")

	keys := m.config.Keys

	// Section helper
	section := func(name string, bindings []struct{ key, desc string }) {
		header := lipgloss.NewStyle().Bold(true).Foreground(HighlightColor()).Render(name)
		content.WriteString(header + "\n")
		for _, b := range bindings {
			keyStyle := lipgloss.NewStyle().Foreground(SuccessColor()).Width(15)
			descStyle := lipgloss.NewStyle().Foreground(TextSecondary())
			content.WriteString(fmt.Sprintf("  %s %s\n", keyStyle.Render(b.key), descStyle.Render(b.desc)))
		}
		content.WriteString("\n")
	}

	// Navigation
	section("Navigation", []struct{ key, desc string }{
		{strings.Join(keys.MoveUp, "/"), "Move up"},
		{strings.Join(keys.MoveDown, "/"), "Move down"},
		{strings.Join(keys.GoTop, "/"), "Go to top"},
		{strings.Join(keys.GoBottom, "/"), "Go to bottom"},
		{strings.Join(keys.NextPage, "/"), "Next page"},
		{strings.Join(keys.PrevPage, "/"), "Previous page"},
		{strings.Join(keys.ScrollLeft, "/"), "Scroll left"},
		{strings.Join(keys.ScrollRight, "/"), "Scroll right"},
	})

	// Actions
	section("Actions", []struct{ key, desc string }{
		{strings.Join(keys.Execute, "/"), "Execute query"},
		{strings.Join(keys.Explain, "/"), "Explain query"},
		{strings.Join(keys.InsertMode, "/"), "Insert mode"},
		{strings.Join(keys.ToggleExpand, "/"), "Toggle expand / Select"},
		{strings.Join(keys.Rerun, "/"), "Rerun query"},
		{strings.Join(keys.Edit, "/"), "Edit query"},
		{strings.Join(keys.Copy, "/"), "Copy query"},
		{strings.Join(keys.Delete, "/"), "Delete entry"},
		{strings.Join(keys.Filter, "/"), "Filter"},
		{strings.Join(keys.Export, "/"), "Export"},
		{strings.Join(keys.Sort, "/"), "Sort"},
	})

	// Panels
	section("Panels", []struct{ key, desc string }{
		{strings.Join(keys.ToggleSchema, "/"), "Toggle schema browser"},
		{strings.Join(keys.ShowProfiles, "/"), "Show profiles"},
		{strings.Join(keys.ToggleStrict, "/"), "Toggle strict mode"},
		{strings.Join(keys.ToggleTheme, "/"), "Toggle theme"},
		{strings.Join(keys.Help, "/"), "Show this help"},
	})

	// Schema Browser
	section("Schema Browser", []struct{ key, desc string }{
		{"t", "Quick query templates"},
		{"e", "Export table"},
		{"o", "Import into table"},
		{"enter", "View columns"},
		{"l/h", "Switch tabs"},
	})

	// Other
	section("Other", []struct{ key, desc string }{
		{strings.Join(keys.Autocomplete, "/"), "Autocomplete"},
		{strings.Join(keys.Undo, "/"), "Undo"},
		{strings.Join(keys.Redo, "/"), "Redo"},
		{strings.Join(keys.Exit, "/"), "Close popup"},
		{strings.Join(keys.Quit, "/"), "Quit"},
	})

	content.WriteString(lipgloss.NewStyle().Faint(true).Render("Press Esc or q to close"))

	// Style popup
	popupWidth := 50
	popupBox := PopupStyle.
		Width(popupWidth).
		MaxHeight(m.height - 4).
		Background(PopupBg()).
		Render(content.String())

	return overlay.Composite(popupBox, main, overlay.Center, overlay.Center, 0, 0)
}
