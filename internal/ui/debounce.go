package ui

import (
	"fmt"
	"strings"
)

// DebounceMsg triggers the actual autocomplete lookup
type DebounceMsg struct {
	ID int
}

// updateSlashSuggestions populates suggestions for commands
func (m Model) updateSlashSuggestions() Model {
	input := m.editor.Value()
	if input == "" || !strings.HasPrefix(input, "/") {
		return m
	}

	// Built-in commands
	commands := []struct {
		Name string
		Desc string
	}{
		{"/profile", "Manage connection profiles"},
		{"/exit", "Exit application"},
		{"/help", "Show help"},
	}

	var suggestions []string

	// Add config aliases
	if m.config != nil && m.config.Commands != nil {
		for alias, query := range m.config.Commands {
			desc := "Alias for: " + query
			if len(desc) > 30 {
				desc = desc[:27] + "..."
			}
			commands = append(commands, struct{ Name, Desc string }{"/" + alias, desc})
		}
	}

	cleanedInput := strings.ToUpper(input)
	for _, cmd := range commands {
		if strings.HasPrefix(strings.ToUpper(cmd.Name), cleanedInput) {
			// Format: "/cmd - Description"
			suggestions = append(suggestions, fmt.Sprintf("%-10s %s", cmd.Name, cmd.Desc))
		}
	}

	m.suggestions = suggestions
	m.suggestionIdx = 0
	return m
}
