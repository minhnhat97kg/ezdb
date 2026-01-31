# EZDB

A fast, modern TUI database client for PostgreSQL, MySQL, and SQLite.

## Features

- **Multi-Database**: PostgreSQL, MySQL, SQLite with unified interface
- **SQL Autocomplete**: Context-aware suggestions from schema introspection
- **Query History**: SQLite-backed with 90-day retention
- **Secure Credentials**: System keyring + AES-256 encryption
- **Schema Browser**: Navigate tables, columns, constraints
- **SSH Tunnel**: Connect to remote databases securely
- **Result Export**: CSV export with pagination
- **Nord Theme**: Customizable via TOML

## Installation

### Quick Install (Linux/macOS)

```bash
curl -sSL https://raw.githubusercontent.com/minhnhat97kg/ezdb/main/install.sh | bash
```

### From Releases

Download from [Releases](https://github.com/minhnhat97kg/ezdb/releases):

```bash
# Linux (amd64)
curl -L https://github.com/minhnhat97kg/ezdb/releases/latest/download/ezdb_linux_amd64 -o ezdb
chmod +x ezdb && sudo mv ezdb /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/minhnhat97kg/ezdb/releases/latest/download/ezdb_darwin_arm64 -o ezdb
chmod +x ezdb && sudo mv ezdb /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/minhnhat97kg/ezdb/releases/latest/download/ezdb_darwin_amd64 -o ezdb
chmod +x ezdb && sudo mv ezdb /usr/local/bin/
```

### From Source

Requires Go 1.24+ and SQLite3 dev libraries:

```bash
# macOS
brew install sqlite3

# Ubuntu/Debian
sudo apt-get install libsqlite3-dev

# Build
git clone https://github.com/minhnhat97kg/ezdb.git
cd ezdb
make build
sudo mv bin/ezdb /usr/local/bin/
```

## Quick Start

```bash
# Launch TUI
ezdb

# First run: create a profile when prompted
# Then write SQL and press Ctrl+D to execute
```

## Configuration

Config: `~/.config/ezdb/config.toml`

```toml
default_profile = "local-postgres"
page_size = 100

[[profiles]]
name = "local-postgres"
type = "postgres"
host = "localhost"
port = 5432
user = "postgres"
database = "mydb"

[[profiles]]
name = "local-sqlite"
type = "sqlite"
database = "/path/to/db.sqlite"

[theme_colors]
accent = "#88C0D0"
bg_primary = "#2E3440"

[keys]
execute = ["ctrl+d"]
exit = ["esc", "ctrl+c", "q"]
```

## Keybindings

| Action | Keys |
|--------|------|
| Execute Query | Ctrl+D |
| Exit | Esc, Ctrl+C, Q |
| Filter Results | / |
| Next/Prev Page | N/B, PgDown/PgUp |
| Scroll Left/Right | H/L, Arrow keys |
| Row Action | Enter, Space |
| Export CSV | E |
| Sort | S |
| Schema Browser | Tab |

## Development

```bash
make build   # Build to ./bin/ezdb
make test    # Run tests
make clean   # Clean artifacts
```

### Creating a Release

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
# GitHub Actions builds and publishes binaries
```

## License

MIT
