// Package popup provides a reusable modal popup component.
package popup

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles for the popup
type Styles struct {
	Box    lipgloss.Style
	Header lipgloss.Style
	Body   lipgloss.Style
	Footer lipgloss.Style
}

// DefaultStyles returns default styling
func DefaultStyles() Styles {
	return Styles{
		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#BD93F9")).
			Padding(1, 2),
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F8F8F2")),
		Body: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2")),
		Footer: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Italic(true),
	}
}

// Model represents the popup state
type Model struct {
	visible    bool
	title      string
	content    string
	footer     string
	width      int
	maxWidth   int
	maxHeight  int
	screenW    int
	screenH    int
	page       int
	totalPages int
	styles     Styles
}

// New creates a new popup model
func New() Model {
	return Model{
		visible:  false,
		maxWidth: 120,
		styles:   DefaultStyles(),
	}
}

// SetStyles sets custom styles
func (m Model) SetStyles(s Styles) Model {
	m.styles = s
	return m
}

// SetScreenSize sets the screen dimensions for centering
func (m Model) SetScreenSize(w, h int) Model {
	m.screenW = w
	m.screenH = h
	m.maxWidth = min(120, w-4)
	m.maxHeight = h - 4
	return m
}

// Show makes the popup visible with content
func (m Model) Show(title, content, footer string) Model {
	m.visible = true
	m.title = title
	m.content = content
	m.footer = footer
	m.page = 0
	return m
}

// Hide hides the popup
func (m Model) Hide() Model {
	m.visible = false
	return m
}

// Visible returns visibility state
func (m Model) Visible() bool {
	return m.visible
}

// SetPage sets current page
func (m Model) SetPage(page int) Model {
	m.page = page
	return m
}

// SetTotalPages sets total pages
func (m Model) SetTotalPages(total int) Model {
	m.totalPages = total
	return m
}

// Page returns current page
func (m Model) Page() int {
	return m.page
}

// NextPage goes to next page
func (m Model) NextPage() Model {
	if m.page < m.totalPages-1 {
		m.page++
	}
	return m
}

// PrevPage goes to previous page
func (m Model) PrevPage() Model {
	if m.page > 0 {
		m.page--
	}
	return m
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "q", "esc":
			m.visible = false
		case "n":
			m = m.NextPage()
		case "p":
			m = m.PrevPage()
		}
	}

	return m, nil
}

// View renders the popup
func (m Model) View() string {
	if !m.visible {
		return ""
	}

	var b strings.Builder

	// Title
	if m.title != "" {
		b.WriteString(m.styles.Header.Render(m.title))
		b.WriteString("\n\n")
	}

	// Content
	b.WriteString(m.styles.Body.Render(m.content))

	// Footer
	if m.footer != "" {
		b.WriteString("\n\n")
		b.WriteString(m.styles.Footer.Render(m.footer))
	}

	box := m.styles.Box.
		Width(m.maxWidth).
		MaxHeight(m.maxHeight).
		Render(b.String())

	// Center on screen
	return lipgloss.Place(m.screenW, m.screenH, lipgloss.Center, lipgloss.Center, box)
}

// RenderOverlay renders popup on top of main content
func (m Model) RenderOverlay(main string) string {
	if !m.visible {
		return main
	}
	return m.View()
}

// Helper for building table-style content
func BuildTableContent(columns []string, rows [][]string, page, pageSize int) (string, int) {
	if len(columns) == 0 {
		return "(No results)", 1
	}

	var b strings.Builder

	// Header
	b.WriteString(strings.Join(columns, " | "))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("-", len(strings.Join(columns, " | "))))
	b.WriteString("\n")

	// Paginated rows
	start := page * pageSize
	end := min(start+pageSize, len(rows))

	for i := start; i < end; i++ {
		b.WriteString(strings.Join(rows[i], " | "))
		b.WriteString("\n")
	}

	totalPages := (len(rows) + pageSize - 1) / pageSize
	b.WriteString(fmt.Sprintf("\nPage %d/%d (%d-%d of %d rows) [n]ext [p]rev",
		page+1, totalPages, start+1, end, len(rows)))

	return b.String(), totalPages
}
