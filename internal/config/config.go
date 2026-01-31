// internal/config/config.go
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
)

// Config represents the application configuration
type Config struct {
	DefaultProfile     string            `toml:"default_profile"`
	PageSize           int               `toml:"page_size"`
	HistoryPreviewRows int               `toml:"history_preview_rows"`
	Pager              string            `toml:"pager"`
	Profiles           []Profile         `toml:"profiles"`
	Commands           map[string]string `toml:"commands"`
	Theme              Theme             `toml:"theme_colors"`
	Keys               KeyMap            `toml:"keys"`
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
}

// KeyMap defines key bindings
type KeyMap struct {
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

// DefaultConfig returns a config with default values
func DefaultConfig() *Config {
	return &Config{
		DefaultProfile:     "",
		PageSize:           100,
		HistoryPreviewRows: 3,
		Pager:              "",
		Profiles:           []Profile{},
		Commands:           make(map[string]string),
		Theme: Theme{
			// Nord Theme Defaults
			TextPrimary:   "#D8DEE9",
			TextSecondary: "#81A1C1",
			TextFaint:     "#4C566A",
			Accent:        "#88C0D0",
			Success:       "#A3BE8C",
			Error:         "#BF616A",
			Highlight:     "#8FBCBB",
			Warning:       "#D08770",
			BgPrimary:     "#2E3440",
			BgSecondary:   "#3B4252",
			CardBg:        "#434C5E",
		},
		Keys: KeyMap{
			Execute:     []string{"ctrl+d"},
			Exit:        []string{"esc", "ctrl+c", "q"},
			Filter:      []string{"/"},
			NextPage:    []string{"n", "pgdown"},
			PrevPage:    []string{"b", "pgup"},
			ScrollLeft:  []string{"h", "left"},
			ScrollRight: []string{"l", "right"},
			RowAction:   []string{"enter", "space"},
			Export:      []string{"e"},
			Sort:        []string{"s"},
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
	if len(cfg.Keys.Execute) == 0 {
		cfg.Keys = defaults.Keys
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
