package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// TableSelectedMsg is sent when a table is selected in schema browser
type TableSelectedMsg struct {
	TableName string
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
