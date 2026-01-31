// internal/ui/app.go
package ui

import (
	"fmt"
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
	showPopup       bool
	showActionPopup bool
	popupEntry      *history.HistoryEntry
	popupResult *db.QueryResult
	popupTable  table.Model

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

	vp := viewport.New(80, 10)

	// Convert config profiles to selector profiles
	selectorProfiles := make([]profileselector.Profile, len(cfg.Profiles))
	for i, p := range cfg.Profiles {
		selectorProfiles[i] = profileselector.Profile{
			Name:     p.Name,
			Type:     p.Type,
			Host:     p.Host,
			Database: p.Database,
			Password: p.Password,
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
		appState:        initialState,
		mode:            VisualMode,
		profile:         profile,
		config:          cfg,
		driver:          driver,
		historyStore:    store,
		profileSelector: ps,
		schemaBrowser:   schemabrowser.New(),
		editor:          ti,
		viewport:        vp,
		history:         []history.HistoryEntry{},
		expandedID:      0,
		selected:        0,
		page:            0,
		columns:         make(map[string][]db.Column),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	if m.appState == StateReady {
		return tea.Batch(
			textarea.Blink,
			m.loadHistoryCmd(),
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
		p, err := config.ParseDSN(msg.Profile.Name, msg.Profile.Database)
		if err != nil {
			m.profileSelector = m.profileSelector.SetStatusMessage(fmt.Sprintf("Error parsing connection string: %v", err))
			return m, nil
		}

		if msg.IsNew {
			// Add new profile
			if err := m.config.AddProfile(p); err != nil {
				m.profileSelector = m.profileSelector.SetStatusMessage(fmt.Sprintf("Error adding profile: %v", err))
			} else {
				m.profileSelector = m.profileSelector.SetStatusMessage(fmt.Sprintf("✓ Added profile: %s", p.Name))
				// Reload profile selector with updated list
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
		} else {
			// Update existing profile
			if err := m.config.UpdateProfile(msg.Profile.Name, p); err != nil {
				m.profileSelector = m.profileSelector.SetStatusMessage(fmt.Sprintf("Error updating profile: %v", err))
			} else {
				m.profileSelector = m.profileSelector.SetStatusMessage(fmt.Sprintf("✓ Updated profile: %s", p.Name))
				// Reload profile selector with updated list
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
			if m.showActionPopup {
				switch msg.String() {
				case "q", "esc":
					m.showActionPopup = false
					return m, nil
				}
				// TODO: Handle action selection
				return m, nil
			}

			switch msg.String() {
			case "q", "esc":
				m.showPopup = false
				return m, nil
			case "a":
				m.showActionPopup = true
				return m, nil
			}

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

			switch msg.String() {
			case "ctrl+space":
				m.autocompleting = true
				m = m.updateSuggestions()
				if len(m.tables) == 0 && !m.loadingTables {
					m.loadingTables = true
					cmd = m.fetchTablesCmd()
					cmds = append(cmds, cmd)
				}
				return m, tea.Batch(cmds...)

			case "ctrl+d": // Execute query
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

			case "ctrl+z": // Undo
				if len(m.undoStack) > 0 {
					// Push current to redo
					m.redoStack = append(m.redoStack, m.editor.Value())
					// Pop from undo
					prev := m.undoStack[len(m.undoStack)-1]
					m.undoStack = m.undoStack[:len(m.undoStack)-1]
					m.editor.SetValue(prev)
				}
				return m, nil

			case "ctrl+y": // Redo
				if len(m.redoStack) > 0 {
					// Push current to undo
					m.undoStack = append(m.undoStack, m.editor.Value())
					// Pop from redo
					next := m.redoStack[len(m.redoStack)-1]
					m.redoStack = m.redoStack[:len(m.redoStack)-1]
					m.editor.SetValue(next)
				}
				return m, nil

			case "esc": // Switch to visual mode
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
				m.loadingTables = true
				return m, m.fetchTablesCmd()
			}
		}
		return m, nil

	case TablesFetchedMsg:
		m.tables = msg.Tables
		m.columns = msg.Columns
		m.loadingTables = false
		if m.autocompleting {
			m = m.updateSuggestions()
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
					m.popupTable = eztable.FromQueryResult(msg.Result).
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
			m.popupTable = eztable.FromQueryResult(msg.Result).
				Focused(true)
			m.updatePopupTable()
			m.showPopup = true
		} else {
			m.errorMsg = msg.Err.Error()
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
	switch msg.String() {
	case "i": // Switch to insert mode
		m.mode = InsertMode
		m.editor.Focus()
		// Ensure cursor is at end?
		return m, textinput.Blink
	case "k", "up": // Move up (older)
		if m.selected > 0 {
			m.selected--
			m = m.ensureSelectionVisible()
		}
	case "j", "down": // Move down (newer)
		if m.selected < len(m.history)-1 {
			m.selected++
			m = m.ensureSelectionVisible()
		}
	case "h", "left": // Scroll left
		if m.expandedID != 0 {
			m.expandedTable = m.expandedTable.ScrollLeft()
		}
	case "l", "right": // Scroll right
		if m.expandedID != 0 {
			m.expandedTable = m.expandedTable.ScrollRight()
		}
	case "g": // gg = go to top (wait for second g)
		// For simplicity, single 'g' goes to top (gg would require state tracking)
		m.selected = 0
		m = m.ensureSelectionVisible()
	case "G": // Go to bottom
		if len(m.history) > 0 {
			m.selected = len(m.history) - 1
			m = m.ensureSelectionVisible()
		}
	case "enter", "space": // Toggle expansion
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
	case "r": // Re-run selected query
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
	case "t": // Toggle strict mode
		m.strictMode = !m.strictMode
		m.errorMsg = "" // Clear error if toggling
		return m, nil
	case "e": // Edit selected query
		if m.selected >= 0 && m.selected < len(m.history) {
			entry := m.history[m.selected]
			m.editor.SetValue(entry.Query)
			m.mode = InsertMode
			m.editor.Focus()
			return m, textinput.Blink
		}
	case "x": // Delete selected query
		if m.selected >= 0 && m.selected < len(m.history) {
			entry := m.history[m.selected]
			m.historyStore.Delete(entry.ID)
			m.history = append(m.history[:m.selected], m.history[m.selected+1:]...)
			if m.selected >= len(m.history) && m.selected > 0 {
				m.selected--
			}
			m = m.ensureSelectionVisible()
		}
	case "y": // Copy selected query to clipboard
		if m.selected >= 0 && m.selected < len(m.history) {
			entry := m.history[m.selected]
			return m, m.copyToClipboardCmd(entry.Query)
		}
	case "/": // Enter search mode
		m.searching = true
		m.searchQuery = ""
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		return m, textinput.Blink
	case "tab": // Toggle schema browser
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
	// Match PopupStyle width logic from popup.go
	popupWidth := m.width - 4
	if popupWidth > 120 {
		popupWidth = 120
	}
	
	// Table width = popup width - padding (2*2) - border (2) - safety (6) = -12
	tableWidth := popupWidth - 12
	if tableWidth < 10 {
		tableWidth = 10
	}

	// Calculate height
	// Window Height
	// - Popup Margin (4)
	// - Popup Border/Padding (4)
	// - Header Text (3 lines)
	// - Footer Text (3 lines)
	// - Table Chrome (~6 lines)
	// - Safety Buffer (8 lines)
	// Total deduction: ~28
	availableHeight := m.height - 28
	if availableHeight < 3 {
		availableHeight = 3
	}

	m.popupTable = m.popupTable.
		WithTargetWidth(tableWidth).
		WithPageSize(availableHeight)
}
