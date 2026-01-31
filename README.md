# EZDB - Terminal Database Client

A fast, modern TUI database client written in Go for PostgreSQL, MySQL, and SQLite with intelligent SQL autocomplete, query history, and secure credential storage.

## Features

- **Multi-Database Support**: PostgreSQL, MySQL, SQLite with unified interface
- **Context-Aware SQL Autocomplete**: Intelligent suggestions based on schema introspection
- **Query History**: SQLite-backed persistence with 90-day retention
- **Secure Credentials**: System keyring + AES-256 encryption
- **Nord Theme**: Full TOML customization with Nord color palette defaults
- **XDG Compliance**: Respects XDG Base Directory specification
- **Syntax Highlighting**: SQL syntax highlighting via Chroma
- **Result Export**: CSV export with pagination support
- **Schema Browser**: Navigate tables, columns, and constraints

## Installation

### Prerequisites
- Go 1.25.4+
- SQLite3 development libraries (for CGO)
- System keyring support (Linux/macOS/Windows)

### Build from Source

```bash
git clone https://github.com/nhath/ezdb.git
cd ezdb
make build
./bin/ezdb
```

## Configuration

Configuration is stored in `$XDG_CONFIG_HOME/ezdb/config.toml` (default: `~/.config/ezdb/config.toml`).

### Example Configuration

```toml
default_profile = "local-postgres"
page_size = 100
history_preview_rows = 3
pager = "less"

# Database Profiles
[[profiles]]
name = "local-postgres"
type = "postgres"
host = "localhost"
port = 5432
user = "postgres"
database = "mydb"
# password is stored encrypted in system keyring

[[profiles]]
name = "local-sqlite"
type = "sqlite"
database = "/path/to/database.db"

# Theme Colors (Nord defaults shown)
[theme_colors]
text_primary = "#D8DEE9"
text_secondary = "#81A1C1"
accent = "#88C0D0"
success = "#A3BE8C"
error = "#BF616A"
bg_primary = "#2E3440"
bg_secondary = "#3B4252"

# Keybindings
[keys]
execute = ["ctrl+d"]
exit = ["esc", "ctrl+c", "q"]
filter = ["/"]
next_page = ["n", "pgdown"]
prev_page = ["b", "pgup"]
scroll_left = ["h", "left"]
scroll_right = ["l", "right"]
row_action = ["enter", "space"]
export = ["e"]
sort = ["s"]
```

## Quick Start

1. Start EZDB:
   ```bash
   ezdb
   ```

2. Select a profile from the list or create a new connection
3. Write SQL in the editor (Ctrl+D to execute)
4. Browse results with pagination (n/b for next/previous)
5. Export results with `e` key
6. View query history with `/history`

## Keybindings

| Action | Keys |
|--------|------|
| Execute Query | Ctrl+D |
| Exit | Esc, Ctrl+C, Q |
| Filter Results | / |
| Next Page | N, PgDown |
| Previous Page | B, PgUp |
| Scroll Left | H, Left Arrow |
| Scroll Right | L, Right Arrow |
| Row Action | Enter, Space |
| Export | E |
| Sort | S |

## Development

### Project Structure

```
ezdb/
├── cmd/ezdb/              # Entry point
├── internal/
│   ├── config/            # Configuration & profiles
│   ├── db/                # Database drivers
│   ├── history/           # Query history
│   └── ui/                # TUI components
├── docs/                  # Documentation
├── Makefile               # Build targets
└── go.mod / go.sum        # Dependency management
```

### Building

```bash
make build      # Build binary to ./bin/ezdb
make run        # Build and run
make test       # Run tests
make clean      # Clean build artifacts
```

### Testing

```bash
make test
```

## Architecture

EZDB uses the Bubble Tea framework with an Elm-inspired architecture:
- **Model**: State management (app state, modes, components)
- **Update**: Message handling and state mutations
- **View**: Terminal rendering via Lipgloss

Database layer uses a driver interface with implementations for PostgreSQL, MySQL, and SQLite.

See `docs/system-architecture.md` for detailed architecture documentation.

## License

MIT License - See LICENSE file for details
