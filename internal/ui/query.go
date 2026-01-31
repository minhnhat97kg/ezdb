package ui

import (
	"context"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nhath/ezdb/internal/config"
	"github.com/nhath/ezdb/internal/db"
	"github.com/nhath/ezdb/internal/history"
)

// copyToClipboardCmd copies text to clipboard using pbcopy (macOS)
func (m Model) copyToClipboardCmd(text string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("pbcopy")
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return ClipboardCopiedMsg{Err: err}
		}

		if err := cmd.Start(); err != nil {
			return ClipboardCopiedMsg{Err: err}
		}

		stdin.Write([]byte(text))
		stdin.Close()

		if err := cmd.Wait(); err != nil {
			return ClipboardCopiedMsg{Err: err}
		}

		return ClipboardCopiedMsg{Text: text}
	}
}

// connectToProfileCmd connects to the selected profile
func (m Model) connectToProfileCmd(profile *config.Profile) tea.Cmd {
	return func() tea.Msg {
		var driverType db.DriverType
		switch profile.Type {
		case "postgres":
			driverType = db.Postgres
		case "mysql":
			driverType = db.MySQL
		case "sqlite":
			driverType = db.SQLite
		default:
			return ProfileConnectedMsg{Err: db.WrapConnectionError(nil)}
		}

		driver, err := db.NewDriver(driverType)
		if err != nil {
			return ProfileConnectedMsg{Err: err}
		}

		// Use password from profile
		password := profile.Password
		if password == "" && profile.Type != "sqlite" {
			// Fallback to keyring for existing profiles not yet migrated to config
			keyringStore, err := config.NewKeyringStore()
			if err == nil {
				password, _ = keyringStore.GetPassword(profile.Name)
			}
		}

		params := db.ConnectParams{
			Host:     profile.Host,
			Port:     profile.Port,
			User:     profile.User,
			Password: password,
			Database: profile.Database,
		}

		if profile.SSHHost != "" {
			params.SSHConfig = &db.SSHConfig{
				Host:     profile.SSHHost,
				Port:     profile.SSHPort,
				User:     profile.SSHUser,
				Password: profile.SSHPassword,
				KeyPath:  profile.SSHKeyPath,
			}
		}

		if err := driver.Connect(params); err != nil {
			return ProfileConnectedMsg{Err: err}
		}

		return ProfileConnectedMsg{Driver: driver}
	}
}

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

// loadHistoryCmd loads query history from SQLite
func (m Model) loadHistoryCmd() tea.Cmd {
	return func() tea.Msg {
		entries, err := m.historyStore.List(m.profile.Name, 100, 0)
		return HistoryLoadedMsg{Entries: entries, Err: err}
	}
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

// fetchTablesCmd fetches tables and columns from the database for autocomplete
