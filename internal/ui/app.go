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
	popupStack         *PopupStack // Stack of popup closers for layered closing
	showPopup          bool
	showActionPopup    bool
	showRowActionPopup bool // NEW: for showing detailed row actions
	showExportPopup    bool
	showHelpPopup      bool   // Show keyboard shortcuts
	showTemplatePopup  bool   // Show query template picker
	templateTable      string // Table name for template
	templateIdx        int    // Selected template index
	exportInput        textinput.Model
	exportTable        string // Table name being exported
	showImportPopup    bool   // Show import dialog
	importInput        textinput.Model
	importTable        string // Table name for import
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
	statusMsg    string // Success/info notifications (shown in status bar, not history)
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

	// Theme selector
	themeSelector ThemeSelector

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
	// Gray out the placeholder text (use TextFaint for darker appearance)
	ti.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Theme.Error))
	ti.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Theme.Error))

	// Initialize Table Filter Input
	tfi := textinput.New()
	tfi.Prompt = "/ "
	tfi.Placeholder = "Filter table..."
	tfi.CharLimit = 100
	tfi.Width = 30

	// Initialize Export Input
	ei := textinput.New()
	ei.Prompt = "Export to: "
	ei.Placeholder = "export.csv"
	ei.CharLimit = 256
	ei.Width = 40

	// Initialize Search Input
	si := textinput.New()
	si.Prompt = "/ "
	si.Placeholder = "Search history..."
	si.CharLimit = 100
	si.Width = 30

	// Initialize Import Input
	ii := textinput.New()
	ii.Prompt = "Import from: "
	ii.Placeholder = "path/to/file.csv"
	ii.CharLimit = 256
	ii.Width = 40

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
	ps := profileselector.New(selectorProfiles, cfg.Theme)

	// Determine initial state
	initialState := StateSelectingProfile
	if driver != nil && profile != nil {
		// Already connected (passed from main.go for backward compatibility)
		initialState = StateReady
	}

	// Initialize eztable global config
	eztable.Init(cfg.Theme, cfg.Keys)

	return Model{
		appState:        initialState,
		mode:            VisualMode,
		profile:         profile,
		config:          cfg,
		driver:          driver,
		historyStore:    store,
		popupStack:      NewPopupStack(),
		profileSelector: ps,
		schemaBrowser: schemabrowser.New().SetStyles(schemabrowser.Styles{
			Container:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(cfg.Theme.Highlight)).Padding(1, 2),
			Title:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(cfg.Theme.Accent)).MarginBottom(1),
			SectionTitle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(cfg.Theme.Highlight)).MarginTop(1).MarginBottom(1),
			Item:          lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Theme.TextPrimary)),
			ItemActive:    lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Theme.Success)).Bold(true),
			TableHeader:   lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Theme.Accent)).Bold(true).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(lipgloss.Color(cfg.Theme.BorderColor)),
			TableCell:     lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Theme.TextPrimary)),
			TableCellKey:  lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Theme.Success)),
			TableCellType: lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Theme.TextFaint)),
			Spinner:       lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Theme.Highlight)),
			TabActive:     lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Theme.Success)).Bold(true).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(lipgloss.Color(cfg.Theme.Success)).Padding(0, 1),
			TabInactive:   lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Theme.TextFaint)).Padding(0, 1),
		}),
		themeSelector:    NewThemeSelector(cfg),
		editor:           ti,
		viewport:         vp,
		history:          []history.HistoryEntry{},
		expandedID:       0,
		selected:         0,
		page:             0,
		columns:          make(map[string][]db.Column),
		tableFilterInput: tfi,
		exportInput:      ei,
		importInput:      ii,
		searchInput:      si,
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
				m.expandedTable = eztable.FromPreview(entry.Preview).WithTargetWidth(m.width - 14)
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
			var cmd tea.Cmd
			m.themeSelector, cmd = m.themeSelector.Update(msg)
			// If theme selector closed itself (e.g., via enter), pop from stack
			if !m.themeSelector.Visible() && m.popupStack.TopName() == "theme" {
				m.popupStack.Pop()
			}
			return m, cmd
		}

		// Profile selector handling
		if m.appState == StateSelectingProfile {
			var cmd tea.Cmd
			m.profileSelector, cmd = m.profileSelector.Update(msg)
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

			if matchKey(msg, m.config.Keys.Autocomplete) {
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
					m.popupTable = eztable.FromQueryResult(msg.Result, 0).
						Focused(true)
					m.updatePopupTable()
					m.openResultsPopup(msg.Entry, msg.Result)
					m.expandedID = msg.Entry.ID // Expand the history entry as well
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
					m.expandedTable = eztable.FromPreview(entry.Preview).WithTargetWidth(m.width - 14)
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
				Foreground(AccentColor()).
				Bold(true)
			status := connectingStyle.Render("Connecting to " + m.profile.Name + "...")
			view = lipgloss.JoinVertical(lipgloss.Center, view, status)
		}
		if m.connectError != "" {
			errorStyle := lipgloss.NewStyle().Foreground(ErrorColor())
			view = lipgloss.JoinVertical(lipgloss.Center, view, errorStyle.Render("Error: "+m.connectError))
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, view)
	}

	// 1. Calculate dynamic editor height based on content
	// Count lines in editor content
	editorContent := m.editor.Value()
	lineCount := strings.Count(editorContent, "\n") + 1

	// Min 3 lines, max half viewport
	minHeight := 3
	maxHeight := m.height / 2
	if maxHeight < minHeight {
		maxHeight = minHeight
	}

	editorHeight := lineCount
	if editorHeight < minHeight {
		editorHeight = minHeight
	}
	if editorHeight > maxHeight {
		editorHeight = maxHeight
	}

	m.editor.SetHeight(editorHeight)

	// 2. Render Components
	inputWidth := m.width - 4
	inputView := InputStyle.Width(inputWidth).Render(m.highlightView(m.editor.View()))

	statusBar := m.renderStatusBar()
	helpText := m.renderHelp()

	// 2. Calculate Content Height
	chromeHeight := lipgloss.Height(statusBar) + lipgloss.Height(helpText) + lipgloss.Height(inputView)
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
		inputView,
		statusBar,
		helpText,
	)

	// Overlay popups if active
	if m.showPopup || m.confirming {
		main = m.renderPopupOverlay(main)
	}

	// Help popup overlay
	if m.showHelpPopup {
		main = m.renderHelpPopup(main)
	}

	// Template popup overlay
	if m.showTemplatePopup {
		main = m.renderTemplatePopup(main)
	}

	// Import popup overlay
	if m.showImportPopup {
		main = m.renderImportPopup(main)
	}

	// Theme Selector Overlay
	if m.themeSelector.Visible() {
		themeView := m.themeSelector.View(m.width, m.height)
		main = overlay.Composite(themeView, main, overlay.Center, overlay.Center, 0, 0)
	}

	// 5. Suggestions Overlay
	if m.autocompleting && m.mode == InsertMode {
		suggestions := m.renderSuggestions()
		if suggestions != "" {
			// Calculate position relative to input area
			// Input input is at the bottom, above status bar and help text
			// Y = height - (StatusBar + Help + InputHeight) + 1 (border)
			// Actually, we want it *above* the input line
			// Input is at: m.height - (StatusBar=1 + Help=1 + InputHeight)
			// So bottom of suggestions should be at top of input?
			// Or below? Standard is usually below, but if we are at bottom of screen, maybe above?
			// Given the layout, the input bar is fixed at the bottom.
			// So suggestions should appear *above* the input box.

			// Let's try absolute positioning from bottom-left anchor
			// Input area is at bottom.
			// cursor Col gives X offset roughly.
			// Use lipgloss.Width to measure text before cursor
			cursorLine := m.editor.Line()
			lines := strings.Split(m.editor.Value(), "\n")
			var lineContent string
			if cursorLine < len(lines) {
				lineContent = lines[cursorLine]
			}
			x := lipgloss.Width(lineContent) + 2 // +2 for border/padding

			// Y: The overlay package coordinates are 0-indexed from top-left.
			// Input box bottom is at m.height - (StatusBar + Help)
			// Input box top is at m.height - (StatusBar + Help + EditorHeight + Border)
			// We want suggestions to appear *above* the current line of text in the editor?
			// Or below? Standard is usually below, but if we are at bottom of screen, maybe above?
			// Given the layout, the input bar is fixed at the bottom.
			// So suggestions should appear *above* the input box.

			inputHeight := lipgloss.Height(inputView)
			bottomOffset := 2 + inputHeight // Status(1) + Help(1) + Input

			// We want to position the suggestions box such that its bottom is at (Height - bottomOffset)
			// overlay.Composite places (0,0) of overlay at (x,y) of bg.
			// So y = m.height - bottomOffset - suggestionHeight

			suggestionHeight := lipgloss.Height(suggestions)
			y := m.height - bottomOffset - suggestionHeight

			if y < 0 {
				y = 0
			} // Clamp to top

			// X position
			// Constrain to width
			suggestionWidth := lipgloss.Width(suggestions)
			if x+suggestionWidth > m.width {
				x = m.width - suggestionWidth
			}

			// Use 0 (Left/Top) for absolute positioning with offsets
			// Assuming overlay.Left and overlay.Top constants exist, or 0/0 works
			// Existing code used overlay.Center, so assuming standardized naming
			// If Left/Top don't exist, we might need another approach, but let's try this.
			main = overlay.Composite(suggestions, main, 0, 0, x, y)
		}
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

		parts = append(parts, profileInfo+lipgloss.NewStyle().Background(CardBg()).Foreground(TextPrimary()).Render(dbInfo))
	} else {
		parts = append(parts, ConnectionStyle.Render(" NO PROFILE "))
	}

	// 3. Strict Mode
	if m.strictMode {
		parts = append(parts, lipgloss.NewStyle().Background(WarningColor()).Foreground(BgPrimary()).Padding(0, 1).Bold(true).Render(" STRICT "))
	}

	// 4. Loading indicator
	if m.loading {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := spinner[int(time.Now().UnixMilli()/100)%len(spinner)]
		loadingStyle := lipgloss.NewStyle().Foreground(AccentColor()).Padding(0, 1)
		parts = append(parts, loadingStyle.Render(frame+" Running..."))
	} else if m.loadingTables {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := spinner[int(time.Now().UnixMilli()/100)%len(spinner)]
		loadingStyle := lipgloss.NewStyle().Foreground(HighlightColor()).Padding(0, 1)
		parts = append(parts, loadingStyle.Render(frame+" Loading schema..."))
	}

	// 5. Status message (success/info)
	if m.statusMsg != "" {
		statusStyle := lipgloss.NewStyle().Background(SuccessColor()).Foreground(BgPrimary()).Padding(0, 1)
		parts = append(parts, statusStyle.Render("✓ "+m.statusMsg))
	}

	// 6. Error indicator
	if m.errorMsg != "" {
		errorStyle := lipgloss.NewStyle().Background(ErrorColor()).Foreground(TextPrimary()).Padding(0, 1)
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
	// Style for key hints - makes keys look like keyboard buttons
	keyStyle := lipgloss.NewStyle().
		Foreground(TextPrimary()).
		Background(CardBg()).
		Padding(0, 1).
		Bold(true)

	sepStyle := lipgloss.NewStyle().Foreground(TextFaint())
	descStyle := lipgloss.NewStyle().Foreground(TextSecondary())

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
		)
	}

	// Always show help and quit
	hints = append(hints,
		hint(key(keys.Help, "?"), "Help"),
		hint(key(keys.Quit, "ctrl+c"), "Quit"),
	)

	return strings.Join(hints, sep)
}

func (m Model) updateHistoryViewport() Model {
	// Calculate dynamic editor height
	editorContent := m.editor.Value()
	lineCount := strings.Count(editorContent, "\n") + 1
	minHeight := 3
	maxHeight := m.height / 2
	if maxHeight < minHeight {
		maxHeight = minHeight
	}
	editorHeight := lineCount
	if editorHeight < minHeight {
		editorHeight = minHeight
	}
	if editorHeight > maxHeight {
		editorHeight = maxHeight
	}
	m.editor.SetHeight(editorHeight)

	// Status bar
	statusBar := m.renderStatusBar()
	// Help
	helpText := m.renderHelp()
	// Input area
	inputWidth := m.width - 4
	inputView := InputStyle.Width(inputWidth).Render(m.highlightView(m.editor.View()))
	// Suggestions (only in insert mode)
	chromeHeight := lipgloss.Height(statusBar) + lipgloss.Height(helpText)
	availableHeight := m.height - chromeHeight
	if availableHeight < 0 {
		availableHeight = 0
	}

	historyHeight := availableHeight - lipgloss.Height(inputView)
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
			f, _ := os.OpenFile("debug_metadata.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
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

// Popup stack helper methods

// openHelpPopup opens the help popup and pushes closer to stack
func (m *Model) openHelpPopup() {
	if m.showHelpPopup {
		return
	}
	m.showHelpPopup = true
	m.popupStack.Push("help", func() bool {
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
	m.templateTable = tableName
	m.templateIdx = 0
	m.popupStack.Push("template", func() bool {
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
	m.popupStack.Push("results", func() bool {
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
	m.popupStack.Push("rowAction", func() bool {
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
	m.exportInput.SetValue(defaultName)
	m.exportInput.Focus()
	m.popupStack.Push("export", func() bool {
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
	m.importInput.SetValue("")
	m.importInput.Focus()
	m.importTable = tableName
	m.popupStack.Push("import", func() bool {
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
	m.popupStack.Push("action", func() bool {
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
	m.popupStack.Push("theme", func() bool {
		m.themeSelector = m.themeSelector.Hide()
		return true
	})
}

// closeTopPopup closes the topmost popup using the stack
func (m *Model) closeTopPopup() bool {
	if m.popupStack == nil {
		return false
	}
	return m.popupStack.CloseTop()
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
