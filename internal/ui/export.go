package ui

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ExportCompleteMsg is sent when export is complete
type ExportCompleteMsg struct {
	Path string
	Err  error
}

// exportTableToPath exports all query results to a specified path
func (m Model) exportTableToPath(filename string) tea.Cmd {
	if m.popupResult == nil {
		return nil
	}

	// Capture result data for the closure
	columns := m.popupResult.Columns
	rows := m.popupResult.Rows

	return func() tea.Msg {
		// Expand path
		exportPath := filename
		if !filepath.IsAbs(exportPath) {
			cwd, err := os.Getwd()
			if err != nil {
				cwd = "."
			}
			exportPath = filepath.Join(cwd, filename)
		}

		// Ensure .csv extension
		if !strings.HasSuffix(strings.ToLower(exportPath), ".csv") {
			exportPath += ".csv"
		}

		// Create file
		f, err := os.Create(exportPath)
		if err != nil {
			return ExportCompleteMsg{Err: err}
		}
		defer f.Close()

		// Write CSV with | separator
		w := csv.NewWriter(f)
		w.Comma = '|'
		defer w.Flush()

		// Write header
		if err := w.Write(columns); err != nil {
			return ExportCompleteMsg{Err: err}
		}

		// Write ALL rows
		for _, row := range rows {
			if err := w.Write(row); err != nil {
				return ExportCompleteMsg{Err: err}
			}
		}

		return ExportCompleteMsg{Path: exportPath}
	}
}

// copyRowAsJSON copies the currently highlighted row as JSON
func (m Model) copyRowAsJSON() tea.Cmd {
	if m.popupResult == nil {
		return nil
	}

	highlightedRow := m.popupTable.HighlightedRow()
	if highlightedRow.Data == nil {
		return nil
	}

	return func() tea.Msg {
		// Convert row data to map
		rowMap := make(map[string]interface{})
		for key, value := range highlightedRow.Data {
			rowMap[key] = value
		}

		// Convert to JSON
		jsonBytes, err := json.MarshalIndent(rowMap, "", "  ")
		if err != nil {
			return ClipboardCopiedMsg{Err: err}
		}

		// Copy to clipboard
		return m.copyToClipboardCmd(string(jsonBytes))()
	}
}

// copyRowAsCSV copies the currently highlighted row as CSV
func (m Model) copyRowAsCSV() tea.Cmd {
	if m.popupResult == nil {
		return nil
	}

	highlightedRow := m.popupTable.HighlightedRow()
	if highlightedRow.Data == nil {
		return nil
	}

	return func() tea.Msg {
		var b strings.Builder
		w := csv.NewWriter(&b)

		// Write row values in column order
		row := make([]string, len(m.popupResult.Columns))
		for i, col := range m.popupResult.Columns {
			if val, ok := highlightedRow.Data[col]; ok {
				row[i] = fmt.Sprintf("%v", val)
			}
		}

		if err := w.Write(row); err != nil {
			return ClipboardCopiedMsg{Err: err}
		}

		w.Flush()

		// Copy to clipboard
		return m.copyToClipboardCmd(b.String())()
	}
}
