// internal/ui/styles.go
package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/nhath/ezdb/internal/config"
)

var (
	// Colors (exported via getter functions below)
	textPrimary   lipgloss.Color
	textSecondary lipgloss.Color
	textFaint     lipgloss.Color

	accentColor    lipgloss.Color
	successColor   lipgloss.Color
	errorColor     lipgloss.Color
	highlightColor lipgloss.Color
	warningColor   lipgloss.Color

	bgPrimary   lipgloss.Color
	bgSecondary lipgloss.Color
	cardBg      lipgloss.Color
	popupBg     lipgloss.Color
	borderColor lipgloss.Color
	selectedBg  lipgloss.Color

	// Styles
	StatusBarStyle          lipgloss.Style
	ModeStyle               lipgloss.Style
	InsertModeStyle         lipgloss.Style
	ConnectionStyle         lipgloss.Style
	QueryStyle              lipgloss.Style
	MetaStyle               lipgloss.Style
	SelectionStyle          lipgloss.Style
	ItemStyle               lipgloss.Style
	InputStyle              lipgloss.Style
	PromptStyle             lipgloss.Style
	SuggestionBoxStyle      lipgloss.Style
	SuggestionItemStyle     lipgloss.Style
	SuggestionSelectedStyle lipgloss.Style
	SuccessStyle            lipgloss.Style
	ErrorStyle              lipgloss.Style
	ErrorGrayStyle          lipgloss.Style
	SystemMessageStyle      lipgloss.Style
	WarningStyle            lipgloss.Style
	PopupStyle              lipgloss.Style
)

// Color getter functions for use in components
func TextPrimary() lipgloss.Color    { return textPrimary }
func TextSecondary() lipgloss.Color  { return textSecondary }
func TextFaint() lipgloss.Color      { return textFaint }
func AccentColor() lipgloss.Color    { return accentColor }
func SuccessColor() lipgloss.Color   { return successColor }
func ErrorColor() lipgloss.Color     { return errorColor }
func HighlightColor() lipgloss.Color { return highlightColor }
func WarningColor() lipgloss.Color   { return warningColor }
func BgPrimary() lipgloss.Color      { return bgPrimary }
func BgSecondary() lipgloss.Color    { return bgSecondary }
func CardBg() lipgloss.Color         { return cardBg }
func PopupBg() lipgloss.Color        { return popupBg }
func BorderColor() lipgloss.Color    { return borderColor }
func SelectedBg() lipgloss.Color     { return selectedBg }

// InitStyles initializes the global styles based on the provided configuration theme
func InitStyles(theme config.Theme) {
	// Initialize Colors
	textPrimary = lipgloss.Color(theme.TextPrimary)
	textSecondary = lipgloss.Color(theme.TextSecondary)
	textFaint = lipgloss.Color(theme.TextFaint)

	accentColor = lipgloss.Color(theme.Accent)
	successColor = lipgloss.Color(theme.Success)
	errorColor = lipgloss.Color(theme.Error)
	highlightColor = lipgloss.Color(theme.Highlight)
	warningColor = lipgloss.Color(theme.Warning)

	bgPrimary = lipgloss.Color(theme.BgPrimary)
	bgSecondary = lipgloss.Color(theme.BgSecondary)
	cardBg = lipgloss.Color(theme.CardBg)
	popupBg = lipgloss.Color(theme.PopupBg)
	borderColor = lipgloss.Color(theme.BorderColor)
	selectedBg = lipgloss.Color(theme.SelectedBg)

	// Initialize Styles
	StatusBarStyle = lipgloss.NewStyle().
		Foreground(textPrimary).
		Background(bgSecondary)

	ModeStyle = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Background(successColor).
		Foreground(bgPrimary)

	InsertModeStyle = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Background(accentColor).
		Foreground(bgPrimary)

	ConnectionStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Background(cardBg).
		Foreground(textPrimary)

	QueryStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(textPrimary)

	MetaStyle = lipgloss.NewStyle().
		Foreground(textFaint).
		Italic(true)

	SelectionStyle = lipgloss.NewStyle()
	ItemStyle = lipgloss.NewStyle()

	InputStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(textFaint).
		Padding(0, 0).
		MarginTop(0)

	PromptStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(accentColor).
		MarginRight(1)

	SuggestionBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(textFaint).
		// Background(bgPrimary).
		Padding(0, 1)

	SuggestionItemStyle = lipgloss.NewStyle().
		Foreground(textPrimary)

	SuggestionSelectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(highlightColor).
		Bold(true)

	SuccessStyle = lipgloss.NewStyle().
		Foreground(successColor)

	ErrorStyle = lipgloss.NewStyle().
		Foreground(errorColor).
		Bold(true)

	ErrorGrayStyle = lipgloss.NewStyle().
		Foreground(textFaint).
		Bold(true)

	SystemMessageStyle = lipgloss.NewStyle().
		Foreground(highlightColor).
		Bold(true)

	WarningStyle = lipgloss.NewStyle().
		Foreground(bgPrimary).
		Background(warningColor).
		Bold(true).
		Padding(0, 1)

	PopupStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(highlightColor).
		Padding(1, 2)
}
