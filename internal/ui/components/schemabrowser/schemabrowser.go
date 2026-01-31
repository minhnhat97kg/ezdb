// Package schemabrowser provides a popup for browsing database schema.
package schemabrowser

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"

	"github.com/nhath/ezdb/internal/db"
	eztable "github.com/nhath/ezdb/internal/ui/components/table"
)

type State int

const (
	StateTables State = iota
	StateColumns
)

type DetailTab int

const (
	TabColumns DetailTab = iota
	TabConstraints
)

// SchemaLoadedMsg is sent when schema is loaded
type SchemaLoadedMsg struct {
	Tables      []string
	Columns     map[string][]db.Column
	Constraints map[string][]db.Constraint
	Err         error
}

// TableSelectedMsg is sent when a table is selected for template
type TableSelectedMsg struct {
	TableName string
}

// ExportTableMsg is sent when a table export is requested
type ExportTableMsg struct {
	TableName string
}

// ImportTableMsg is sent when a table import is requested
type ImportTableMsg struct {
	TableName string
}

// Styles for the browser
type Styles struct {
	Container     lipgloss.Style
	Title         lipgloss.Style
	SectionTitle  lipgloss.Style
	Item          lipgloss.Style
	ItemActive    lipgloss.Style
	TableHeader   lipgloss.Style
	TableCell     lipgloss.Style
	TableCellKey  lipgloss.Style
	TableCellType lipgloss.Style
	Spinner       lipgloss.Style
	TabActive     lipgloss.Style
	TabInactive   lipgloss.Style
}

// DefaultStyles returns default styling using Nord palette
func DefaultStyles() Styles {
	// Nord color palette (matching OpenCode)
	textPrimary := lipgloss.Color("#D8DEE9")    // Nord4: Light gray
	textFaint := lipgloss.Color("#4C566A")      // Nord3: Dark gray
	accentColor := lipgloss.Color("#88C0D0")    // Nord8: Cyan blue
	successColor := lipgloss.Color("#A3BE8C")   // Nord14: Green
	highlightColor := lipgloss.Color("#8FBCBB") // Nord7: Teal

	return Styles{
		Container: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(highlightColor). // Nord7: Teal border
			Padding(1, 2),
		// No Background - transparent, inherits from terminal
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor). // Nord8: Cyan
			MarginBottom(1),
		SectionTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(highlightColor). // Nord7: Teal
			MarginTop(1).
			MarginBottom(1),
		Item: lipgloss.NewStyle().
			Foreground(textPrimary), // Nord4: Light gray
		ItemActive: lipgloss.NewStyle().
			Foreground(successColor). // Nord14: Green
			Bold(true),
		TableHeader: lipgloss.NewStyle().
			Foreground(accentColor). // Nord8: Cyan
			Bold(true).
			Border(lipgloss.NormalBorder(), false, false, true, false),
		TableCell: lipgloss.NewStyle().
			Foreground(textPrimary), // Nord4: Light gray
		TableCellKey: lipgloss.NewStyle().
			Foreground(successColor), // Nord14: Green
		TableCellType: lipgloss.NewStyle().
			Foreground(textFaint), // Nord3: Dark gray
		Spinner: lipgloss.NewStyle().
			Foreground(highlightColor), // Nord7: Teal
		TabActive: lipgloss.NewStyle().
			Foreground(successColor). // Nord14: Green
			Bold(true).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(successColor). // Nord14: Green underline
			Padding(0, 1),
		TabInactive: lipgloss.NewStyle().
			Foreground(textFaint). // Nord3: Dark gray
			Padding(0, 1),
	}
}

// Model represents the schema browser state
type Model struct {
	visible          bool
	state            State
	tables           []string
	columns          map[string][]db.Column
	constraints      map[string][]db.Constraint
	selectedTable    string
	selectedIdx      int
	width            int
	height           int
	activeTab        DetailTab
	styles           Styles
	viewport         viewport.Model
	spinner          spinner.Model
	columnsTable     table.Model
	constraintsTable table.Model
	loading          bool
}

// New creates a new schema browser
func New() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6"))

	return Model{
		visible:     false,
		state:       StateTables,
		columns:     make(map[string][]db.Column),
		constraints: make(map[string][]db.Constraint),
		styles:      DefaultStyles(),
		viewport:    viewport.New(0, 0),
		spinner:     s,
	}
}

// SetSize sets the available size
func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h
	return m.updateViewportDimensions()
}

func (m Model) updateViewportDimensions() Model {
	// Calculate popup size
	popupWidth := int(float64(m.width) * 0.9)
	if popupWidth > 100 {
		popupWidth = 100
	}
	popupHeight := int(float64(m.height) * 0.8)
	if popupHeight > 35 {
		popupHeight = 35
	}

	m.viewport.Width = popupWidth - 6
	if m.state == StateColumns {
		m.viewport.Height = popupHeight - 7
	} else {
		m.viewport.Height = popupHeight - 4
	}
	return m
}

// Toggle toggles visibility
func (m Model) Toggle() Model {
	m.visible = !m.visible
	if m.visible {
		m.state = StateTables
		m.selectedIdx = 0
	}
	return m
}

// IsVisible returns visibility state
func (m Model) IsVisible() bool {
	return m.visible
}

// StartLoading begins loading state
func (m Model) StartLoading() (Model, tea.Cmd) {
	m.loading = true
	return m, m.spinner.Tick
}

// SetSchema sets the schema data and stops loading
func (m Model) SetSchema(tables []string, columns map[string][]db.Column, constraints map[string][]db.Constraint) Model {
	m.tables = tables
	m.columns = columns
	m.constraints = constraints
	m.loading = false
	return m
}

// LoadSchemaCmd loads schema from driver
func LoadSchemaCmd(driver db.Driver) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		f, _ := os.OpenFile("schema_loader_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if f != nil {
			fmt.Fprintf(f, "Step 1: LoadSchemaCmd started\n")
		}

		tables, err := driver.GetTables(ctx)
		if err != nil {
			if f != nil {
				fmt.Fprintf(f, "Step 1.5: GetTables error: %v\n", err)
				f.Close()
			}
			return SchemaLoadedMsg{Err: err}
		}

		if f != nil {
			fmt.Fprintf(f, "Step 2: Found tables: %v\n", tables)
		}

		columns := make(map[string][]db.Column)
		constraints := make(map[string][]db.Constraint)
		var mu sync.Mutex

		// Use a semaphore to limit concurrency
		sem := make(chan struct{}, 20)
		var wg sync.WaitGroup

		for _, table := range tables {
			wg.Add(1)
			go func(t string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				cols, err := driver.GetColumns(ctx, t)
				cons, err2 := driver.GetConstraints(ctx, t)

				mu.Lock()
				defer mu.Unlock()
				if err == nil {
					columns[t] = cols
				}
				if err2 == nil {
					constraints[t] = cons
				}
			}(table)
		}

		wg.Wait()

		if f != nil {
			fmt.Fprintf(f, "Step 3: Finished fetching columns. Columns map size: %d\n", len(columns))
			f.Close()
		}

		return SchemaLoadedMsg{Tables: tables, Columns: columns, Constraints: constraints}
	}
}

// Update handles input
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible && !m.loading {
		return m, nil
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.state == StateTables {
				if m.selectedIdx > 0 {
					m.selectedIdx--
					m = m.ensureSelectionVisible()
				}
				return m, nil
			} else {
				m.viewport.LineUp(1)
				return m, nil
			}
		case "down", "j":
			if m.state == StateTables {
				if m.selectedIdx < len(m.tables)-1 {
					m.selectedIdx++
					m = m.ensureSelectionVisible()
				}
				return m, nil
			} else {
				m.viewport.LineDown(1)
				return m, nil
			}
		case "left", "h":
			if m.state == StateColumns {
				m.activeTab = TabColumns
				m.viewport.YOffset = 0
				m.viewport.SetContent(m.renderContent())
			}
		case "right", "l":
			if m.state == StateColumns {
				m.activeTab = TabConstraints
				m.viewport.YOffset = 0
				m.viewport.SetContent(m.renderContent())
			}
		case "t": // Template quick query
			var tableName string
			if m.state == StateTables && len(m.tables) > 0 {
				tableName = m.tables[m.selectedIdx]
			} else if m.state == StateColumns {
				tableName = m.selectedTable
			}

			if tableName != "" {
				m.visible = false
				return m, func() tea.Msg {
					return TableSelectedMsg{TableName: tableName}
				}
			}
		case "e": // Export table
			var tableName string
			if m.state == StateTables && len(m.tables) > 0 {
				tableName = m.tables[m.selectedIdx]
			} else if m.state == StateColumns {
				tableName = m.selectedTable
			}

			if tableName != "" {
				m.visible = false
				return m, func() tea.Msg {
					return ExportTableMsg{TableName: tableName}
				}
			}
		case "o": // Import (open) data into table
			var tableName string
			if m.state == StateTables && len(m.tables) > 0 {
				tableName = m.tables[m.selectedIdx]
			} else if m.state == StateColumns {
				tableName = m.selectedTable
			}

			if tableName != "" {
				m.visible = false
				return m, func() tea.Msg {
					return ImportTableMsg{TableName: tableName}
				}
			}
		case "enter":
			if m.state == StateTables && len(m.tables) > 0 {
				m.selectedTable = m.tables[m.selectedIdx]
				m.state = StateColumns
				m.selectedIdx = 0
				m.activeTab = TabColumns
				m.viewport.YOffset = 0
				// Initialize rich tables - non-paginated and unfocused for viewport scrolling
				m.columnsTable = eztable.FromSchemaColumns(m.columns[m.selectedTable]).WithNoPagination().Focused(false)
				m.constraintsTable = eztable.FromConstraints(m.constraints[m.selectedTable]).WithNoPagination().Focused(false)

				// Synchronize viewport dimensions and content immediately
				m = m.updateViewportDimensions()
				m.viewport.SetContent(m.renderContent())
			}
		case "backspace", "esc":
			if m.state == StateColumns {
				m.state = StateTables
				// Find index of selected table
				for i, t := range m.tables {
					if t == m.selectedTable {
						m.selectedIdx = i
						break
					}
				}
				m = m.updateViewportDimensions()
				m = m.ensureSelectionVisible()
				m.viewport.SetContent(m.renderContent())
			} else {
				m.visible = false
			}
		case "tab":
			m.visible = false
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	if m.state == StateColumns && !m.loading {
		m.columnsTable, cmd = m.columnsTable.Update(msg)
		cmds = append(cmds, cmd)
		m.constraintsTable, cmd = m.constraintsTable.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) ensureSelectionVisible() Model {
	if m.viewport.Height <= 0 {
		return m
	}

	if m.selectedIdx < m.viewport.YOffset {
		m.viewport.YOffset = m.selectedIdx
	} else if m.selectedIdx >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.YOffset = m.selectedIdx - m.viewport.Height + 1
	}
	return m
}

// SetStyles sets custom styles
func (m Model) SetStyles(s Styles) Model {
	m.styles = s
	return m
}

// View renders the browser popup
func (m Model) View() string {
	if !m.visible && !m.loading {
		return ""
	}

	var view strings.Builder

	if m.loading {
		return m.styles.Container.
			Width(40).
			Height(5).
			Render(fmt.Sprintf("\n  %s Loading Schema...", m.spinner.View()))
	}

	popupWidth, popupHeight := m.getPopupSize()

	title := " Tables"
	if m.state == StateColumns {
		title = " Table: " + m.selectedTable
	}
	view.WriteString(m.styles.Title.Render(title))
	view.WriteString("\n")

	m = m.updateViewportDimensions()
	if m.state == StateColumns {
		// Render tabs
		tabs := []string{}
		colStyle := m.styles.TabInactive
		if m.activeTab == TabColumns {
			colStyle = m.styles.TabActive
		}
		tabs = append(tabs, colStyle.Render(" Columns"))
		conStyle := m.styles.TabInactive
		if m.activeTab == TabConstraints {
			conStyle = m.styles.TabActive
		}
		tabs = append(tabs, conStyle.Render(" Constraints"))

		view.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
		view.WriteString("\n\n")
	}

	m.viewport.SetContent(m.renderContent())
	m.viewport.SetContent(m.renderContent())
	view.WriteString(m.viewport.View())

	// Help footer
	view.WriteString("\n")
	view.WriteString(lipgloss.NewStyle().Faint(true).Render("enter: details • t: template • e: export • o: import • ?: help"))
	if m.state == StateColumns {
		view.WriteString(lipgloss.NewStyle().Faint(true).Render(" • l/h: tabs • esc: back"))
	} else {
		view.WriteString(lipgloss.NewStyle().Faint(true).Render(" • tab: close"))
	}

	return m.styles.Container.
		Width(popupWidth).
		Height(popupHeight).
		Render(view.String())
}

func (m Model) getPopupSize() (int, int) {
	popupWidth := int(float64(m.width) * 0.9)
	if popupWidth > 100 {
		popupWidth = 100
	}
	popupHeight := int(float64(m.height) * 0.8)
	if popupHeight > 35 {
		popupHeight = 35
	}
	return popupWidth, popupHeight
}

func (m Model) renderContent() string {
	var content strings.Builder
	popupWidth, _ := m.getPopupSize()

	if m.state == StateTables {
		for i, table := range m.tables {
			style := m.styles.Item
			prefix := "  "
			if i == m.selectedIdx {
				style = m.styles.ItemActive
				prefix = " "
			}
			content.WriteString(style.Render(prefix + table))
			content.WriteString("\n")
		}
		if len(m.tables) == 0 {
			content.WriteString(m.styles.Item.Render("  (No tables found)"))
		}
	} else {
		if m.activeTab == TabColumns {
			m.columnsTable = m.columnsTable.WithTargetWidth(popupWidth - 8)
			content.WriteString(m.columnsTable.View())
		} else {
			cons := m.constraints[m.selectedTable]
			if len(cons) == 0 {
				content.WriteString(m.styles.TableCell.Render("  (No constraints found)"))
				content.WriteString("\n")
			} else {
				m.constraintsTable = m.constraintsTable.WithTargetWidth(popupWidth - 8)
				content.WriteString(m.constraintsTable.View())
			}
		}
	}
	return content.String()
}

// Width for popups is handled internally, but we can return 0 to app.go
func (m Model) Width() int {
	return 0
}
