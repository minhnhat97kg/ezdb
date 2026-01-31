package ui

import (
	"encoding/csv"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nhath/ezdb/internal/db"
)

// openPager opens the result in an external pager
func (m Model) openPager(result *db.QueryResult) tea.Cmd {
	if m.config.Pager == "" || result == nil || len(result.Rows) == 0 {
		return nil
	}

	// Create temp file
	f, err := os.CreateTemp("", "ezdb-*.csv")
	if err != nil {
		return func() tea.Msg { return PagerFinishedMsg{Err: err} }
	}
	defer f.Close()

	// Write CSV
	w := csv.NewWriter(f)
	w.Comma = ','

	if err := w.Write(result.Columns); err != nil {
		return func() tea.Msg { return PagerFinishedMsg{Err: err} }
	}
	for _, row := range result.Rows {
		if err := w.Write(row); err != nil {
			return func() tea.Msg { return PagerFinishedMsg{Err: err} }
		}
	}
	w.Flush()

	// Command
	parts := strings.Fields(m.config.Pager)
	cmdName := parts[0]
	args := append(parts[1:], f.Name())

	c := exec.Command(cmdName, args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		os.Remove(f.Name()) // Cleanup
		return PagerFinishedMsg{Err: err}
	})
}
