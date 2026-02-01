// internal/ui/handle_profile.go
// Handles profile-related messages: selection, save, management, connection.
package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nhath/ezdb/internal/config"
	"github.com/nhath/ezdb/internal/ui/components/profileselector"
	"github.com/nhath/ezdb/internal/ui/components/schemabrowser"
	"github.com/nhath/ezdb/internal/ui/icons"
)

// handleProfileSelected processes a profile selection from the profile selector.
func (m Model) handleProfileSelected(msg profileselector.SelectedMsg) (Model, tea.Cmd) {
	if msg.Index >= 0 && msg.Index < len(m.config.Profiles) {
		m.appState = StateConnecting
		selectedProfile := &m.config.Profiles[msg.Index]
		if msg.Password != "" {
			selectedProfile.Password = msg.Password
			m.config.Save()
		}
		m.profile = selectedProfile
		return m, m.connectToProfileCmd(selectedProfile)
	}
	return m, nil
}

// handleProfileSaved processes a saved profile (new or updated).
func (m Model) handleProfileSaved(msg profileselector.ProfileSavedMsg) (Model, tea.Cmd) {
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
		if err := m.config.AddProfile(p); err != nil {
			m.profileSelector = m.profileSelector.SetStatusMessage(fmt.Sprintf("Error adding profile: %v", err))
		} else {
			m.statusMsg = fmt.Sprintf("%s Added profile: %s", icons.IconSuccess, p.Name)
			m.reloadProfiles()
			m.profileSelector = m.profileSelector.ResetState()
		}
	} else {
		if err := m.config.UpdateProfile(msg.Profile.Name, p); err != nil {
			m.profileSelector = m.profileSelector.SetStatusMessage(fmt.Sprintf("Error updating profile: %v", err))
		} else {
			m.statusMsg = fmt.Sprintf("%s Updated profile: %s", icons.IconSuccess, p.Name)
			m.reloadProfiles()
			m.profileSelector = m.profileSelector.ResetState()
		}
	}
	return m, nil
}

// handleProfileManagement processes profile management actions (e.g. delete).
func (m Model) handleProfileManagement(msg profileselector.ManagementMsg) (Model, tea.Cmd) {
	switch msg.Action {
	case profileselector.ActionDelete:
		if msg.Profile != nil {
			if err := m.config.DeleteProfile(msg.Profile.Name); err != nil {
				m.profileSelector = m.profileSelector.SetStatusMessage(fmt.Sprintf("Error deleting profile: %v", err))
			} else {
				m.profileSelector = m.profileSelector.SetStatusMessage(fmt.Sprintf("%s Deleted profile: %s", icons.IconSuccess, msg.Profile.Name))
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
}

// handleProfileConnected processes the result of a connection attempt.
func (m Model) handleProfileConnected(msg ProfileConnectedMsg) (Model, tea.Cmd) {
	if msg.Err != nil {
		m.connectError = msg.Err.Error()
		m.appState = StateSelectingProfile
		return m, nil
	}
	m.driver = msg.Driver
	m.appState = StateReady
	m.connectError = ""
	m.loadingTables = true
	return m, tea.Batch(
		tea.ClearScreen,
		textarea.Blink,
		m.loadHistoryCmd(),
		schemabrowser.LoadSchemaCmd(m.driver),
	)
}

// reloadProfiles updates the profile selector with the current config profiles.
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
