package table

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	bbtable "github.com/evertras/bubble-table/table"
	"github.com/nhath/ezdb/internal/db"
)

// Nord colors (matching OpenCode theme)
const (
	ColorForeground = "#D8DEE9" // Nord4: Light gray
	ColorComment    = "#4C566A" // Nord3: Dark gray
	ColorCyan       = "#88C0D0" // Nord8: Cyan blue
	ColorGreen      = "#A3BE8C" // Nord14: Green
	ColorOrange     = "#D08770" // Nord12: Orange
	ColorPink       = "#B48EAD" // Nord15: Pink
	ColorPurple     = "#B48EAD" // Nord15: Purple
	ColorRed        = "#BF616A" // Nord11: Red
	ColorYellow     = "#EBCB8B" // Nord13: Yellow
	ColorTeal       = "#8FBCBB" // Nord7: Teal
)

// New creates a new bubble-table with Nord theme (no background)
func New(cols []bbtable.Column) bbtable.Model {
	return bbtable.New(cols).
		WithBaseStyle(lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorForeground))).
		HeaderStyle(lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTeal)).
			Bold(true)).
		HighlightStyle(lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGreen)).
			Bold(true)).
		Focused(true).
		BorderRounded()
}

// FromQueryResult builds a table from a QueryResult with type-specific coloring
func FromQueryResult(res *db.QueryResult) bbtable.Model {
	if res == nil {
		return bbtable.New(nil)
	}

	widths := calculateColumnWidths(res.Columns, res.Rows)
	var cols []bbtable.Column
	for _, c := range res.Columns {
		w := widths[c]
		if w > 40 {
			w = 40 // Cap max width
		}
		cols = append(cols, bbtable.NewColumn(c, c, w))
	}

	var rows []bbtable.Row
	for _, r := range res.Rows {
		rowData := bbtable.RowData{}
		for i, val := range r {
			rowData[res.Columns[i]] = bbtable.NewStyledCell(val, GetValueStyle(val))
		}
		rows = append(rows, bbtable.NewRow(rowData))
	}

	return New(cols).
		WithRows(rows).
		WithPageSize(20).
		WithStaticFooter("Press 'q' to close, 'a' for actions, '/' to filter")
}

// FromSchemaColumns builds a table for database columns metadata
func FromSchemaColumns(cols []db.Column) bbtable.Model {
	headers := []string{"Name", "Type", "Null", "Key", "Default"}
	var rowsData [][]string
	for _, c := range cols {
		nullStr := "YES"
		if !c.Nullable {
			nullStr = "NO"
		}
		rowsData = append(rowsData, []string{c.Name, c.Type, nullStr, c.Key, c.Default})
	}

	widths := calculateColumnWidths(headers, rowsData)
	tableCols := []bbtable.Column{}
	for _, h := range headers {
		tableCols = append(tableCols, bbtable.NewColumn(h, h, widths[h]))
	}

	var rows []bbtable.Row
	for _, rd := range rowsData {
		rows = append(rows, bbtable.NewRow(bbtable.RowData{
			"Name":    rd[0],
			"Type":    rd[1],
			"Null":    rd[2],
			"Key":     bbtable.NewStyledCell(rd[3], lipgloss.NewStyle().Foreground(lipgloss.Color(ColorYellow))),
			"Default": rd[4],
		}))
	}

	return New(tableCols).WithRows(rows)
}

// FromConstraints builds a table for database constraints metadata
func FromConstraints(constraints []db.Constraint) bbtable.Model {
	headers := []string{"Name", "Type", "Definition"}
	var rowsData [][]string
	for _, c := range constraints {
		rowsData = append(rowsData, []string{c.Name, c.Type, c.Definition})
	}

	widths := calculateColumnWidths(headers, rowsData)
	cols := []bbtable.Column{}
	for _, h := range headers {
		w := widths[h]
		if h == "Definition" && w > 50 {
			w = 50
		}
		cols = append(cols, bbtable.NewColumn(h, h, w))
	}

	var rows []bbtable.Row
	for _, rd := range rowsData {
		rows = append(rows, bbtable.NewRow(bbtable.RowData{
			"Name":       rd[0],
			"Type":       rd[1],
			"Definition": rd[2],
		}))
	}

	return New(cols).WithRows(rows)
}

// FromPreview builds a table from a preview string (columns | columns\nrow | row)
func FromPreview(preview string) bbtable.Model {
	lines := strings.Split(preview, "\n")
	if len(lines) < 2 {
		return bbtable.New(nil)
	}

	// First line is columns
	colNames := strings.Split(lines[0], " | ")
	for i, name := range colNames {
		colNames[i] = strings.TrimSpace(name)
	}

	var rowDataStrings [][]string
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || line == "..." {
			continue
		}
		parts := strings.Split(line, " | ")
		for j, p := range parts {
			parts[j] = strings.TrimSpace(p)
		}
		rowDataStrings = append(rowDataStrings, parts)
	}

	widths := calculateColumnWidths(colNames, rowDataStrings)
	var cols []bbtable.Column
	for _, name := range colNames {
		w := widths[name]
		if w > 40 {
			w = 40
		}
		cols = append(cols, bbtable.NewColumn(name, name, w))
	}

	var rows []bbtable.Row
	for _, rds := range rowDataStrings {
		// Special handling for truncation marker
		if len(rds) == 1 && rds[0] == "..." {
			rowData := bbtable.RowData{}
			for _, name := range colNames {
				rowData[name] = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorComment)).Render("...")
			}
			rows = append(rows, bbtable.NewRow(rowData))
			continue
		}

		rowData := bbtable.RowData{}
		for j, val := range rds {
			if j < len(colNames) {
				rowData[colNames[j]] = bbtable.NewStyledCell(val, GetValueStyle(val))
			}
		}
		rows = append(rows, bbtable.NewRow(rowData))
	}

	return New(cols).WithRows(rows).WithNoPagination()
}

func calculateColumnWidths(headers []string, rows [][]string) map[string]int {
	widths := make(map[string]int)
	for _, h := range headers {
		widths[h] = len(h)
	}

	for _, row := range rows {
		for i, val := range row {
			if i < len(headers) {
				if len(val) > widths[headers[i]] {
					widths[headers[i]] = len(val)
				}
			}
		}
	}

	// Add padding
	for h := range widths {
		widths[h] += 2
	}

	return widths
}

// GetValueStyle returns a lipgloss style based on value content
func GetValueStyle(val string) lipgloss.Style {
	if val == "" || strings.ToUpper(val) == "NULL" || val == "<nil>" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPink)).Italic(true)
	}
	if _, err := fmt.Sscanf(val, "%f", new(float64)); err == nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPurple))
	}
	lower := strings.ToLower(val)
	if lower == "true" || lower == "false" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorOrange))
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorYellow))
}
