// Package profileselector provides a profile selection popup for app startup.
package profileselector

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nhath/ezdb/internal/config"
	"github.com/nhath/ezdb/internal/ui/icons"
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
	Subtitle      lipgloss.Style
	Item          lipgloss.Style
	ItemName      lipgloss.Style
	ItemType      lipgloss.Style
	ItemHost      lipgloss.Style
	ItemSSH       lipgloss.Style
	Selected      lipgloss.Style
	SelectedName  lipgloss.Style
	SelectedType  lipgloss.Style
	SelectedHost  lipgloss.Style
	Hint          lipgloss.Style
	HintKey       lipgloss.Style
	PasswordLabel lipgloss.Style
	SectionTitle  lipgloss.Style
	FieldLabel    lipgloss.Style
	FieldLabelAct lipgloss.Style
	StatusSuccess lipgloss.Style
	StatusError   lipgloss.Style
	Divider       lipgloss.Style
	ProfileIcon   lipgloss.Style
	Logo          lipgloss.Style
	MenuIcon      lipgloss.Style
	HelpKey       lipgloss.Style
	Footer        lipgloss.Style
}

// DefaultStyles returns the default styling
func DefaultStyles(theme config.Theme) Styles {
	return Styles{
		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Highlight)).
			Padding(1, 2).
			Width(80),
		Logo: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Accent)).
			Bold(true).
			Align(lipgloss.Center).
			MarginBottom(2),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.TextPrimary)).
			Background(lipgloss.Color(theme.Highlight)).
			Padding(0, 1).
			MarginBottom(1),
		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextFaint)).
			Italic(true).
			MarginBottom(1),
		Item: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextSecondary)).
			PaddingLeft(2).
			MarginBottom(0),
		ItemName: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextPrimary)).
			Bold(true),
		ItemType: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Highlight)).
			Italic(true),
		ItemHost: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextFaint)),
		ItemSSH: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Warning)),
		Selected: lipgloss.NewStyle().
			Border(lipgloss.ThickBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color(theme.Accent)).
			Background(lipgloss.Color(theme.SelectedBg)).
			PaddingLeft(1).
			PaddingRight(1).
			MarginBottom(0),
		SelectedName: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Success)).
			Bold(true),
		SelectedType: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Highlight)),
		SelectedHost: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextSecondary)),
		Hint: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextFaint)),
		HintKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextPrimary)).
			Background(lipgloss.Color(theme.CardBg)).
			Padding(0, 1).
			Bold(true),
		PasswordLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Accent)).
			Bold(true),
		SectionTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Highlight)).
			Bold(true).
			MarginTop(1).
			MarginBottom(1),
		FieldLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextFaint)).
			Width(16),
		FieldLabelAct: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Accent)).
			Bold(true).
			Width(16),
		StatusSuccess: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Success)).
			Bold(true),
		StatusError: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Error)).
			Bold(true),
		Divider: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.BorderColor)),
		ProfileIcon: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Accent)),
		MenuIcon: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Success)),
		HelpKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextFaint)).
			Faint(true),
		Footer: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextPrimary)).
			Bold(false).
			Align(lipgloss.Center),
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

	newPasswordInput := func(placeholder string, width int) textinput.Model {
		t := textinput.New()
		t.Placeholder = placeholder
		t.Width = width
		t.EchoMode = textinput.EchoPassword
		t.EchoCharacter = '•'
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
		passwordFormInput: newPasswordInput("Password (optional)", 30),

		sshHostInput:     newInput("SSH Host", 40),
		sshPortInput:     newInput("SSH Port (22)", 10),
		sshUserInput:     newInput("SSH User", 30),
		sshKeyInput:      newInput("SSH Key Path (~/.ssh/id_rsa)", 50),
		sshPasswordInput: newPasswordInput("SSH Password (optional)", 30),

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

// ResetState goes back to selection mode
func (m Model) ResetState() Model {
	m.state = StateSelectingProfile
	m.menuSelected = 0
	m.clearInputs()
	m.editingProfile = nil
	m.statusMessage = ""
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
			case "ctrl+c":
				return m, tea.Quit
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

	// Handle non-key messages (like TickMsg for blinking)
	var cmds []tea.Cmd
	var cmd tea.Cmd

	// Update password prompt if in that state
	if m.state == StateEnteringPassword {
		m.passwordInput, cmd = m.passwordInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	// For form inputs:
	// If it's a key message, it was already handled in the focused field switch above.
	// If it's NOT a key message (e.g. TickMsg), we should update all of them
	// to ensure blinking/other animations work.
	if _, ok := msg.(tea.KeyMsg); !ok {
		m.nameInput, cmd = m.nameInput.Update(msg)
		cmds = append(cmds, cmd)
		m.typeInput, cmd = m.typeInput.Update(msg)
		cmds = append(cmds, cmd)
		m.hostInput, cmd = m.hostInput.Update(msg)
		cmds = append(cmds, cmd)
		m.portInput, cmd = m.portInput.Update(msg)
		cmds = append(cmds, cmd)
		m.userInput, cmd = m.userInput.Update(msg)
		cmds = append(cmds, cmd)
		m.databaseInput, cmd = m.databaseInput.Update(msg)
		cmds = append(cmds, cmd)
		m.passwordFormInput, cmd = m.passwordFormInput.Update(msg)
		cmds = append(cmds, cmd)
		m.sshHostInput, cmd = m.sshHostInput.Update(msg)
		cmds = append(cmds, cmd)
		m.sshPortInput, cmd = m.sshPortInput.Update(msg)
		cmds = append(cmds, cmd)
		m.sshUserInput, cmd = m.sshUserInput.Update(msg)
		cmds = append(cmds, cmd)
		m.sshKeyInput, cmd = m.sshKeyInput.Update(msg)
		cmds = append(cmds, cmd)
		m.sshPasswordInput, cmd = m.sshPasswordInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the selector
func (m Model) View() string {
	var b strings.Builder

	// Logo for Home Screen
	logo := `
  ███████╗███████╗██████╗ ██████╗ 
  ██╔════╝╚══███╔╝██╔══██╗██╔══██╗
  █████╗    ███╔╝ ██║  ██║██████╔╝
  ██╔══╝   ███╔╝  ██║  ██║██╔══██╗
  ███████╗███████╗██████╔╝██████╔╝
  ╚══════╝╚══════╝╚═════╝ ╚═════╝ `

	itemWidth := m.styles.Box.GetWidth() - 6

	if m.state == StateSelectingProfile {
		// Center the logo
		centeredLogo := lipgloss.NewStyle().
			Width(m.styles.Box.GetWidth() - 4).
			Align(lipgloss.Center).
			Render(m.styles.Logo.Render(logo))
		b.WriteString(centeredLogo)
		b.WriteString("\n\n")

		// Center the title
		centeredTitle := lipgloss.NewStyle().
			Width(m.styles.Box.GetWidth() - 4).
			Align(lipgloss.Center).
			Render(m.styles.Title.Render(" SELECT PROFILE "))
		b.WriteString(centeredTitle)
		b.WriteString("\n\n")

		for i, p := range m.profiles {
			style := m.styles.Item.Copy().Width(itemWidth)
			nameStyle := m.styles.ItemName
			hostStyle := m.styles.ItemHost

			// Selection indicator and icon
			icon := icons.GetDatabaseIcon(p.Type)

			prefix := "   "
			if i == m.selected {
				style = m.styles.Selected.Copy().Width(itemWidth)
				nameStyle = m.styles.SelectedName
				hostStyle = m.styles.SelectedHost
				prefix = " " + icons.IconSelect + " "
			}

			// First row: icon + name
			nameRow := prefix + icon + " " + nameStyle.Render(p.Name)

			// Second row: faint connection info
			hostStr := ""
			if p.Type == "sqlite" {
				hostStr = p.Database
			} else {
				if p.Host != "" {
					hostStr = limitString(p.Host, 20)
					if p.Port != 0 {
						hostStr += fmt.Sprintf(":%d", p.Port)
					}
				}
				if p.Database != "" {
					if hostStr != "" {
						hostStr += "/"
					}
					hostStr += p.Database

				}
			}
			if p.SSHHost != "" {
				hostStr += fmt.Sprintf(" %s %s", icons.IconLock, p.SSHHost)
			}

			// Indent the second row to align with name (prefix width + icon width)
			indent := "      "
			hostRow := indent + hostStyle.Render(hostStr)

			b.WriteString(style.Render(nameRow+"\n"+hostRow) + "\n")
		}

		b.WriteString("\n")
		// Footer hints - inline format
		hints := []struct{ key, desc string }{
			{"↑↓", "Navigate"},
			{"Enter", "Select"},
			{"m", "Manage"},
			{"q", "Quit"},
			{"?", "Help"},
		}
		var hintParts []string
		for _, h := range hints {
			// Clear any margins for inline display
			desc := m.styles.Hint.Copy().Margin(0).Render(h.desc)
			key := m.styles.HintKey.Copy().Margin(0).Render(h.key)
			hintParts = append(hintParts, key+" "+desc)
		}

		// Footer bar - just center it without a background row
		footerRow := strings.Join(hintParts, icons.IconSeparator)
		footer := m.styles.Footer.Copy().
			Width(itemWidth + 2).
			MarginTop(1).
			Render(footerRow)

		b.WriteString(footer)
		b.WriteString("\n")

	} else if m.state == StateManagementMenu {
		// Management menu view
		centeredTitle := lipgloss.NewStyle().
			Width(m.styles.Box.GetWidth() - 4).
			Align(lipgloss.Center).
			Render(m.styles.Title.Render(" PROFILE MANAGEMENT "))
		b.WriteString(centeredTitle)
		b.WriteString("\n\n")

		p := m.SelectedProfile()
		if p != nil {
			centeredSelected := lipgloss.NewStyle().
				Width(m.styles.Box.GetWidth() - 4).
				Align(lipgloss.Center).
				Render(m.styles.Hint.Render("Selected: " + p.Name))
			b.WriteString(centeredSelected)
			b.WriteString("\n\n")
		}

		menuItems := []string{
			icons.IconAdd + " Add New Profile",
			icons.IconEdit + " Edit Profile",
			icons.IconDelete + " Delete Profile",
			icons.IconCancel + " Back",
		}
		itemWidth := m.styles.Box.GetWidth() - 6
		for i, item := range menuItems {
			style := m.styles.Item.Copy().Width(itemWidth)
			prefix := "   "
			if i == m.menuSelected {
				style = m.styles.Selected.Copy().Width(itemWidth)
				prefix = " " + icons.IconSelect + " "
			}
			b.WriteString(style.Render(prefix + item))
			b.WriteString("\n")
		}

		b.WriteString("\n")
		footerRow := m.styles.HintKey.Render("Enter") + " " + m.styles.Hint.Copy().Margin(0).Render("Select") +
			icons.IconSeparator +
			m.styles.HintKey.Render("Esc") + " " + m.styles.Hint.Copy().Margin(0).Render("Back")

		footer := m.styles.Footer.Copy().
			Width(itemWidth + 2).
			MarginTop(1).
			Render(footerRow)
		b.WriteString(footer)

	} else if m.state == StateEnteringPassword {
		// Password input view
		p := m.SelectedProfile()
		centeredTitle := lipgloss.NewStyle().
			Width(m.styles.Box.GetWidth() - 4).
			Align(lipgloss.Center).
			Render(m.styles.Title.Render(" AUTHENTICATION "))
		b.WriteString(centeredTitle)
		b.WriteString("\n\n")

		centeredLabel := lipgloss.NewStyle().
			Width(m.styles.Box.GetWidth() - 4).
			Align(lipgloss.Center).
			Render(m.styles.PasswordLabel.Render(fmt.Sprintf("Enter password for %s (%s)", p.Name, p.Type)))
		b.WriteString(centeredLabel)
		b.WriteString("\n\n")

		// Center text input? Bubbles textinput isn't easily centered inside lipgloss.Place usually,
		// but we can pad it.
		tiView := lipgloss.NewStyle().
			Width(m.styles.Box.GetWidth() - 4).
			Align(lipgloss.Center).
			Render(m.passwordInput.View())
		b.WriteString(tiView)

		b.WriteString("\n\n")
		footerRow := m.styles.HintKey.Render("Enter") + " " + m.styles.Hint.Copy().Margin(0).Render("Connect") +
			icons.IconSeparator +
			m.styles.HintKey.Render("Esc") + " " + m.styles.Hint.Copy().Margin(0).Render("Back")

		footer := m.styles.Footer.Copy().
			Width(itemWidth + 2).
			MarginTop(1).
			Render(footerRow)
		b.WriteString(footer)

	} else if m.state == StateAddingProfile || m.state == StateEditingProfile {
		// Profile form view (Add or Edit)
		title := " NEW PROFILE "
		if m.state == StateEditingProfile {
			title = " EDIT PROFILE "
		}
		centeredTitle := lipgloss.NewStyle().
			Width(m.styles.Box.GetWidth() - 4).
			Align(lipgloss.Center).
			Render(m.styles.Title.Render(title))
		b.WriteString(centeredTitle)
		b.WriteString("\n\n")

		// Helper to render field
		renderField := func(label string, input textinput.Model, idx int) {
			prefix := "  "
			styleLabel := m.styles.FieldLabel
			if m.formFocused == idx {
				prefix = icons.IconPointer
				styleLabel = m.styles.FieldLabelAct
			}
			// Render label + input
			b.WriteString(fmt.Sprintf("%2s %s %s\n", prefix, styleLabel.Render(label), input.View()))
		}

		renderField("Name", m.nameInput, 0)
		renderField("Type", m.typeInput, 1)
		b.WriteString(m.styles.Divider.Render("──────────────────────────────────────────────────") + "\n")
		renderField("Host", m.hostInput, 2)
		renderField("Port", m.portInput, 3)
		renderField("User", m.userInput, 4)
		renderField("Database", m.databaseInput, 5)
		renderField("Password", m.passwordFormInput, 6)

		b.WriteString("\n" + m.styles.SectionTitle.Render(" SSH Tunnel (Optional) ") + "\n")

		renderField("SSH Host", m.sshHostInput, 7)
		renderField("SSH Port", m.sshPortInput, 8)
		renderField("SSH User", m.sshUserInput, 9)
		renderField("SSH Key", m.sshKeyInput, 10)
		renderField("SSH Password", m.sshPasswordInput, 11)

		b.WriteString("\n")
		footerRow := m.styles.HintKey.Render("Tab") + " " + m.styles.Hint.Copy().Margin(0).Render("Next") +
			icons.IconSeparator +
			m.styles.HintKey.Render("Enter") + " " + m.styles.Hint.Copy().Margin(0).Render("Save") +
			icons.IconSeparator +
			m.styles.HintKey.Render("Esc") + " " + m.styles.Hint.Copy().Margin(0).Render("Cancel")

		footer := m.styles.Footer.Copy().
			Width(itemWidth + 2).
			MarginTop(1).
			Render(footerRow)
		b.WriteString(footer)
	}

	// Show status message if set
	if m.statusMessage != "" {
		b.WriteString("\n\n")
		statusStyle := m.styles.StatusSuccess
		if strings.HasPrefix(m.statusMessage, "Error") {
			statusStyle = m.styles.StatusError
		}
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

func limitString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// relace middle with ...
	half := (maxLen - 3) / 2
	return s[:half] + "..." + s[len(s)-half:]
}
