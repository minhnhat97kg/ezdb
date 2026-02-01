package highlight

import (
	"strings"
)

// SQL keywords
var sqlKeywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "AND": true, "OR": true,
	"INSERT": true, "INTO": true, "VALUES": true, "UPDATE": true, "SET": true,
	"DELETE": true, "CREATE": true, "TABLE": true, "DROP": true, "ALTER": true,
	"JOIN": true, "LEFT": true, "RIGHT": true, "INNER": true, "OUTER": true,
	"ON": true, "AS": true, "ORDER": true, "BY": true, "GROUP": true,
	"HAVING": true, "LIMIT": true, "OFFSET": true, "DISTINCT": true,
	"NULL": true, "NOT": true, "IN": true, "LIKE": true, "BETWEEN": true,
	"IS": true, "TRUE": true, "FALSE": true, "ASC": true, "DESC": true,
	"UNION": true, "ALL": true, "EXISTS": true, "CASE": true, "WHEN": true,
	"THEN": true, "ELSE": true, "END": true, "COUNT": true, "SUM": true,
	"AVG": true, "MIN": true, "MAX": true,
}

// ANSI foreground color codes (no background, no reset issues)
const (
	fgCyan   = "\x1b[38;5;110m" // Keywords - light cyan
	fgPurple = "\x1b[38;5;183m" // Numbers - purple
	fgGreen  = "\x1b[38;5;150m" // Strings - green
	fgOrange = "\x1b[38;5;209m" // Wildcards - orange
	fgGray   = "\x1b[38;5;253m" // Default - light gray
	fgReset  = "\x1b[39m"       // Reset foreground only (not all attributes)
)

// SQL returns syntax highlighted SQL using foreground-only ANSI codes
// This is used for plain text (history view queries)
func SQL(sql string) string {
	var result strings.Builder
	i := 0

	for i < len(sql) {
		c := sql[i]

		// Whitespace
		if c == ' ' || c == '\t' || c == '\n' {
			result.WriteByte(c)
			i++
			continue
		}

		// Star wildcard
		if c == '*' {
			result.WriteString(fgOrange)
			result.WriteByte('*')
			result.WriteString(fgReset)
			i++
			continue
		}

		// String literals
		if c == '\'' || c == '"' {
			quote := c
			j := i + 1
			for j < len(sql) && sql[j] != quote {
				j++
			}
			if j < len(sql) {
				j++ // include closing quote
			}
			result.WriteString(fgGreen)
			result.WriteString(sql[i:j])
			result.WriteString(fgReset)
			i = j
			continue
		}

		// Numbers
		if c >= '0' && c <= '9' {
			j := i
			for j < len(sql) && ((sql[j] >= '0' && sql[j] <= '9') || sql[j] == '.') {
				j++
			}
			result.WriteString(fgPurple)
			result.WriteString(sql[i:j])
			result.WriteString(fgReset)
			i = j
			continue
		}

		// Words (keywords or identifiers)
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' {
			j := i
			for j < len(sql) && ((sql[j] >= 'a' && sql[j] <= 'z') || (sql[j] >= 'A' && sql[j] <= 'Z') || (sql[j] >= '0' && sql[j] <= '9') || sql[j] == '_') {
				j++
			}
			word := sql[i:j]
			if sqlKeywords[strings.ToUpper(word)] {
				result.WriteString(fgCyan)
				result.WriteString(word)
				result.WriteString(fgReset)
			} else {
				result.WriteString(fgGray)
				result.WriteString(word)
				result.WriteString(fgReset)
			}
			i = j
			continue
		}

		// Other characters
		result.WriteByte(c)
		i++
	}

	return result.String()
}

// SQLPreserveANSI highlights SQL while preserving existing ANSI escape sequences
// This is used for textarea views that already contain cursor/styling ANSI codes
func SQLPreserveANSI(text string) string {
	var result strings.Builder
	i := 0

	for i < len(text) {
		c := text[i]

		// Preserve existing ANSI escape sequences
		if c == '\x1b' && i+1 < len(text) && text[i+1] == '[' {
			j := i + 2
			for j < len(text) && !((text[j] >= 'A' && text[j] <= 'Z') || (text[j] >= 'a' && text[j] <= 'z')) {
				j++
			}
			if j < len(text) {
				j++ // include the terminating letter
			}
			result.WriteString(text[i:j])
			i = j
			continue
		}

		// Whitespace
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			result.WriteByte(c)
			i++
			continue
		}

		// Star wildcard
		if c == '*' {
			result.WriteString(fgOrange)
			result.WriteByte('*')
			result.WriteString(fgReset)
			i++
			continue
		}

		// String literals (be careful with ANSI codes inside)
		if c == '\'' || c == '"' {
			quote := c
			result.WriteString(fgGreen)
			result.WriteByte(c)
			i++
			for i < len(text) && text[i] != quote {
				if text[i] == '\x1b' {
					// Preserve ANSI inside string
					j := i + 2
					for j < len(text) && !((text[j] >= 'A' && text[j] <= 'Z') || (text[j] >= 'a' && text[j] <= 'z')) {
						j++
					}
					if j < len(text) {
						j++
					}
					result.WriteString(text[i:j])
					i = j
				} else {
					result.WriteByte(text[i])
					i++
				}
			}
			if i < len(text) {
				result.WriteByte(text[i]) // closing quote
				i++
			}
			result.WriteString(fgReset)
			continue
		}

		// Numbers
		if c >= '0' && c <= '9' {
			result.WriteString(fgPurple)
			for i < len(text) && ((text[i] >= '0' && text[i] <= '9') || text[i] == '.') {
				result.WriteByte(text[i])
				i++
			}
			result.WriteString(fgReset)
			continue
		}

		// Words (keywords or identifiers)
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' {
			j := i
			for j < len(text) && ((text[j] >= 'a' && text[j] <= 'z') || (text[j] >= 'A' && text[j] <= 'Z') || (text[j] >= '0' && text[j] <= '9') || text[j] == '_') {
				j++
			}
			word := text[i:j]
			if sqlKeywords[strings.ToUpper(word)] {
				result.WriteString(fgCyan)
				result.WriteString(word)
				result.WriteString(fgReset)
			} else {
				result.WriteString(fgGray)
				result.WriteString(word)
				result.WriteString(fgReset)
			}
			i = j
			continue
		}

		// Other characters
		result.WriteByte(c)
		i++
	}

	return result.String()
}
