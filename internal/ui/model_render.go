package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nhath/ezdb/internal/ui/styles"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Show profile selector if not connected
	if m.appState == StateSelectingProfile || m.appState == StateConnecting {
		view := m.profileSelector.View()
		if m.appState == StateConnecting {
			// Show connecting status
			connectingStyle := lipgloss.NewStyle().
				Foreground(styles.AccentColor()).
				Bold(true)
			status := connectingStyle.Render("Connecting to " + m.profile.Name + "...")
			view = lipgloss.JoinVertical(lipgloss.Center, view, status)
		}
		if m.connectError != "" {
			errorStyle := lipgloss.NewStyle().Foreground(styles.ErrorColor())
			view = lipgloss.JoinVertical(lipgloss.Center, view, errorStyle.Render("Error: "+m.connectError))
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, view)
	}

	// 1. Calculate dynamic editor height based on content
	// Count lines in editor content
	editorContent := m.editor.Value()
	lineCount := strings.Count(editorContent, "\n") + 1

	// Min 3 lines, max half viewport
	minHeight := 3
	maxHeight := m.height / 2
	if maxHeight < minHeight {
		maxHeight = minHeight
	}

	editorHeight := lineCount
	if editorHeight < minHeight {
		editorHeight = minHeight
	}
	if editorHeight > maxHeight {
		editorHeight = maxHeight
	}

	m.editor.SetHeight(editorHeight)

	// 2. Render Components
	inputWidth := m.width - 4
	inputView := styles.InputStyle.Width(inputWidth).Render(m.highlightView(m.editor.View()))

	statusBar := m.renderStatusBar()
	helpText := m.renderHelp()

	// 2. Calculate Content Height
	chromeHeight := lipgloss.Height(statusBar) + lipgloss.Height(helpText) + lipgloss.Height(inputView)
	availableHeight := m.height - chromeHeight
	if availableHeight < 0 {
		availableHeight = 0
	}

	historyHeight := availableHeight
	if historyHeight < 0 {
		historyHeight = 0
	}

	// 3. Render History Content (Viewport)
	m.viewport.Height = historyHeight
	historyView := m.viewport.View()

	// 4. Final Layout
	main := lipgloss.JoinVertical(lipgloss.Left,
		historyView,
		inputView,
		statusBar,
		helpText,
	)

	// Overlay popups if active
	if m.showPopup || m.confirming {
		main = m.renderPopupOverlay(main)
	}

	// Template popup overlay
	if m.showTemplatePopup {
		main = m.renderTemplatePopup(main)
	}

	// Import popup overlay
	if m.showImportPopup {
		main = m.renderImportPopup(main)
	}

	// Export popup overlay
	if m.showExportPopup {
		main = m.renderExportPopup(main)
	}

	// Theme Selector Overlay
	if m.themeSelector.Visible() {
		themeView := m.themeSelector.View(m.width, m.height)
		main = overlay.Composite(themeView, main, overlay.Center, overlay.Center, 0, 0)
	}

	// 5. Suggestions Overlay
	hasPopup := m.hasOpenPopup() || m.showPopup || m.showHelpPopup || m.showTemplatePopup ||
		m.showImportPopup || m.showExportPopup || m.showRowActionPopup || m.showActionPopup ||
		m.themeSelector.Visible()

	if m.autocompleting && m.mode == InsertMode && !hasPopup {
		suggestions := m.renderSuggestions()
		if suggestions != "" {
			// Calculate position relative to input area
			// Input input is at the bottom, above status bar and help text
			// Y = height - (StatusBar + Help + InputHeight) + 1 (border)
			// Actually, we want it *above* the input line
			// Input is at: m.height - (StatusBar=1 + Help=1 + InputHeight)
			// So bottom of suggestions should be at top of input?
			// Or below? Standard is usually below, but if we are at bottom of screen, maybe above?
			// Given the layout, the input bar is fixed at the bottom.
			// So suggestions should appear *above* the input box.

			// Let's try absolute positioning from bottom-left anchor
			// Input area is at bottom.
			// cursor Col gives X offset roughly.
			// Use lipgloss.Width to measure text before cursor
			cursorLine := m.editor.Line()
			lines := strings.Split(m.editor.Value(), "\n")
			var lineContent string
			if cursorLine < len(lines) {
				lineContent = lines[cursorLine]
			}
			x := lipgloss.Width(lineContent) + 2 // +2 for border/padding

			// Y: The overlay package coordinates are 0-indexed from top-left.
			// Input box bottom is at m.height - (StatusBar + Help)
			// Input box top is at m.height - (StatusBar + Help + EditorHeight + Border)
			// We want suggestions to appear *above* the current line of text in the editor?
			// Or below? Standard is usually below, but if we are at bottom of screen, maybe above?
			// Given the layout, the input bar is fixed at the bottom.
			// So suggestions should appear *above* the input box.

			inputHeight := lipgloss.Height(inputView)
			bottomOffset := 2 + inputHeight // Status(1) + Help(1) + Input

			// We want to position the suggestions box such that its bottom is at (Height - bottomOffset)
			// overlay.Composite places (0,0) of overlay at (x,y) of bg.
			// So y = m.height - bottomOffset - suggestionHeight

			suggestionHeight := lipgloss.Height(suggestions)
			y := m.height - bottomOffset - suggestionHeight

			if y < 0 {
				y = 0
			} // Clamp to top

			// X position
			// Constrain to width
			suggestionWidth := lipgloss.Width(suggestions)
			if x+suggestionWidth > m.width {
				x = m.width - suggestionWidth
			}

			// Use 0 (Left/Top) for absolute positioning with offsets
			// Assuming overlay.Left and overlay.Top constants exist, or 0/0 works
			// Existing code used overlay.Center, so assuming standardized naming
			// If Left/Top don't exist, we might need another approach, but let's try this.
			main = overlay.Composite(suggestions, main, 0, 0, x, y)
		}
	}

	if m.schemaBrowser.IsVisible() || m.loadingTables { // Show if visible OR loading (for spinner)
		m.schemaBrowser = m.schemaBrowser.SetSize(m.width, m.height)
		browser := m.schemaBrowser.View()
		if browser != "" {
			// Use bubbletea-overlay to composite schema browser over main content
			main = overlay.Composite(browser, main, overlay.Center, overlay.Center, 0, 0)
		}
	}

	// Help popup overlay (render last to be on top)
	if m.showHelpPopup {
		main = m.renderHelpPopup(main)
	}

	return main
}
