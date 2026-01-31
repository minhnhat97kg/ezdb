package ui

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nhath/ezdb/internal/config"
	"github.com/nhath/ezdb/internal/ui/components/popup"
)

type ThemeSelector struct {
	visible  bool
	themes   []string
	selected int
	popup    popup.Model
	config   *config.Config
}

func NewThemeSelector(cfg *config.Config) ThemeSelector {
	themes := config.GetThemes()
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	sort.Strings(names)

	return ThemeSelector{
		visible:  false,
		themes:   names,
		selected: 0,
		popup:    popup.New(cfg.Theme),
		config:   cfg,
	}
}

func (m ThemeSelector) UpdateTheme(theme config.Theme) ThemeSelector {
	m.popup = popup.New(theme)
	return m
}

func (m ThemeSelector) Show() ThemeSelector {
	m.visible = true
	// Find current theme index
	for i, name := range m.themes {
		if name == m.config.ThemeName {
			m.selected = i
			break
		}
	}
	return m
}

func (m ThemeSelector) Hide() ThemeSelector {
	m.visible = false
	return m
}

func (m ThemeSelector) Visible() bool {
	return m.visible
}

func (m ThemeSelector) Update(msg tea.Msg) (ThemeSelector, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.themes)-1 {
				m.selected++
			}
		case "enter":
			m.visible = false
			themeName := m.themes[m.selected]
			theme := config.GetThemes()[themeName]
			return m, func() tea.Msg {
				return ThemeSelectedMsg{ThemeName: themeName, Theme: theme}
			}
		case "esc", "q":
			m.visible = false
			return m, nil
		}
	}
	return m, nil
}

func (m ThemeSelector) View(w, h int) string {
	if !m.visible {
		return ""
	}

	m.popup = m.popup.SetScreenSize(w, h)

	var content string
	for i, name := range m.themes {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.TextPrimary))
		prefix := "  "
		if i == m.selected {
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.config.Theme.BgPrimary)).
				Background(lipgloss.Color(m.config.Theme.Success)).
				Bold(true)
			prefix = "> "
		}
		content += style.Render(prefix+name) + "\n"
	}

	m.popup = m.popup.Show("Select Theme", content, "Enter: Select • Esc: Cancel • ?: Help")
	return m.popup.View()
}
