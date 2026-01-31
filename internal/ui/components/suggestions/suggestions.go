// Package suggestions provides a reusable autocomplete dropdown component.
package suggestions

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles for the suggestions dropdown
type Styles struct {
	Box      lipgloss.Style
	Item     lipgloss.Style
	Selected lipgloss.Style
	Loading  lipgloss.Style
}

// DefaultStyles returns default styling
func DefaultStyles() Styles {
	return Styles{
		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#6272A4")).
			Padding(0, 1),
		Item: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2")),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#282A36")).
			Background(lipgloss.Color("#8BE9FD")),
		Loading: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Italic(true),
	}
}

// Model represents the suggestions state
type Model struct {
	items    []string
	selected int
	visible  bool
	loading  bool
	maxShow  int
	styles   Styles
}

// New creates a new suggestions model
func New() Model {
	return Model{
		items:    []string{},
		selected: 0,
		visible:  false,
		loading:  false,
		maxShow:  5,
		styles:   DefaultStyles(),
	}
}

// SetItems sets the suggestion items
func (m Model) SetItems(items []string) Model {
	m.items = items
	if m.selected >= len(items) {
		m.selected = 0
	}
	if len(items) > 0 && m.selected < 0 {
		m.selected = 0
	}
	return m
}

// SetStyles sets custom styles
func (m Model) SetStyles(s Styles) Model {
	m.styles = s
	return m
}

// SetMaxShow sets maximum visible items
func (m Model) SetMaxShow(n int) Model {
	m.maxShow = n
	return m
}

// Show makes the dropdown visible
func (m Model) Show() Model {
	m.visible = true
	return m
}

// Hide hides the dropdown
func (m Model) Hide() Model {
	m.visible = false
	m.selected = 0
	return m
}

// SetLoading sets loading state
func (m Model) SetLoading(loading bool) Model {
	m.loading = loading
	return m
}

// Visible returns visibility state
func (m Model) Visible() bool {
	return m.visible
}

// Loading returns loading state
func (m Model) Loading() bool {
	return m.loading
}

// Selected returns the selected index
func (m Model) Selected() int {
	return m.selected
}

// SelectedItem returns the selected item string
func (m Model) SelectedItem() string {
	if m.selected >= 0 && m.selected < len(m.items) {
		return m.items[m.selected]
	}
	return ""
}

// Items returns all items
func (m Model) Items() []string {
	return m.items
}

// Len returns number of items
func (m Model) Len() int {
	return len(m.items)
}

// MoveUp moves selection up
func (m Model) MoveUp() Model {
	if m.selected > 0 {
		m.selected--
	}
	return m
}

// MoveDown moves selection down
func (m Model) MoveDown() Model {
	if m.selected < len(m.items)-1 {
		m.selected++
	}
	return m
}

// Update handles messages (currently passthrough)
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

// View renders the suggestions dropdown
func (m Model) View() string {
	if !m.visible {
		return ""
	}

	if m.loading {
		return m.styles.Box.Render(m.styles.Loading.Render("Loading..."))
	}

	if len(m.items) == 0 {
		return ""
	}

	// Calculate visible window
	start := 0
	if m.selected > m.maxShow/2 {
		start = m.selected - m.maxShow/2
	}
	end := start + m.maxShow
	if end > len(m.items) {
		end = len(m.items)
		if end-m.maxShow >= 0 {
			start = end - m.maxShow
		} else {
			start = 0
		}
	}

	var views []string
	for i := start; i < end; i++ {
		item := m.items[i]
		style := m.styles.Item
		prefix := "  "
		if i == m.selected {
			style = m.styles.Selected
			prefix = "> "
		}
		views = append(views, style.Render(prefix+item))
	}

	return m.styles.Box.Render(strings.Join(views, "\n"))
}
