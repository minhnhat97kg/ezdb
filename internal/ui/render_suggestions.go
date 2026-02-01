package ui

import (
	"strings"

	"github.com/nhath/ezdb/internal/ui/autocomplete"
	"github.com/nhath/ezdb/internal/ui/icons"
	"github.com/nhath/ezdb/internal/ui/styles"
)

// renderSuggestions renders the suggestion dropdown with type indicators
func (m Model) renderSuggestions() string {
	if len(m.suggestions) == 0 && !m.loadingTables {
		return ""
	}

	if m.loadingTables {
		return styles.SuggestionBoxStyle.Render("Loading schema...")
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
		style := styles.SuggestionItemStyle
		prefix := "  "
		if i == m.suggestionIdx {
			style = styles.SuggestionSelectedStyle
			prefix = " " + icons.IconSelect + " "
		}

		// Add type indicator
		typeIndicator := ""
		if i < len(m.suggestionTypes) {
			switch m.suggestionTypes[i] {
			case autocomplete.SuggestKeyword:
				typeIndicator = " " + icons.IconKeyword
			case autocomplete.SuggestTable:
				typeIndicator = " " + icons.IconTypeT
			case autocomplete.SuggestColumn:
				typeIndicator = " " + icons.IconTypeC
			case autocomplete.SuggestFunction:
				typeIndicator = " " + icons.IconTypeF
			default:
				typeIndicator = " " + icons.IconBullet
			}
		}

		// Add detail (column type)
		detail := ""
		if i < len(m.suggestionDetails) && m.suggestionDetails[i] != "" {
			detail = " : " + m.suggestionDetails[i]
		}

		views = append(views, style.Render(prefix+s+typeIndicator+detail))
	}

	return styles.SuggestionBoxStyle.Render(strings.Join(views, "\n"))
}
