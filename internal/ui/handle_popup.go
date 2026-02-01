// internal/ui/handle_popup.go
// Popup key-handling dispatch and opener/closer helpers.
package ui

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nhath/ezdb/internal/db"
	"github.com/nhath/ezdb/internal/history"
)

// handlePopupKeys processes key events that target open popups.
// Returns (model, cmd, handled). If handled is false the caller must
// continue dispatching.
func (m Model) handlePopupKeys(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	// Universal popup close handler
	isExitKey := matchKey(msg, m.config.Keys.Exit) || msg.String() == "esc" || msg.String() == "q"
	hasPopup := m.hasOpenPopup() || m.showPopup || m.showHelpPopup || m.showTemplatePopup ||
		m.showImportPopup || m.showExportPopup || m.showRowActionPopup || m.showActionPopup ||
		m.themeSelector.Visible()

	if hasPopup && isExitKey {
		f, _ := os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		fmt.Fprintf(f, "Exit key pressed. Stack len: %d. Top: %s\n", m.popupStack.Len(), m.popupStack.TopName())
		f.Close()
		if m.closeTopPopup() {
			return m, nil, true
		}
		// Fallback: close any open popup directly
		if m.showRowActionPopup {
			m.showRowActionPopup = false
			return m, nil, true
		}
		if m.showExportPopup {
			m.showExportPopup = false
			m.exportInput.Blur()
			return m, nil, true
		}
		if m.showActionPopup {
			m.showActionPopup = false
			return m, nil, true
		}
		if m.showPopup {
			m.showPopup = false
			m.tableFilterInput.Blur()
			m.tableFilterInput.SetValue("")
			return m, nil, true
		}
		if m.showTemplatePopup {
			m.showTemplatePopup = false
			m.templateTable = ""
			m.templateIdx = 0
			return m, nil, true
		}
		if m.showImportPopup {
			m.showImportPopup = false
			m.importInput.Blur()
			m.importTable = ""
			return m, nil, true
		}
		if m.showHelpPopup {
			m.showHelpPopup = false
			return m, nil, true
		}
		if m.themeSelector.Visible() {
			m.themeSelector = m.themeSelector.Hide()
			return m, nil, true
		}
	}

	// Confirming prompt (y/n for destructive queries)
	if m.confirming {
		switch msg.String() {
		case "y", "Y":
			m.confirming = false
			m.loading = true
			query := m.pendingQuery
			m.pendingQuery = ""
			return m, m.executeQueryCmd(query), true
		case "n", "N", "esc":
			m.confirming = false
			m.pendingQuery = ""
			return m, nil, true
		}
		return m, nil, true
	}

	// Theme selector
	if m.themeSelector.Visible() {
		if matchKey(msg, m.config.Keys.Help) {
			m.openHelpPopup()
			return m, nil, true
		}
		var cmd tea.Cmd
		m.themeSelector, cmd = m.themeSelector.Update(msg)
		if !m.themeSelector.Visible() && m.popupStack.TopName() == "theme" {
			m.popupStack.Pop()
		}
		return m, cmd, true
	}

	// Help popup (blocks all other keys)
	if m.showHelpPopup {
		if matchKey(msg, m.config.Keys.Help) {
			m.closeTopPopup()
			return m, nil, true
		}
		return m, nil, true
	}

	// Template popup
	if m.showTemplatePopup {
		switch msg.String() {
		case "up", "k":
			if m.templateIdx > 0 {
				m.templateIdx--
			}
			return m, nil, true
		case "down", "j":
			if m.templateIdx < len(m.config.QueryTemplates)-1 {
				m.templateIdx++
			}
			return m, nil, true
		case "enter":
			m.popupStack.Pop()
			model, cmd := m.executeTemplate()
			return model, cmd, true
		case "i":
			m.popupStack.Pop()
			m = m.insertTemplate()
			return m, nil, true
		}
		return m, nil, true
	}

	// Import popup
	if m.showImportPopup {
		if msg.String() == "enter" {
			filename := m.importInput.Value()
			if filename != "" {
				m.popupStack.Pop()
				m.showImportPopup = false
				m.importInput.Blur()
				m.importTable = ""
				m.loading = true
				return m, m.importTableCmd(m.importTable, filename), true
			}
			return m, nil, true
		}
		var cmd tea.Cmd
		m.importInput, cmd = m.importInput.Update(msg)
		return m, cmd, true
	}

	// Export popup
	if m.showExportPopup {
		if msg.String() == "enter" {
			filename := m.exportInput.Value()
			if filename == "" {
				filename = "export.csv"
			}
			m.popupStack.Pop()
			m.showExportPopup = false
			m.exportInput.Blur()
			if m.exportTable != "" {
				m.loading = true
				return m, m.exportTableCmd(m.exportTable, filename), true
			}
			return m, m.exportTableToPath(filename), true
		}
		var cmd tea.Cmd
		m.exportInput, cmd = m.exportInput.Update(msg)
		return m, cmd, true
	}

	// Results table popup (and its nested sub-popups)
	if m.showPopup {
		// Filter input active
		if m.tableFilterActive {
			if msg.Type == tea.KeyEnter || msg.Type == tea.KeyEsc {
				m.tableFilterActive = false
				m.tableFilterInput.Blur()
				return m, nil, true
			}
			var cmd tea.Cmd
			m.tableFilterInput, cmd = m.tableFilterInput.Update(msg)
			m.popupTable = m.popupTable.WithFilterInputValue(m.tableFilterInput.Value())
			return m, cmd, true
		}

		// Row action sub-popup
		if m.showRowActionPopup {
			switch msg.String() {
			case "1":
				m.popupStack.Pop()
				model, cmd := m.selectRowAsQuery()
				return model, cmd, true
			case "2":
				m.popupStack.Pop()
				model, cmd := m.viewFullRow()
				return model, cmd, true
			case "3":
				m.popupStack.Pop()
				m.showRowActionPopup = false
				return m, m.copyRowAsJSON(), true
			case "4":
				m.popupStack.Pop()
				m.showRowActionPopup = false
				return m, m.copyRowAsCSV(), true
			}
			return m, nil, true
		}

		// Action menu sub-popup
		if m.showActionPopup {
			return m, nil, true
		}

		// Table popup keys
		if msg.String() == "a" {
			m.openActionPopup()
			return m, nil, true
		} else if matchKey(msg, m.config.Keys.Filter) {
			m.tableFilterActive = true
			m.tableFilterInput.Focus()
			return m, textinput.Blink, true
		} else if matchKey(msg, m.config.Keys.RowAction) {
			m.openRowActionPopup()
			return m, nil, true
		} else if matchKey(msg, m.config.Keys.Export) {
			m.openExportPopup("export.csv")
			return m, textinput.Blink, true
		} else if matchKey(msg, m.config.Keys.Help) {
			m.openHelpPopup()
			return m, nil, true
		}

		// Pass remaining keys to the popup table for navigation
		var cmd tea.Cmd
		m.popupTable, cmd = m.popupTable.Update(msg)
		return m, cmd, true
	}

	return m, nil, false // not handled
}

// --- Popup opener / closer helpers ---

// openHelpPopup opens the help popup and pushes it onto the stack.
func (m *Model) openHelpPopup() {
	if m.showHelpPopup {
		return
	}
	m.showHelpPopup = true
	m.autocompleting = false
	f, _ := os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	fmt.Fprintf(f, "Pushing help. Stack len before: %d\n", m.popupStack.Len())
	f.Close()
	m.popupStack.Push("help", func(m *Model) bool {
		m.showHelpPopup = false
		return true
	})
}

// openTemplatePopup opens the template popup for a given table.
func (m *Model) openTemplatePopup(tableName string) {
	if m.showTemplatePopup {
		return
	}
	m.showTemplatePopup = true
	m.autocompleting = false
	m.templateTable = tableName
	m.templateIdx = 0
	m.popupStack.Push("template", func(m *Model) bool {
		m.showTemplatePopup = false
		m.templateTable = ""
		m.templateIdx = 0
		return true
	})
}

// openResultsPopup opens the query-results popup.
func (m *Model) openResultsPopup(entry *history.HistoryEntry, result *db.QueryResult) {
	if m.showPopup {
		return
	}
	m.popupEntry = entry
	m.popupResult = result
	m.showPopup = true
	m.autocompleting = false
	f, _ := os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	fmt.Fprintf(f, "Pushing results. Stack len before: %d\n", m.popupStack.Len())
	f.Close()
	m.popupStack.Push("results", func(m *Model) bool {
		m.showPopup = false
		m.tableFilterInput.Blur()
		m.tableFilterInput.SetValue("")
		m.popupTable = m.popupTable.WithFilterInputValue("")
		return true
	})
}

// openRowActionPopup opens the row-action sub-popup.
func (m *Model) openRowActionPopup() {
	if m.showRowActionPopup {
		return
	}
	m.showRowActionPopup = true
	m.autocompleting = false
	m.popupStack.Push("rowAction", func(m *Model) bool {
		m.showRowActionPopup = false
		return true
	})
}

// openExportPopup opens the export filename input popup.
func (m *Model) openExportPopup(defaultName string) {
	if m.showExportPopup {
		return
	}
	m.showExportPopup = true
	m.autocompleting = false
	m.exportInput.SetValue(defaultName)
	m.exportInput.Focus()
	m.popupStack.Push("export", func(m *Model) bool {
		m.showExportPopup = false
		m.exportInput.Blur()
		return true
	})
}

// openImportPopup opens the import filename input popup for a table.
func (m *Model) openImportPopup(tableName string) {
	if m.showImportPopup {
		return
	}
	m.showImportPopup = true
	m.autocompleting = false
	m.importInput.SetValue("")
	m.importInput.Focus()
	m.importTable = tableName
	m.popupStack.Push("import", func(m *Model) bool {
		m.showImportPopup = false
		m.importInput.Blur()
		m.importTable = ""
		return true
	})
}

// openActionPopup opens the action-menu popup.
func (m *Model) openActionPopup() {
	if m.showActionPopup {
		return
	}
	m.showActionPopup = true
	m.autocompleting = false
	m.popupStack.Push("action", func(m *Model) bool {
		m.showActionPopup = false
		return true
	})
}

// openThemeSelector opens the theme-selector popup.
func (m *Model) openThemeSelector() {
	if m.themeSelector.Visible() {
		return
	}
	m.themeSelector = m.themeSelector.Show()
	m.autocompleting = false
	m.popupStack.Push("theme", func(m *Model) bool {
		m.themeSelector = m.themeSelector.Hide()
		return true
	})
}

// closeTopPopup closes the topmost popup via the stack.
func (m *Model) closeTopPopup() bool {
	if m.popupStack == nil {
		return false
	}
	return m.popupStack.CloseTop(m)
}

// hasOpenPopup reports whether any popup is currently open.
func (m *Model) hasOpenPopup() bool {
	if m.popupStack == nil {
		return false
	}
	return !m.popupStack.IsEmpty()
}

// updatePopupTable updates the popup table dimensions and freezing.
func (m *Model) updatePopupTable() {
	if m.width == 0 || m.height == 0 {
		return
	}
	availableHeight := m.height - 28
	if availableHeight < 3 {
		availableHeight = 3
	}
	popupWidth := m.width - 10
	if popupWidth < 60 {
		popupWidth = 60
	}
	maxTableWidth := popupWidth - 10

	m.popupTable = m.popupTable.
		WithPageSize(availableHeight).
		WithMaxTotalWidth(maxTableWidth).
		WithHorizontalFreezeColumnCount(1)
}

// selectRowAsQuery takes the highlighted row in the popup table,
// attempts to find its primary key, and constructs a SELECT query to fetch that specific row.
func (m Model) selectRowAsQuery() (Model, tea.Cmd) {
	if m.popupTable.HighlightedRow().Data == nil {
		return m, nil
	}

	query := m.popupEntry.Query
	re := regexp.MustCompile(`(?i)from\s+["'\[]?([a-zA-Z0-9._]+)["'\]]?`)
	matches := re.FindStringSubmatch(query)
	if len(matches) < 2 {
		m.errorMsg = "Could not determine table name from query"
		return m, nil
	}
	tableName := matches[1]

	var cols []db.Column
	var ok bool

	if cols, ok = m.columns[tableName]; !ok {
		for realName, c := range m.columns {
			if strings.EqualFold(realName, tableName) {
				tableName = realName
				cols = c
				ok = true
				break
			}
		}
		if !ok {
			suffix := "." + strings.ToLower(tableName)
			for realName, c := range m.columns {
				if strings.HasSuffix(strings.ToLower(realName), suffix) {
					tableName = realName
					cols = c
					ok = true
					break
				}
			}
		}
	}

	if !ok {
		f, _ := os.OpenFile("debug_metadata.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if f != nil {
			fmt.Fprintf(f, "Timestamp: %s\nTable: %s\nLoaded Tables Count: %d\nAll tables: %v\n\n",
				time.Now(), tableName, len(m.tables), m.tables)
			f.Close()
		}
		m.errorMsg = fmt.Sprintf("Metadata missing for %s (Tabs: %d). See debug_metadata.log", tableName, len(m.tables))
		return m, nil
	}

	var pkCols []db.Column
	for _, c := range cols {
		if c.Key == "PRI" {
			pkCols = append(pkCols, c)
		}
	}
	if len(pkCols) == 0 {
		m.errorMsg = fmt.Sprintf("No primary key found for table %s", tableName)
		return m, nil
	}

	var whereParts []string
	row := m.popupTable.HighlightedRow().Data
	for _, col := range pkCols {
		val, ok := row[col.Name]
		if !ok {
			continue
		}
		val = unwrapCellValue(val)
		val = unwrapCellValue(val)

		valStr := fmt.Sprintf("'%v'", val)
		typeUpper := strings.ToUpper(col.Type)
		if strings.Contains(typeUpper, "INT") ||
			strings.Contains(typeUpper, "FLOAT") ||
			strings.Contains(typeUpper, "DOUBLE") ||
			strings.Contains(typeUpper, "DECIMAL") ||
			strings.Contains(typeUpper, "NUMERIC") ||
			strings.Contains(typeUpper, "REAL") ||
			strings.Contains(typeUpper, "BOOL") {
			valStr = fmt.Sprintf("%v", val)
		}
		whereParts = append(whereParts, fmt.Sprintf("%s = %s", col.Name, valStr))
	}

	if len(whereParts) == 0 {
		m.errorMsg = "Could not construct WHERE clause from row data"
		return m, nil
	}

	newQuery := fmt.Sprintf("SELECT * FROM %s WHERE %s;", tableName, strings.Join(whereParts, " AND "))
	m.editor.SetValue(newQuery)
	m.showPopup = false
	m.showRowActionPopup = false
	m.showActionPopup = false
	m.mode = InsertMode
	return m, nil
}

// viewFullRow displays all columns and values for the highlighted row.
func (m Model) viewFullRow() (Model, tea.Cmd) {
	highlightedRow := m.popupTable.HighlightedRow()
	if highlightedRow.Data == nil || m.popupResult == nil {
		return m, nil
	}

	var content strings.Builder
	content.WriteString("-- Row Details --\n")
	for _, col := range m.popupResult.Columns {
		if val, ok := highlightedRow.Data[col]; ok {
			val = unwrapCellValue(val)
			content.WriteString(fmt.Sprintf("%s: %v\n", col, val))
		}
	}

	m.editor.SetValue(content.String())
	m.showPopup = false
	m.showRowActionPopup = false
	m.mode = InsertMode
	return m, nil
}
