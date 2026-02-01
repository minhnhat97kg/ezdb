// internal/ui/model_types.go
// Type definitions for the UI layer following superfile patterns
package ui

// Mode represents the current UI mode (vim-style)
type Mode string

// AppState represents the overall application state
type AppState string

const (
	InsertMode Mode = "INSERT"
	VisualMode Mode = "VISUAL"
)

const (
	StateSelectingProfile AppState = "SELECTING_PROFILE"
	StateConnecting       AppState = "CONNECTING"
	StateReady            AppState = "READY"
)

// HelpContext represents the current UI context for help display
type HelpContext int

const (
	HelpContextVisual HelpContext = iota
	HelpContextInsert
	HelpContextPopup
	HelpContextSchema
)
