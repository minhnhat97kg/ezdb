// Package historylist provides a reusable scrollable list component
// for displaying history entries with selection and expansion.
package historylist

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Item represents a single list item
type Item interface {
	ID() int64
	Query() string
	QueryPreview(maxLen int) string
	Status() string
	ErrorMessage() string
	Preview() string
	DurationMs() int64
	RowCount() int
	ExecutedAtFormatted() string
}

// Styles for the list
type Styles struct {
	Item          lipgloss.Style
	Selected      lipgloss.Style
	Prompt        lipgloss.Style
	Meta          lipgloss.Style
	Error         lipgloss.Style
	SystemMessage lipgloss.Style
	SuccessIcon   lipgloss.Style
	ErrorIcon     lipgloss.Style
	InfoIcon      lipgloss.Style
	Faint         lipgloss.Style
}

// DefaultStyles returns default styling
func DefaultStyles() Styles {
	textFaint := lipgloss.Color("#6272A4")
	successColor := lipgloss.Color("#50FA7B")
	errorColor := lipgloss.Color("#FF5555")
	highlightColor := lipgloss.Color("#8BE9FD")

	return Styles{
		Item:          lipgloss.NewStyle().PaddingLeft(1),
		Selected:      lipgloss.NewStyle().PaddingLeft(1).Background(lipgloss.Color("#44475A")),
		Prompt:        lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true),
		Meta:          lipgloss.NewStyle().Foreground(textFaint),
		Error:         lipgloss.NewStyle().Foreground(errorColor),
		SystemMessage: lipgloss.NewStyle().Foreground(highlightColor).Italic(true),
		SuccessIcon:   lipgloss.NewStyle().Foreground(successColor),
		ErrorIcon:     lipgloss.NewStyle().Foreground(errorColor),
		InfoIcon:      lipgloss.NewStyle().Foreground(highlightColor),
		Faint:         lipgloss.NewStyle().Foreground(textFaint),
	}
}

// Model represents the list state
type Model struct {
	items    []Item
	selected int
	expanded map[int64]bool
	width    int
	height   int
	viewport viewport.Model
	styles   Styles

	// Callbacks
	highlightFunc func(string) string
}

// New creates a new list model
func New() Model {
	vp := viewport.New(80, 10)
	return Model{
		items:    []Item{},
		selected: 0,
		expanded: make(map[int64]bool),
		viewport: vp,
		styles:   DefaultStyles(),
	}
}

// SetItems replaces the items in the list
func (m Model) SetItems(items []Item) Model {
	m.items = items
	if m.selected >= len(items) && len(items) > 0 {
		m.selected = len(items) - 1
	}
	m.updateViewport()
	return m
}

// SetSize sets the component dimensions
func (m Model) SetSize(width, height int) Model {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
	m.updateViewport()
	return m
}

// SetStyles sets custom styles
func (m Model) SetStyles(s Styles) Model {
	m.styles = s
	return m
}

// SetHighlightFunc sets a custom syntax highlighting function
func (m Model) SetHighlightFunc(fn func(string) string) Model {
	m.highlightFunc = fn
	return m
}

// Selected returns the currently selected index
func (m Model) Selected() int {
	return m.selected
}

// SelectedItem returns the currently selected item
func (m Model) SelectedItem() Item {
	if m.selected >= 0 && m.selected < len(m.items) {
		return m.items[m.selected]
	}
	return nil
}

// IsExpanded returns whether an item is expanded
func (m Model) IsExpanded(id int64) bool {
	return m.expanded[id]
}

// ToggleExpanded toggles expansion for the selected item
func (m Model) ToggleExpanded() Model {
	if item := m.SelectedItem(); item != nil {
		m.expanded[item.ID()] = !m.expanded[item.ID()]
		m.updateViewport()
		m = m.ensureVisible()
	}
	return m
}

// SetExpanded sets the expansion state for an item
func (m Model) SetExpanded(id int64, expanded bool) Model {
	m.expanded[id] = expanded
	m.updateViewport()
	return m
}

// MoveUp moves selection up
func (m Model) MoveUp() Model {
	if m.selected > 0 {
		m.selected--
		m = m.ensureVisible()
	}
	return m
}

// MoveDown moves selection down
func (m Model) MoveDown() Model {
	if m.selected < len(m.items)-1 {
		m.selected++
		m = m.ensureVisible()
	}
	return m
}

// SelectLast selects the last item
func (m Model) SelectLast() Model {
	if len(m.items) > 0 {
		m.selected = len(m.items) - 1
		m.updateViewport()
		m.viewport.GotoBottom()
	}
	return m
}

// GotoBottom scrolls viewport to bottom
func (m Model) GotoBottom() Model {
	m.viewport.GotoBottom()
	return m
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the list
func (m Model) View() string {
	return m.viewport.View()
}

// updateViewport refreshes the viewport content
func (m *Model) updateViewport() {
	var sections []string
	for i := range m.items {
		sections = append(sections, m.renderItem(i))
	}
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Bottom anchor: pad top if content is shorter
	h := lipgloss.Height(content)
	if h < m.height {
		padding := strings.Repeat("\n", m.height-h)
		content = padding + content
	}

	m.viewport.SetContent(content)
}

// renderItem renders a single item
func (m *Model) renderItem(i int) string {
	if i < 0 || i >= len(m.items) {
		return ""
	}
	item := m.items[i]
	isSelected := i == m.selected
	isExpanded := m.expanded[item.ID()]

	// Select style
	style := m.styles.Item
	if isSelected {
		style = m.styles.Selected
	}
	style = style.Width(m.width - 2)

	var content strings.Builder

	// Query Line
	if item.Status() != "info" {
		content.WriteString(m.styles.Prompt.Render(">"))
	}

	queryText := item.QueryPreview(m.width - 10)
	if isExpanded {
		queryText = item.Query()
	}

	if item.Status() == "info" {
		content.WriteString(m.styles.SystemMessage.Render(queryText))
	} else {
		// Apply highlighting if available
		if m.highlightFunc != nil {
			queryText = m.highlightFunc(queryText)
		}
		content.WriteString(lipgloss.NewStyle().Bold(true).Render(queryText))
	}
	content.WriteString("\n")

	// Meta Line
	var statusIcon string
	var iconStyle lipgloss.Style
	switch item.Status() {
	case "error":
		statusIcon = ""
		iconStyle = m.styles.ErrorIcon
	case "info":
		statusIcon = ""
		iconStyle = m.styles.InfoIcon
	default:
		statusIcon = ""
		iconStyle = m.styles.SuccessIcon
	}

	var metaInfo string
	if item.Status() == "info" {
		metaInfo = " " + item.ExecutedAtFormatted()
	} else {
		metaInfo = fmt.Sprintf(" %dms | %d rows | %s",
			item.DurationMs(), item.RowCount(), item.ExecutedAtFormatted())
	}
	content.WriteString(iconStyle.Render("  "+statusIcon) + m.styles.Meta.Render(metaInfo))
	content.WriteString("\n")

	// Error message
	if item.ErrorMessage() != "" {
		content.WriteString(m.styles.Error.Render("  " + item.ErrorMessage()))
		content.WriteString("\n")
	}

	// Preview
	if item.Preview() != "" {
		previewLines := strings.Split(item.Preview(), "\n")
		var previewBody strings.Builder
		for _, line := range previewLines {
			previewBody.WriteString(line + "\n")
		}

		styledPreview := lipgloss.NewStyle().
			Foreground(m.styles.Faint.GetForeground()).
			Padding(0, 4). // Left/Right padding for the preview text
			Render(previewBody.String())

		content.WriteString(styledPreview)
		content.WriteString("\n")
	}

	return style.Render(content.String())
}

// ensureVisible keeps the selected item in view
func (m Model) ensureVisible() Model {
	if len(m.items) == 0 {
		return m
	}

	// Calculate top Y of selected item
	top := 0
	for i := 0; i < m.selected; i++ {
		top += lipgloss.Height(m.renderItem(i))
	}

	itemHeight := lipgloss.Height(m.renderItem(m.selected))
	bottom := top + itemHeight

	vTop := m.viewport.YOffset
	vBottom := vTop + m.viewport.Height

	if top < vTop {
		m.viewport.SetYOffset(top)
	} else if bottom > vBottom {
		m.viewport.SetYOffset(bottom - m.viewport.Height)
	}

	return m
}
