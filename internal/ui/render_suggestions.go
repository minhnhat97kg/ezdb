package ui

import "strings"

// renderSuggestions renders the suggestion dropdown with type indicators
func (m Model) renderSuggestions() string {
	if len(m.suggestions) == 0 && !m.loadingTables {
		return ""
	}

	if m.loadingTables {
		return SuggestionBoxStyle.Render("Loading schema...")
	}

	var views []string
	// Limit to 8 suggestions for better visibility
	limit := 8
	start := 0
	if m.suggestionIdx > 3 {
		start = m.suggestionIdx - 3
	}
	end := start + limit
	if end > len(m.suggestions) {
		end = len(m.suggestions)
		if end-limit >= 0 {
			start = end - limit
		} else {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		s := m.suggestions[i]
		style := SuggestionItemStyle
		prefix := "  "
		if i == m.suggestionIdx {
			style = SuggestionSelectedStyle
			prefix = " ▶ "
		}

		// Add type indicator
		typeIndicator := ""
		if i < len(m.suggestionTypes) {
			switch m.suggestionTypes[i] {
			case SuggestKeyword:
				typeIndicator = " [K]"
			case SuggestTable:
				typeIndicator = " [T]"
			case SuggestColumn:
				typeIndicator = " [C]"
			case SuggestFunction:
				typeIndicator = " [F]"
			}
		}

		// Add detail (column type)
		detail := ""
		if i < len(m.suggestionDetails) && m.suggestionDetails[i] != "" {
			detail = " : " + m.suggestionDetails[i]
		}

		views = append(views, style.Render(prefix+s+typeIndicator+detail))
	}

	return SuggestionBoxStyle.Render(strings.Join(views, "\n"))
}
