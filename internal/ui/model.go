// internal/ui/model.go
// Root Model struct, constructor, and Init -- following superfile split pattern
package ui

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/evertras/bubble-table/table"
	"github.com/nhath/ezdb/internal/config"
	"github.com/nhath/ezdb/internal/db"
	"github.com/nhath/ezdb/internal/history"
	"github.com/nhath/ezdb/internal/ui/autocomplete"
	"github.com/nhath/ezdb/internal/ui/components/profileselector"
	"github.com/nhath/ezdb/internal/ui/components/schemabrowser"
	eztable "github.com/nhath/ezdb/internal/ui/components/table"
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
	suggestionDetails []string                      // Column types, function signatures
	suggestionTypes   []autocomplete.SuggestionType // Type indicators for suggestions
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
