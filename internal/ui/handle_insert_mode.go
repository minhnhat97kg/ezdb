// internal/ui/handle_insert_mode.go
// Key handling for insert (typing) mode and autocomplete integration.
package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nhath/ezdb/internal/db"
	"github.com/nhath/ezdb/internal/ui/autocomplete"
)

// handleInsertMode processes keys while in insert mode.
// cmds is the accumulated command slice from the caller; it is returned
// appended-to so the caller can batch everything.
func (m Model) handleInsertMode(msg tea.KeyMsg, cmds []tea.Cmd) (Model, []tea.Cmd) {
	var cmd tea.Cmd

	hasPopup := m.hasOpenPopup() || m.showPopup || m.showHelpPopup || m.showTemplatePopup ||
		m.showImportPopup || m.showExportPopup || m.showRowActionPopup || m.showActionPopup ||
		m.themeSelector.Visible()

	// Autocomplete navigation / apply
	if m.autocompleting && !hasPopup {
		switch msg.String() {
		case "up", "ctrl+p":
			if m.suggestionIdx > 0 {
				m.suggestionIdx--
			}
			return m, cmds
		case "down", "ctrl+n":
			if m.suggestionIdx < len(m.suggestions)-1 {
				m.suggestionIdx++
			}
			return m, cmds
		case "enter", "tab":
			m = m.applySuggestion()
			m.autocompleting = false
			return m, cmds
		case "esc":
			m.autocompleting = false
			return m, cmds
		}
	}

	// Ctrl+Space – open autocomplete
	if matchKey(msg, m.config.Keys.Autocomplete) && !hasPopup {
		m.autocompleting = true
		m = m.updateSuggestions()
		return m, cmds
	}

	// Ctrl+D – execute
	if matchKey(msg, m.config.Keys.Execute) {
		query := strings.TrimSpace(m.editor.Value())
		if query != "" {
			m.editor.SetValue("")
			m.editor.Reset()

			if m.strictMode && isModifyingQuery(query) {
				m.confirming = true
				m.pendingQuery = query
				return m, cmds
			}
			m.loading = true
			cmds = append(cmds, m.executeQueryCmd(query))
		}
		return m, cmds
	}

	// Ctrl+E – explain
	if matchKey(msg, m.config.Keys.Explain) {
		query := strings.TrimSpace(m.editor.Value())
		if query != "" && m.driver != nil {
			explainQuery := "EXPLAIN " + query
			if m.driver.Type() == db.SQLite {
				explainQuery = "EXPLAIN QUERY PLAN " + query
			}
			m.loading = true
			cmds = append(cmds, m.executeQueryCmd(explainQuery))
		}
		return m, cmds
	}

	// Undo
	if matchKey(msg, m.config.Keys.Undo) {
		if len(m.undoStack) > 0 {
			m.redoStack = append(m.redoStack, m.editor.Value())
			prev := m.undoStack[len(m.undoStack)-1]
			m.undoStack = m.undoStack[:len(m.undoStack)-1]
			m.editor.SetValue(prev)
		}
		return m, cmds
	}

	// Redo
	if matchKey(msg, m.config.Keys.Redo) {
		if len(m.redoStack) > 0 {
			m.undoStack = append(m.undoStack, m.editor.Value())
			next := m.redoStack[len(m.redoStack)-1]
			m.redoStack = m.redoStack[:len(m.redoStack)-1]
			m.editor.SetValue(next)
		}
		return m, cmds
	}

	// Esc – back to visual mode
	if matchKey(msg, m.config.Keys.Exit) || msg.String() == "esc" {
		m.mode = VisualMode
		m.editor.Blur()
		if len(m.history) > 0 {
			m.selected = len(m.history) - 1
			m = m.ensureSelectionVisible()
		}
		return m, cmds
	}

	// Pass key to the textarea editor
	m.editor, cmd = m.editor.Update(msg)
	cmds = append(cmds, cmd)

	// --- Post-keystroke autocomplete logic ---
	val := m.editor.Value()

	// Empty input: clear suggestions
	if strings.TrimSpace(val) == "" {
		m.autocompleting = false
		m.suggestions = nil
		m.debounceID++
		return m, cmds
	}

	// Debounce: schedule suggestion refresh after 1 s
	m.debounceID++
	id := m.debounceID
	cmd = tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return DebounceMsg{ID: id}
	})
	cmds = append(cmds, cmd)

	return m, cmds
}

// updateSuggestions refreshes autocomplete suggestions based on cursor position.
func (m Model) updateSuggestions() Model {
	text := m.editor.Value()
	row := m.editor.Line()
	lines := strings.Split(text, "\n")
	if row >= len(lines) {
		return m
	}

	// Calculate cursor position in full text
	cursorPos := 0
	for i := 0; i < row; i++ {
		cursorPos += len(lines[i]) + 1 // +1 for newline
	}
	cursorPos += len(lines[row])

	// Get the word being typed
	line := lines[row]
	col := len(line)
	word, _, _ := autocomplete.GetWordAtCursor(line, col)

	// Parse SQL context and fetch suggestions
	ctx := autocomplete.ParseSQLContext(text, cursorPos)
	suggestions := autocomplete.GetSuggestions(ctx, m.tables, m.columns, word)

	// Convert to display slices
	m.suggestions = make([]string, len(suggestions))
	m.suggestionDetails = make([]string, len(suggestions))
	m.suggestionTypes = make([]autocomplete.SuggestionType, len(suggestions))
	for i, s := range suggestions {
		m.suggestions[i] = s.Text
		m.suggestionDetails[i] = s.Detail
		m.suggestionTypes[i] = s.Type
	}

	m.suggestionIdx = 0
	return m
}

// applySuggestion inserts the currently selected suggestion at the cursor.
func (m Model) applySuggestion() Model {
	if len(m.suggestions) == 0 || m.suggestionIdx >= len(m.suggestions) {
		return m
	}
	selected := m.suggestions[m.suggestionIdx]

	row := m.editor.Line()
	lines := strings.Split(m.editor.Value(), "\n")
	if row >= len(lines) {
		return m
	}
	line := lines[row]
	col := len(line)
	_, start, end := autocomplete.GetWordAtCursor(line, col)

	// Replace word with suggestion
	prefix := line[:start]
	suffix := ""
	if end < len(line) {
		suffix = line[end:]
	}
	lines[row] = prefix + selected + suffix
	m.editor.SetValue(strings.Join(lines, "\n"))

	// Move cursor to end of inserted text
	newCol := start + len(selected)
	cursorIdx := 0
	for i := 0; i < row; i++ {
		cursorIdx += len(lines[i]) + 1
	}
	cursorIdx += newCol
	m.editor.SetCursor(cursorIdx)
	return m
}
