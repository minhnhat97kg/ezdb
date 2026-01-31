package ui

import (
	"strings"
	"unicode"

	"github.com/nhath/ezdb/internal/db"
)

// SuggestionType indicates what kind of completion to show
type SuggestionType int

const (
	SuggestKeyword SuggestionType = iota
	SuggestTable
	SuggestColumn
	SuggestFunction
	SuggestAlias
)

// Suggestion represents a single autocomplete suggestion
type Suggestion struct {
	Text     string
	Type     SuggestionType
	Detail   string // e.g., column type, function signature
	Priority int    // Lower is higher priority
}

// SQL keywords organized by context
var (
	// Keywords that start statements
	statementKeywords = []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER",
		"TRUNCATE", "EXPLAIN", "DESCRIBE", "SHOW", "USE", "BEGIN", "COMMIT", "ROLLBACK",
	}

	// Keywords after SELECT
	selectKeywords = []string{
		"DISTINCT", "ALL", "TOP", "AS", "FROM",
	}

	// Keywords after FROM/JOIN
	fromKeywords = []string{
		"WHERE", "JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "CROSS", "NATURAL",
		"ON", "USING", "GROUP", "ORDER", "LIMIT", "OFFSET", "UNION", "EXCEPT", "INTERSECT",
	}

	// Keywords after WHERE/AND/OR
	whereKeywords = []string{
		"AND", "OR", "NOT", "IN", "BETWEEN", "LIKE", "ILIKE", "IS", "NULL",
		"TRUE", "FALSE", "EXISTS", "ANY", "ALL", "SOME",
	}

	// Keywords after GROUP BY
	groupKeywords = []string{
		"HAVING", "ORDER", "LIMIT", "OFFSET",
	}

	// Keywords after ORDER BY
	orderKeywords = []string{
		"ASC", "DESC", "NULLS", "FIRST", "LAST", "LIMIT", "OFFSET",
	}

	// Aggregate functions
	aggregateFunctions = []string{
		"COUNT", "SUM", "AVG", "MIN", "MAX", "GROUP_CONCAT", "STRING_AGG",
	}

	// Common SQL functions
	commonFunctions = []string{
		"COALESCE", "NULLIF", "CAST", "CONVERT", "CONCAT", "SUBSTRING", "LENGTH",
		"UPPER", "LOWER", "TRIM", "LTRIM", "RTRIM", "REPLACE", "NOW", "CURRENT_DATE",
		"CURRENT_TIME", "CURRENT_TIMESTAMP", "DATE", "TIME", "DATETIME", "YEAR", "MONTH", "DAY",
		"ROUND", "FLOOR", "CEIL", "ABS", "MOD", "IFNULL", "NVL", "CASE", "WHEN", "THEN", "ELSE", "END",
	}

	// All keywords combined for legacy compatibility
	SQLKeywords = append(append(append(append(append(append(
		statementKeywords, selectKeywords...), fromKeywords...), whereKeywords...),
		groupKeywords...), orderKeywords...), aggregateFunctions...)
)

// SQLContext represents the parsed context of a SQL statement
type SQLContext struct {
	StatementType string   // SELECT, INSERT, UPDATE, DELETE, etc.
	LastKeyword   string   // Most recent keyword before cursor
	Tables        []string // Tables referenced in the query (with aliases)
	TableAliases  map[string]string // alias -> table name
	InSelect      bool
	InFrom        bool
	InWhere       bool
	InJoin        bool
	InGroupBy     bool
	InOrderBy     bool
	InHaving      bool
	InInsert      bool
	InUpdate      bool
	InSet         bool
	AfterDot      bool     // After a "." for qualified names
	Qualifier     string   // Table/alias before the dot
}

// ParseSQLContext analyzes SQL text up to cursor position to determine context
func ParseSQLContext(sql string, cursorPos int) SQLContext {
	if cursorPos > len(sql) {
		cursorPos = len(sql)
	}
	textBeforeCursor := strings.ToUpper(sql[:cursorPos])

	ctx := SQLContext{
		TableAliases: make(map[string]string),
	}

	// Tokenize
	tokens := tokenizeSQL(textBeforeCursor)

	// Find statement type
	for _, t := range tokens {
		switch t {
		case "SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER":
			ctx.StatementType = t
			break
		}
		if ctx.StatementType != "" {
			break
		}
	}

	// Track context based on keywords
	for i, t := range tokens {
		switch t {
		case "SELECT":
			ctx.InSelect = true
			ctx.InFrom = false
			ctx.InWhere = false
		case "FROM", "INTO":
			ctx.InSelect = false
			ctx.InFrom = true
			ctx.InWhere = false
		case "JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "CROSS":
			ctx.InJoin = true
			ctx.InFrom = true
		case "ON", "USING":
			ctx.InJoin = false
		case "WHERE":
			ctx.InFrom = false
			ctx.InWhere = true
			ctx.InJoin = false
		case "GROUP":
			if i+1 < len(tokens) && tokens[i+1] == "BY" {
				ctx.InGroupBy = true
				ctx.InWhere = false
			}
		case "ORDER":
			if i+1 < len(tokens) && tokens[i+1] == "BY" {
				ctx.InOrderBy = true
				ctx.InGroupBy = false
			}
		case "HAVING":
			ctx.InHaving = true
			ctx.InGroupBy = false
		case "SET":
			ctx.InSet = true
			ctx.InFrom = false
		case "INSERT":
			ctx.InInsert = true
		case "UPDATE":
			ctx.InUpdate = true
		case "VALUES":
			ctx.InInsert = false
		}
		ctx.LastKeyword = t
	}

	// Extract tables and aliases from the query
	ctx.Tables, ctx.TableAliases = extractTables(sql[:cursorPos])

	// Check if we're after a dot (qualified name)
	trimmed := strings.TrimRight(sql[:cursorPos], " \t\n")
	if strings.HasSuffix(trimmed, ".") {
		ctx.AfterDot = true
		// Find the qualifier before the dot
		words := strings.Fields(trimmed)
		if len(words) > 0 {
			lastWord := words[len(words)-1]
			ctx.Qualifier = strings.TrimSuffix(lastWord, ".")
		}
	}

	return ctx
}

// tokenizeSQL splits SQL into tokens (simplified)
func tokenizeSQL(sql string) []string {
	var tokens []string
	var current strings.Builder

	for _, r := range sql {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// extractTables parses SQL to find table names and aliases
func extractTables(sql string) ([]string, map[string]string) {
	var tables []string
	aliases := make(map[string]string)

	upper := strings.ToUpper(sql)
	tokens := tokenizeSQL(sql) // Use original case for table names
	upperTokens := tokenizeSQL(upper)

	// Find tables after FROM, JOIN, UPDATE, INTO
	for i := 0; i < len(upperTokens); i++ {
		t := upperTokens[i]
		if t == "FROM" || t == "JOIN" || t == "UPDATE" || t == "INTO" {
			if i+1 < len(tokens) {
				tableName := tokens[i+1]
				// Skip keywords
				if isKeyword(strings.ToUpper(tableName)) {
					continue
				}
				tables = append(tables, tableName)

				// Check for alias
				if i+2 < len(upperTokens) {
					if upperTokens[i+2] == "AS" && i+3 < len(tokens) {
						alias := tokens[i+3]
						aliases[alias] = tableName
					} else if !isKeyword(upperTokens[i+2]) &&
						upperTokens[i+2] != "," &&
						upperTokens[i+2] != "ON" &&
						upperTokens[i+2] != "WHERE" {
						// Implicit alias
						alias := tokens[i+2]
						aliases[alias] = tableName
					}
				}
			}
		}
	}

	return tables, aliases
}

// isKeyword checks if a token is a SQL keyword
func isKeyword(s string) bool {
	for _, kw := range SQLKeywords {
		if s == kw {
			return true
		}
	}
	return false
}

// GetSuggestions returns context-aware suggestions
func GetSuggestions(ctx SQLContext, tables []string, columns map[string][]db.Column, input string) []Suggestion {
	var suggestions []Suggestion
	inputUpper := strings.ToUpper(input)

	// After a dot - suggest columns for the qualified table/alias
	if ctx.AfterDot {
		tableName := ctx.Qualifier
		// Check if it's an alias
		if actual, ok := ctx.TableAliases[tableName]; ok {
			tableName = actual
		}
		// Also check uppercase version
		if actual, ok := ctx.TableAliases[strings.ToUpper(tableName)]; ok {
			tableName = actual
		}

		if cols, ok := columns[tableName]; ok {
			for _, col := range cols {
				suggestions = append(suggestions, Suggestion{
					Text:     col.Name,
					Type:     SuggestColumn,
					Detail:   col.Type,
					Priority: 1,
				})
			}
		}
		// Also try lowercase table name
		if cols, ok := columns[strings.ToLower(tableName)]; ok {
			for _, col := range cols {
				suggestions = append(suggestions, Suggestion{
					Text:     col.Name,
					Type:     SuggestColumn,
					Detail:   col.Type,
					Priority: 1,
				})
			}
		}
		return filterSuggestionsTyped(suggestions, input)
	}

	// Context-specific suggestions
	switch {
	case ctx.StatementType == "" || ctx.LastKeyword == "":
		// Start of query - suggest statement keywords
		for _, kw := range statementKeywords {
			suggestions = append(suggestions, Suggestion{Text: kw, Type: SuggestKeyword, Priority: 1})
		}

	case ctx.InSelect:
		// After SELECT - suggest columns, functions, tables (for table.*)
		// Add columns from referenced tables
		for _, tbl := range ctx.Tables {
			if cols, ok := columns[tbl]; ok {
				for _, col := range cols {
					suggestions = append(suggestions, Suggestion{
						Text:     col.Name,
						Type:     SuggestColumn,
						Detail:   col.Type,
						Priority: 2,
					})
				}
			}
		}
		// Add aggregate functions
		for _, fn := range aggregateFunctions {
			suggestions = append(suggestions, Suggestion{Text: fn + "(", Type: SuggestFunction, Priority: 3})
		}
		// Add common functions
		for _, fn := range commonFunctions {
			suggestions = append(suggestions, Suggestion{Text: fn + "(", Type: SuggestFunction, Priority: 4})
		}
		// Add tables for qualified references
		for _, tbl := range tables {
			suggestions = append(suggestions, Suggestion{Text: tbl, Type: SuggestTable, Priority: 5})
		}
		// Add SELECT keywords
		for _, kw := range selectKeywords {
			suggestions = append(suggestions, Suggestion{Text: kw, Type: SuggestKeyword, Priority: 6})
		}

	case ctx.InFrom || ctx.InJoin:
		// After FROM/JOIN - suggest tables
		for _, tbl := range tables {
			suggestions = append(suggestions, Suggestion{Text: tbl, Type: SuggestTable, Priority: 1})
		}
		// Add FROM/JOIN keywords
		for _, kw := range fromKeywords {
			suggestions = append(suggestions, Suggestion{Text: kw, Type: SuggestKeyword, Priority: 3})
		}

	case ctx.InWhere || ctx.InHaving:
		// After WHERE - suggest columns, operators, functions
		for _, tbl := range ctx.Tables {
			if cols, ok := columns[tbl]; ok {
				for _, col := range cols {
					suggestions = append(suggestions, Suggestion{
						Text:     col.Name,
						Type:     SuggestColumn,
						Detail:   col.Type,
						Priority: 1,
					})
				}
			}
		}
		// Also add table-qualified columns
		for _, tbl := range ctx.Tables {
			if cols, ok := columns[tbl]; ok {
				for _, col := range cols {
					suggestions = append(suggestions, Suggestion{
						Text:     tbl + "." + col.Name,
						Type:     SuggestColumn,
						Detail:   col.Type,
						Priority: 2,
					})
				}
			}
		}
		// Add functions
		for _, fn := range commonFunctions {
			suggestions = append(suggestions, Suggestion{Text: fn + "(", Type: SuggestFunction, Priority: 3})
		}
		// Add WHERE keywords
		for _, kw := range whereKeywords {
			suggestions = append(suggestions, Suggestion{Text: kw, Type: SuggestKeyword, Priority: 4})
		}

	case ctx.InGroupBy:
		// After GROUP BY - suggest columns
		for _, tbl := range ctx.Tables {
			if cols, ok := columns[tbl]; ok {
				for _, col := range cols {
					suggestions = append(suggestions, Suggestion{
						Text:     col.Name,
						Type:     SuggestColumn,
						Detail:   col.Type,
						Priority: 1,
					})
				}
			}
		}
		for _, kw := range groupKeywords {
			suggestions = append(suggestions, Suggestion{Text: kw, Type: SuggestKeyword, Priority: 3})
		}

	case ctx.InOrderBy:
		// After ORDER BY - suggest columns and ASC/DESC
		for _, tbl := range ctx.Tables {
			if cols, ok := columns[tbl]; ok {
				for _, col := range cols {
					suggestions = append(suggestions, Suggestion{
						Text:     col.Name,
						Type:     SuggestColumn,
						Detail:   col.Type,
						Priority: 1,
					})
				}
			}
		}
		for _, kw := range orderKeywords {
			suggestions = append(suggestions, Suggestion{Text: kw, Type: SuggestKeyword, Priority: 3})
		}

	case ctx.InSet:
		// After SET in UPDATE - suggest columns
		for _, tbl := range ctx.Tables {
			if cols, ok := columns[tbl]; ok {
				for _, col := range cols {
					suggestions = append(suggestions, Suggestion{
						Text:     col.Name,
						Type:     SuggestColumn,
						Detail:   col.Type,
						Priority: 1,
					})
				}
			}
		}

	default:
		// General suggestions - keywords + tables
		for _, kw := range statementKeywords {
			suggestions = append(suggestions, Suggestion{Text: kw, Type: SuggestKeyword, Priority: 5})
		}
		for _, tbl := range tables {
			suggestions = append(suggestions, Suggestion{Text: tbl, Type: SuggestTable, Priority: 3})
		}
	}

	// Filter by input prefix
	filtered := filterSuggestionsTyped(suggestions, input)

	// If input looks like it might be a keyword, boost keyword matches
	if len(inputUpper) >= 2 {
		for i := range filtered {
			if filtered[i].Type == SuggestKeyword && strings.HasPrefix(filtered[i].Text, inputUpper) {
				filtered[i].Priority = 0
			}
		}
	}

	return filtered
}

// filterSuggestionsTyped filters suggestions by prefix
func filterSuggestionsTyped(suggestions []Suggestion, input string) []Suggestion {
	if input == "" {
		return suggestions
	}

	var matches []Suggestion
	inputUpper := strings.ToUpper(input)

	for _, s := range suggestions {
		if strings.HasPrefix(strings.ToUpper(s.Text), inputUpper) {
			matches = append(matches, s)
		}
	}

	// Sort by priority
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Priority < matches[i].Priority {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	return matches
}

// GetWordAtCursor returns the word under the cursor and its start/end indices
func GetWordAtCursor(text string, cursor int) (string, int, int) {
	if cursor < 0 || cursor > len(text) {
		return "", 0, 0
	}

	// Find start
	start := cursor
	for start > 0 {
		r := rune(text[start-1])
		if !isValidIdentifierChar(r) {
			break
		}
		start--
	}

	// Find end
	end := cursor
	for end < len(text) {
		r := rune(text[end])
		if !isValidIdentifierChar(r) {
			break
		}
		end++
	}

	return text[start:end], start, end
}

func isValidIdentifierChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.'
}

// Legacy function for backward compatibility
func filterSuggestions(input string, candidates []string) []string {
	var matches []string
	input = strings.ToUpper(input)
	for _, c := range candidates {
		if strings.HasPrefix(strings.ToUpper(c), input) {
			matches = append(matches, c)
		}
	}
	return matches
}

// updateSuggestions updates the autocomplete suggestions based on cursor position
func (m Model) updateSuggestions() Model {
	text := m.editor.Value()
	row := m.editor.Line()
	lines := strings.Split(text, "\n")
	if row >= len(lines) {
		return m
	}

	// Calculate cursor position in full text
	cursorPos := 0
	for i := 0; i < row; i++ {
		cursorPos += len(lines[i]) + 1 // +1 for newline
	}
	cursorPos += len(lines[row]) // Assume cursor at end of line

	// Get the word being typed
	line := lines[row]
	col := len(line)
	word, _, _ := GetWordAtCursor(line, col)

	// Parse SQL context
	ctx := ParseSQLContext(text, cursorPos)

	// Get context-aware suggestions
	suggestions := GetSuggestions(ctx, m.tables, m.columns, word)

	// Convert to string slice for display
	m.suggestions = make([]string, len(suggestions))
	m.suggestionDetails = make([]string, len(suggestions))
	m.suggestionTypes = make([]SuggestionType, len(suggestions))

	for i, s := range suggestions {
		m.suggestions[i] = s.Text
		m.suggestionDetails[i] = s.Detail
		m.suggestionTypes[i] = s.Type
	}

	m.suggestionIdx = 0
	return m
}

// applySuggestion inserts the selected suggestion
func (m Model) applySuggestion() Model {
	if len(m.suggestions) == 0 || m.suggestionIdx >= len(m.suggestions) {
		return m
	}
	selected := m.suggestions[m.suggestionIdx]

	row := m.editor.Line()
	lines := strings.Split(m.editor.Value(), "\n")
	if row >= len(lines) {
		return m
	}
	line := lines[row]
	col := len(line)
	_, start, end := GetWordAtCursor(line, col)

	// Replace
	prefix := line[:start]
	suffix := ""
	if end < len(line) {
		suffix = line[end:]
	}
	newLine := prefix + selected + suffix
	lines[row] = newLine

	m.editor.SetValue(strings.Join(lines, "\n"))

	// Calculate new cursor position (linear index)
	newCol := start + len(selected)
	cursorIdx := 0
	for i := 0; i < row; i++ {
		cursorIdx += len(lines[i]) + 1 // +1 for newline
	}
	cursorIdx += newCol

	m.editor.SetCursor(cursorIdx)
	return m
}

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
			prefix = "> "
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
