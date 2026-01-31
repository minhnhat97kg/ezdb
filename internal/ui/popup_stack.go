package ui

// PopupCloser is a function that closes a popup and returns true if it was open
type PopupCloser func(*Model) bool

// PopupStack manages a stack of popup closers for proper layered popup handling
// When Esc/q is pressed, the topmost popup is closed first
type PopupStack struct {
	closers []PopupCloser
	names   []string // For debugging
}

// NewPopupStack creates a new popup stack
func NewPopupStack() *PopupStack {
	return &PopupStack{
		closers: make([]PopupCloser, 0),
		names:   make([]string, 0),
	}
}

// Push adds a closer to the stack
func (s *PopupStack) Push(name string, closer PopupCloser) {
	s.closers = append(s.closers, closer)
	s.names = append(s.names, name)
}

// Pop removes and returns the topmost closer, returns nil if empty
func (s *PopupStack) Pop() PopupCloser {
	if len(s.closers) == 0 {
		return nil
	}
	closer := s.closers[len(s.closers)-1]
	s.closers = s.closers[:len(s.closers)-1]
	s.names = s.names[:len(s.names)-1]
	return closer
}

// CloseTop closes the topmost popup and removes it from the stack
// Returns true if a popup was closed, false if stack was empty
func (s *PopupStack) CloseTop(m *Model) bool {
	closer := s.Pop()
	if closer == nil {
		return false
	}
	return closer(m)
}

// IsEmpty returns true if no popups are open
func (s *PopupStack) IsEmpty() bool {
	return len(s.closers) == 0
}

// Len returns the number of open popups
func (s *PopupStack) Len() int {
	return len(s.closers)
}

// Clear closes all popups and clears the stack
// func (s *PopupStack) Clear() {
// 	for !s.IsEmpty() {
// 		s.CloseTop()
// 	}
// }

// TopName returns the name of the topmost popup for debugging
func (s *PopupStack) TopName() string {
	if len(s.names) == 0 {
		return ""
	}
	return s.names[len(s.names)-1]
}
