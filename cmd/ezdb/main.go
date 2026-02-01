// cmd/ezdb/main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nhath/ezdb/internal/config"
	"github.com/nhath/ezdb/internal/history"
	"github.com/nhath/ezdb/internal/ui"
	"github.com/nhath/ezdb/internal/ui/components/table"
	"github.com/nhath/ezdb/internal/ui/styles"
)

func main() {
	// Parse flags
	debug := flag.Bool("debug", false, "Enable debug logging to debug.log")
	flag.Parse()

	// Setup logging if debug enabled
	if *debug {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Printf("fatal: could not open debug log: %v", err)
			os.Exit(1)
		}
		defer f.Close()
		log.SetOutput(f) // Redirect standard log to the same file
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize UI styles
	styles.Init(cfg.Theme)
	table.Init(cfg.Theme, cfg.Keys)

	// Initialize history store
	historyStore, err := history.NewStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize history: %v\n", err)
		os.Exit(1)
	}
	defer historyStore.Close()

	// Create TUI with profile selector (no pre-connection)
	// The TUI will handle profile selection and connection
	model := ui.NewModel(cfg, nil, nil, historyStore)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}

	// Clear any leftover output from pagers (pspg, less, etc.)
	// by printing a clear screen sequence
	fmt.Print("\033[H\033[2J")
}
