package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nhath/ezdb/internal/ui/styles"
)

func (m Model) renderHelp() string {
	// Style for key hints - makes keys look like keyboard buttons
	keyStyle := lipgloss.NewStyle().
		Foreground(styles.TextPrimary()).
		Background(styles.CardBg()).
		Padding(0, 1).
		Bold(true)

	sepStyle := lipgloss.NewStyle().Foreground(styles.TextFaint())
	descStyle := lipgloss.NewStyle().Foreground(styles.TextSecondary())

	// Helper to format a single hint
	hint := func(key, desc string) string {
		return keyStyle.Render(key) + descStyle.Render(" "+desc)
	}

	// Helper to get first key or fallback
	key := func(bindings []string, fallback string) string {
		if len(bindings) > 0 {
			return bindings[0]
		}
		return fallback
	}

	sep := sepStyle.Render("  ")

	keys := m.config.Keys

	// Context-aware hints based on current state
	var hints []string

	if m.mode == InsertMode {
		hints = append(hints,
			hint(key(keys.Execute, "ctrl+d"), "Run"),
			hint(key(keys.Explain, "X"), "Explain"),
			hint(key(keys.Exit, "esc"), "Visual"),
			hint(key(keys.Autocomplete, "ctrl+space"), "Complete"),
		)
	} else {
		// Visual mode
		hints = append(hints,
			hint(key(keys.InsertMode, "i"), "Insert"),
			hint(key(keys.MoveUp, "k")+"/"+key(keys.MoveDown, "j"), "Nav"),
			hint(key(keys.ToggleExpand, "enter"), "Expand"),
			hint(key(keys.Rerun, "r"), "Rerun"),
			hint(key(keys.Edit, "e"), "Edit"),
			hint(key(keys.ToggleSchema, "tab"), "Schema"),
			hint(key(keys.ToggleTheme, "t"), "Theme"),
		)
	}

	// Always show help and quit
	hints = append(hints,
		hint(key(keys.Help, "?"), "Help"),
		hint(key(keys.Quit, "ctrl+c"), "Quit"),
	)

	return strings.Join(hints, sep)
}
