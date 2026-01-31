// internal/ui/app.go
package ui

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/evertras/bubble-table/table"
	"github.com/nhath/ezdb/internal/config"
	"github.com/nhath/ezdb/internal/db"
	"github.com/nhath/ezdb/internal/history"
	"github.com/nhath/ezdb/internal/ui/components/profileselector"
	"github.com/nhath/ezdb/internal/ui/components/schemabrowser"
	eztable "github.com/nhath/ezdb/internal/ui/components/table"
)

// Update handles messages and updates model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.editor.SetWidth(msg.Width - 4)
		m.profileSelector = m.profileSelector.SetSize(msg.Width, msg.Height)
		m.schemaBrowser = m.schemaBrowser.SetSize(msg.Width, msg.Height)
		if m.expandedID != 0 {
			m.expandedTable = m.expandedTable.WithMaxTotalWidth(msg.Width - 14)
		}
		m.updatePopupTable()
		m = m.updateHistoryViewport()
		return m, nil

	case profileselector.SelectedMsg:
		// Profile selected, connect to it
		if msg.Index >= 0 && msg.Index < len(m.config.Profiles) {
			m.appState = StateConnecting
			selectedProfile := &m.config.Profiles[msg.Index]
			// Store password from prompt in profile
			if msg.Password != "" {
				selectedProfile.Password = msg.Password
				m.config.Save() // Persist to config.toml (encrypted)
			}
			m.profile = selectedProfile
			return m, m.connectToProfileCmd(selectedProfile)
		}
		return m, nil

	case profileselector.ProfileSavedMsg:
		// Handle saved profile (add or edit)
		// Map UI profile to Config profile
		p := config.Profile{
			Name:        msg.Profile.Name,
			Type:        msg.Profile.Type,
			Host:        msg.Profile.Host,
			Port:        msg.Profile.Port,
			User:        msg.Profile.User,
			Database:    msg.Profile.Database,
			Password:    msg.Profile.Password,
			SSHHost:     msg.Profile.SSHHost,
			SSHPort:     msg.Profile.SSHPort,
			SSHUser:     msg.Profile.SSHUser,
			SSHKeyPath:  msg.Profile.SSHKeyPath,
			SSHPassword: msg.Profile.SSHPassword,
		}

		if msg.IsNew {
			// Add new profile
			if err := m.config.AddProfile(p); err != nil {
				m.profileSelector = m.profileSelector.SetStatusMessage(fmt.Sprintf("Error adding profile: %v", err))
			} else {
				m.statusMsg = fmt.Sprintf("✓ Added profile: %s", p.Name)
				// Reload profile selector with updated list and reset its state
				m.reloadProfiles()
				m.profileSelector = m.profileSelector.ResetState()
			}
		} else {
			// Update existing profile
			if err := m.config.UpdateProfile(msg.Profile.Name, p); err != nil {
				m.profileSelector = m.profileSelector.SetStatusMessage(fmt.Sprintf("Error updating profile: %v", err))
			} else {
				m.statusMsg = fmt.Sprintf("✓ Updated profile: %s", p.Name)
				// Reload profile selector with updated list and reset its state
				m.reloadProfiles()
				m.profileSelector = m.profileSelector.ResetState()
			}
		}
		return m, nil

	case profileselector.ManagementMsg:
		// Handle profile management actions
		switch msg.Action {
		case profileselector.ActionDelete:
			// Delete profile
			if msg.Profile != nil {
				if err := m.config.DeleteProfile(msg.Profile.Name); err != nil {
					m.profileSelector = m.profileSelector.SetStatusMessage(fmt.Sprintf("Error deleting profile: %v", err))
				} else {
					m.profileSelector = m.profileSelector.SetStatusMessage(fmt.Sprintf("✓ Deleted profile: %s", msg.Profile.Name))
					// Reload profile selector with updated list
					profiles := make([]profileselector.Profile, len(m.config.Profiles))
					for i, p := range m.config.Profiles {
						profiles[i] = profileselector.Profile{
							Name:     p.Name,
							Type:     p.Type,
							Host:     p.Host,
							Database: p.Database,
							Password: p.Password,
						}
					}
					m.profileSelector = m.profileSelector.SetProfiles(profiles)
				}
			}
		}
		return m, nil

	case ProfileConnectedMsg:
		if msg.Err != nil {
			m.connectError = msg.Err.Error()
			m.appState = StateSelectingProfile // Go back to selection
			return m, nil
		}
		m.driver = msg.Driver
		m.appState = StateReady
		m.connectError = ""
		m.loadingTables = true // Start loading schema (shown in status bar)
		// Clear screen and load history AND schema in background
		return m, tea.Batch(
			tea.ClearScreen,
			textarea.Blink,
			m.loadHistoryCmd(),
			schemabrowser.LoadSchemaCmd(m.driver),
		)

	case ClipboardCopiedMsg:
		if msg.Err != nil {
			m.errorMsg = fmt.Sprintf("Clipboard error: %v", msg.Err)
			m.statusMsg = ""
		} else {
			m.errorMsg = ""
			m.statusMsg = "Copied to clipboard"
		}
		return m, nil

	case schemabrowser.SchemaLoadedMsg:
		if msg.Err == nil {
			m.schemaBrowser = m.schemaBrowser.SetSchema(msg.Tables, msg.Columns, msg.Constraints)
			m.tables = msg.Tables
			m.columns = msg.Columns
			m.statusMsg = fmt.Sprintf("Loaded %d tables", len(msg.Tables))
		} else {
			m.errorMsg = fmt.Sprintf("Schema load failed: %v", msg.Err)
		}
		m.loadingTables = false
		if m.autocompleting {
			m = m.updateSuggestions()
		}
		return m, nil

	case schemabrowser.TableSelectedMsg:
		// Show template popup for selected table
		m.openTemplatePopup(msg.TableName)
		return m, nil

	case schemabrowser.ExportTableMsg:
		// Export table - show export popup with suggested filename
		m.exportTable = msg.TableName
		m.openExportPopup(msg.TableName + ".csv")
		return m, textinput.Blink

	case schemabrowser.ImportTableMsg:
		// Import into table - show import popup
		m.openImportPopup(msg.TableName)
		return m, textinput.Blink

	case ThemeSelectedMsg:
		m.config.Theme = msg.Theme
		m.config.ThemeName = msg.ThemeName
		InitStyles(m.config.Theme)

		// Update sub-components with new theme
		m.profileSelector = m.profileSelector.SetStyles(profileselector.DefaultStyles(m.config.Theme))
		m.themeSelector = m.themeSelector.UpdateTheme(m.config.Theme)

		// Update global eztable - this affects new tables
		eztable.Init(m.config.Theme, m.config.Keys)

		// Recreate existing tables with new theme
		if m.expandedID != 0 && m.selected >= 0 && m.selected < len(m.history) {
			entry := m.history[m.selected]
			if strings.Contains(entry.Preview, " | ") {
				m.expandedTable = eztable.FromPreview(entry.Preview).
					WithMaxTotalWidth(m.width - 14).
					WithHorizontalFreezeColumnCount(1)
			}
		}
		if m.popupResult != nil {
			m.popupTable = eztable.FromQueryResult(m.popupResult, 0).Focused(true)
			m.updatePopupTable()
		}

		// Re-initialize schema browser styles
		m.schemaBrowser = m.schemaBrowser.SetStyles(schemabrowser.Styles{
			Container:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(m.config.Theme.Highlight)).Padding(1, 2),
			Title:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.config.Theme.Accent)).MarginBottom(1),
			SectionTitle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.config.Theme.Highlight)).MarginTop(1).MarginBottom(1),
			Item:          lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.TextPrimary)),
			ItemActive:    lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Success)).Bold(true),
			TableHeader:   lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Accent)).Bold(true).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(lipgloss.Color(m.config.Theme.BorderColor)),
			TableCell:     lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.TextPrimary)),
			TableCellKey:  lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Success)),
			TableCellType: lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.TextFaint)),
			Spinner:       lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Highlight)),
			TabActive:     lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Success)).Bold(true).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(lipgloss.Color(m.config.Theme.Success)).Padding(0, 1),
			TabInactive:   lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.TextFaint)).Padding(0, 1),
		})

		// Save config
		m.config.Save()
		// Pop theme selector from stack since it closed itself
		if m.popupStack.TopName() == "theme" {
			m.popupStack.Pop()
		}
		return m, tea.ClearScreen

	case ExportTableCompleteMsg:
		m.loading = false
		if msg.Err != nil {
			m.errorMsg = fmt.Sprintf("Export failed: %v", msg.Err)
		} else {
			m.statusMsg = fmt.Sprintf("Exported %d rows to %s", msg.Rows, msg.Filename)
		}
		m.exportTable = ""
		return m, nil

	case ImportTableCompleteMsg:
		m.loading = false
		if msg.Err != nil {
			m.errorMsg = fmt.Sprintf("Import failed: %v", msg.Err)
		} else {
			m.statusMsg = fmt.Sprintf("Imported %d rows", msg.Rows)
		}
		m.importTable = ""
		return m, nil
	}

	// Handle schema browser updates (spinner ticks etc)
	// KeyMsg is specifically handled below
	if _, ok := msg.(tea.KeyMsg); !ok {
		var sbCmd tea.Cmd
		m.schemaBrowser, sbCmd = m.schemaBrowser.Update(msg)
		cmds = append(cmds, sbCmd)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Clear status message on any key press
		m.statusMsg = ""

		// DRIVER-FIRST HANDLING: If we are selecting a profile, delegate immediately
		// to avoid global keys (like 't' for theme) intercepting input in the Add/Edit form.
		if m.appState == StateSelectingProfile {
			if matchKey(msg, m.config.Keys.Help) {
				m.openHelpPopup()
				return m, nil
			}
			var cmd tea.Cmd
			m.profileSelector, cmd = m.profileSelector.Update(msg)
			return m, cmd
		}

		// Global keys that should intercept everything (but not typing in InsertMode)
		// Try to run them only if NOT in InsertMode, OR if they are modifier keys (Ctrl+...)
		// Don't intercept 't' when schema browser or theme selector is visible
		if m.mode != InsertMode && !m.schemaBrowser.IsVisible() && !m.themeSelector.Visible() && matchKey(msg, m.config.Keys.ToggleTheme) {
			m.openThemeSelector()
			return m, nil
		}

		// Universal popup close handler - Esc/q closes topmost popup via stack
		// Fallback to hardcoded "esc"/"q" if Keys.Exit is empty
		isExitKey := matchKey(msg, m.config.Keys.Exit) || msg.String() == "esc" || msg.String() == "q"
		// Check both stack AND boolean flags for robustness
		hasPopup := m.hasOpenPopup() || m.showPopup || m.showHelpPopup || m.showTemplatePopup ||
			m.showImportPopup || m.showExportPopup || m.showRowActionPopup || m.showActionPopup ||
			m.themeSelector.Visible()
		if hasPopup && isExitKey {
			f, _ := os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			fmt.Fprintf(f, "Exit key pressed. Stack len: %d. Top: %s\n", m.popupStack.Len(), m.popupStack.TopName())
			f.Close()
			// Try stack first
			if m.closeTopPopup() {
				return m, nil
			}
			// Fallback: close any open popup directly
			if m.showRowActionPopup {
				m.showRowActionPopup = false
				return m, nil
			}
			if m.showExportPopup {
				m.showExportPopup = false
				m.exportInput.Blur()
				return m, nil
			}
			if m.showActionPopup {
				m.showActionPopup = false
				return m, nil
			}
			if m.showPopup {
				m.showPopup = false
				m.tableFilterInput.Blur()
				m.tableFilterInput.SetValue("")
				return m, nil
			}
			if m.showTemplatePopup {
				m.showTemplatePopup = false
				m.templateTable = ""
				m.templateIdx = 0
				return m, nil
			}
			if m.showImportPopup {
				m.showImportPopup = false
				m.importInput.Blur()
				m.importTable = ""
				return m, nil
			}
			if m.showHelpPopup {
				m.showHelpPopup = false
				return m, nil
			}
			if m.themeSelector.Visible() {
				m.themeSelector = m.themeSelector.Hide()
				return m, nil
			}
		}

		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				m.loading = true
				query := m.pendingQuery
				m.pendingQuery = ""
				return m, m.executeQueryCmd(query)
			case "n", "N", "esc":
				m.confirming = false
				m.pendingQuery = ""
				return m, nil
			}
			return m, nil
		}

		// Theme selector handling
		if m.themeSelector.Visible() {
			if matchKey(msg, m.config.Keys.Help) {
				m.openHelpPopup()
				return m, nil
			}
			var cmd tea.Cmd
			m.themeSelector, cmd = m.themeSelector.Update(msg)
			// If theme selector closed itself (e.g., via enter), pop from stack
			if !m.themeSelector.Visible() && m.popupStack.TopName() == "theme" {
				m.popupStack.Pop()
			}
			return m, cmd
		}

		// Help popup handling (? toggles)
		if m.showHelpPopup {
			if matchKey(msg, m.config.Keys.Help) {
				m.closeTopPopup()
				return m, nil
			}
			return m, nil // Block other keys while help is shown
		}

		// Template popup handling
		if m.showTemplatePopup {
			switch msg.String() {
			case "up", "k":
				if m.templateIdx > 0 {
					m.templateIdx--
				}
				return m, nil
			case "down", "j":
				if m.templateIdx < len(m.config.QueryTemplates)-1 {
					m.templateIdx++
				}
				return m, nil
			case "enter":
				m.popupStack.Pop() // Remove from stack before executing
				return m.executeTemplate()
			case "i":
				m.popupStack.Pop() // Remove from stack before inserting
				m = m.insertTemplate()
				return m, nil
			}
			return m, nil
		}

		// Import popup handling
		if m.showImportPopup {
			if msg.String() == "enter" {
				filename := m.importInput.Value()
				if filename != "" {
					m.popupStack.Pop() // Remove from stack
					m.showImportPopup = false
					m.importInput.Blur()
					m.importTable = ""
					m.loading = true
					return m, m.importTableCmd(m.importTable, filename)
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.importInput, cmd = m.importInput.Update(msg)
			return m, cmd
		}

		// Export popup (innermost layer) - Esc handled by stack
		if m.showExportPopup {
			if msg.String() == "enter" {
				filename := m.exportInput.Value()
				if filename == "" {
					filename = "export.csv"
				}
				m.popupStack.Pop() // Remove export from stack
				m.showExportPopup = false
				m.exportInput.Blur()
				if m.exportTable != "" {
					m.loading = true
					return m, m.exportTableCmd(m.exportTable, filename)
				}
				return m, m.exportTableToPath(filename)
			}
			var cmd tea.Cmd
			m.exportInput, cmd = m.exportInput.Update(msg)
			return m, cmd
		}

		// Popup handling (results table popup)
		if m.showPopup {
			// Handle filter input if active
			if m.tableFilterActive {
				if msg.Type == tea.KeyEnter || msg.Type == tea.KeyEsc {
					m.tableFilterActive = false
					m.tableFilterInput.Blur()
					return m, nil
				}
				var cmd tea.Cmd
				m.tableFilterInput, cmd = m.tableFilterInput.Update(msg)
				m.popupTable = m.popupTable.WithFilterInputValue(m.tableFilterInput.Value())
				return m, cmd
			}

			// Row action popup (innermost layer) - Esc handled by stack
			if m.showRowActionPopup {
				switch msg.String() {
				case "1": // Select this row (Query)
					m.popupStack.Pop()
					return m.selectRowAsQuery()
				case "2": // View full row
					m.popupStack.Pop()
					return m.viewFullRow()
				case "3": // Copy as JSON
					m.popupStack.Pop()
					m.showRowActionPopup = false
					return m, m.copyRowAsJSON()
				case "4": // Copy as CSV
					m.popupStack.Pop()
					m.showRowActionPopup = false
					return m, m.copyRowAsCSV()
				}
				return m, nil
			}

			// Action menu popup (middle layer) - Esc handled by stack
			if m.showActionPopup {
				// TODO: Handle action selection
				return m, nil
			}

			// Table popup (outer layer) - Esc handled by stack
			if msg.String() == "a" {
				m.openActionPopup()
				return m, nil
			} else if matchKey(msg, m.config.Keys.Filter) {
				m.tableFilterActive = true
				m.tableFilterInput.Focus()
				return m, textinput.Blink
			} else if matchKey(msg, m.config.Keys.RowAction) {
				m.openRowActionPopup()
				return m, nil
			} else if matchKey(msg, m.config.Keys.Export) {
				m.openExportPopup("export.csv")
				return m, textinput.Blink
			} else if matchKey(msg, m.config.Keys.Help) {
				m.openHelpPopup()
				return m, nil
			}

			// Pass other keys to table for navigation
			var cmd tea.Cmd
			m.popupTable, cmd = m.popupTable.Update(msg)
			return m, cmd
		}

		// Global keys
		if matchKey(msg, m.config.Keys.Quit) {
			return m, tea.Quit
		}

		// Tab key toggles schema browser
		if matchKey(msg, m.config.Keys.ToggleSchema) && m.mode == VisualMode {
			m.schemaBrowser = m.schemaBrowser.Toggle()
			if m.schemaBrowser.IsVisible() && m.driver != nil {
				return m, schemabrowser.LoadSchemaCmd(m.driver)
			}
			return m, nil
		}

		// ? key toggles help popup (only when no other popups are visible)
		if matchKey(msg, m.config.Keys.Help) && !m.hasOpenPopup() {
			m.openHelpPopup()
			return m, nil
		}

		// P key shows profile selector (reconnect)
		if matchKey(msg, m.config.Keys.ShowProfiles) && m.mode == VisualMode {
			if m.driver != nil {
				m.driver.Close()
				m.driver = nil
			}
			m.appState = StateSelectingProfile
			m.reloadProfiles()
			return m, nil
		}

		// Schema browser handling (when visible)
		if m.schemaBrowser.IsVisible() {
			var cmd tea.Cmd
			m.schemaBrowser, cmd = m.schemaBrowser.Update(msg)
			return m, cmd
		}

		// Mode-specific handling
		if m.mode == InsertMode {
			var cmd tea.Cmd

			// Autocomplete interaction
			hasPopup := m.hasOpenPopup() || m.showPopup || m.showHelpPopup || m.showTemplatePopup ||
				m.showImportPopup || m.showExportPopup || m.showRowActionPopup || m.showActionPopup ||
				m.themeSelector.Visible()

			if m.autocompleting && !hasPopup {
				switch msg.String() {
				case "up", "ctrl+p":
					if m.suggestionIdx > 0 {
						m.suggestionIdx--
					}
					return m, nil
				case "down", "ctrl+n":
					if m.suggestionIdx < len(m.suggestions)-1 {
						m.suggestionIdx++
					}
					return m, nil
				case "enter", "tab":
					m = m.applySuggestion()
					m.autocompleting = false
					return m, nil
				case "esc":
					m.autocompleting = false
					return m, nil
				}
			}

			if matchKey(msg, m.config.Keys.Autocomplete) && !hasPopup {
				m.autocompleting = true
				m = m.updateSuggestions()
				if len(m.tables) == 0 && !m.loadingTables {
					// m.loadingTables = true
					// cmd = m.fetchTablesCmd()
					// cmds = append(cmds, cmd)
				}
				return m, tea.Batch(cmds...)
			} else if matchKey(msg, m.config.Keys.Execute) {
				query := strings.TrimSpace(m.editor.Value())
				if query != "" {
					m.editor.SetValue("")
					m.editor.Reset() // Reset cursor to top-left

					if m.strictMode && isModifyingQuery(query) {
						m.confirming = true
						m.pendingQuery = query
						return m, nil
					}
					m.loading = true
					cmds = append(cmds, m.executeQueryCmd(query))
					// Scroll to bottom handled by history update
				}
				return m, tea.Batch(cmds...)
			} else if matchKey(msg, m.config.Keys.Explain) {
				query := strings.TrimSpace(m.editor.Value())
				if query != "" && m.driver != nil {
					explainQuery := "EXPLAIN " + query
					if m.driver.Type() == db.SQLite {
						explainQuery = "EXPLAIN QUERY PLAN " + query
					}
					m.loading = true
					cmds = append(cmds, m.executeQueryCmd(explainQuery))
				}
				return m, tea.Batch(cmds...)
			} else if matchKey(msg, m.config.Keys.Undo) {
				if len(m.undoStack) > 0 {
					// Push current to redo
					m.redoStack = append(m.redoStack, m.editor.Value())
					// Pop from undo
					prev := m.undoStack[len(m.undoStack)-1]
					m.undoStack = m.undoStack[:len(m.undoStack)-1]
					m.editor.SetValue(prev)
				}
				return m, nil
			} else if matchKey(msg, m.config.Keys.Redo) {
				if len(m.redoStack) > 0 {
					// Push current to undo
					m.undoStack = append(m.undoStack, m.editor.Value())
					// Pop from redo
					next := m.redoStack[len(m.redoStack)-1]
					m.redoStack = m.redoStack[:len(m.redoStack)-1]
					m.editor.SetValue(next)
				}
				return m, nil
			} else if matchKey(msg, m.config.Keys.Exit) || msg.String() == "esc" {
				m.mode = VisualMode
				m.editor.Blur()
				if len(m.history) > 0 {
					m.selected = len(m.history) - 1
					m = m.ensureSelectionVisible()
				}
				return m, nil
			}

			// Pass other keys to editor
			m.editor, cmd = m.editor.Update(msg)
			cmds = append(cmds, cmd)

			// Autocomplete Logic
			val := m.editor.Value()

			// 1. Empty Input: Clear suggestions
			if strings.TrimSpace(val) == "" {
				m.autocompleting = false
				m.suggestions = nil
				m.debounceID++ // Cancel pending callbacks
				return m, tea.Batch(cmds...)
			}

			// 3. SQL Autocomplete: Debounce 1s
			m.debounceID++
			id := m.debounceID
			cmd = tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
				return DebounceMsg{ID: id}
			})
			cmds = append(cmds, cmd)

			return m, tea.Batch(cmds...)
		}
		return m.handleVisualMode(msg)

	case DebounceMsg:
		if msg.ID == m.debounceID {
			m = m.updateSuggestions()
			if len(m.suggestions) > 0 {
				m.autocompleting = true
			} else {
				m.autocompleting = false
			}

			if m.autocompleting && len(m.tables) == 0 && !m.loadingTables {
				// Tables should be loaded by schema browser on startup
				// If not, we could trigger it, but for now let's avoid redundant fetches
				// m.loadingTables = true
			}
		}
		return m, nil

	case PagerFinishedMsg:
		if msg.Err != nil {
			m.errorMsg = fmt.Sprintf("Pager error: %v", msg.Err)
		}
		return m, nil

	case QueryResultMsg:
		m.loading = false
		if msg.Err != nil {
			m.errorMsg = msg.Err.Error()
			if msg.Entry != nil {
				m.history = append(m.history, *msg.Entry)
				m.selected = len(m.history) - 1
				m.expandedID = msg.Entry.ID
				if strings.Contains(msg.Entry.Preview, " | ") {
					m.expandedTable = eztable.FromPreview(msg.Entry.Preview).
						WithMaxTotalWidth(m.width - 14).
						WithHorizontalFreezeColumnCount(1)
				}
			}
		} else {
			m.results = msg.Result
			m.page = 0
			if msg.Entry != nil {
				// Append to end (newest)
				m.history = append(m.history, *msg.Entry)
				// Auto-select the new item
				m.selected = len(m.history) - 1

				if msg.Result.IsSelect {
					if m.config.Pager != "" {
						// Use external pager for results
						return m, m.openPager(msg.Result)
					}
					// No pager - show popup with full result table
					m.popupTable = eztable.FromQueryResult(msg.Result, 0).
						Focused(true)
					m.updatePopupTable()
					m.openResultsPopup(msg.Entry, msg.Result)
					m.expandedID = msg.Entry.ID // Expand the history entry as well
				} else {
					// Non-SELECT (INSERT, UPDATE, DELETE) - expand with preview
					m.expandedID = msg.Entry.ID
					if strings.Contains(msg.Entry.Preview, " | ") {
						m.expandedTable = eztable.FromPreview(msg.Entry.Preview).
							WithMaxTotalWidth(m.width - 14).
							WithHorizontalFreezeColumnCount(1)
					}
				}
			}
			m.errorMsg = ""
		}
		// Update viewport content to ensure correct scrolling
		m = m.updateHistoryViewport()
		// Scroll viewport to bottom to show new entry
		m.viewport.GotoBottom()
		m = m.ensureSelectionVisible()
		return m, nil

	case HistoryLoadedMsg:
		if msg.Err == nil {
			// Reverse entries so they are Oldest -> Newest
			// Store returns Newest -> Oldest (DESC)
			entries := msg.Entries
			for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
				entries[i], entries[j] = entries[j], entries[i]
			}
			m.history = entries

			if len(m.history) > 0 {
				m.selected = len(m.history) - 1         // Select newest (bottom)
				m.expandedID = m.history[m.selected].ID // Expand newest by default
				if strings.Contains(m.history[m.selected].Preview, " | ") {
					m.expandedTable = eztable.FromPreview(m.history[m.selected].Preview).
						WithMaxTotalWidth(m.width - 14).
						WithHorizontalFreezeColumnCount(1)
				}
				m = m.updateHistoryViewport()
				m.viewport.GotoBottom()
			}
		}
		return m, nil

	case RerunResultMsg:
		m.loading = false
		if msg.Err == nil {
			// Create table without width constraints for scrolling
			m.popupTable = eztable.FromQueryResult(msg.Result, 0).
				Focused(true)
			m.updatePopupTable()
			m.openResultsPopup(msg.Entry, msg.Result)
		} else {
			m.errorMsg = msg.Err.Error()
		}
		return m, nil

	case ExportCompleteMsg:
		if msg.Err != nil {
			m.errorMsg = fmt.Sprintf("Export failed: %v", msg.Err)
		} else {
			m.statusMsg = fmt.Sprintf("Exported to: %s", msg.Path)
		}
		return m, nil

	case ExportTableCompleteMsg:
		m.loading = false
		if msg.Err != nil {
			m.errorMsg = fmt.Sprintf("Export failed: %v", msg.Err)
		} else {
			m.statusMsg = fmt.Sprintf("Exported %d rows to %s", msg.Rows, msg.Filename)
		}
		return m, nil
	}

	// Update editor if in insert mode
	if m.mode == InsertMode {
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleVisualMode handles keys in visual mode
func (m Model) handleVisualMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if matchKey(msg, m.config.Keys.InsertMode) {
		m.mode = InsertMode
		m.editor.Focus()
		// Ensure cursor is at end?
		return m, textinput.Blink
	} else if matchKey(msg, m.config.Keys.MoveUp) { // Move up (older)
		if m.selected > 0 {
			m.selected--
			m = m.ensureSelectionVisible()
		}
	} else if matchKey(msg, m.config.Keys.MoveDown) { // Move down (newer)
		if m.selected < len(m.history)-1 {
			m.selected++
			m = m.ensureSelectionVisible()
		}
	} else if matchKey(msg, m.config.Keys.ScrollLeft) { // Scroll left
		if m.expandedID != 0 {
			m.expandedTable = m.expandedTable.ScrollLeft()
		}
	} else if matchKey(msg, m.config.Keys.ScrollRight) { // Scroll right
		if m.expandedID != 0 {
			m.expandedTable = m.expandedTable.ScrollRight()
		}
	} else if matchKey(msg, m.config.Keys.GoTop) {
		// For simplicity, single 'g' goes to top (gg would require state tracking)
		m.selected = 0
		m = m.ensureSelectionVisible()
	} else if matchKey(msg, m.config.Keys.GoBottom) {
		if len(m.history) > 0 {
			m.selected = len(m.history) - 1
			m = m.ensureSelectionVisible()
		}
	} else if matchKey(msg, m.config.Keys.ToggleExpand) { // Toggle expansion
		if m.selected >= 0 && m.selected < len(m.history) {
			entry := m.history[m.selected]
			if m.expandedID == entry.ID {
				m.expandedID = 0 // Collapse
				m.expandedTable = table.Model{}
			} else {
				m.expandedID = entry.ID // Expand
				if strings.Contains(entry.Preview, " | ") {
					m.expandedTable = eztable.FromPreview(entry.Preview).
						WithMaxTotalWidth(m.width - 14).
						WithHorizontalFreezeColumnCount(1)
				}
			}
			// Re-calculate visibility after expansion change
			m = m.ensureSelectionVisible()
		}
	} else if matchKey(msg, m.config.Keys.Rerun) { // Re-run selected query
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
	} else if matchKey(msg, m.config.Keys.ToggleStrict) { // Toggle strict mode
		m.strictMode = !m.strictMode
		m.errorMsg = "" // Clear error if toggling
		return m, nil
	} else if matchKey(msg, m.config.Keys.Edit) { // Edit selected query
		if m.selected >= 0 && m.selected < len(m.history) {
			entry := m.history[m.selected]
			m.editor.SetValue(entry.Query)
			m.mode = InsertMode
			m.editor.Focus()
			return m, textinput.Blink
		}
	} else if matchKey(msg, m.config.Keys.Delete) { // Delete selected query
		if m.selected >= 0 && m.selected < len(m.history) {
			entry := m.history[m.selected]
			m.historyStore.Delete(entry.ID)
			m.history = append(m.history[:m.selected], m.history[m.selected+1:]...)
			if m.selected >= len(m.history) && m.selected > 0 {
				m.selected--
			}
			m = m.ensureSelectionVisible()
		}
	} else if matchKey(msg, m.config.Keys.Copy) { // Copy selected query to clipboard
		if m.selected >= 0 && m.selected < len(m.history) {
			entry := m.history[m.selected]
			return m, m.copyToClipboardCmd(entry.Query)
		}
	} else if matchKey(msg, m.config.Keys.Filter) { // Enter search mode
		m.searching = true
		m.searchQuery = ""
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		return m, textinput.Blink
	} else if matchKey(msg, m.config.Keys.ToggleSchema) { // Toggle schema browser
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

func (m *Model) updatePopupTable() {
	if m.width == 0 || m.height == 0 {
		return
	}

	// Set table dimensions
	// Height: based on available vertical space
	// Width: constrain to prevent popup overflow, table will be horizontally scrollable
	availableHeight := m.height - 28
	if availableHeight < 3 {
		availableHeight = 3
	}

	// Table width should fit within popup (which is m.width - 10)
	// Account for popup padding (4) + borders (4) + table borders (2)
	popupWidth := m.width - 10
	if popupWidth < 60 {
		popupWidth = 60
	}
	maxTableWidth := popupWidth - 10 // Extra margin for borders and padding

	// Use WithMaxTotalWidth for proper horizontal scrolling (like bubble-table example)
	// WithHorizontalFreezeColumnCount(1) keeps first column visible when scrolling
	m.popupTable = m.popupTable.
		WithPageSize(availableHeight).
		WithMaxTotalWidth(maxTableWidth).
		WithHorizontalFreezeColumnCount(1)
}

func (m Model) selectRowAsQuery() (Model, tea.Cmd) {
	if m.popupTable.HighlightedRow().Data == nil {
		return m, nil
	}

	// Extract table name from query
	// This handles both "table" and "schema.table"
	query := m.popupEntry.Query
	re := regexp.MustCompile(`(?i)from\s+["'\[]?([a-zA-Z0-9._]+)["'\]]?`)
	matches := re.FindStringSubmatch(query)
	if len(matches) < 2 {
		m.errorMsg = "Could not determine table name from query"
		return m, nil
	}
	tableName := matches[1]

	// Find the columns for this table, handling schema prefixes
	var cols []db.Column
	var ok bool

	// 1. Direct match
	if cols, ok = m.columns[tableName]; !ok {
		// 2. Case-insensitive
		for realName, c := range m.columns {
			if strings.EqualFold(realName, tableName) {
				tableName = realName
				cols = c
				ok = true
				break
			}
		}

		if !ok {
			// 3. Suffix match (lookup "users" in "public.users")
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
		// Log full debug info to file since UI might truncate it
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

	// Construct WHERE clause
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

		// Don't quote numbers or booleans
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
	// Close popups
	m.showPopup = false
	m.showRowActionPopup = false
	m.showActionPopup = false
	m.mode = InsertMode

	return m, nil
}

// viewFullRow displays all columns and values for the highlighted row
func (m Model) viewFullRow() (Model, tea.Cmd) {
	highlightedRow := m.popupTable.HighlightedRow()
	if highlightedRow.Data == nil || m.popupResult == nil {
		return m, nil
	}

	// Build formatted view of all row data
	var content strings.Builder
	content.WriteString("-- Row Details --\n")

	for _, col := range m.popupResult.Columns {
		if val, ok := highlightedRow.Data[col]; ok {
			val = unwrapCellValue(val)
			content.WriteString(fmt.Sprintf("%s: %v\n", col, val))
		}
	}

	// Put the row details in the editor for viewing/copying
	m.editor.SetValue(content.String())
	m.showPopup = false
	m.showRowActionPopup = false
	m.mode = InsertMode

	return m, nil
}

// Popup stack helper methods

// openHelpPopup opens the help popup and pushes closer to stack
func (m *Model) openHelpPopup() {
	if m.showHelpPopup {
		return
	}
	// Add to stack so it can be closed with q/Esc
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

// openTemplatePopup opens template popup for a table
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

// openResultsPopup opens the results popup
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

// openRowActionPopup opens row action popup
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

// openExportPopup opens export popup
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

// openImportPopup opens import popup for a table
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

// openActionPopup opens action menu popup
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

// openThemeSelector opens theme selector popup
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

// closeTopPopup closes the topmost popup using the stack
func (m *Model) closeTopPopup() bool {
	if m.popupStack == nil {
		return false
	}
	return m.popupStack.CloseTop(m)
}

// hasOpenPopup returns true if any popup is open
func (m *Model) hasOpenPopup() bool {
	if m.popupStack == nil {
		return false
	}
	return !m.popupStack.IsEmpty()
}

// reloadProfiles updates the profile selector with current config
func (m *Model) reloadProfiles() {
	profiles := make([]profileselector.Profile, len(m.config.Profiles))
	for i, cp := range m.config.Profiles {
		profiles[i] = profileselector.Profile{
			Name:     cp.Name,
			Type:     cp.Type,
			Host:     cp.Host,
			Port:     cp.Port,
			User:     cp.User,
			Database: cp.Database,
			Password: cp.Password,
		}
	}
	m.profileSelector = m.profileSelector.SetProfiles(profiles)
}

// addSystemMessage adds a system message to the history
func (m Model) addSystemMessage(msg string) Model {
	entry := history.HistoryEntry{
		ID:         time.Now().UnixNano(),
		Query:      msg,
		Status:     "info",
		ExecutedAt: time.Now(),
	}
	m.history = append(m.history, entry)
	m.selected = len(m.history) - 1
	m.viewport.SetContent(m.renderHistoryContent(m.viewport.Height))
	m.viewport.GotoBottom()
	return m
}

