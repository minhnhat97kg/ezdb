package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

// TableSelectedMsg is sent when a table is selected in schema browser
type TableSelectedMsg struct {
	TableName string
}

func (m Model) renderTemplatePopup(main string) string {
	var content strings.Builder

	// Title
	title := lipgloss.NewStyle().Bold(true).Foreground(AccentColor()).Render(
		fmt.Sprintf("📋 Quick Queries for: %s", m.templateTable))
	content.WriteString(title)
	content.WriteString("\n\n")

	// List templates
	for i, t := range m.config.QueryTemplates {
		style := lipgloss.NewStyle().Foreground(TextSecondary())
		prefix := "  "
		if i == m.templateIdx {
			style = lipgloss.NewStyle().Foreground(SuccessColor()).Bold(true)
			prefix = "▸ "
		}
		// Show template with replaced table name for preview
		preview := strings.ReplaceAll(t.Query, "<table>", m.templateTable)
		if len(preview) > 50 {
			preview = preview[:47] + "..."
		}
		content.WriteString(fmt.Sprintf("%s%s\n", prefix, style.Render(t.Name)))
		content.WriteString(fmt.Sprintf("    %s\n\n", lipgloss.NewStyle().Faint(true).Render(preview)))
	}

	content.WriteString(lipgloss.NewStyle().Faint(true).Render("Enter: execute • i: insert into editor • Esc: cancel"))

	// Style popup
	popupWidth := 60
	if popupWidth > m.width-10 {
		popupWidth = m.width - 10
	}
	popupBox := PopupStyle.
		Width(popupWidth).
		MaxHeight(m.height - 4).
		Background(PopupBg()).
		Render(content.String())

	return overlay.Composite(popupBox, main, overlay.Center, overlay.Center, 0, 0)
}

func (m Model) executeTemplate() (Model, tea.Cmd) {
	if m.templateIdx < 0 || m.templateIdx >= len(m.config.QueryTemplates) {
		return m, nil
	}

	template := m.config.QueryTemplates[m.templateIdx]
	query := strings.ReplaceAll(template.Query, "<table>", m.templateTable)

	m.showTemplatePopup = false
	m.templateTable = ""
	m.templateIdx = 0

	// Execute the query
	m.loading = true
	return m, m.executeQueryCmd(query)
}

func (m Model) insertTemplate() Model {
	if m.templateIdx < 0 || m.templateIdx >= len(m.config.QueryTemplates) {
		return m
	}

	template := m.config.QueryTemplates[m.templateIdx]
	query := strings.ReplaceAll(template.Query, "<table>", m.templateTable)

	m.showTemplatePopup = false
	m.templateTable = ""
	m.templateIdx = 0

	// Insert query into editor
	m.editor.SetValue(query)
	m.mode = InsertMode
	m.editor.Focus()
	return m
}
