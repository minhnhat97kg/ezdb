# Tech Stack - EzDB Universal Database TUI Client

**Project:** EzDB - Universal Database Client TUI
**Language:** Go (Golang)
**Date:** 2026-01-30

## Core Technologies

### TUI Framework
- **[charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea)** - Elm-inspired TUI framework
  - Model-View-Update pattern
  - Event-driven architecture
  - No goroutines needed (framework handles concurrency)
- **[charmbracelet/bubbles](https://github.com/charmbracelet/bubbles)** - Pre-built components
  - Table component for query results
  - TextInput for SQL editor
  - Viewport for scrollable content
  - List for navigation
  - Key bindings for vim mode
- **[charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss)** - Styling & layout
  - CSS-like styling
  - Color schemes (dark/light detection)
  - Responsive layouts

### Database Drivers
All use `database/sql` standard interface:

- **PostgreSQL:** [lib/pq](https://github.com/lib/pq)
  - Pure Go implementation
  - 213k+ dependents, battle-tested
  - Connection string: `postgres://user:pass@host:port/db?sslmode=disable`

- **MySQL:** [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql)
  - Industry standard MySQL driver
  - Pure Go, no CGO
  - Connection string: `user:pass@tcp(host:port)/db`

- **SQLite:** [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3)
  - Most popular SQLite driver
  - CGO-based (requires C compiler)
  - Connection string: `file:path/to/db.sqlite3`

### Persistence & Configuration
- **[BurntSushi/toml](https://github.com/BurntSushi/toml)** - TOML parsing (human-editable configs)
- **[adrg/xdg](https://github.com/adrg/xdg)** - XDG Base Directory standard
  - Config: `$XDG_CONFIG_HOME/ezdb/config.toml`
  - History: `$XDG_DATA_HOME/ezdb/history.db`
  - Cache: `$XDG_CACHE_HOME/ezdb/cache/`
- **[99designs/keyring](https://github.com/99designs/keyring)** - Secure credential storage
  - macOS Keychain
  - Linux Secret Service
  - Windows Credential Manager
- **SQLite** (via mattn/go-sqlite3) - Query history database
  - Searchable history with indexes
  - 1000-entry limit per connection
  - 90-day auto-cleanup

## Architecture Principles

### YAGNI (You Aren't Gonna Need It)
- Start with core features only
- No premature abstractions
- Add complexity when needed

### KISS (Keep It Simple, Stupid)
- Minimal dependencies
- Clear separation of concerns
- Straightforward patterns

### DRY (Don't Repeat Yourself)
- Database driver interface abstraction
- Reusable TUI components
- Shared configuration utilities

## Project Structure

```
ezdb/
├── cmd/
│   └── ezdb/           # Main entry point
│       └── main.go
├── internal/
│   ├── ui/             # Bubble Tea UI components
│   │   ├── app.go      # Root model
│   │   ├── editor.go   # SQL input component
│   │   ├── history.go  # Query history list
│   │   ├── results.go  # Table results view
│   │   └── popup.go    # Modal popup
│   ├── db/             # Database layer
│   │   ├── driver.go   # Driver interface
│   │   ├── postgres.go
│   │   ├── mysql.go
│   │   └── sqlite.go
│   ├── config/         # Configuration management
│   │   ├── config.go   # Load/save TOML
│   │   ├── profiles.go # Connection profiles
│   │   └── keyring.go  # Credential storage
│   └── history/        # Query history
│       ├── store.go    # SQLite persistence
│       └── entry.go    # History entry model
├── docs/               # Documentation
├── plans/              # Implementation plans
└── go.mod
```

## Dependencies List

```go
require (
    github.com/charmbracelet/bubbletea v0.25.0
    github.com/charmbracelet/bubbles v0.18.0
    github.com/charmbracelet/lipgloss v0.9.1
    github.com/lib/pq v1.10.9
    github.com/go-sql-driver/mysql v1.7.1
    github.com/mattn/go-sqlite3 v1.14.18
    github.com/BurntSushi/toml v1.3.2
    github.com/adrg/xdg v0.4.0
    github.com/99designs/keyring v1.2.2
)
```

## Testing Strategy

- **Unit Tests:** Go's built-in testing framework
- **Database Tests:** SQLite in-memory for integration tests
- **TUI Tests:** Bubble Tea's testing utilities
- **Mocking:** Interface-based mocking for database drivers

## Build Requirements

- **Go:** 1.21+ (for generics and latest std lib features)
- **CGO:** Required for SQLite driver (mattn/go-sqlite3)
- **Platforms:** Linux, macOS, Windows

## Performance Targets

- **Query execution:** <100ms for results display
- **History load:** <50ms for 500 entries
- **Table rendering:** 60 FPS for smooth scrolling
- **Pagination:** 100 rows per page (configurable)

## Security Considerations

- No plaintext password storage (use keyring)
- Parameterized queries only (prevent SQL injection)
- TLS/SSL support for PostgreSQL/MySQL
- Secure file permissions for config (0600)

## Future Extensibility

- Plugin system for additional database drivers
- Export results (CSV, JSON)
- Query autocomplete
- Syntax highlighting
- Schema browser

---

**Rationale:** Tech stack chosen based on comprehensive research of Go ecosystem, Bubble Tea best practices, database driver maturity, and security standards. All choices align with YAGNI/KISS/DRY principles while ensuring production-ready quality.
