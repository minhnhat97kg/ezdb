# EZDB Codebase Summary

## Directory Structure

```
ezdb/
├── cmd/
│   └── ezdb/
│       └── main.go                    # Application entry point
├── internal/
│   ├── config/                        # Configuration management (4 files)
│   │   ├── config.go                  # Config struct, Load/Save, TOML parsing
│   │   ├── profiles.go                # Database profile CRUD, DSN builders
│   │   ├── crypto.go                  # AES-256 encryption/decryption
│   │   └── keyring.go                 # System keyring abstraction
│   ├── db/                            # Database abstraction layer (6 files)
│   │   ├── driver.go                  # Driver interface, factory pattern
│   │   ├── errors.go                  # ConnectionError, QueryError types
│   │   ├── sqlite.go                  # SQLite driver with optimizations
│   │   ├── postgres.go                # PostgreSQL driver with connection pooling
│   │   ├── mysql.go                   # MySQL driver with INFORMATION_SCHEMA
│   │   └── sqlite_test.go             # Basic unit tests
│   ├── history/                       # Query history management (2 files)
│   │   ├── entry.go                   # HistoryEntry struct
│   │   └── store.go                   # SQLite-backed persistence, 90-day retention
│   └── ui/                            # Terminal UI layer (14 files + 6 component dirs)
│       ├── app.go                     # Root BubbleTea Model, state machine, lifecycle
│       ├── autocomplete.go            # SQL context parser, suggestion engine
│       ├── query.go                   # Async query execution commands
│       ├── commands.go                # /profile, /export command handlers
│       ├── popup.go                   # Result, action, row-level popup components
│       ├── history_view.go            # Scrollable query history display
│       ├── highlight.go               # SQL syntax highlighting via Chroma
│       ├── styles.go                  # Lipgloss theme system, color management
│       ├── export.go                  # CSV export functionality
│       ├── messages.go                # Custom tea.Msg types (5 message types)
│       ├── debounce.go                # Debounced autocomplete trigger
│       ├── adapters.go                # Component interop bridges
│       ├── render_input.go            # Input field with syntax highlighting
│       └── components/                # Reusable UI components (6 subdirectories)
│           ├── table/                 # Table rendering wrapper
│           │   └── table.go
│           ├── popup/                 # Generic popup overlay
│           │   └── popup.go
│           ├── historylist/           # History list component
│           │   └── historylist.go
│           ├── profileselector/       # Profile selection dialog
│           │   └── profileselector.go
│           ├── schemabrowser/         # Schema navigation sidebar
│           │   └── schemabrowser.go
│           └── suggestions/           # Autocomplete dropdown
│               └── suggestions.go
├── docs/                              # Documentation (this directory)
├── plans/                             # Implementation plans
├── Makefile                           # Build targets, test commands
├── go.mod / go.sum                    # Go module dependencies (11 direct, 40 indirect)
├── CLAUDE.md                          # AI agent guidelines
├── README.md                          # User-facing documentation
└── LICENSE                            # MIT License

Total: 41 text files, 56,240 tokens
```

## Module Responsibilities

### cmd/ezdb (Entry Point)
**File**: `main.go` (64 lines)

**Responsibility**: Application initialization and lifecycle
- Load configuration from XDG directory
- Initialize theme and keybindings
- Create history store for query tracking
- Check for database profiles (first-run guidance)
- Launch Bubble Tea program with TUI

**Key Functions**:
- `main()`: Application entry, orchestration of startup sequence

**Error Handling**: Exits with status 1 on config, history, or TUI errors

---

### internal/config (Configuration Management)

#### config.go (202 lines)
**Responsibility**: Configuration loading, saving, and defaults

**Key Types**:
- `Config`: Main configuration with profiles, theme, keybindings
- `Theme`: Color palette struct (10 colors using hex codes)
- `KeyMap`: Keybinding configuration (8 action types)
- `Profile`: Database connection profile (name, type, host, port, user, database)

**Key Functions**:
- `Load()`: Load from TOML or create defaults
- `Save()`: Persist configuration with password encryption
- `DefaultConfig()`: Provide Nord theme defaults
- `ConfigPath()`: XDG-compliant path resolution

**Features**:
- Automatic config migration for missing fields
- Password encryption before save
- Password decryption after load
- Secure file permissions (0600)

#### profiles.go (lines)
**Responsibility**: Profile CRUD operations and DSN generation

**Key Functions**:
- Profile connection string builders by type
- Profile validation
- DSN formatting for each database type

#### crypto.go (lines)
**Responsibility**: Password encryption and key management

**Key Functions**:
- `GetMasterKey()`: Retrieve or generate master key from keyring
- `Encrypt()`: AES-256-GCM encryption with random nonce
- `Decrypt()`: AES-256-GCM decryption with nonce extraction

**Features**:
- Uses system keyring when available
- Generates 32-byte random key on first run
- Hex encoding for keyring storage
- Secure nonce management

#### keyring.go (lines)
**Responsibility**: Abstraction over system keyring implementations

**Key Types**:
- `KeyringStore`: Wraps 99designs/keyring

**Key Functions**:
- `NewKeyringStore()`: Create platform-specific keyring
- `GetPassword()`: Retrieve credential
- `SetPassword()`: Store credential securely

---

### internal/db (Database Abstraction Layer)

#### driver.go (100 lines)
**Responsibility**: Driver interface and factory pattern

**Key Types**:
- `Driver`: Interface defining database operations
  - `Connect()`, `Close()`, `Execute()`, `Ping()`
  - `GetTables()`, `GetColumns()`, `GetConstraints()`
- `DriverType`: Enum (Postgres, MySQL, SQLite)
- `Column`: Table column metadata
- `Constraint`: Table constraint metadata
- `QueryResult`: Uniform result representation

**Key Functions**:
- `NewDriver()`: Factory method by type
- `executeQuery()`: Dispatch to SELECT or DML execution
- `executeSelect()`: SELECT query with column scanning
- `executeDML()`: INSERT/UPDATE/DELETE/CREATE/ALTER

**Design Pattern**: Factory creates concrete driver, all code uses Driver interface

#### errors.go (lines)
**Responsibility**: Database error types and wrapping

**Key Types**:
- `ConnectionError`: Connection/authentication failures
- `QueryError`: SQL execution errors

#### sqlite.go (lines)
**Responsibility**: SQLite driver implementation

**Key Features**:
- PRAGMA optimizations (journal_mode, cache_size, foreign_keys)
- File path normalization
- Constraint detection from sqlite_master

#### postgres.go (lines)
**Responsibility**: PostgreSQL driver implementation

**Key Features**:
- Connection pooling
- Prepared statement caching
- Information schema for column metadata
- Transaction handling

#### mysql.go (lines)
**Responsibility**: MySQL driver implementation

**Key Features**:
- INFORMATION_SCHEMA queries
- Column metadata extraction
- Key type detection (PRI, UNI, MUL)
- Collation handling

#### sqlite_test.go (lines)
**Responsibility**: Unit tests for database operations

**Coverage**:
- Driver creation
- Connection lifecycle
- Query execution (SELECT, INSERT, UPDATE, DELETE)
- Error scenarios

---

### internal/history (Query History)

#### entry.go (lines)
**Responsibility**: History entry data structure

**Key Types**:
- `HistoryEntry`: Query with metadata
  - Query text
  - Execution timestamp
  - Execution time
  - Row count
  - Database profile name

#### store.go (lines)
**Responsibility**: SQLite-backed history persistence

**Key Functions**:
- `NewStore()`: Initialize history database
- `Add()`: Persist query to history
- `GetAll()`: Retrieve history with pagination
- `GetRecent()`: Most recent queries
- `Cleanup()`: Remove entries older than 90 days
- `Close()`: Close database connection

**Features**:
- SQLite storage in XDG cache directory
- Automatic index on timestamp
- Bulk cleanup on startup
- Prepared statements for performance

---

### internal/ui (Terminal User Interface)

#### app.go (500+ lines)
**Responsibility**: Root BubbleTea Model, state machine, lifecycle

**Key Types**:
- `Model`: Root BubbleTea model with all state
- `Mode`: Insert vs Visual mode
- `AppState`: SelectingProfile, Connecting, Ready

**Key Components**:
- Profile selector (initial connection)
- SQL editor (textarea with highlights)
- Results table with pagination
- Query history view
- Schema browser sidebar
- Autocomplete suggestions
- Multiple popup overlays

**State Management**:
- AppState machine for connection lifecycle
- Mode switching for editing/navigation
- Popup state for overlays
- Debounced autocomplete

**Key Functions**:
- `NewModel()`: Initialize UI with config
- `Update()`: Message handling and state mutations
- `View()`: Render TUI frame

#### autocomplete.go (175+ lines)
**Responsibility**: SQL context parsing and suggestion generation

**Features**:
- Recursive descent parser for SQL
- Context detection (table.column, WHERE clause, etc.)
- Real-time schema loading
- Suggestion filtering and ranking
- Column type hints

**Key Functions**:
- `ParseContext()`: Extract cursor position context
- `GetSuggestions()`: Generate suggestions for context
- `LoadTables()`: Async schema discovery

#### query.go (lines)
**Responsibility**: Async query execution

**Key Functions**:
- `ExecuteQuery()`: Wrapped in tea.Cmd for async execution
- Error propagation via messages
- Result formatting for table display

#### commands.go (lines)
**Responsibility**: Special commands (/, /profile, /export)

**Key Functions**:
- `/profile`: Switch database profile
- `/export`: Export results to CSV
- `/history`: Browse query history
- `/clear`: Clear results

#### popup.go (lines)
**Responsibility**: Result, action, and row-detail popups

**Components**:
- Result popup: Display full query results
- Action popup: Available row actions
- Row detail popup: Full cell content expansion

**Features**:
- Modal overlay with keyboard navigation
- Dismiss with Esc
- Nested popup support

#### history_view.go (lines)
**Responsibility**: Scrollable query history browser

**Features**:
- List of recent queries
- Query preview on hover
- Jump to query
- Filter by date range

#### highlight.go (lines)
**Responsibility**: SQL syntax highlighting

**Features**:
- Chroma lexer for SQL
- Color mapping to theme
- Line-by-line highlighting

#### styles.go (lines)
**Responsibility**: Lipgloss theme system

**Features**:
- Global style definitions
- Theme color mapping
- Component style builders
- Dynamic styling based on theme

#### export.go (lines)
**Responsibility**: CSV export functionality

**Key Functions**:
- `ExportCSV()`: Format results as CSV
- Column escaping
- File output

#### messages.go (lines)
**Responsibility**: Custom BubbleTea message types

**Key Message Types**:
- `QueryExecutedMsg`: Query results ready
- `QueryErrorMsg`: Query execution failed
- `TablesLoadedMsg`: Schema loaded
- `HistoryLoadedMsg`: History retrieved
- `KeyMsg`: Keyboard input handling

#### debounce.go (lines)
**Responsibility**: Debounced autocomplete trigger

**Features**:
- 300ms debounce timer
- Cancel in-flight requests
- Prevent excessive schema loads

#### adapters.go (lines)
**Responsibility**: Component interop bridges

**Functions**:
- Translate between component models
- Route messages between components
- State synchronization

#### render_input.go (lines)
**Responsibility**: Input field with syntax highlighting wrapper

**Features**:
- Highlighted SQL input
- Cursor position tracking
- Selection support

### UI Components

#### table/table.go (lines)
**Responsibility**: Table rendering wrapper

**Features**:
- Pagination support
- Horizontal scrolling
- Column alignment
- Header styling
- Row selection

#### popup/popup.go (lines)
**Responsibility**: Generic modal popup

**Features**:
- Centered positioning
- Keyboard navigation
- Border styling
- Dismiss callbacks

#### historylist/historylist.go (lines)
**Responsibility**: History list display

**Features**:
- Query preview
- Timestamp display
- Selection tracking

#### profileselector/profileselector.go (lines)
**Responsibility**: Profile selection dialog

**Features**:
- List of available profiles
- Keyboard navigation
- Profile details display
- New profile option

#### schemabrowser/schemabrowser.go (lines)
**Responsibility**: Schema navigation sidebar

**Features**:
- Table list
- Column details
- Constraint information
- Hierarchical view

#### suggestions/suggestions.go (lines)
**Responsibility**: Autocomplete dropdown

**Features**:
- Scrollable list
- Type indicators
- Detail display
- Keyboard selection

---

## Key Patterns & Conventions

### Driver Factory Pattern
All database operations use the `Driver` interface, with concrete implementations (PostgreSQL, MySQL, SQLite). Clients only depend on the interface, enabling easy testing and extension.

### State Machine (AppState)
Application lifecycle: SelectingProfile → Connecting → Ready, ensuring proper initialization order and connection handling.

### Elm Architecture (Bubble Tea)
BubbleTea enforces Model-Update-View pattern: pure functions for state mutations, unidirectional message flow.

### Async Operations (tea.Cmd)
Long-running operations (database queries, schema loading) return `tea.Cmd` to prevent UI blocking.

### Secure Defaults
- Passwords encrypted with AES-256-GCM
- Master key in system keyring
- Config files with 0600 permissions
- No plaintext password logging

### Component Composition
Large UI (app.go) composed of smaller components (table, popup, history_view, etc.) with message routing via adapters.go.

---

## Dependency Graph

```
cmd/ezdb
  └── internal/config
      └── internal/db
      └── internal/history
      └── internal/ui
          ├── internal/config
          ├── internal/db
          ├── internal/history
          └── components/
              ├── table
              ├── popup
              ├── profileselector
              ├── schemabrowser
              ├── historylist
              └── suggestions
```

---

## Critical Dependencies

| Package | Purpose | Version |
|---------|---------|---------|
| charmbracelet/bubbletea | TUI framework (Elm arch) | 1.3.10 |
| charmbracelet/bubbles | Input components | 0.21.0 |
| charmbracelet/lipgloss | Terminal styling | 1.1.0 |
| evertras/bubble-table | Table widget | 0.19.2 |
| alecthomas/chroma/v2 | Syntax highlighting | 2.23.1 |
| 99designs/keyring | System keyring | 1.2.2 |
| BurntSushi/toml | Config parsing | 1.6.0 |
| adrg/xdg | XDG compliance | 0.5.3 |
| lib/pq | PostgreSQL | 1.11.1 |
| go-sql-driver/mysql | MySQL | 1.9.3 |
| mattn/go-sqlite3 | SQLite | 1.14.33 |

---

## Code Statistics

- **Total Files**: 41 text files
- **Total Tokens**: 56,240 (Repomix measurement)
- **Total Characters**: 207,363
- **Largest File**: internal/ui/app.go (9,436 tokens, 33,972 chars, 16.8%)
- **Smallest Components**: Individual driver implementations 500-1000 lines each

---

## Development Focus Areas

### High-Complexity Modules
1. **app.go**: 500+ lines, state management, message routing
2. **autocomplete.go**: SQL parsing, context detection
3. **profileselector.go**: Profile selection with async loading
4. **schemabrowser.go**: Hierarchical schema navigation

### Core Abstractions
1. **Driver interface**: Database abstraction (6 implementations)
2. **QueryResult struct**: Uniform result representation
3. **History store**: SQLite persistence layer

### Performance Critical
1. **Query execution**: Async with timeout
2. **Autocomplete**: Debounced, schema caching
3. **Table rendering**: Lazy pagination, viewport management
