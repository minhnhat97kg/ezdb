package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nhath/ezdb/internal/config"
	"github.com/nhath/ezdb/internal/db"
)

// connectToProfileCmd connects to the selected profile
func (m Model) connectToProfileCmd(profile *config.Profile) tea.Cmd {
	return func() tea.Msg {
		var driverType db.DriverType
		switch profile.Type {
		case "postgres":
			driverType = db.Postgres
		case "mysql":
			driverType = db.MySQL
		case "sqlite":
			driverType = db.SQLite
		default:
			return ProfileConnectedMsg{Err: db.WrapConnectionError(nil)}
		}

		driver, err := db.NewDriver(driverType)
		if err != nil {
			return ProfileConnectedMsg{Err: err}
		}

		// Use password from profile
		password := profile.Password
		if password == "" && profile.Type != "sqlite" {
			// Fallback to keyring for existing profiles not yet migrated to config
			keyringStore, err := config.NewKeyringStore()
			if err == nil {
				password, _ = keyringStore.GetPassword(profile.Name)
			}
		}

		params := db.ConnectParams{
			Host:     profile.Host,
			Port:     profile.Port,
			User:     profile.User,
			Password: password,
			Database: profile.Database,
		}

		if profile.SSHHost != "" {
			params.SSHConfig = &db.SSHConfig{
				Host:     profile.SSHHost,
				Port:     profile.SSHPort,
				User:     profile.SSHUser,
				Password: profile.SSHPassword,
				KeyPath:  profile.SSHKeyPath,
			}
		}

		if err := driver.Connect(params); err != nil {
			return ProfileConnectedMsg{Err: err}
		}

		return ProfileConnectedMsg{Driver: driver}
	}
}
