# EZDB Code Standards & Conventions

## Go Idioms & Style

### File Organization
- **Max file size**: 200 lines (excluding tests)
- **Rationale**: Improves readability, reduces context switching, easier mental model
- **Exception**: Large switch statements or tables (use comments to organize)

### Naming Conventions

#### Packages
- Lowercase, single word: `config`, `db`, `history`, `ui`
- No underscores or dashes in package names
- Descriptive purpose: `internal/ui/components/table` (not `widgets` or `views`)

#### Variables & Functions
- **Short-lived**: `i`, `j`, `err`, `ok`
- **Package-scoped**: `config.DefaultPageSize` (exported), `defaultPageSize` (private)
- **Receiver names**: `c *Config`, `m *Model` (1-2 letters, meaningful)
- **Interfaces**: Verb-based or -er suffix: `Driver`, `Reader`, `Closer`, `Validator`
- **Booleans**: `is*`, `has*`, `should*`: `isModifyingQuery()`, `hasError`

#### Types
- **Structs**: PascalCase: `QueryResult`, `HistoryEntry`, `Profile`
- **Enums/Constants**: PascalCase: `InsertMode`, `StateReady`, `Postgres`
- **Unexported**: lowercase: `mode`, `appState`, `driverType`

### Import Organization
```go
import (
    // Standard library (alphabetical)
    "context"
    "database/sql"
    "fmt"
    "os"
    "strings"
    "time"

    // Third-party (alphabetical by import path)
    "github.com/BurntSushi/toml"
    "github.com/charmbracelet/bubbles/textarea"
    tea "github.com/charmbracelet/bubbletea"

    // Internal (alphabetical)
    "github.com/nhath/ezdb/internal/config"
    "github.com/nhath/ezdb/internal/db"
)
```

## Error Handling

### Error Wrapping
- **Use error types** for distinguishing error categories:
  ```go
  type ConnectionError struct {
      Message string
      Err     error
  }

  func (e *ConnectionError) Error() string {
      return fmt.Sprintf("connection error: %v", e.Err)
  }
  ```

### Error Propagation
- **Immediate return** if error is fatal:
  ```go
  rows, err := db.QueryContext(ctx, query)
  if err != nil {
      return nil, WrapQueryError(err)
  }
  ```

- **Silent fallback** if error is recoverable:
  ```go
  key, err := GetMasterKey()
  if err == nil {  // Only proceed if no error
      for i := range cfg.Profiles {
          if cfg.Profiles[i].EncryptedPassword != "" {
              decrypted, err := Decrypt(cfg.Profiles[i].EncryptedPassword, key)
              if err == nil {
                  cfg.Profiles[i].Password = decrypted
              }
          }
      }
  }
  ```

### Error Messages
- **Context-rich**: Include what was attempted and why it failed
- **User-facing**: Use simple language in UI errors
- **Developer-facing**: Include stack context in logs

Example:
```go
fmt.Sprintf("Failed to load config from %s: %v", path, err)
```

### Panic Usage
- **NO panics** in user-facing code
- **Panic only** for programming errors (invariant violations)
- **Never panic** on external input (files, network, user input)

## Database Driver Interface

### Driver Contract
All database drivers must implement:
```go
type Driver interface {
    Connect(dsn string) error
    Close() error
    Execute(ctx context.Context, query string) (*QueryResult, error)
    Ping(ctx context.Context) error
    Type() DriverType
    GetTables(ctx context.Context) ([]string, error)
    GetColumns(ctx context.Context, tableName string) ([]Column, error)
    GetConstraints(ctx context.Context, tableName string) ([]Constraint, error)
}
```

### QueryResult Standard
All drivers return consistent `QueryResult`:
```go
type QueryResult struct {
    Columns      []string    // Column names
    Rows         [][]string  // All values as strings
    ExecTime     time.Duration
    RowCount     int         // Number of rows returned
    IsSelect     bool        // SELECT vs DML
    AffectedRows int64       // For DML operations
}
```

### Connection Management
- **Single connection per driver instance**
- **Timeout context**: All queries use context with timeout
- **Graceful close**: Defer Close() in main, handle connection pool cleanup

## Configuration Management

### TOML Structure
```toml
default_profile = "profile-name"
page_size = 100

[[profiles]]
name = "unique-profile-name"
type = "postgres|mysql|sqlite"
host = "localhost"        # Required for postgres/mysql, ignored for sqlite
port = 5432               # Required for postgres/mysql
user = "username"         # Required for postgres/mysql
database = "db-name"      # Database name or file path for sqlite
password = ""             # Encrypted in keyring, not stored here

[theme_colors]
text_primary = "#D8DEE9"
# ... 10 color definitions total

[keys]
execute = ["ctrl+d"]
exit = ["esc", "ctrl+c", "q"]
# ... 8 keybinding categories
```

### Config Validation
- **At load time**: Check required fields for each profile type
- **At save time**: Encrypt passwords before writing to disk
- **At startup**: Warn if config is missing, provide example

## UI & Message Handling

### Message Types
All messages must be types (not interfaces) for type switching:
```go
type QueryExecutedMsg struct {
    Result *db.QueryResult
}

type QueryErrorMsg struct {
    Err error
}

type TablesLoadedMsg struct {
    Tables []string
}
```

### Bubble Tea Patterns
```go
// Update function signature
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
    // 1. Type assert the message
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // 2. Handle key inputs
        return m.handleKeyMsg(msg)
    case QueryExecutedMsg:
        // 3. Update state with results
        m.results = msg.Result
        return m, nil
    }
    return m, nil
}

// Return tea.Cmd for async operations
func executeQuery(driver db.Driver, query string) tea.Cmd {
    return func() tea.Msg {
        result, err := driver.Execute(context.Background(), query)
        if err != nil {
            return QueryErrorMsg{Err: err}
        }
        return QueryExecutedMsg{Result: result}
    }
}
```

### Component Composition
- **Small components**: < 100 lines for View()
- **Nested messages**: Use type-switching in parent's Update()
- **State encapsulation**: Components manage their own state

## Testing Requirements

### Unit Tests
- **Database drivers**: Test with test databases (SQLite in-memory recommended)
- **Config parsing**: Test TOML serialization/deserialization
- **Crypto functions**: Test encryption/decryption round-trips
- **History store**: Test SQLite persistence

Example:
```go
func TestPostgresDriver_Execute(t *testing.T) {
    // Setup
    driver := &PostgresDriver{}
    err := driver.Connect(testDSN)
    if err != nil {
        t.Fatal(err)
    }
    defer driver.Close()

    // Test
    result, err := driver.Execute(context.Background(), "SELECT 1")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    // Assert
    if result.RowCount != 1 {
        t.Errorf("expected 1 row, got %d", result.RowCount)
    }
}
```

### Integration Tests
- **End-to-end**: Full query cycle with real database
- **Keybindings**: Verify key mapping to actions
- **Configuration**: Test loading and saving

### Coverage Goals
- **Critical paths**: > 80% coverage
- **Error handling**: All error branches tested
- **Edge cases**: Empty results, large datasets, special characters

### Test Naming
- `Test<Function><Scenario>`: `TestConfigLoad_FileNotFound`
- `Benchmark<Function>`: `BenchmarkExecuteQuery`
- `TestHelper_<Name>`: `TestHelper_createTestDriver`

## Concurrency

### Guidelines
- **No goroutines** in synchronous functions
- **Async via tea.Cmd**: Let Bubble Tea manage goroutines
- **No shared state**: Each goroutine gets its own context copy
- **No race conditions**: Use value types, not pointers, for concurrent passes

Example (CORRECT):
```go
func executeQueryCmd(driver db.Driver, query string) tea.Cmd {
    return func() tea.Msg {
        // Goroutine spawned by Bubble Tea
        result, err := driver.Execute(context.Background(), query)
        if err != nil {
            return QueryErrorMsg{Err: err}
        }
        return QueryExecutedMsg{Result: result}
    }
}
```

Example (WRONG):
```go
// Don't do this - race conditions
go func() {
    m.results = driver.Execute(query)  // Concurrent map write!
}()
```

## Security

### Credentials
- **Never log** passwords, keys, or connection strings
- **Never print** plaintext passwords to terminal
- **Always encrypt** before disk persistence
- **Use keyring** when available, encryption fallback

### File Permissions
- **Config files**: 0600 (owner read/write only)
- **History database**: 0600 (owner read/write only)
- **Cache directory**: 0700 (owner read/write/execute only)

### SQL Injection Prevention
- **Never concatenate** user input into SQL strings
- **Use parameterized queries** (if supported in future)
- **Validate input** before passing to database
- **Example**: Config profiles are trusted, user queries are trusted (read-only mode)

## Performance Guidelines

### Query Execution
- **Timeout**: 30 seconds default (configurable)
- **Result size**: Limit to page_size (100 rows) by default
- **Paging**: Never load entire result set into memory

### Autocomplete
- **Debounce**: 300ms minimum between schema loads
- **Caching**: Cache table/column lists in memory
- **Lazy loading**: Load tables only when needed

### UI Rendering
- **Viewport**: Never render off-screen content
- **Redraws**: Minimize full redraws, use updates
- **String allocation**: Reuse buffers where possible

## Documentation

### Code Comments
- **Why, not what**: Comments explain intent, not implementation
- **Public APIs**: Document exported functions/types with godoc comments
- **Complex logic**: Comment non-obvious algorithms

Example (GOOD):
```go
// GetMasterKey retrieves the encryption key from the system keyring.
// If no key exists, generates a new 32-byte key and stores it.
func GetMasterKey() ([]byte, error) {
```

Example (BAD):
```go
// Get the master key
func GetMasterKey() ([]byte, error) {
```

### Godoc Comments
All exported functions/types must have godoc comments:
```go
// Config represents the application configuration.
// It is loaded from $XDG_CONFIG_HOME/ezdb/config.toml at startup.
type Config struct {
    DefaultProfile string
    PageSize       int
}

// Load loads the configuration from disk or creates defaults.
// Returns Config and error if load fails.
func Load() (*Config, error) {
```

### README Comments in Code
Mark non-obvious sections with `// NOTE:` or `// BUG:`:
```go
// NOTE: This loop modifies cfg.Profiles during iteration,
// which is safe because we're iterating over a slice copy.
for i := range cfg.Profiles {
    cfg.Profiles[i].Password = decrypted
}
```

## Linting & Formatting

### Go Formatting
- **gofmt** automatically on save
- **Run before commit**: `gofmt -w ./...`

### Static Analysis
- **golangci-lint** for style issues
- **No strict enforcement**: Focus on readability over strict linting

### Pre-commit Checks
```bash
# Verify formatting
gofmt -l ./...

# Run tests
go test ./...

# Build successfully
go build ./cmd/ezdb
```

## Commit Messages

### Format
Use conventional commit format:
```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code restructuring
- `docs`: Documentation changes
- `test`: Test additions/changes
- `chore`: Build, deps, CI/CD

### Examples
```
feat(autocomplete): add context-aware table suggestions

Implement SQL parser to detect current context (FROM, WHERE, JOIN)
and provide relevant table/column suggestions.

Closes #45
```

```
fix(ui): prevent popup cutoff in narrow terminals

Use stricter sizing constraints to ensure popup never exceeds
available terminal height, with minimum 10 rows for content.

Fixes #52
```

## Development Environment

### Required Tools
- **Go 1.25.4+**
- **CGO_ENABLED=1** for SQLite support
- **SQLite3 development libraries**
- **git** for version control

### Build Commands
```bash
# Build to ./bin/ezdb
make build

# Build and run
make run

# Run tests
make test

# Clean build artifacts
make clean
```

### Recommended Setup
- **Editor**: VS Code with Go extension
- **Debugger**: Delve (`dlv`)
- **Formatter**: golangci-lint
- **Git hooks**: Pre-commit for linting

## Decision Log

| Decision | Rationale | Status |
|----------|-----------|--------|
| Bubble Tea + Lipgloss | TUI framework with Elm arch, active community | Active |
| Go standard library | Minimize dependencies, built-in database/sql | Active |
| SQLite for history | Cross-platform, no external service, XDG compliant | Active |
| XDG Base Directory | Standard on Linux, supports macOS/Windows | Active |
| AES-256-GCM for crypto | AEAD cipher, authenticated encryption | Active |
| Read-only mode (future) | Safety first for interactive TUI | Planned |
