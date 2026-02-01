package ui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nhath/ezdb/internal/db"
	"github.com/nhath/ezdb/internal/history"
)

// executeQueryCmd executes a query (or multiple queries split by ;) asynchronously
func (m Model) executeQueryCmd(query string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Split by semicolon for multi-statement execution
		statements := splitStatements(query)
		if len(statements) == 0 {
			return QueryResultMsg{Err: db.WrapQueryError(nil)}
		}

		var lastResult *db.QueryResult
		var lastEntry *history.HistoryEntry
		var allEntries []*history.HistoryEntry

		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}

			start := time.Now()
			result, err := m.driver.Execute(ctx, stmt)
			if err != nil {
				// Save error to history
				entry := &history.HistoryEntry{
					ProfileName:  m.profile.Name,
					Query:        stmt,
					ExecutedAt:   time.Now(),
					DurationMs:   time.Since(start).Milliseconds(),
					RowCount:     0,
					Status:       "error",
					ErrorMessage: err.Error(),
				}
				m.historyStore.Add(entry)
				return QueryResultMsg{Err: err, Entry: entry}
			}

			var previewBuilder strings.Builder
			if len(result.Rows) > 0 {
				previewBuilder.WriteString(strings.Join(result.Columns, " | "))
				previewBuilder.WriteString("\n")
				limit := m.config.HistoryPreviewRows
				if len(result.Rows) < limit {
					limit = len(result.Rows)
				}
				for i := 0; i < limit; i++ {
					previewBuilder.WriteString(strings.Join(result.Rows[i], " | "))
					previewBuilder.WriteString("\n")
				}
				if len(result.Rows) > m.config.HistoryPreviewRows {
					previewBuilder.WriteString("...")
				}
			}

			entry := &history.HistoryEntry{
				ProfileName: m.profile.Name,
				Query:       stmt,
				ExecutedAt:  time.Now(),
				DurationMs:  result.ExecTime.Milliseconds(),
				RowCount:    result.RowCount,
				Status:      "success",
				Preview:     strings.TrimSpace(previewBuilder.String()),
			}
			m.historyStore.Add(entry)
			allEntries = append(allEntries, entry)
			lastResult = result
			lastEntry = entry
		}

		// Return last result for display
		return QueryResultMsg{Result: lastResult, Entry: lastEntry, AllEntries: allEntries}
	}
}

// splitStatements splits a query string by semicolons, respecting quotes
func splitStatements(query string) []string {
	var statements []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false

	for i := 0; i < len(query); i++ {
		c := query[i]

		// Handle escape sequences
		if (inSingleQuote || inDoubleQuote) && c == '\\' && i+1 < len(query) {
			current.WriteByte(c)
			i++
			current.WriteByte(query[i])
			continue
		}

		// Toggle quote state
		if c == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
		} else if c == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
		}

		// Split on semicolon outside quotes
		if c == ';' && !inSingleQuote && !inDoubleQuote {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
			continue
		}

		current.WriteByte(c)
	}

	// Don't forget the last statement
	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}

// rerunQueryCmd re-runs a query from history
func (m Model) rerunQueryCmd(entry *history.HistoryEntry) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := m.driver.Execute(ctx, entry.Query)
		if err != nil {
			return RerunResultMsg{Err: err, Entry: entry}
		}

		return RerunResultMsg{Result: result, Entry: entry}
	}
}
