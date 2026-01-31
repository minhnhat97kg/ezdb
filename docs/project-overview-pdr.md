# EZDB Project Overview & Product Development Requirements

## Executive Summary

EZDB is a Terminal User Interface (TUI) database client for modern developers who prefer working in the terminal. It provides a fast, intuitive interface for querying PostgreSQL, MySQL, and SQLite databases with features like intelligent SQL autocomplete, secure credential management, and persistent query history.

## Vision & Target Users

**Vision**: Make database querying as seamless in the terminal as using dedicated GUI clients, with better developer workflow integration.

**Target Users**:
- Backend developers and data engineers using terminal-first workflows
- DevOps engineers managing multiple database instances
- Data analysts requiring quick schema exploration and ad-hoc queries
- System administrators who prefer CLI-based tools

## Core Features

### 1. Multi-Database Support
- PostgreSQL, MySQL, and SQLite drivers
- Unified query interface across database types
- Automatic schema introspection for autocomplete

### 2. Intelligent SQL Autocomplete
- Context-aware suggestions based on SQL parsing
- Table and column name completion
- Function signature hints
- Real-time schema loading with debouncing

### 3. Query History & Persistence
- SQLite-backed history storage
- 90-day retention with automatic cleanup
- Quick-access history browser with preview
- Search and filter capabilities

### 4. Secure Credential Management
- System keyring integration (macOS Keychain, Linux Secret Service, Windows Credential Manager)
- AES-256 encryption fallback for encrypted passwords
- Master key generation and rotation
- No plaintext passwords in config files

### 5. Configuration & Customization
- TOML-based configuration files
- XDG Base Directory compliance
- Customizable keybindings
- Theme system with Nord palette defaults
- Multiple connection profiles

### 6. Result Browsing & Export
- Paginated table view with horizontal scrolling
- Column filtering and sorting
- CSV export functionality
- Popup menus for row-level actions
- Syntax highlighting for query code

### 7. Schema Navigation
- Schema browser component for table exploration
- Column metadata display (type, nullability, keys)
- Constraint information
- Context-sensitive suggestions

## Non-Functional Requirements

### Performance
- Query execution < 5s for typical operations
- Autocomplete suggestions within 500ms
- Smooth pagination for 100+ row results
- Terminal rendering without flicker

### Reliability
- Graceful error handling with clear messages
- Connection retry logic
- Transaction safety (read-only query mode for safety)
- Secure cleanup on exit

### Security
- No hardcoded credentials
- Encrypted credential storage
- Secure file permissions (0600 for config)
- No password logging or debugging output

### Usability
- Vim-like keybindings (hjkl for navigation)
- Intuitive mode switching (insert/visual)
- Command-line style (/) for special operations
- Consistent with terminal conventions

### Maintainability
- Modular component architecture
- Clear separation of concerns (config, db, ui)
- Comprehensive error types
- Testable driver interface

## Architecture Constraints

1. **Bubble Tea Framework**: TUI rendering based on Elm architecture (Model-Update-View)
2. **Driver Interface**: Pluggable database drivers with common QueryResult struct
3. **State Machine**: AppState manages connection lifecycle (SelectingProfile → Connecting → Ready)
4. **Single-Threaded UI**: Async operations via tea.Cmd pattern
5. **Terminal-Only**: No GUI dependencies, pure terminal rendering

## Scope

### In Scope
- Query execution and result display
- Database connection management
- Configuration persistence
- Query history tracking
- Schema introspection
- Credential encryption
- Result export

### Out of Scope
- GUI client (web or desktop)
- Data migration tools
- Database administration (create/drop databases)
- Query optimization suggestions
- Scheduled/batch operations
- Multi-connection simultaneous queries

## Success Metrics

1. **User Adoption**: Successfully handles 100+ queries per session
2. **Performance**: 95% of queries return results in < 5s
3. **Reliability**: Zero crashes on graceful exit and error scenarios
4. **Security**: All credentials encrypted with no unencrypted storage
5. **Developer Experience**: Context-aware autocomplete improves query speed by 30%

## Known Limitations

1. No transaction support (auto-commit mode for safety)
2. Large result sets (10,000+ rows) may have performance impact
3. No real-time query cancellation (terminal signal only)
4. Limited to single database connection per session
5. No query result caching between sessions
6. Binary data display limited to hex representation

## Future Roadmap

### Phase 2
- Multi-connection management (tab support)
- Query result caching and filtering
- Advanced schema navigation (relationships, indexes)
- SQL formatting and beautification

### Phase 3
- Query execution profiling
- Custom snippets and saved queries
- Database comparison tools
- Time-series result visualization

### Phase 4
- Plugin system for custom drivers
- API server for programmatic access
- Cloud database support (managed services)
- Remote connection via SSH tunnels

## Dependencies

### Direct Dependencies (11)
- `charmbracelet/bubbletea` (1.3.10): TUI framework
- `charmbracelet/bubbles` (0.21.0): Input components
- `charmbracelet/lipgloss` (1.1.0): Terminal styling
- `evertras/bubble-table` (0.19.2): Table widget
- `alecthomas/chroma/v2` (2.23.1): Syntax highlighting
- `99designs/keyring` (1.2.2): System keyring
- `BurntSushi/toml` (1.6.0): Config parsing
- `adrg/xdg` (0.5.3): XDG compliance
- `lib/pq` (1.11.1): PostgreSQL driver
- `go-sql-driver/mysql` (1.9.3): MySQL driver
- `mattn/go-sqlite3` (1.14.33): SQLite driver

### Build Requirements
- Go 1.25.4+
- CGO enabled (for SQLite)
- SQLite3 dev libraries

## Development Guidelines

### Code Standards
- Go idioms and conventions (follow `docs/code-standards.md`)
- Max file size 200 lines for maintainability
- Clear error types with context
- Comprehensive error handling

### Testing
- Unit tests for business logic
- Driver interface tests
- Config encryption/decryption tests
- Happy path and error path coverage

### Documentation
- Architecture diagrams in ASCII
- Code comments for non-obvious logic
- README examples for common tasks
- Inline documentation for public APIs
