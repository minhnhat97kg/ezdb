package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nhath/ezdb/internal/ui/icons"
	"github.com/nhath/ezdb/internal/ui/styles"
)

func (m Model) renderStatusBar() string {
	var parts []string

	// 1. Mode
	modeStr := strings.ToUpper(string(m.mode))
	modeStyle := styles.ModeStyle
	if m.mode == InsertMode {
		modeStyle = styles.InsertModeStyle
	}
	parts = append(parts, modeStyle.Render(modeStr))

	// 2. Connection Info
	if m.profile != nil {
		icon := icons.GetDatabaseIcon(m.profile.Type)
		profileInfo := styles.ConnectionStyle.Render(fmt.Sprintf(" %s %s ", icon, m.profile.Name))

		dbInfo := fmt.Sprintf(" %s@%s:%d/%s ", m.profile.User, limitString(m.profile.Host, 20), m.profile.Port, m.profile.Database)
		if m.profile.Type == "sqlite" {
			dbInfo = fmt.Sprintf(" sqlite:%s ", m.profile.Database)
		}

		parts = append(parts, profileInfo+lipgloss.NewStyle().Background(styles.CardBg()).Foreground(styles.TextPrimary()).Render(dbInfo))
	} else {
		parts = append(parts, styles.ConnectionStyle.Render(" NO PROFILE "))
	}

	// 3. Strict Mode
	if m.strictMode {
		parts = append(parts, lipgloss.NewStyle().Background(styles.WarningColor()).Foreground(styles.BgPrimary()).Padding(0, 1).Bold(true).Render(icons.IconLock+" STRICT "))
	}

	// 4. Loading indicator
	if m.loading {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := spinner[int(time.Now().UnixMilli()/100)%len(spinner)]
		loadingStyle := lipgloss.NewStyle().Foreground(styles.AccentColor()).Padding(0, 1)
		parts = append(parts, loadingStyle.Render(frame+" Running..."))
	} else if m.loadingTables {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := spinner[int(time.Now().UnixMilli()/100)%len(spinner)]
		loadingStyle := lipgloss.NewStyle().Foreground(styles.HighlightColor()).Padding(0, 1)
		parts = append(parts, loadingStyle.Render(frame+" Loading schema..."))
	}

	// 5. Status message (success/info)
	if m.statusMsg != "" {
		statusStyle := lipgloss.NewStyle().Background(styles.SuccessColor()).Foreground(styles.BgPrimary()).Padding(0, 1)
		parts = append(parts, statusStyle.Render(icons.IconSuccess+" "+m.statusMsg))
	}

	// 6. Error indicator
	if m.errorMsg != "" {
		errorStyle := lipgloss.NewStyle().Background(styles.ErrorColor()).Foreground(styles.TextPrimary()).Padding(0, 1)
		truncated := m.errorMsg
		if len(truncated) > 40 {
			truncated = truncated[:37] + "..."
		}
		parts = append(parts, errorStyle.Render(icons.IconError+" "+truncated))
	}

	content := lipgloss.JoinHorizontal(lipgloss.Left, parts...)
	return styles.StatusBarStyle.Width(m.width).Render(content)
}
