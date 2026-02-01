package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// loadHistoryCmd loads query history from SQLite
func (m Model) loadHistoryCmd() tea.Cmd {
	return func() tea.Msg {
		entries, err := m.historyStore.List(m.profile.Name, 100, 0)
		return HistoryLoadedMsg{Entries: entries, Err: err}
	}
}
