package autocomplete

import (
	"strings"
	"unicode"

	"github.com/nhath/ezdb/internal/db"
)

// Suggestion represents a single autocomplete suggestion
type Suggestion struct {
	Text     string
	Type     SuggestionType
	Detail   string // e.g., column type, function signature
	Priority int    // Lower is higher priority
}

// SuggestionType indicates what kind of completion to show
type SuggestionType int

const (
	SuggestKeyword SuggestionType = iota
	SuggestTable
	SuggestColumn
	SuggestFunction
	SuggestAlias
)

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
	StatementType string            // SELECT, INSERT, UPDATE, DELETE, etc.
	LastKeyword   string            // Most recent keyword before cursor
	Tables        []string          // Tables referenced in the query (with aliases)
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
	AfterDot      bool   // After a "." for qualified names
	Qualifier     string // Table/alias before the dot
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
	_, start, _ := GetWordAtCursor(sql, cursorPos)
	if start > 0 && sql[start-1] == '.' {
		ctx.AfterDot = true
		// Find the qualifier before the dot
		qualifierWord, _, _ := GetWordAtCursor(sql, start-1)
		ctx.Qualifier = qualifierWord
	} else if strings.HasSuffix(strings.TrimRight(sql[:cursorPos], " \t\n"), ".") {
		// Just typed a dot
		ctx.AfterDot = true
		trimmed := strings.TrimRight(sql[:cursorPos], " \t\n")
		qualifierWord, _, _ := GetWordAtCursor(sql, len(trimmed))
		ctx.Qualifier = qualifierWord
	}

	return ctx
}

// tokenizeSQL splits SQL into tokens (simplified)
func tokenizeSQL(sql string) []string {
	var tokens []string
	var current strings.Builder

	for _, r := range sql {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' {
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

// findTableColumns finds columns for a table name, handling schema prefixes and case sensitivity
func findTableColumns(tableName string, columns map[string][]db.Column) ([]db.Column, bool) {
	// 1. Exact match
	if cols, ok := columns[tableName]; ok {
		return cols, true
	}

	lowerName := strings.ToLower(tableName)

	// 2. Case-insensitive exact match
	for k, v := range columns {
		if strings.ToLower(k) == lowerName {
			return v, true
		}
	}

	// 3. Suffix match (e.g., "users" matches "public.users")
	suffix := "." + lowerName
	for k, v := range columns {
		if strings.HasSuffix(strings.ToLower(k), suffix) {
			return v, true
		}
	}

	return nil, false
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
		} else if actual, ok := ctx.TableAliases[strings.ToUpper(tableName)]; ok {
			tableName = actual
		}

		if cols, ok := findTableColumns(tableName, columns); ok {
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
			if cols, ok := findTableColumns(tbl, columns); ok {
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
			if cols, ok := findTableColumns(tbl, columns); ok {
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
			if cols, ok := findTableColumns(tbl, columns); ok {
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
			if cols, ok := findTableColumns(tbl, columns); ok {
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
			if cols, ok := findTableColumns(tbl, columns); ok {
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
			if cols, ok := findTableColumns(tbl, columns); ok {
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
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
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
