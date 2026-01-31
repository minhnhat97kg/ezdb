# EZDB System Architecture

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      Terminal User                      │
└─────────────────────────────────────────────────────────┘
                           │
                           ↓
┌─────────────────────────────────────────────────────────┐
│                    Bubble Tea Program                    │
│                   (TUI Framework)                        │
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │  Model (State Management)                       │   │
│  │  - AppState (SelectingProfile→Connecting→Ready) │   │
│  │  - Mode (Insert ↔ Visual)                       │   │
│  │  - Editor, Results, History, Popups             │   │
│  └─────────────────────────────────────────────────┘   │
│                           │                             │
│                           ↓                             │
│  ┌─────────────────────────────────────────────────┐   │
│  │  Update(msg tea.Msg) → (Model, tea.Cmd)        │   │
│  │  - Message dispatch                             │   │
│  │  - State mutations                              │   │
│  │  - Async command generation                     │   │
│  └─────────────────────────────────────────────────┘   │
│                           │                             │
│                           ↓                             │
│  ┌─────────────────────────────────────────────────┐   │
│  │  View() → string (Terminal Rendering)           │   │
│  │  - Lipgloss styling                             │   │
│  │  - Component composition                        │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
         │             │             │
         ↓             ↓             ↓
    ┌────────┐  ┌────────┐  ┌────────────┐
    │ Config │  │   DB   │  │  History   │
    │ Layer  │  │ Layer  │  │   Layer    │
    └────────┘  └────────┘  └────────────┘
         │             │             │
    [TOML File]   [PostgreSQL  [SQLite DB]
                   MySQL        (Query
                   SQLite]      History)
```

---

## Component Architecture

### 1. UI Layer (internal/ui)

#### State Machine: AppState

```
┌────────────────────────────────────────────────┐
│         SelectingProfile (Initial)             │
│  - Display list of configured profiles         │
│  - Handle profile selection                    │
└────────────────────────────────────────────────┘
                    │ (Selected)
                    ↓
┌────────────────────────────────────────────────┐
│            Connecting (Progress)               │
│  - Establish connection to database            │
│  - Load tables and schema                      │
│  - Initialize autocomplete                     │
└────────────────────────────────────────────────┘
                    │ (Connected)
                    ↓
┌────────────────────────────────────────────────┐
│             Ready (Normal)                     │
│  - Accept SQL input                            │
│  - Execute queries                             │
│  - Display results                             │
│  - Provide autocomplete                        │
└────────────────────────────────────────────────┘
                    │ (Disconnect)
                    ↓
            Back to SelectingProfile
```

#### Mode Switching

```
Input Focus (Editor)
        ↑
        │
    Insert Mode ←→ Visual Mode
    (Editing SQL)  (Browsing Results)
```

### 2. Configuration Layer (internal/config)

```
┌──────────────────────────────────────────────┐
│           Config Lifecycle                   │
└──────────────────────────────────────────────┘

Startup:
  config.Load()
    ├─ Check $XDG_CONFIG_HOME/ezdb/config.toml
    ├─ If not exists: Create default
    ├─ Parse TOML
    ├─ Decrypt passwords from keyring
    └─ Return Config struct

Shutdown:
  config.Save()
    ├─ Encrypt passwords
    ├─ Serialize to TOML
    ├─ Set 0600 permissions
    └─ Write to config file
```

#### Profile Resolution

```
Postgres Profile:
  DSN = "postgres://user:pass@host:port/database?sslmode=disable"

MySQL Profile:
  DSN = "user:pass@tcp(host:port)/database"

SQLite Profile:
  DSN = "file:/path/to/database.db"
```

#### Credential Security

```
Credential Storage:
  1. User enters password → RAM only
  2. Save triggered → Encrypt with master key
  3. Master key stored in:
     - System keyring (preferred)
     - XDG cache as encrypted file (fallback)
  4. Load triggered → Decrypt password
  5. Decrypt successful → Password in RAM
  6. Password in config.TOML → Ciphertext only
```

### 3. Database Layer (internal/db)

#### Driver Interface

```
┌─────────────────────────────────────────────┐
│           Driver Interface                  │
├─────────────────────────────────────────────┤
│ Connect(dsn) error                          │
│ Close() error                               │
│ Execute(ctx, query) (*QueryResult, error)   │
│ Ping(ctx) error                             │
│ Type() DriverType                           │
│ GetTables(ctx) ([]string, error)            │
│ GetColumns(ctx, table) ([]Column, error)    │
│ GetConstraints(ctx, table) ([]Constraint)   │
└─────────────────────────────────────────────┘
         │              │              │
         ↓              ↓              ↓
    ┌────────┐   ┌────────┐   ┌────────┐
    │Postgres│   │ MySQL  │   │SQLite  │
    │Driver  │   │Driver  │   │Driver  │
    └────────┘   └────────┘   └────────┘
```

#### Query Execution Flow

```
Execute(query) {
    Start Timer
    ↓
    Detect Query Type (SELECT vs DML)
    ├─ SELECT/WITH → executeSelect()
    └─ INSERT/UPDATE/DELETE/CREATE/ALTER → executeDML()
    ↓
    For SELECT:
        Scan columns
        Iterate rows
        Convert to [][]string
        Return QueryResult{Columns, Rows, IsSelect: true}
    ↓
    For DML:
        Execute statement
        Get affected rows
        Return QueryResult{AffectedRows, IsSelect: false}
    ↓
    Calculate duration
    Return QueryResult with timing
}
```

#### Result Representation

```go
type QueryResult struct {
    Columns      []string    // ["id", "name", "email"]
    Rows         [][]string  // [["1", "Alice", "alice@"], ...]
    ExecTime     time.Duration  // 125ms
    RowCount     int         // 1000
    IsSelect     bool        // true for SELECT
    AffectedRows int64       // For DML
}
```

### 4. History Layer (internal/history)

#### Persistence

```
History Storage:
  Location: $XDG_CACHE_HOME/ezdb/history.db (SQLite)

  Table: query_history
  ├─ id (PRIMARY KEY, AUTO_INCREMENT)
  ├─ query (TEXT)
  ├─ profile_name (TEXT)
  ├─ executed_at (TIMESTAMP)
  ├─ execution_time_ms (INTEGER)
  ├─ row_count (INTEGER)
  └─ INDEX on executed_at

Retention:
  - Automatic cleanup on startup
  - Delete entries older than 90 days
  - Keep last N entries (1000 default)
```

#### Access Patterns

```
Store.Add(entry *HistoryEntry) error
  - Insert new query with timestamp
  - Executed at current time
  - Execution time in milliseconds

Store.GetAll(limit, offset int) ([]HistoryEntry, error)
  - Paginated retrieval
  - Most recent first
  - Useful for history browser

Store.Cleanup() error
  - Delete old entries
  - Run on startup
  - Safe to call multiple times

Store.Close() error
  - Close database connection
  - Called on app exit
```

---

## Message Flow & State Management

### Key Message Types

```go
QueryExecutedMsg       // ← Query results ready
QueryErrorMsg          // ← Query execution failed
TablesLoadedMsg        // ← Schema introspection complete
HistoryLoadedMsg       // ← Query history retrieved
UpdateEditorMsg        // ← Text input changed
KeyMsg                 // ← Keyboard input
ProfileSelectedMsg     // ← Profile selection confirmed
ConnectionErrorMsg     // ← Connection failed
```

### Update Cycle

```
1. User input (KeyMsg)
   ↓
2. Model.Update(msg)
   ├─ Type assert message
   ├─ Modify state
   └─ Return (Model, tea.Cmd)
   ↓
3. Tea.Cmd execution (async)
   ├─ Run in goroutine
   └─ Return new message
   ↓
4. New message queued
   ↓
5. View() called with updated Model
   ↓
6. Terminal re-rendered
```

### Popup Overlay System

```
Base View (Results Table)
        ↓
    Can overlay:
    ├─ ResultPopup (Display full result set)
    ├─ ActionPopup (Available row actions)
    ├─ ProfileSelector (Change database)
    └─ HistoryBrowser (View query history)

Popup Model:
  - Receives KeyMsg from parent
  - Returns modified state or dismissal signal
  - Parent re-renders with popup overlay
```

---

## Autocomplete Engine

### Context Detection

```
Input: "SELECT * FROM users WHERE id = "
        │                                 │
        └─ Cursor position (25)

Parse Logic:
1. Find FROM clause → tables context
2. Scan to current position
3. Detect scope (SELECT, WHERE, JOIN)
4. Identify preceding keyword
5. Return SuggestionContext

Result: Context{
    Type: "column",
    Table: "users",
    Scope: "WHERE",
}
```

### Suggestion Generation

```
For each table in database:
  1. Load columns (cached)
  2. Match prefix to current word
  3. Rank by:
     - Exact match (higher priority)
     - Frequency in history
     - Column order (primary key first)
  4. Return top 10 suggestions

Details provided:
  - Column type (INTEGER, VARCHAR, etc.)
  - Nullable status
  - Default value
  - Key type (PRIMARY, FOREIGN, etc.)
```

### Debouncing

```
User types character
    ↓
Schedule debounce (300ms)
    ├─ Cancel previous timer
    └─ Start new timer
    ↓
Wait 300ms without input
    ↓
Trigger TableLoadMsg if schema not cached
    ↓
Generate suggestions
    ↓
Display autocomplete dropdown
```

---

## Terminal Rendering Pipeline

```
Model {
    editor: textarea.Model,
    results: *db.QueryResult,
    resultsTable: table.Model,
    popups: []*PopupModel,
}
    ↓
View() {
    // Layout
    mainArea := editor + resultsTable

    // Style application
    styledArea := applyTheme(mainArea)

    // Overlay popups
    if showPopup {
        return renderWithPopup(styledArea, popup)
    }
    return styledArea
}
    ↓
Terminal buffer (Lipgloss)
    ↓
System terminal (ANSI escape codes)
    ↓
User sees styled, interactive interface
```

### Component Tree

```
Model (Root)
├─ ProfileSelector
│  └─ List of profiles
├─ Editor (textarea)
│  └─ SQL input with highlights
├─ ResultsTable (bubble-table)
│  ├─ Column headers
│  ├─ Rows (paginated)
│  └─ Status line
├─ HistoryView
│  ├─ History list
│  └─ Preview pane
├─ SchemaBrowser
│  ├─ Table list
│  └─ Column details
├─ PopupOverlay
│  ├─ ResultPopup
│  ├─ ActionPopup
│  └─ RowDetailPopup
└─ StatusBar
   ├─ Connection status
   ├─ Mode indicator
   └─ Error messages
```

---

## Data Flow Examples

### Example 1: Execute Query

```
User: "SELECT * FROM users"
   ↓
Press Ctrl+D (execute key)
   ↓
Model.Update(KeyMsg{String: "ctrl+d"})
   ↓
Detect in InsertMode
   ↓
Return (Model, executeQueryCmd(driver, sql))
   ↓
executeQueryCmd runs in goroutine:
  - driver.Execute(context.Background(), sql)
  - Returns QueryResult or error
  - Wraps in QueryExecutedMsg or QueryErrorMsg
   ↓
Message sent back to Model.Update()
   ↓
Update populates m.results = msg.Result
   ↓
View() renders results in table widget
   ↓
Terminal shows results with pagination
```

### Example 2: Autocomplete

```
User types "SELECT * FROM us"
          │                   │
          └─ Triggers keypress
                 ↓
Model.Update(KeyMsg{Rune: 's'})
   ↓
Not in Insert mode? Stop.
   ↓
Schedule debounce(300ms)
   ↓
Wait 300ms (user pauses)
   ↓
Debounce fires:
  - No schema cached? Load tables
  - Return TablesLoadedMsg
   ↓
Model.Update(TablesLoadedMsg{Tables: [...]})
   ↓
Generate suggestions for "us":
  - Match: "users", "user_profiles", "user_roles"
   ↓
m.suggestions = [...]
   ↓
View() renders suggestions dropdown
   ↓
User presses Down arrow to select "users"
   ↓
View() highlights selected suggestion
```

### Example 3: Profile Change

```
User: Press "ctrl+p" (profile selector key)
   ↓
Model.Update(KeyMsg{String: "ctrl+p"})
   ↓
Set m.appState = StateSelectingProfile
   ↓
Return (Model, nil)
   ↓
View() renders ProfileSelector component
   ↓
User navigates and selects "production-db"
   ↓
Model receives ProfileSelectedMsg{Profile: ...}
   ↓
Disconnect from current driver
   ↓
driver.NewDriver(profile.Type)
   ↓
driver.Connect(profile.DSN())
   ↓
Load schema (tables, columns)
   ↓
m.appState = StateReady
   ↓
View() renders editor + empty results
   ↓
User can now execute queries on new database
```

---

## Concurrency Model

### Key Principle
**No shared mutable state between goroutines**

### Async Pattern (Tea.Cmd)

```
tea.Cmd is a function: func() tea.Msg

Pattern:
1. User action triggers async operation
2. Model.Update() returns tea.Cmd
3. Bubble Tea spawns goroutine
4. Goroutine executes, produces tea.Msg
5. Message sent back to Update()
6. Update() applies result to Model
7. View() called with new state

No race conditions because:
- Each goroutine has its own context
- Model mutations only in Update()
- Update() is single-threaded
```

### Thread-Safe Operations

```
✓ Reading from Driver
✓ Each query execution gets new context
✓ Each async operation independent
✓ Config loaded once at startup (read-only)
✓ History Store has internal SQLite sync

✗ Never: Modify Model.results in goroutine
✗ Never: Shared channel between goroutines
✗ Never: Concurrent database.Execute() calls
```

---

## Performance Characteristics

### Query Execution
- **Network latency** dominates (ms to s)
- **Result set size** affects memory (string conversion)
- **Pagination** prevents loading entire result

### Autocomplete
- **Schema load** cached after first access (< 1s for typical DB)
- **Suggestion matching** O(n) where n = number of tables
- **Debounce** prevents excessive database queries

### UI Rendering
- **Bubble Tea** single-threaded, 60+ FPS possible
- **Large result sets** (10k+ rows) stutter without pagination
- **Viewport** rendering only visible rows

### Memory Usage
- **Config**: ~1-10 KB
- **Schema cache**: ~100 KB (typical DB)
- **Query results**: O(rows * columns) in RAM
- **History**: Paginated, only N entries in memory

---

## Error Recovery

### Connection Errors
```
User attempts to connect
    ↓
Driver.Connect() fails
    ↓
Model.Update(ConnectionErrorMsg)
    ↓
m.errorMsg = "Failed to connect: ..."
m.appState = StateSelectingProfile (reset)
    ↓
View() displays error message
    ↓
User can select different profile or retry
```

### Query Errors
```
User executes query
    ↓
Driver.Execute() returns error
    ↓
Model.Update(QueryErrorMsg)
    ↓
m.results = nil
m.errorMsg = "SQL Syntax Error: ..."
    ↓
View() displays error, preserves editor content
    ↓
User can edit query and retry
```

### Recovery Behavior
- **Connection errors**: Reset to profile selector
- **Query errors**: Keep in ready state, display error
- **Timeout errors**: Cancel query, return to ready
- **Memory errors**: Exit gracefully (panic avoided)

---

## Security Architecture

### Secret Storage

```
Credential Lifecycle:

1. First Run (No Master Key):
   └─ Generate 32-byte random key
   └─ Save to system keyring
       ├─ macOS: Keychain
       ├─ Linux: Secret Service (GNOME Keyring)
       └─ Windows: Credential Manager

2. Subsequent Runs:
   └─ Retrieve key from keyring
   └─ Use key for all encryption/decryption

3. Config File (Encrypted):
   └─ Store passwords as AES-256-GCM ciphertext
   └─ File permissions: 0600
   └─ Not executable, no scripts
```

### Encryption Details

```go
// Encrypt(plaintext, key) → ciphertext
// - Generate random 12-byte nonce
// - AES-256-GCM cipher from key
// - Encrypt plaintext with nonce
// - Prepend nonce to ciphertext
// - Hex encode for storage

// Decrypt(ciphertext, key) → plaintext
// - Hex decode
// - Extract 12-byte nonce
// - AES-256-GCM cipher from key
// - Decrypt ciphertext
// - Verify authentication tag
```

### Terminal Safety

```
Secure Practices:
✓ Passwords not printed in errors
✓ Connection strings not logged
✓ Query history doesn't store sensitive data
✓ Temp files cleaned up on exit
✓ No environment variable secrets
✓ No shell history (read-only from TUI)

Risky Practices (Avoided):
✗ Hardcoded credentials
✗ Plaintext password files
✗ Shell command execution
✗ System password managers (eval)
```

---

## Extension Points

### Adding a New Database Driver

1. **Implement Driver interface**:
   ```go
   type NewDBDriver struct {
       db *sql.DB
   }

   func (d *NewDBDriver) Connect(dsn string) error { ... }
   func (d *NewDBDriver) Close() error { ... }
   // ... implement all Driver methods
   ```

2. **Register in factory**:
   ```go
   func NewDriver(t DriverType) (Driver, error) {
       case NewDB:
           return &NewDBDriver{}, nil
   }
   ```

3. **Test thoroughly**:
   - Connection lifecycle
   - All query types (SELECT, INSERT, etc.)
   - Schema introspection
   - Error handling

### Adding a New UI Component

1. **Create component package**:
   ```
   internal/ui/components/newcomponent/
   └─ newcomponent.go
   ```

2. **Implement BubbleTea interface**:
   ```go
   type Model struct { ... }
   func (m Model) Init() tea.Cmd { ... }
   func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) { ... }
   func (m Model) View() string { ... }
   ```

3. **Integrate into app.go**:
   - Add field to root Model
   - Route messages in Update()
   - Render in View()

---

## Testing Strategy

### Unit Tests
- Config serialization/deserialization
- Driver query execution
- History store persistence
- Crypto functions

### Integration Tests
- Full query cycle (connect → query → display)
- Profile switching
- Error recovery

### Manual Testing
- Terminal compatibility (width, height changes)
- Long result sets (pagination)
- Autocomplete accuracy
- Keybinding conflicts
