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

	"github.com/nhath/ezdb/internal/config"
	"github.com/nhath/ezdb/internal/db"
	"github.com/nhath/ezdb/internal/history"
)

// handleCommand processes commands starting with /
func (m Model) handleCommand(input string) (Model, tea.Cmd) {
	parts := strings.Fields(strings.TrimPrefix(input, "/"))
	if len(parts) == 0 {
		return m, nil
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "profile":
		return m.handleProfileCommand(args)
	case "export":
		return m.handleExportCommand(args)
	default:
		// Check for aliases in config
		if query, ok := m.config.Commands[cmd]; ok {
			m.loading = true
			return m, m.executeQueryCmd(query)
		}

		return m.addSystemMessage(fmt.Sprintf("Command '/%s' not found", cmd)), nil
	}
}

// handleProfileCommand handles /profile subcommands
func (m Model) handleProfileCommand(args []string) (Model, tea.Cmd) {
	if len(args) == 0 {
		// List profiles
		return m.listProfiles()
	}

	subCmd := args[0]
	switch subCmd {
	case "add":
		// /profile add <name> <dsn>
		if len(args) < 3 {
			return m.addSystemMessage("Usage: /profile add <name> <connection_string>"), nil
		}
		name := args[1]
		dsn := strings.Join(args[2:], " ")
		p, err := config.ParseDSN(name, dsn)
		if err != nil {
			return m.addSystemMessage(fmt.Sprintf("Error parsing DSN: %v", err)), nil
		}

		if err := m.config.AddProfile(p); err != nil {
			return m.addSystemMessage(fmt.Sprintf("Error adding profile: %v", err)), nil
		}
		return m.addSystemMessage(fmt.Sprintf("Profile '%s' added.", name)), nil

	case "delete":
		if len(args) < 2 {
			return m.addSystemMessage("Usage: /profile delete <name>"), nil
		}
		name := args[1]
		if err := m.config.DeleteProfile(name); err != nil {
			return m.addSystemMessage(fmt.Sprintf("Error deleting profile: %v", err)), nil
		}
		if m.profile != nil && m.profile.Name == name {
			return m.addSystemMessage(fmt.Sprintf("Profile '%s' deleted. Warning: Active profile removed.", name)), nil
		}
		return m.addSystemMessage(fmt.Sprintf("Profile '%s' deleted.", name)), nil

	case "list":
		return m.listProfiles()

	default:
		// Assume switch: /profile <name>
		name := subCmd
		p, err := m.config.GetProfile(name)
		if err != nil {
			return m.addSystemMessage(fmt.Sprintf("Profile '%s' not found.", name)), nil
		}

		// Create new driver
		var newDriver db.Driver
		switch p.Type {
		case "postgres":
			newDriver = &db.PostgresDriver{}
		case "mysql":
			newDriver = &db.MySQLDriver{}
		case "sqlite":
			newDriver = &db.SQLiteDriver{}
		default:
			return m.addSystemMessage(fmt.Sprintf("Unknown driver type: %s", p.Type)), nil
		}

		if m.driver != nil {
			m.driver.Close()
		}

		// Connect
		params := db.ConnectParams{
			Host:     p.Host,
			Port:     p.Port,
			User:     p.User,
			Password: p.Password,
			Database: p.Database,
		}

		if p.SSHHost != "" {
			params.SSHConfig = &db.SSHConfig{
				Host:     p.SSHHost,
				Port:     p.SSHPort,
				User:     p.SSHUser,
				Password: p.SSHPassword,
				KeyPath:  p.SSHKeyPath,
			}
		}

		if err := newDriver.Connect(params); err != nil {
			return m.addSystemMessage(fmt.Sprintf("Error connecting to '%s': %v", name, err)), nil
		}

		m.profile = p
		m.driver = newDriver
		return m.addSystemMessage(fmt.Sprintf("Switched to profile '%s'.", name)), nil
	}
}

func (m Model) listProfiles() (Model, tea.Cmd) {
	var lines []string
	lines = append(lines, "Available Profiles:")
	for i, p := range m.config.Profiles {
		active := ""
		if m.profile != nil && m.profile.Name == p.Name {
			active = " (active)"
		}
		lines = append(lines, fmt.Sprintf("%d. %s%s (%s)", i+1, p.Name, active, p.Type))
	}
	return m.addSystemMessage(strings.Join(lines, "\n")), nil
}

// handleExportCommand handles /export csv|json [filename]
func (m Model) handleExportCommand(args []string) (Model, tea.Cmd) {
	if m.results == nil || len(m.results.Rows) == 0 {
		return m.addSystemMessage("No results to export. Run a query first."), nil
	}

	format := "csv"
	filename := ""
	if len(args) > 0 {
		format = strings.ToLower(args[0])
	}
	if len(args) > 1 {
		filename = args[1]
	}

	// Generate filename if not provided
	if filename == "" {
		timestamp := time.Now().Format("20060102_150405")
		filename = fmt.Sprintf("export_%s.%s", timestamp, format)
	}

	// Ensure it has the right extension
	if format == "json" && !strings.HasSuffix(filename, ".json") {
		filename += ".json"
	} else if format == "csv" && !strings.HasSuffix(filename, ".csv") {
		filename += ".csv"
	}

	// Get absolute path (current directory)
	absPath, _ := filepath.Abs(filename)

	switch format {
	case "csv":
		return m.exportCSV(absPath)
	case "json":
		return m.exportJSON(absPath)
	default:
		return m.addSystemMessage(fmt.Sprintf("Unknown format '%s'. Use csv or json.", format)), nil
	}
}

// exportCSV exports the current results to a CSV file
func (m Model) exportCSV(filename string) (Model, tea.Cmd) {
	file, err := os.Create(filename)
	if err != nil {
		return m.addSystemMessage(fmt.Sprintf("Error creating file: %v", err)), nil
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write(m.results.Columns); err != nil {
		return m.addSystemMessage(fmt.Sprintf("Error writing header: %v", err)), nil
	}

	// Write rows
	for _, row := range m.results.Rows {
		if err := writer.Write(row); err != nil {
			return m.addSystemMessage(fmt.Sprintf("Error writing row: %v", err)), nil
		}
	}

	return m.addSystemMessage(fmt.Sprintf("Exported %d rows to %s", len(m.results.Rows), filename)), nil
}

// exportJSON exports the current results to a JSON file
func (m Model) exportJSON(filename string) (Model, tea.Cmd) {
	// Convert to array of maps
	data := make([]map[string]interface{}, len(m.results.Rows))
	for i, row := range m.results.Rows {
		record := make(map[string]interface{})
		for j, col := range m.results.Columns {
			if j < len(row) {
				record[col] = row[j]
			}
		}
		data[i] = record
	}

	file, err := os.Create(filename)
	if err != nil {
		return m.addSystemMessage(fmt.Sprintf("Error creating file: %v", err)), nil
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return m.addSystemMessage(fmt.Sprintf("Error encoding JSON: %v", err)), nil
	}

	return m.addSystemMessage(fmt.Sprintf("Exported %d rows to %s", len(m.results.Rows), filename)), nil
}

// addSystemMessage adds a system message to the history
func (m Model) addSystemMessage(msg string) Model {
	entry := history.HistoryEntry{
		ID:         time.Now().UnixNano(),
		Query:      msg,
		Status:     "info",
		ExecutedAt: time.Now(),
	}
	m.history = append(m.history, entry)

	m.selected = len(m.history) - 1

	m.viewport.SetContent(m.renderHistoryContent(m.viewport.Height))
	m.viewport.GotoBottom()
	return m
}
