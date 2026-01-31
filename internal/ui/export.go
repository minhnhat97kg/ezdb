package ui

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ExportCompleteMsg is sent when export is complete
type ExportCompleteMsg struct {
	Path string
	Err  error
}

// exportTable exports the currently displayed table data to a file
func (m Model) exportTable() tea.Cmd {
	if m.popupResult == nil {
		return nil
	}

	return func() tea.Msg {
		// Generate filename with timestamp
		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("export_%s_%s.csv", m.popupEntry.ProfileName, timestamp)
		filepath := filepath.Join(os.TempDir(), filename)

		// Create file
		f, err := os.Create(filepath)
		if err != nil {
			return ExportCompleteMsg{Err: err}
		}
		defer f.Close()

		// Write CSV
		w := csv.NewWriter(f)
		defer w.Flush()

		// Write header
		if err := w.Write(m.popupResult.Columns); err != nil {
			return ExportCompleteMsg{Err: err}
		}

		// Write rows
		for _, row := range m.popupResult.Rows {
			if err := w.Write(row); err != nil {
				return ExportCompleteMsg{Err: err}
			}
		}

		return ExportCompleteMsg{Path: filepath}
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
