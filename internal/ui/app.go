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
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/rmhubbert/bubbletea-overlay"

	"github.com/evertras/bubble-table/table"
	"github.com/nhath/ezdb/internal/config"
	"github.com/nhath/ezdb/internal/db"
	"github.com/nhath/ezdb/internal/history"
	"github.com/nhath/ezdb/internal/ui/components/profileselector"
	"github.com/nhath/ezdb/internal/ui/components/schemabrowser"
	eztable "github.com/nhath/ezdb/internal/ui/components/table"
)

// Mode represents the current UI mode
type Mode string

// AppState represents the overall application state
type AppState string

const (
	InsertMode Mode = "INSERT"
	VisualMode Mode = "VISUAL"
)

const (
	StateSelectingProfile AppState = "SELECTING_PROFILE"
	StateConnecting       AppState = "CONNECTING"
	StateReady            AppState = "READY"
)

// Model is the root Bubble Tea model
type Model struct {
	// App state
	appState AppState

	// Core state
	mode          Mode
	width, height int
	profile       *config.Profile
	driver        db.Driver
	historyStore  *history.Store
	config        *config.Config

	// Profile selector
	profileSelector profileselector.Model

	// Components
	editor        textarea.Model
	viewport      viewport.Model
	history       []history.HistoryEntry
	expandedID    int64 // ID of the currently expanded history item
	expandedTable table.Model
	selected      int // selected history item in visual mode

	// Results
	results      *db.QueryResult
	resultsTable table.Model
	page         int // current results page

	// Popup state
	showPopup          bool
	showActionPopup    bool
	showRowActionPopup bool // NEW: for showing detailed row actions
	popupEntry         *history.HistoryEntry
	popupResult        *db.QueryResult
	popupTable         table.Model

	// Autocomplete
	autocompleting    bool
	suggestions       []string
	suggestionDetails []string         // Column types, function signatures
	suggestionTypes   []SuggestionType // Type indicators for suggestions
	suggestionIdx     int
	tables            []string
	columns           map[string][]db.Column // table -> columns
	loadingTables     bool

	// Status
	loading      bool
	errorMsg     string
	connectError string

	// Search mode
	searching   bool
	searchQuery string
	searchInput textinput.Model

	// Cursor tracking for manual rendering
	cursorIndex int

	// Table filtering
	tableFilterActive bool
	tableFilterInput  textinput.Model

	// Debounce
	debounceID int

	// Schema browser sidebar
	schemaBrowser schemabrowser.Model

	// Undo/Redo history
	undoStack []string
	redoStack []string

	// Strict mode
	strictMode   bool
	confirming   bool
	pendingQuery string
}

func isModifyingQuery(query string) bool {
	q := strings.TrimSpace(strings.ToUpper(query))
	modifyingOps := []string{
		"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "TRUNCATE", "CREATE", "REPLACE",
	}
	for _, op := range modifyingOps {
		if strings.HasPrefix(q, op) {
			return true
		}
	}
	return false
	return false
}

func matchKey(msg tea.KeyMsg, keys []string) bool {
	keyStr := msg.String()
	for _, k := range keys {
		if k == keyStr {
			return true
		}
	}
	return false
}

// NewModel creates a new UI model
func NewModel(cfg *config.Config, profile *config.Profile, driver db.Driver, store *history.Store) Model {
	ti := textarea.New()
	ti.Placeholder = "Enter SQL query (Ctrl+D to execute, Esc for visual mode)..."
	ti.Focus()
	ti.CharLimit = 5000
	ti.SetHeight(3)
	ti.SetWidth(80)
	ti.ShowLineNumbers = false
	// Remove cursor line background - keep it transparent
	ti.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ti.BlurredStyle.CursorLine = lipgloss.NewStyle()

	// Initialize Table Filter Input
	tfi := textinput.New()
	tfi.Prompt = "/ "
	tfi.Placeholder = "Filter table..."
	tfi.CharLimit = 100
	tfi.Width = 30

	vp := viewport.New(80, 10)

	// Convert config profiles to selector profiles
	selectorProfiles := make([]profileselector.Profile, len(cfg.Profiles))
	for i, p := range cfg.Profiles {
		selectorProfiles[i] = profileselector.Profile{
			Name:        p.Name,
			Type:        p.Type,
			Host:        p.Host,
			Port:        p.Port,
			User:        p.User,
			Database:    p.Database,
			Password:    p.Password,
			SSHHost:     p.SSHHost,
			SSHPort:     p.SSHPort,
			SSHUser:     p.SSHUser,
			SSHKeyPath:  p.SSHKeyPath,
			SSHPassword: p.SSHPassword,
		}
	}
	ps := profileselector.New(selectorProfiles)

	// Determine initial state
	initialState := StateSelectingProfile
	if driver != nil && profile != nil {
		// Already connected (passed from main.go for backward compatibility)
		initialState = StateReady
	}

	return Model{
		appState:         initialState,
		mode:             VisualMode,
		profile:          profile,
		config:           cfg,
		driver:           driver,
		historyStore:     store,
		profileSelector:  ps,
		schemaBrowser:    schemabrowser.New(),
		editor:           ti,
		viewport:         vp,
		history:          []history.HistoryEntry{},
		expandedID:       0,
		selected:         0,
		page:             0,
		columns:          make(map[string][]db.Column),
		tableFilterInput: tfi,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	if m.appState == StateReady {
		return tea.Batch(
			textarea.Blink,
			m.loadHistoryCmd(),
			schemabrowser.LoadSchemaCmd(m.driver),
		)
	}
	// In profile selection state, just wait for input
	return nil
}

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
			m.expandedTable = m.expandedTable.WithTargetWidth(msg.Width - 14)
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
				m.profileSelector = m.profileSelector.SetStatusMessage(fmt.Sprintf("✓ Added profile: %s", p.Name))
				// Reload profile selector with updated list
				m.reloadProfiles()
			}
		} else {
			// Update existing profile
			if err := m.config.UpdateProfile(msg.Profile.Name, p); err != nil {
				m.profileSelector = m.profileSelector.SetStatusMessage(fmt.Sprintf("Error updating profile: %v", err))
			} else {
				m.profileSelector = m.profileSelector.SetStatusMessage(fmt.Sprintf("✓ Updated profile: %s", p.Name))
				// Reload profile selector with updated list
				m.reloadProfiles()
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
		// Load history AND schema
		sb, sbCmd := m.schemaBrowser.StartLoading()
		m.schemaBrowser = sb
		return m, tea.Batch(
			textarea.Blink,
			m.loadHistoryCmd(),
			schemabrowser.LoadSchemaCmd(m.driver),
			sbCmd,
		)

	case ClipboardCopiedMsg:
		if msg.Err != nil {
			m.errorMsg = fmt.Sprintf("Clipboard error: %v", msg.Err)
		} else {
			m.errorMsg = ""
			// Show success message
			m = m.addSystemMessage("Query copied to clipboard")
		}
		return m, nil

	case schemabrowser.SchemaLoadedMsg:
		if msg.Err == nil {
			m.schemaBrowser = m.schemaBrowser.SetSchema(msg.Tables, msg.Columns, msg.Constraints)
		}
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

		// Profile selector handling
		if m.appState == StateSelectingProfile {
			var cmd tea.Cmd
			m.profileSelector, cmd = m.profileSelector.Update(msg)
			return m, cmd
		}

		// Popup handling
		if m.showPopup {
			// Handle filter input if active
			if m.tableFilterActive {
				if msg.Type == tea.KeyEnter || msg.Type == tea.KeyEsc {
					m.tableFilterActive = false
					m.tableFilterInput.Blur()
					return m, nil
				}
				var cmd tea.Cmd
				// Convert KeyMsg back to generic Msg for textinput.Update
				// textinput.Update expects tea.Msg
				// Since we are inside case tea.KeyMsg, msg is tea.KeyMsg
				// We need to pass it as tea.Msg
				m.tableFilterInput, cmd = m.tableFilterInput.Update(msg)
				m.popupTable = m.popupTable.WithFilterInputValue(m.tableFilterInput.Value())
				return m, cmd
			}

			// Row action popup (innermost layer)
			if m.showRowActionPopup {
				if matchKey(msg, m.config.Keys.Exit) {
					m.showRowActionPopup = false
					return m, nil
				}
				switch msg.String() {
				case "1": // View full row
					// TODO: Implement full row view
					return m, nil
				case "2": // Copy as JSON
					return m, m.copyRowAsJSON()
				case "3": // Copy as CSV
					return m, m.copyRowAsCSV()
				case "4": // Select this row (Query)
					return m.selectRowAsQuery()
				}
				return m, nil
			}

			// Action menu popup (middle layer)
			if m.showActionPopup {
				if matchKey(msg, m.config.Keys.Exit) {
					m.showActionPopup = false
					return m, nil
				}
				// TODO: Handle action selection
				return m, nil
			}

			// Table popup (outer layer)
			// Check for special keys BEFORE passing to table
			if matchKey(msg, m.config.Keys.Exit) {
				m.showPopup = false
				m.tableFilterInput.Blur()
				m.tableFilterInput.SetValue("")
				m.popupTable = m.popupTable.WithFilterInputValue("")
				return m, nil
			} else if msg.String() == "a" {
				m.showActionPopup = true
				return m, nil
			} else if matchKey(msg, m.config.Keys.Filter) {
				m.tableFilterActive = true
				m.tableFilterInput.Focus()
				// Don't clear value so user can refine filter
				return m, textinput.Blink
			} else if matchKey(msg, m.config.Keys.RowAction) {
				// Show row action popup for highlighted row
				m.showRowActionPopup = true
				return m, nil
			} else if matchKey(msg, m.config.Keys.Export) {
				// Export table
				return m, m.exportTable()
			}

			// Pass other keys to table for navigation, filtering, etc.
			var cmd tea.Cmd
			m.popupTable, cmd = m.popupTable.Update(msg)
			return m, cmd
		}

		// Global keys
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Tab key toggles schema browser
		if msg.String() == "tab" && m.mode == VisualMode {
			m.schemaBrowser = m.schemaBrowser.Toggle()
			if m.schemaBrowser.IsVisible() && m.driver != nil {
				return m, schemabrowser.LoadSchemaCmd(m.driver)
			}
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
			if m.autocompleting {
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

			keyStr := msg.String()

			if keyStr == "ctrl+space" {
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

					// Check for commands
					if strings.HasPrefix(query, "/") {
						var cmd tea.Cmd
						m, cmd = m.handleCommand(query)
						cmds = append(cmds, cmd)
					} else {
						if m.strictMode && isModifyingQuery(query) {
							m.confirming = true
							m.pendingQuery = query
							return m, nil
						}
						m.loading = true
						cmds = append(cmds, m.executeQueryCmd(query))
					}
					// Scroll to bottom handled by history update
				}
				return m, tea.Batch(cmds...)
			} else if keyStr == "ctrl+z" {
				if len(m.undoStack) > 0 {
					// Push current to redo
					m.redoStack = append(m.redoStack, m.editor.Value())
					// Pop from undo
					prev := m.undoStack[len(m.undoStack)-1]
					m.undoStack = m.undoStack[:len(m.undoStack)-1]
					m.editor.SetValue(prev)
				}
				return m, nil
			} else if keyStr == "ctrl+y" {
				if len(m.redoStack) > 0 {
					// Push current to undo
					m.undoStack = append(m.undoStack, m.editor.Value())
					// Pop from redo
					next := m.redoStack[len(m.redoStack)-1]
					m.redoStack = m.redoStack[:len(m.redoStack)-1]
					m.editor.SetValue(next)
				}
				return m, nil
			} else if matchKey(msg, m.config.Keys.Exit) {
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

			// 2. Slash Commands: Immediate
			if strings.HasPrefix(val, "/") {
				m = m.updateSlashSuggestions()
				if len(m.suggestions) > 0 {
					m.autocompleting = true
				} else {
					m.autocompleting = false
				}
				m.debounceID++ // Cancel pending SQL callbacks
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

	case schemabrowser.SchemaLoadedMsg:
		if msg.Err != nil {
			m.errorMsg = fmt.Sprintf("Schema load failed: %v", msg.Err)
		} else {
			m.tables = msg.Tables
			m.columns = msg.Columns
		}
		m.loadingTables = false
		if m.autocompleting {
			m = m.updateSuggestions()
		}
		// Also pass to schema browser component
		var cmd tea.Cmd
		m.schemaBrowser, cmd = m.schemaBrowser.Update(msg)
		return m, cmd

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
					m.expandedTable = eztable.FromPreview(msg.Entry.Preview).WithTargetWidth(m.width - 14)
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
					m.popupEntry = msg.Entry
					m.popupResult = msg.Result
					// Create table without width constraints for scrolling
					m.popupTable = eztable.FromQueryResult(msg.Result, 0).
						Focused(true)
					m.updatePopupTable()
					m.showPopup = true
				} else {
					// Non-SELECT (INSERT, UPDATE, DELETE) - expand with preview
					m.expandedID = msg.Entry.ID
					if strings.Contains(msg.Entry.Preview, " | ") {
						m.expandedTable = eztable.FromPreview(msg.Entry.Preview).WithTargetWidth(m.width - 14)
					}
				}
			}
			m.errorMsg = ""
		}
		// Update viewport content to ensure correct scrolling
		m = m.updateHistoryViewport()
		// Scroll viewport to bottom to show new entry
		m.viewport.GotoBottom()
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
					m.expandedTable = eztable.FromPreview(m.history[m.selected].Preview).WithTargetWidth(m.width - 14)
				}
				m = m.updateHistoryViewport()
				m.viewport.GotoBottom()
			}
		}
		return m, nil

	case RerunResultMsg:
		m.loading = false
		if msg.Err == nil {
			m.popupEntry = msg.Entry
			m.popupResult = msg.Result
			// Create table without width constraints for scrolling
			m.popupTable = eztable.FromQueryResult(msg.Result, 0).
				Focused(true)
			m.updatePopupTable()
			m.showPopup = true
		} else {
			m.errorMsg = msg.Err.Error()
		}
		return m, nil

	case ExportCompleteMsg:
		if msg.Err != nil {
			m.errorMsg = fmt.Sprintf("Export failed: %v", msg.Err)
		} else {
			m = m.addSystemMessage(fmt.Sprintf("Exported to: %s", msg.Path))
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
	// switch msg.String() {
	keyStr := msg.String()

	if keyStr == "i" {
		m.mode = InsertMode
		m.editor.Focus()
		// Ensure cursor is at end?
		return m, textinput.Blink
	} else if keyStr == "k" || keyStr == "up" { // Move up (older)
		if m.selected > 0 {
			m.selected--
			m = m.ensureSelectionVisible()
		}
	} else if keyStr == "j" || keyStr == "down" { // Move down (newer)
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
	} else if keyStr == "g" {
		// For simplicity, single 'g' goes to top (gg would require state tracking)
		m.selected = 0
		m = m.ensureSelectionVisible()
	} else if keyStr == "G" {
		if len(m.history) > 0 {
			m.selected = len(m.history) - 1
			m = m.ensureSelectionVisible()
		}
	} else if keyStr == "enter" || keyStr == "space" { // Toggle expansion
		if m.selected >= 0 && m.selected < len(m.history) {
			entry := m.history[m.selected]
			if m.expandedID == entry.ID {
				m.expandedID = 0 // Collapse
				m.expandedTable = table.Model{}
			} else {
				m.expandedID = entry.ID // Expand
				if strings.Contains(entry.Preview, " | ") {
					m.expandedTable = eztable.FromPreview(entry.Preview).WithTargetWidth(m.width - 14)
				}
			}
			// Re-calculate visibility after expansion change
			m = m.ensureSelectionVisible()
		}
	} else if keyStr == "r" { // Re-run selected query
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
	} else if keyStr == "t" { // Toggle strict mode
		m.strictMode = !m.strictMode
		m.errorMsg = "" // Clear error if toggling
		return m, nil
	} else if keyStr == "e" { // Edit selected query
		if m.selected >= 0 && m.selected < len(m.history) {
			entry := m.history[m.selected]
			m.editor.SetValue(entry.Query)
			m.mode = InsertMode
			m.editor.Focus()
			return m, textinput.Blink
		}
	} else if keyStr == "x" { // Delete selected query
		if m.selected >= 0 && m.selected < len(m.history) {
			entry := m.history[m.selected]
			m.historyStore.Delete(entry.ID)
			m.history = append(m.history[:m.selected], m.history[m.selected+1:]...)
			if m.selected >= len(m.history) && m.selected > 0 {
				m.selected--
			}
			m = m.ensureSelectionVisible()
		}
	} else if keyStr == "y" { // Copy selected query to clipboard
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
	} else if keyStr == "tab" { // Toggle schema browser
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

// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Show profile selector if not connected
	if m.appState == StateSelectingProfile || m.appState == StateConnecting {
		view := m.profileSelector.View()
		if m.appState == StateConnecting {
			// Show connecting status
			connectingStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8BE9FD")).
				Bold(true)
			status := connectingStyle.Render("Connecting to " + m.profile.Name + "...")
			view = lipgloss.JoinVertical(lipgloss.Center, view, status)
		}
		if m.connectError != "" {
			errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
			view = lipgloss.JoinVertical(lipgloss.Center, view, errorStyle.Render("Error: "+m.connectError))
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, view)
	}

	// 1. Render Components
	inputWidth := m.width - 4
	inputView := InputStyle.Width(inputWidth).Render(m.highlightView(m.editor.View()))

	var suggestionsView string
	if m.autocompleting {
		suggestionsView = m.renderSuggestions()
	}

	statusBar := m.renderStatusBar()
	helpText := m.renderHelp()

	// 2. Calculate Content Height
	chromeHeight := lipgloss.Height(statusBar) + lipgloss.Height(helpText) + lipgloss.Height(inputView) + lipgloss.Height(suggestionsView)
	availableHeight := m.height - chromeHeight
	if availableHeight < 0 {
		availableHeight = 0
	}

	historyHeight := availableHeight
	if historyHeight < 0 {
		historyHeight = 0
	}

	// 3. Render History Content (Viewport)
	m.viewport.Height = historyHeight
	historyView := m.viewport.View()

	// 4. Final Layout
	main := lipgloss.JoinVertical(lipgloss.Left,
		historyView,
		suggestionsView,
		inputView,
		statusBar,
		helpText,
	)

	// Overlay popups if active
	if m.showPopup || m.confirming {
		main = m.renderPopupOverlay(main)
	}

	if m.schemaBrowser.IsVisible() || m.loadingTables { // Show if visible OR loading (for spinner)
		m.schemaBrowser = m.schemaBrowser.SetSize(m.width, m.height)
		browser := m.schemaBrowser.View()
		if browser != "" {
			// Use bubbletea-overlay to composite schema browser over main content
			main = overlay.Composite(browser, main, overlay.Center, overlay.Center, 0, 0)
		}
	}

	return main
}

func (m Model) renderStatusBar() string {
	var parts []string

	// 1. Mode
	modeStr := strings.ToUpper(string(m.mode))
	modeStyle := ModeStyle
	if m.mode == InsertMode {
		modeStyle = InsertModeStyle
	}
	parts = append(parts, modeStyle.Render(modeStr))

	// 2. Connection Info
	if m.profile != nil {
		profileInfo := ConnectionStyle.Render(fmt.Sprintf(" %s ", m.profile.Name))

		dbInfo := fmt.Sprintf(" %s@%s:%d/%s ", m.profile.User, m.profile.Host, m.profile.Port, m.profile.Database)
		if m.profile.Type == "sqlite" {
			dbInfo = fmt.Sprintf(" sqlite:%s ", m.profile.Database)
		}

		parts = append(parts, profileInfo+lipgloss.NewStyle().Background(lipgloss.Color("#282A36")).Foreground(lipgloss.Color("#F8F8F2")).Render(dbInfo))
	} else {
		parts = append(parts, ConnectionStyle.Render(" NO PROFILE "))
	}

	// 3. Strict Mode
	if m.strictMode {
		parts = append(parts, lipgloss.NewStyle().Background(lipgloss.Color("#FFB86C")).Foreground(lipgloss.Color("#000000")).Padding(0, 1).Bold(true).Render(" STRICT "))
	}

	// 4. Loading indicator
	if m.loading {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := spinner[int(time.Now().UnixMilli()/100)%len(spinner)]
		loadingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD")).Padding(0, 1)
		parts = append(parts, loadingStyle.Render(frame+" Running..."))
	}

	// 4. Error indicator
	if m.errorMsg != "" {
		errorStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FF5555")).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1)
		truncated := m.errorMsg
		if len(truncated) > 40 {
			truncated = truncated[:37] + "..."
		}
		parts = append(parts, errorStyle.Render("⚠ "+truncated))
	}

	content := lipgloss.JoinHorizontal(lipgloss.Left, parts...)
	return StatusBarStyle.Width(m.width).Render(content)
}

func (m Model) renderHelp() string {
	helpStyle := lipgloss.NewStyle().Foreground(textFaint)
	if m.mode == InsertMode {
		return helpStyle.Render("Ctrl+D: Execute | Esc: Visual mode | Ctrl+C: Quit")
	}
	return helpStyle.Render("i: Insert mode | k/j: Navigate | Enter: Expand | r: Re-run | e: Edit | x: Delete | Ctrl+C: Quit")
}

func (m Model) updateHistoryViewport() Model {
	// Status bar
	statusBar := m.renderStatusBar()
	// Help
	helpText := m.renderHelp()
	// Input area
	inputWidth := m.width - 4
	inputView := InputStyle.Width(inputWidth).Render(m.highlightView(m.editor.View()))
	// Suggestions
	var suggestionsView string
	if m.autocompleting {
		suggestionsView = m.renderSuggestions()
	}

	chromeHeight := lipgloss.Height(statusBar) + lipgloss.Height(helpText)
	availableHeight := m.height - chromeHeight
	if availableHeight < 0 {
		availableHeight = 0
	}

	historyHeight := availableHeight - lipgloss.Height(inputView) - lipgloss.Height(suggestionsView)
	if historyHeight < 0 {
		historyHeight = 0
	}

	m.viewport.Width = m.width
	m.viewport.Height = historyHeight
	m.viewport.SetContent(m.renderHistoryContent(historyHeight))
	return m
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

	// Table width should fit within terminal with room for popup borders and padding
	// Aggressive margin (30) to absolutely ensure no overflow
	maxTableWidth := m.width - 30
	if maxTableWidth < 40 {
		maxTableWidth = 40
	}

	m.popupTable = m.popupTable.
		WithPageSize(availableHeight).
		WithTargetWidth(maxTableWidth)
}

func (m Model) selectRowAsQuery() (Model, tea.Cmd) {
	if m.popupTable.HighlightedRow().Data == nil {
		return m, nil
	}

	// Extract table name from query
	// This is naive but works for simple "SELECT * FROM table" queries
	query := m.popupEntry.Query
	re := regexp.MustCompile(`(?i)from\s+["']?([a-zA-Z0-9_]+)["']?`)
	matches := re.FindStringSubmatch(query)
	if len(matches) < 2 {
		m.errorMsg = "Could not determine table name from query"
		return m, nil
	}
	tableName := matches[1]

	// Find PK columns
	// Try direct match first
	cols, ok := m.columns[tableName]
	if !ok {
		// Try case-insensitive match
		found := false
		for realName, c := range m.columns {
			if strings.EqualFold(realName, tableName) {
				tableName = realName
				cols = c
				found = true
				break
			}
		}
		if !found {
			// Log full debug info to file since UI might truncate it
			f, _ := os.OpenFile("debug_metadata.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if f != nil {
				fmt.Fprintf(f, "Timestamp: %s\nTable: %s\nLoaded Tables Count: %d\nColumns found for table: %v\nAll tables: %v\n\n",
					time.Now(), tableName, len(m.tables), m.columns[tableName], m.tables)
				f.Close()
			}

			m.errorMsg = fmt.Sprintf("Metadata missing for %s (Tabs: %d). See debug_metadata.log", tableName, len(m.tables))
			return m, nil
		}
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

// unwrapCellValue extracts the raw value from a bubble-table StyledCell if necessary
// Since StyledCell fields might be unexported or hard to access, we use a robust string check
func unwrapCellValue(val interface{}) interface{} {
	if _, ok := val.(table.StyledCell); ok {
		// Formatted struct looks like {Value Style ...}
		// e.g. {3 [38;2;...}
		s := fmt.Sprintf("%v", val)
		s = strings.TrimPrefix(s, "{")
		if idx := strings.Index(s, " "); idx != -1 {
			return s[:idx]
		}
		// Fallback: return the whole string if parsing fails, but cleaner
		return strings.TrimSuffix(s, "}")
	}
	return val
}

// reloadProfiles updates the profile selector with current config
func (m *Model) reloadProfiles() {
	profiles := make([]profileselector.Profile, len(m.config.Profiles))
	for i, cp := range m.config.Profiles {
		profiles[i] = profileselector.Profile{
			Name:        cp.Name,
			Type:        cp.Type,
			Host:        cp.Host,
			Port:        cp.Port,
			User:        cp.User,
			Database:    cp.Database,
			Password:    cp.Password,
			SSHHost:     cp.SSHHost,
			SSHPort:     cp.SSHPort,
			SSHUser:     cp.SSHUser,
			SSHKeyPath:  cp.SSHKeyPath,
			SSHPassword: cp.SSHPassword,
		}
	}
	m.profileSelector = m.profileSelector.SetProfiles(profiles)
}
