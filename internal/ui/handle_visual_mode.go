// internal/ui/handle_visual_mode.go
// Key handling for visual (vim-normal) mode.
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"

	"github.com/nhath/ezdb/internal/ui/components/schemabrowser"
	eztable "github.com/nhath/ezdb/internal/ui/components/table"
)

// handleVisualMode handles keys in visual mode.
func (m Model) handleVisualMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if matchKey(msg, m.config.Keys.InsertMode) {
		m.mode = InsertMode
		m.editor.Focus()
		return m, textinput.Blink
	} else if matchKey(msg, m.config.Keys.MoveUp) {
		if m.selected > 0 {
			m.selected--
			m = m.ensureSelectionVisible()
		}
	} else if matchKey(msg, m.config.Keys.MoveDown) {
		if m.selected < len(m.history)-1 {
			m.selected++
			m = m.ensureSelectionVisible()
		}
	} else if matchKey(msg, m.config.Keys.ScrollLeft) {
		if m.expandedID != 0 {
			m.expandedTable = m.expandedTable.ScrollLeft()
		}
	} else if matchKey(msg, m.config.Keys.ScrollRight) {
		if m.expandedID != 0 {
			m.expandedTable = m.expandedTable.ScrollRight()
		}
	} else if matchKey(msg, m.config.Keys.GoTop) {
		m.selected = 0
		m = m.ensureSelectionVisible()
	} else if matchKey(msg, m.config.Keys.GoBottom) {
		if len(m.history) > 0 {
			m.selected = len(m.history) - 1
			m = m.ensureSelectionVisible()
		}
	} else if matchKey(msg, m.config.Keys.ToggleExpand) {
		if m.selected >= 0 && m.selected < len(m.history) {
			entry := m.history[m.selected]
			if m.expandedID == entry.ID {
				m.expandedID = 0
				m.expandedTable = table.Model{}
			} else {
				m.expandedID = entry.ID
				if strings.Contains(entry.Preview, " | ") {
					m.expandedTable = eztable.FromPreview(entry.Preview).
						WithMaxTotalWidth(m.width - 14).
						WithHorizontalFreezeColumnCount(1)
				}
			}
			m = m.ensureSelectionVisible()
		}
	} else if matchKey(msg, m.config.Keys.Rerun) {
		if m.selected >= 0 && m.selected < len(m.history) {
			entry := m.history[m.selected]
			if m.strictMode && isModifyingQuery(entry.Query) {
				m.confirming = true
				m.pendingQuery = entry.Query
				return m, nil
			}
			m.loading = true
			return m, m.executeQueryCmd(entry.Query)
		}
	} else if matchKey(msg, m.config.Keys.ToggleStrict) {
		m.strictMode = !m.strictMode
		m.errorMsg = ""
		return m, nil
	} else if matchKey(msg, m.config.Keys.Edit) {
		if m.selected >= 0 && m.selected < len(m.history) {
			entry := m.history[m.selected]
			m.editor.SetValue(entry.Query)
			m.mode = InsertMode
			m.editor.Focus()
			return m, textinput.Blink
		}
	} else if matchKey(msg, m.config.Keys.Delete) {
		if m.selected >= 0 && m.selected < len(m.history) {
			entry := m.history[m.selected]
			m.historyStore.Delete(entry.ID)
			m.history = append(m.history[:m.selected], m.history[m.selected+1:]...)
			if m.selected >= len(m.history) && m.selected > 0 {
				m.selected--
			}
			m = m.ensureSelectionVisible()
		}
	} else if matchKey(msg, m.config.Keys.Copy) {
		if m.selected >= 0 && m.selected < len(m.history) {
			entry := m.history[m.selected]
			return m, m.copyToClipboardCmd(entry.Query)
		}
	} else if matchKey(msg, m.config.Keys.Filter) {
		m.searching = true
		m.searchQuery = ""
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		return m, textinput.Blink
	} else if matchKey(msg, m.config.Keys.ToggleSchema) {
		m.schemaBrowser = m.schemaBrowser.Toggle()
		if m.schemaBrowser.IsVisible() && m.driver != nil {
			sb, sbCmd := m.schemaBrowser.StartLoading()
			m.schemaBrowser = sb
			return m, tea.Batch(schemabrowser.LoadSchemaCmd(m.driver), sbCmd)
		}
		return m, nil
	}
	m = m.updateHistoryViewport()
	return m, nil
}
