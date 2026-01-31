// internal/ui/model_helpers.go
// Small helper functions used across the UI layer
package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
)

// isModifyingQuery returns true if the SQL statement is a write operation
func isModifyingQuery(query string) bool {
	q := strings.TrimSpace(strings.ToUpper(query))
	modifyingOps := []string{
		"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "TRUNCATE", "CREATE", "REPLACE",
	}
	for _, op := range modifyingOps {
		if strings.HasPrefix(q, op) {
			return true
		}
	}
	return false
}

// matchKey returns true if the key message matches any of the provided key strings
func matchKey(msg tea.KeyMsg, keys []string) bool {
	keyStr := msg.String()
	for _, k := range keys {
		if k == keyStr {
			return true
		}
	}
	return false
}

// unwrapCellValue extracts the raw value from a bubble-table StyledCell if necessary.
// Since StyledCell fields might be unexported or hard to access, we use a robust string check.
func unwrapCellValue(val interface{}) interface{} {
	if _, ok := val.(table.StyledCell); ok {
		// Formatted struct looks like {Value Style ...}
		// e.g. {3 [38;2;...}
		s := fmt.Sprintf("%v", val)
		s = strings.TrimPrefix(s, "{")
		if idx := strings.Index(s, " "); idx != -1 {
			return s[:idx]
		}
		// Fallback: return the whole string if parsing fails, but cleaner
		return strings.TrimSuffix(s, "}")
	}
	return val
}

// limitString truncates s to maxLen by replacing the middle with "..."
func limitString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// replace middle with ...
	half := (maxLen - 3) / 2
	return s[:half] + "..." + s[len(s)-half:]
}
