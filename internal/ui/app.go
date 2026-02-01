package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nhath/ezdb/internal/history"
	"github.com/nhath/ezdb/internal/ui/components/profileselector"
	"github.com/nhath/ezdb/internal/ui/components/schemabrowser"
	eztable "github.com/nhath/ezdb/internal/ui/components/table"
	"github.com/nhath/ezdb/internal/ui/styles"
)

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// --- Non-key messages (structural / async results) ---
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
		return m.handleProfileSelected(msg)

	case profileselector.ProfileSavedMsg:
		return m.handleProfileSaved(msg)

	case profileselector.ManagementMsg:
		return m.handleProfileManagement(msg)

	case ProfileConnectedMsg:
		return m.handleProfileConnected(msg)

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
		m.openTemplatePopup(msg.TableName)
		return m, nil

	case schemabrowser.ExportTableMsg:
		m.exportTable = msg.TableName
		m.openExportPopup(msg.TableName + ".csv")
		return m, nil

	case schemabrowser.ImportTableMsg:
		m.openImportPopup(msg.TableName)
		return m, nil

	case ThemeSelectedMsg:
		return m.handleThemeSelected(msg)

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

	case DebounceMsg:
		if msg.ID == m.debounceID {
			m = m.updateSuggestions()
			if len(m.suggestions) > 0 {
				m.autocompleting = true
			} else {
				m.autocompleting = false
			}
		}
		return m, nil

	case PagerFinishedMsg:
		if msg.Err != nil {
			m.errorMsg = fmt.Sprintf("Pager error: %v", msg.Err)
		}
		return m, nil

	case QueryResultMsg:
		return m.handleQueryResult(msg)

	case HistoryLoadedMsg:
		return m.handleHistoryLoaded(msg)

	case RerunResultMsg:
		m.loading = false
		if msg.Err == nil {
			m.popupTable = eztable.FromQueryResult(msg.Result, 0).Focused(true)
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
	}

	// Forward non-KeyMsg to schema browser (spinner ticks, etc.)
	if _, ok := msg.(tea.KeyMsg); !ok {
		var sbCmd tea.Cmd
		m.schemaBrowser, sbCmd = m.schemaBrowser.Update(msg)
		cmds = append(cmds, sbCmd)
	}

	// --- Key messages ---
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.statusMsg = "" // clear status on any key

		// Profile-selection state: delegate immediately
		if m.appState == StateSelectingProfile {
			if matchKey(msg, m.config.Keys.Help) {
				m.openHelpPopup()
				return m, nil
			}
			var cmd tea.Cmd
			m.profileSelector, cmd = m.profileSelector.Update(msg)
			return m, cmd
		}

		// Toggle theme (only outside insert mode and when schema/theme not visible)
		if m.mode != InsertMode && !m.schemaBrowser.IsVisible() && !m.themeSelector.Visible() && matchKey(msg, m.config.Keys.ToggleTheme) {
			m.openThemeSelector()
			return m, nil
		}

		// Let popup layer handle its keys first
		if m2, cmd, handled := m.handlePopupKeys(msg); handled {
			return m2, cmd
		}

		// Global quit
		if matchKey(msg, m.config.Keys.Quit) {
			return m, tea.Quit
		}

		// Tab toggles schema browser (visual mode only, outside schema browser)
		if matchKey(msg, m.config.Keys.ToggleSchema) && m.mode == VisualMode {
			m.schemaBrowser = m.schemaBrowser.Toggle()
			if m.schemaBrowser.IsVisible() && m.driver != nil {
				return m, schemabrowser.LoadSchemaCmd(m.driver)
			}
			return m, nil
		}

		// Help shortcut (when no popup open)
		if matchKey(msg, m.config.Keys.Help) && !m.hasOpenPopup() {
			m.openHelpPopup()
			return m, nil
		}

		// P – reconnect / show profile selector
		if matchKey(msg, m.config.Keys.ShowProfiles) && m.mode == VisualMode {
			if m.driver != nil {
				m.driver.Close()
				m.driver = nil
			}
			m.appState = StateSelectingProfile
			m.reloadProfiles()
			return m, nil
		}

		// Schema browser consumes keys when visible
		if m.schemaBrowser.IsVisible() {
			var cmd tea.Cmd
			m.schemaBrowser, cmd = m.schemaBrowser.Update(msg)
			return m, cmd
		}

		// Mode dispatch
		if m.mode == InsertMode {
			var updatedCmds []tea.Cmd
			m, updatedCmds = m.handleInsertMode(msg, cmds)
			return m, tea.Batch(updatedCmds...)
		}
		return m.handleVisualMode(msg)
	}

	// Fallback: forward to editor if in insert mode
	if m.mode == InsertMode {
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// --- Result / history message handlers ---

// handleQueryResult processes a completed query execution.
func (m Model) handleQueryResult(msg QueryResultMsg) (Model, tea.Cmd) {
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
			m.history = append(m.history, *msg.Entry)
			m.selected = len(m.history) - 1

			if msg.Result.IsSelect {
				if m.config.Pager != "" {
					return m, m.openPager(msg.Result)
				}
				m.popupTable = eztable.FromQueryResult(msg.Result, 0).Focused(true)
				m.updatePopupTable()
				m.openResultsPopup(msg.Entry, msg.Result)
				m.expandedID = msg.Entry.ID
			} else {
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
	m = m.updateHistoryViewport()
	m.viewport.GotoBottom()
	m = m.ensureSelectionVisible()
	return m, nil
}

// handleHistoryLoaded processes loaded history entries.
func (m Model) handleHistoryLoaded(msg HistoryLoadedMsg) (Model, tea.Cmd) {
	if msg.Err == nil {
		// Reverse: store returns newest-first, we want oldest-first
		entries := msg.Entries
		for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
			entries[i], entries[j] = entries[j], entries[i]
		}
		m.history = entries

		if len(m.history) > 0 {
			m.selected = len(m.history) - 1
			m.expandedID = m.history[m.selected].ID
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
}

// handleThemeSelected processes a theme change.
func (m Model) handleThemeSelected(msg ThemeSelectedMsg) (Model, tea.Cmd) {
	m.config.Theme = msg.Theme
	m.config.ThemeName = msg.ThemeName
	styles.Init(m.config.Theme)

	m.profileSelector = m.profileSelector.SetStyles(profileselector.DefaultStyles(m.config.Theme))
	m.themeSelector = m.themeSelector.UpdateTheme(m.config.Theme)
	eztable.Init(m.config.Theme, m.config.Keys)

	// Recreate expanded table with new theme
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

	// Re-init schema browser styles
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

	m.config.Save()
	if m.popupStack.TopName() == "theme" {
		m.popupStack.Pop()
	}
	return m, tea.ClearScreen
}

// addSystemMessage appends an informational entry to the visible history.
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
