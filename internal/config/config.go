// internal/config/config.go
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
)

// QueryTemplate defines a predefined query with <table> placeholder
type QueryTemplate struct {
	Name  string `toml:"name"`
	Query string `toml:"query"`
}

// Config represents the application configuration
type Config struct {
	DefaultProfile     string          `toml:"default_profile"`
	PageSize           int             `toml:"page_size"`
	HistoryPreviewRows int             `toml:"history_preview_rows"`
	Pager              string          `toml:"pager"`
	Profiles           []Profile       `toml:"profiles"`
	ThemeName          string          `toml:"theme_name"`
	Theme              Theme           `toml:"theme_colors"`
	Keys               KeyMap          `toml:"keys"`
	QueryTemplates     []QueryTemplate `toml:"query_templates"`
}

// Theme defines the color palette
type Theme struct {
	TextPrimary   string `toml:"text_primary"`
	TextSecondary string `toml:"text_secondary"`
	TextFaint     string `toml:"text_faint"`
	Accent        string `toml:"accent"`
	Success       string `toml:"success"`
	Error         string `toml:"error"`
	Highlight     string `toml:"highlight"`
	Warning       string `toml:"warning"`
	BgPrimary     string `toml:"bg_primary"`
	BgSecondary   string `toml:"bg_secondary"`
	CardBg        string `toml:"card_bg"`
	PopupBg       string `toml:"popup_bg"`
	BorderColor   string `toml:"border_color"`
	SelectedBg    string `toml:"selected_bg"`
}

// KeyMap defines key bindings
type KeyMap struct {
	// Existing keys
	Execute     []string `toml:"execute"`
	Exit        []string `toml:"exit"`
	Filter      []string `toml:"filter"`
	NextPage    []string `toml:"next_page"`
	PrevPage    []string `toml:"prev_page"`
	ScrollLeft  []string `toml:"scroll_left"`
	ScrollRight []string `toml:"scroll_right"`
	RowAction   []string `toml:"row_action"`
	Export      []string `toml:"export"`
	Sort        []string `toml:"sort"`
	ToggleTheme []string `toml:"toggle_theme"`
	// Navigation keys
	InsertMode   []string `toml:"insert_mode"`
	MoveUp       []string `toml:"move_up"`
	MoveDown     []string `toml:"move_down"`
	GoTop        []string `toml:"go_top"`
	GoBottom     []string `toml:"go_bottom"`
	ToggleExpand []string `toml:"toggle_expand"`
	// Action keys
	Rerun        []string `toml:"rerun"`
	Edit         []string `toml:"edit"`
	Delete       []string `toml:"delete"`
	Copy         []string `toml:"copy"`
	ToggleStrict []string `toml:"toggle_strict"`
	ToggleSchema []string `toml:"toggle_schema"`
	ShowProfiles []string `toml:"show_profiles"`
	Help         []string `toml:"help"`
	Explain      []string `toml:"explain"`
	// Modifier keys
	Autocomplete []string `toml:"autocomplete"`
	Undo         []string `toml:"undo"`
	Redo         []string `toml:"redo"`
	Quit         []string `toml:"quit"`
}

// Profile represents a database connection profile
type Profile struct {
	Name     string `toml:"name"`
	Type     string `toml:"type"` // postgres, mysql, sqlite
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Database string `toml:"database"`
	// Password is kept in memory for usage
	Password string `toml:"-"`
	// EncryptedPassword is the one persisted in the config file
	EncryptedPassword string `toml:"password"`

	// SSH Tunnel Configuration
	SSHHost     string `toml:"ssh_host,omitempty"`
	SSHPort     int    `toml:"ssh_port,omitempty"`
	SSHUser     string `toml:"ssh_user,omitempty"`
	SSHPassword string `toml:"-,omitempty"` // In-memory
	SSHKeyPath  string `toml:"ssh_key_path,omitempty"`

	// EncryptedSSHPassword persisted in config
	EncryptedSSHPassword string `toml:"ssh_password,omitempty"`
}

const defaultHistoryFile = "history.txt"

// DefaultConfig returns a config with default values
func DefaultConfig() *Config {
	return &Config{
		DefaultProfile:     "",
		PageSize:           100,
		HistoryPreviewRows: 3,
		Pager:              "",
		Profiles:           []Profile{},
		ThemeName:          "JetBrains Darcula",
		Theme: Theme{
			// JetBrains Darcula Theme
			TextPrimary:   "#A9B7C6", // Default foreground
			TextSecondary: "#6897BB", // Blue (numbers, etc)
			TextFaint:     "#6272A4", // Comments/faint
			Accent:        "#CC7832", // Orange (keywords)
			Success:       "#6A8759", // Green (strings)
			Error:         "#FF6B68", // Red
			Highlight:     "#9876AA", // Purple (types)
			Warning:       "#FFC66D", // Yellow
			BgPrimary:     "#2B2B2B", // Background
			BgSecondary:   "#3C3F41", // Selection/secondary
			CardBg:        "#313335", // Tool window
			PopupBg:       "#1E1E1E", // Popup background
			BorderColor:   "#5E5E5E", // Borders
			SelectedBg:    "#214283", // Selection
		},
		Keys: KeyMap{
			// Existing keys
			Execute:     []string{"ctrl+d"},
			Exit:        []string{"esc", "q"},
			Filter:      []string{"/"},
			NextPage:    []string{"n", "pgdown"},
			PrevPage:    []string{"b", "pgup"},
			ScrollLeft:  []string{"h", "left"},
			ScrollRight: []string{"l", "right"},
			RowAction:   []string{"enter", "space"},
			Export:      []string{"ctrl+e"},
			Sort:        []string{"s"},
			ToggleTheme: []string{"t"},
			// Navigation keys
			InsertMode:   []string{"i"},
			MoveUp:       []string{"k", "up"},
			MoveDown:     []string{"j", "down"},
			GoTop:        []string{"g"},
			GoBottom:     []string{"G"},
			ToggleExpand: []string{"enter", "space"},
			// Action keys
			Rerun:        []string{"r"},
			Edit:         []string{"e"},
			Delete:       []string{"x"},
			Copy:         []string{"y"},
			ToggleStrict: []string{"m"},
			ToggleSchema: []string{"tab"},
			ShowProfiles: []string{"P"},
			Help:         []string{"?"},
			Explain:      []string{"X"},
			// Modifier keys
			Autocomplete: []string{"ctrl+space"},
			Undo:         []string{"ctrl+z"},
			Redo:         []string{"ctrl+y"},
			Quit:         []string{"ctrl+c"},
		},
		QueryTemplates: []QueryTemplate{
			{Name: "SELECT 10", Query: "SELECT * FROM <table> LIMIT 10"},
			{Name: "SELECT 100", Query: "SELECT * FROM <table> LIMIT 100"},
			{Name: "COUNT", Query: "SELECT COUNT(*) FROM <table>"},
			{Name: "DESCRIBE", Query: "DESCRIBE <table>"},
		},
	}
}

// ConfigPath returns the XDG-compliant config file path
func ConfigPath() (string, error) {
	return xdg.ConfigFile("ezdb/config.toml")
}

// Load loads the config from disk or creates default
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// First run: create default
		cfg := DefaultConfig()
		if err := cfg.Save(); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}

	// Populate defaults for missing fields (migration)
	defaults := DefaultConfig()
	updated := false

	if cfg.Theme.TextPrimary == "" {
		cfg.Theme = defaults.Theme
		updated = true
	}
	// Migrate new theme colors
	if cfg.Theme.PopupBg == "" {
		cfg.Theme.PopupBg = defaults.Theme.PopupBg
		cfg.Theme.BorderColor = defaults.Theme.BorderColor
		cfg.Theme.SelectedBg = defaults.Theme.SelectedBg
		updated = true
	}

	// Migrate existing keys
	if len(cfg.Keys.Execute) == 0 {
		cfg.Keys.Execute = defaults.Keys.Execute
		updated = true
	}
	if len(cfg.Keys.Exit) == 0 {
		cfg.Keys.Exit = defaults.Keys.Exit
		updated = true
	}
	if len(cfg.Keys.RowAction) == 0 {
		cfg.Keys.RowAction = defaults.Keys.RowAction
		updated = true
	}
	if len(cfg.Keys.Filter) == 0 {
		cfg.Keys.Filter = defaults.Keys.Filter
		updated = true
	}
	if len(cfg.Keys.Export) == 0 {
		cfg.Keys.Export = defaults.Keys.Export
		updated = true
	}
	if len(cfg.Keys.NextPage) == 0 {
		cfg.Keys.NextPage = defaults.Keys.NextPage
		updated = true
	}
	if len(cfg.Keys.PrevPage) == 0 {
		cfg.Keys.PrevPage = defaults.Keys.PrevPage
		updated = true
	}
	if len(cfg.Keys.ScrollLeft) == 0 {
		cfg.Keys.ScrollLeft = defaults.Keys.ScrollLeft
		updated = true
	}
	if len(cfg.Keys.ScrollRight) == 0 {
		cfg.Keys.ScrollRight = defaults.Keys.ScrollRight
		updated = true
	}
	if len(cfg.Keys.Sort) == 0 {
		cfg.Keys.Sort = defaults.Keys.Sort
		updated = true
	}
	if len(cfg.Keys.ToggleTheme) == 0 {
		cfg.Keys.ToggleTheme = defaults.Keys.ToggleTheme
		updated = true
	}
	// Migrate new keys individually
	if len(cfg.Keys.InsertMode) == 0 {
		cfg.Keys.InsertMode = defaults.Keys.InsertMode
		updated = true
	}
	if len(cfg.Keys.MoveUp) == 0 {
		cfg.Keys.MoveUp = defaults.Keys.MoveUp
		updated = true
	}
	if len(cfg.Keys.MoveDown) == 0 {
		cfg.Keys.MoveDown = defaults.Keys.MoveDown
		updated = true
	}
	if len(cfg.Keys.GoTop) == 0 {
		cfg.Keys.GoTop = defaults.Keys.GoTop
		updated = true
	}
	if len(cfg.Keys.GoBottom) == 0 {
		cfg.Keys.GoBottom = defaults.Keys.GoBottom
		updated = true
	}
	if len(cfg.Keys.ToggleExpand) == 0 {
		cfg.Keys.ToggleExpand = defaults.Keys.ToggleExpand
		updated = true
	}
	if len(cfg.Keys.Rerun) == 0 {
		cfg.Keys.Rerun = defaults.Keys.Rerun
		updated = true
	}
	if len(cfg.Keys.Edit) == 0 {
		cfg.Keys.Edit = defaults.Keys.Edit
		updated = true
	}
	if len(cfg.Keys.Delete) == 0 {
		cfg.Keys.Delete = defaults.Keys.Delete
		updated = true
	}
	if len(cfg.Keys.Copy) == 0 {
		cfg.Keys.Copy = defaults.Keys.Copy
		updated = true
	}
	if len(cfg.Keys.ToggleStrict) == 0 {
		cfg.Keys.ToggleStrict = defaults.Keys.ToggleStrict
		updated = true
	}
	if len(cfg.Keys.ToggleSchema) == 0 {
		cfg.Keys.ToggleSchema = defaults.Keys.ToggleSchema
		updated = true
	}
	if len(cfg.Keys.ShowProfiles) == 0 {
		cfg.Keys.ShowProfiles = defaults.Keys.ShowProfiles
		updated = true
	}
	if len(cfg.Keys.Autocomplete) == 0 {
		cfg.Keys.Autocomplete = defaults.Keys.Autocomplete
		updated = true
	}
	if len(cfg.Keys.Undo) == 0 {
		cfg.Keys.Undo = defaults.Keys.Undo
		updated = true
	}
	if len(cfg.Keys.Redo) == 0 {
		cfg.Keys.Redo = defaults.Keys.Redo
		updated = true
	}
	if len(cfg.Keys.Quit) == 0 {
		cfg.Keys.Quit = defaults.Keys.Quit
		updated = true
	}
	if len(cfg.Keys.Help) == 0 {
		cfg.Keys.Help = defaults.Keys.Help
		updated = true
	}
	if len(cfg.Keys.Explain) == 0 {
		cfg.Keys.Explain = defaults.Keys.Explain
		updated = true
	}

	if len(cfg.QueryTemplates) == 0 {
		cfg.QueryTemplates = []QueryTemplate{
			{Name: "SELECT 100", Query: "SELECT * FROM <table> LIMIT 100"},
			{Name: "COUNT", Query: "SELECT COUNT(*) FROM <table>"},
			{Name: "DESCRIBE", Query: "DESCRIBE <table>"},
			{Name: "INSERT DEFAULT", Query: "INSERT INTO <table> DEFAULT VALUES"},
		}
		updated = true
	}

	if updated {
		// Save updated config to persist defaults so user can see/edit them
		if err := cfg.Save(); err != nil {
			// Proceed with in-memory defaults even if save fails
			// Maybe log warning?
		}
	}

	// Decrypt passwords
	key, err := GetMasterKey()
	if err == nil {
		for i := range cfg.Profiles {
			if cfg.Profiles[i].EncryptedPassword != "" {
				decrypted, err := Decrypt(cfg.Profiles[i].EncryptedPassword, key)
				if err == nil {
					cfg.Profiles[i].Password = decrypted
				}
			}
			if cfg.Profiles[i].EncryptedSSHPassword != "" {
				decrypted, err := Decrypt(cfg.Profiles[i].EncryptedSSHPassword, key)
				if err == nil {
					cfg.Profiles[i].SSHPassword = decrypted
				}
			}
		}
	}

	return &cfg, nil
}

// Save writes the config to disk
func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	// Ensure directory exists with secure permissions
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Create/truncate file with secure permissions (owner read/write only)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	// Encrypt passwords before saving
	key, err := GetMasterKey()
	if err == nil {
		for i := range c.Profiles {
			if c.Profiles[i].Password != "" {
				encrypted, err := Encrypt(c.Profiles[i].Password, key)
				if err == nil {
					c.Profiles[i].EncryptedPassword = encrypted
				}
			}
			if c.Profiles[i].SSHPassword != "" {
				encrypted, err := Encrypt(c.Profiles[i].SSHPassword, key)
				if err == nil {
					c.Profiles[i].EncryptedSSHPassword = encrypted
				}
			}
		}
	}

	return toml.NewEncoder(f).Encode(c)
}
