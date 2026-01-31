// Package profileselector provides a profile selection popup for app startup.
package profileselector

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nhath/ezdb/internal/config"
)

// State represents the component state
type State int

const (
	StateSelectingProfile State = iota
	StateEnteringPassword
	StateManagementMenu
	StateAddingProfile
	StateEditingProfile
)

// Profile represents a selectable profile
type Profile struct {
	Name     string
	Type     string // postgres, mysql, sqlite
	Host     string
	Port     int
	User     string
	Database string
	Password string

	// SSH tunneling
	SSHHost     string
	SSHPort     int
	SSHUser     string
	SSHPassword string
	SSHKeyPath  string
}

// BuildDSN builds a URI connection string from profile for display
func (p *Profile) BuildDSN(password string) string {
	if password == "" {
		password = p.Password
	}
	switch p.Type {
	case "postgres":
		if password != "" {
			return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", p.User, password, p.Host, p.Port, p.Database)
		}
		return fmt.Sprintf("postgres://%s@%s:%d/%s", p.User, p.Host, p.Port, p.Database)
	case "mysql":
		if password != "" {
			return fmt.Sprintf("mysql://%s:%s@%s:%d/%s", p.User, password, p.Host, p.Port, p.Database)
		}
		return fmt.Sprintf("mysql://%s@%s:%d/%s", p.User, p.Host, p.Port, p.Database)
	case "sqlite":
		return fmt.Sprintf("sqlite://%s", p.Database)
	default:
		return p.Database // Return as-is for unknown types
	}
}

// Styles for the selector
type Styles struct {
	Box           lipgloss.Style
	Title         lipgloss.Style
	Item          lipgloss.Style
	Selected      lipgloss.Style
	Hint          lipgloss.Style
	PasswordLabel lipgloss.Style
}

// DefaultStyles returns the default styling using Nord palette
func DefaultStyles(theme config.Theme) Styles {
	return Styles{
		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Highlight)).
			Padding(1, 2),
		// No Background - transparent, inherits from terminal
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.TextPrimary)).
			MarginBottom(1),
		Item: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextPrimary)).
			PaddingLeft(2),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.BgPrimary)).
			Background(lipgloss.Color(theme.Success)).
			Bold(true).
			PaddingLeft(2),
		Hint: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextFaint)).
			Italic(true).
			MarginTop(1),
		PasswordLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Accent)).
			Bold(true),
	}
}

// SelectedMsg is sent when a profile is selected (with optional password)
type SelectedMsg struct {
	Index    int
	Password string
}

// ManagementAction represents profile management actions
type ManagementAction int

const (
	ActionAdd ManagementAction = iota
	ActionEdit
	ActionDelete
	ActionCancel
)

// ManagementMsg is sent when a management action is requested
type ManagementMsg struct {
	Action  ManagementAction
	Profile *Profile // Only set for Edit/Delete
}

// StatusMsg sets a temporary status message
type StatusMsg struct {
	Message string
}

// ProfileSavedMsg is sent when a profile is added or updated
type ProfileSavedMsg struct {
	Profile Profile
	IsNew   bool // true for add, false for edit
}

// Model represents the selector state
type Model struct {
	profiles      []Profile
	selected      int
	menuSelected  int
	state         State
	passwordInput textinput.Model // For connection password prompt

	// Form inputs
	nameInput         textinput.Model
	typeInput         textinput.Model // simple text for now, could be list
	hostInput         textinput.Model
	portInput         textinput.Model
	userInput         textinput.Model
	databaseInput     textinput.Model // serves as path for sqlite
	passwordFormInput textinput.Model // For saving password

	// SSH Form inputs
	sshHostInput     textinput.Model
	sshPortInput     textinput.Model
	sshUserInput     textinput.Model
	sshKeyInput      textinput.Model
	sshPasswordInput textinput.Model

	formFocused    int      // Index of focused field
	editingProfile *Profile // Profile being edited (nil for add)
	width          int
	height         int
	styles         Styles
	showManagement bool   // Flag to show management actions
	statusMessage  string // Temporary status message to display
}

// New creates a new selector
func New(profiles []Profile, theme config.Theme) Model {
	// Password input (prompt)
	ti := textinput.New()
	ti.Placeholder = "Enter password..."
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Width = 50

	// Helper to create inputs
	newInput := func(placeholder string, width int) textinput.Model {
		t := textinput.New()
		t.Placeholder = placeholder
		t.Width = width
		return t
	}

	return Model{
		profiles:      profiles,
		selected:      0,
		state:         StateSelectingProfile,
		passwordInput: ti,

		nameInput:         newInput("Profile Name", 50),
		typeInput:         newInput("Type (postgres, mysql, sqlite)", 30),
		hostInput:         newInput("Host (localhost)", 40),
		portInput:         newInput("Port (5432)", 10),
		userInput:         newInput("User", 30),
		databaseInput:     newInput("Database / Path", 40),
		passwordFormInput: newInput("Password (optional)", 30),

		sshHostInput:     newInput("SSH Host", 40),
		sshPortInput:     newInput("SSH Port (22)", 10),
		sshUserInput:     newInput("SSH User", 30),
		sshKeyInput:      newInput("SSH Key Path (~/.ssh/id_rsa)", 50),
		sshPasswordInput: newInput("SSH Password (optional)", 30),

		formFocused: 0,
		styles:      DefaultStyles(theme),
	}
}

// SetProfiles updates the profile list
func (m Model) SetProfiles(profiles []Profile) Model {
	m.profiles = profiles
	if m.selected >= len(profiles) {
		m.selected = 0
	}
	return m
}

// SetSize sets the screen size for centering
func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h
	return m
}

// SetStyles sets custom styles
func (m Model) SetStyles(s Styles) Model {
	m.styles = s
	return m
}

// SetStatusMessage sets a temporary status message
func (m Model) SetStatusMessage(msg string) Model {
	m.statusMessage = msg
	return m
}

// Selected returns the currently selected index
func (m Model) Selected() int {
	return m.selected
}

// SelectedProfile returns the selected profile
func (m Model) SelectedProfile() *Profile {
	if m.selected >= 0 && m.selected < len(m.profiles) {
		return &m.profiles[m.selected]
	}
	return nil
}

// NeedsPassword returns true if the selected profile needs a password
func (m Model) NeedsPassword() bool {
	p := m.SelectedProfile()
	return p != nil && p.Type != "sqlite" && p.Password == ""
}

// Update handles input
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle form input states (Add/Edit profile)
		if m.state == StateAddingProfile || m.state == StateEditingProfile {
			switch msg.String() {
			case "tab":
				// Cycle next
				m.blurField(m.formFocused)
				m.formFocused = (m.formFocused + 1) % 12 // 12 inputs
				m.focusField(m.formFocused)
				return m, nil
			case "shift+tab":
				// Cycle prev
				m.blurField(m.formFocused)
				m.formFocused--
				if m.formFocused < 0 {
					m.formFocused = 11
				}
				m.focusField(m.formFocused)
				return m, nil
			case "enter":
				// Submit form
				name := strings.TrimSpace(m.nameInput.Value())
				dbType := strings.TrimSpace(m.typeInput.Value())
				host := strings.TrimSpace(m.hostInput.Value())
				portStr := strings.TrimSpace(m.portInput.Value())
				user := strings.TrimSpace(m.userInput.Value())
				database := strings.TrimSpace(m.databaseInput.Value())
				password := strings.TrimSpace(m.passwordFormInput.Value())

				sshHost := strings.TrimSpace(m.sshHostInput.Value())
				sshPortStr := strings.TrimSpace(m.sshPortInput.Value())
				sshUser := strings.TrimSpace(m.sshUserInput.Value())
				sshKey := strings.TrimSpace(m.sshKeyInput.Value())
				sshPass := strings.TrimSpace(m.sshPasswordInput.Value())

				// Basic validtion
				if name == "" {
					m.statusMessage = "Profile name is required"
					return m, nil
				}
				if dbType != "sqlite" && host == "" {
					m.statusMessage = "Host is required for non-sqlite"
					return m, nil
				}

				port := 0
				if portStr != "" {
					fmt.Sscanf(portStr, "%d", &port)
				}
				sshPort := 22
				if sshPortStr != "" {
					fmt.Sscanf(sshPortStr, "%d", &sshPort)
				}

				// Return ProfileSavedMsg to app for processing
				return m, func() tea.Msg {
					return ProfileSavedMsg{
						Profile: Profile{
							Name:        name,
							Type:        dbType,
							Host:        host,
							Port:        port,
							User:        user,
							Database:    database,
							Password:    password,
							SSHHost:     sshHost,
							SSHPort:     sshPort,
							SSHUser:     sshUser,
							SSHKeyPath:  sshKey,
							SSHPassword: sshPass,
						},
						IsNew: m.state == StateAddingProfile,
					}
				}
			case "esc":
				// Cancel form
				m.state = StateManagementMenu
				m.clearInputs()
				m.editingProfile = nil
				m.statusMessage = ""
				return m, nil
			default:
				// Update the focused input
				var cmd tea.Cmd
				switch m.formFocused {
				case 0:
					m.nameInput, cmd = m.nameInput.Update(msg)
				case 1:
					m.typeInput, cmd = m.typeInput.Update(msg)
				case 2:
					m.hostInput, cmd = m.hostInput.Update(msg)
				case 3:
					m.portInput, cmd = m.portInput.Update(msg)
				case 4:
					m.userInput, cmd = m.userInput.Update(msg)
				case 5:
					m.databaseInput, cmd = m.databaseInput.Update(msg)
				case 6:
					m.passwordFormInput, cmd = m.passwordFormInput.Update(msg)
				case 7:
					m.sshHostInput, cmd = m.sshHostInput.Update(msg)
				case 8:
					m.sshPortInput, cmd = m.sshPortInput.Update(msg)
				case 9:
					m.sshUserInput, cmd = m.sshUserInput.Update(msg)
				case 10:
					m.sshKeyInput, cmd = m.sshKeyInput.Update(msg)
				case 11:
					m.sshPasswordInput, cmd = m.sshPasswordInput.Update(msg)
				}
				return m, cmd
			}
		}

		if m.state == StateEnteringPassword {
			switch msg.String() {
			case "enter":
				// Submit with password
				return m, func() tea.Msg {
					return SelectedMsg{
						Index:    m.selected,
						Password: m.passwordInput.Value(),
					}
				}
			case "esc":
				// Go back to profile selection
				m.state = StateSelectingProfile
				m.passwordInput.SetValue("")
				m.passwordInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.passwordInput, cmd = m.passwordInput.Update(msg)
				return m, cmd
			}
		}

		// Management menu mode
		if m.state == StateManagementMenu {
			switch msg.String() {
			case "up", "k":
				if m.menuSelected > 0 {
					m.menuSelected--
				}
			case "down", "j":
				if m.menuSelected < 3 { // 0-3: Add, Edit, Delete, Cancel
					m.menuSelected++
				}
			case "enter":
				switch m.menuSelected {
				case 0: // Add
					m.state = StateAddingProfile
					m.editingProfile = nil
					m.clearInputs()
					m.formFocused = 0
					m.nameInput.Focus()
					m.statusMessage = ""
					return m, textinput.Blink
				case 1: // Edit
					p := m.SelectedProfile()
					if p != nil {
						m.state = StateEditingProfile
						m.editingProfile = p
						m.populateInputs(p)
						m.formFocused = 0
						m.nameInput.Focus()
						m.statusMessage = ""
						return m, textinput.Blink
					}
					return m, nil
				case 2: // Delete
					p := m.SelectedProfile()
					return m, func() tea.Msg {
						return ManagementMsg{Action: ActionDelete, Profile: p}
					}
				case 3: // Cancel
					m.state = StateSelectingProfile
					m.menuSelected = 0
					return m, nil
				}
			case "esc":
				m.state = StateSelectingProfile
				m.menuSelected = 0
				return m, nil
			}
			return m, nil
		}

		// Profile selection mode
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.profiles)-1 {
				m.selected++
			}
		case "enter":
			if m.NeedsPassword() {
				// Show password input
				m.state = StateEnteringPassword
				m.passwordInput.Focus()
				return m, textinput.Blink
			}
			// SQLite doesn't need password
			return m, func() tea.Msg {
				return SelectedMsg{Index: m.selected, Password: ""}
			}
		case "m", "M":
			// Open management menu
			m.state = StateManagementMenu
			m.menuSelected = 0
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the selector
func (m Model) View() string {
	var b strings.Builder

	if m.state == StateEnteringPassword {
		// Password input view
		p := m.SelectedProfile()
		b.WriteString(m.styles.Title.Render("Enter Password"))
		b.WriteString("\n")
		b.WriteString(m.styles.PasswordLabel.Render("Profile: " + p.Name + " (" + p.Type + ")"))
		b.WriteString("\n\n")
		b.WriteString(m.passwordInput.View())
		b.WriteString("\n\n")
		b.WriteString(m.styles.Hint.Render("Enter: Connect  Esc: Back"))
	} else if m.state == StateAddingProfile || m.state == StateEditingProfile {
		// Profile form view (Add or Edit)
		title := "Add New Profile"
		if m.state == StateEditingProfile {
			title = "Edit Profile: " + m.editingProfile.Name
		}
		b.WriteString(m.styles.Title.Render(title))
		b.WriteString("\n\n")

		// Helper to render field
		renderField := func(label string, input textinput.Model, idx int) {
			prefix := "  "
			if m.formFocused == idx {
				prefix = "→ "
				label = m.styles.PasswordLabel.Render(label)
			} else {
				label = m.styles.Hint.Render(label)
			}
			b.WriteString(prefix + label + "\n")
			b.WriteString(input.View() + "\n")
		}

		renderField("Name:", m.nameInput, 0)
		renderField("Type:", m.typeInput, 1)
		renderField("Host:", m.hostInput, 2)
		renderField("Port:", m.portInput, 3)
		renderField("User:", m.userInput, 4)
		renderField("Database:", m.databaseInput, 5)
		renderField("Password:", m.passwordFormInput, 6)

		b.WriteString("\n" + m.styles.Title.Render("SSH Tunnel (Optional)") + "\n")

		renderField("SSH Host:", m.sshHostInput, 7)
		renderField("SSH Port:", m.sshPortInput, 8)
		renderField("SSH User:", m.sshUserInput, 9)
		renderField("SSH Key:", m.sshKeyInput, 10)
		renderField("SSH Password:", m.sshPasswordInput, 11)

		b.WriteString("\n")
		b.WriteString(m.styles.Hint.Render("Tab: Next field  Enter: Save  Esc: Cancel"))
	} else if m.state == StateManagementMenu {
		// Management menu view
		b.WriteString(m.styles.Title.Render("Profile Management"))
		b.WriteString("\n")

		p := m.SelectedProfile()
		if p != nil {
			b.WriteString(m.styles.Hint.Render("Managing: " + p.Name))
			b.WriteString("\n\n")
		}

		menuItems := []string{"Add New Profile", "Edit Profile", "Delete Profile", "Cancel"}
		for i, item := range menuItems {
			style := m.styles.Item
			prefix := "  "
			if i == m.menuSelected {
				style = m.styles.Selected
				prefix = "> "
			}
			b.WriteString(style.Render(prefix + item))
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(m.styles.Hint.Render("↑/↓: Navigate  Enter: Select  Esc: Back"))
	} else {
		// Profile selection view
		b.WriteString(m.styles.Title.Render("Select Connection Profile"))
		b.WriteString("\n")

		for i, p := range m.profiles {
			style := m.styles.Item
			prefix := "  "
			if i == m.selected {
				style = m.styles.Selected
				prefix = "> "
			}

			line := prefix + p.Name + " (" + p.Type + ")"
			if p.Host != "" {
				line += " - " + p.Host
			}
			if p.SSHHost != "" {
				line += " (via SSH)"
			}

			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}

		b.WriteString(m.styles.Hint.Render("↑/↓: Navigate  Enter: Select  m: Manage  q: Quit"))
	}

	// Show status message if set
	if m.statusMessage != "" {
		b.WriteString("\n")
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A3BE8C")). // Nord14: Green
			Bold(true)
		b.WriteString(statusStyle.Render(m.statusMessage))
	}

	box := m.styles.Box.Render(b.String())

	// Center on screen
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// Helpers

func (m *Model) focusField(idx int) {
	switch idx {
	case 0:
		m.nameInput.Focus()
	case 1:
		m.typeInput.Focus()
	case 2:
		m.hostInput.Focus()
	case 3:
		m.portInput.Focus()
	case 4:
		m.userInput.Focus()
	case 5:
		m.databaseInput.Focus()
	case 6:
		m.passwordFormInput.Focus()
	case 7:
		m.sshHostInput.Focus()
	case 8:
		m.sshPortInput.Focus()
	case 9:
		m.sshUserInput.Focus()
	case 10:
		m.sshKeyInput.Focus()
	case 11:
		m.sshPasswordInput.Focus()
	}
}

func (m *Model) blurField(idx int) {
	switch idx {
	case 0:
		m.nameInput.Blur()
	case 1:
		m.typeInput.Blur()
	case 2:
		m.hostInput.Blur()
	case 3:
		m.portInput.Blur()
	case 4:
		m.userInput.Blur()
	case 5:
		m.databaseInput.Blur()
	case 6:
		m.passwordFormInput.Blur()
	case 7:
		m.sshHostInput.Blur()
	case 8:
		m.sshPortInput.Blur()
	case 9:
		m.sshUserInput.Blur()
	case 10:
		m.sshKeyInput.Blur()
	case 11:
		m.sshPasswordInput.Blur()
	}
}

func (m *Model) clearInputs() {
	m.nameInput.SetValue("")
	m.typeInput.SetValue("")
	m.hostInput.SetValue("")
	m.portInput.SetValue("")
	m.userInput.SetValue("")
	m.databaseInput.SetValue("")
	m.passwordFormInput.SetValue("")
	m.sshHostInput.SetValue("")
	m.sshPortInput.SetValue("")
	m.sshUserInput.SetValue("")
	m.sshKeyInput.SetValue("")
	m.sshPasswordInput.SetValue("")
}

func (m *Model) populateInputs(p *Profile) {
	m.nameInput.SetValue(p.Name)
	m.typeInput.SetValue(p.Type)
	m.hostInput.SetValue(p.Host)
	m.portInput.SetValue(fmt.Sprintf("%d", p.Port))
	m.userInput.SetValue(p.User)
	m.databaseInput.SetValue(p.Database)
	m.passwordFormInput.SetValue(p.Password)
	m.sshHostInput.SetValue(p.SSHHost)
	m.sshPortInput.SetValue(fmt.Sprintf("%d", p.SSHPort))
	m.sshUserInput.SetValue(p.SSHUser)
	m.sshKeyInput.SetValue(p.SSHKeyPath)
	m.sshPasswordInput.SetValue(p.SSHPassword)
}
