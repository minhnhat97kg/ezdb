# EZDB Documentation Index

Complete documentation for the EZDB Terminal Database Client project.

## Core Documentation

### [README.md](../README.md)
**User-facing project overview**
- Installation and setup instructions
- Quick start guide
- Configuration example
- Keybindings reference
- Build and test commands
- Basic architecture overview

### [project-overview-pdr.md](project-overview-pdr.md)
**Product Development Requirements & Vision**
- Executive summary
- Vision and target users
- Core features (7 major areas)
- Non-functional requirements
- Architecture constraints
- Success metrics and roadmap

### [codebase-summary.md](codebase-summary.md)
**Technical codebase overview & module responsibilities**
- Complete directory structure with descriptions
- 4 internal packages (config, db, history, ui)
- 6 UI components with purposes
- File-by-file summaries
- Key patterns and conventions
- Dependency graph
- Code statistics

### [code-standards.md](code-standards.md)
**Go idioms, naming conventions, error handling, testing**
- File organization (max 200 lines)
- Naming conventions (packages, functions, types)
- Import organization
- Error handling patterns
- Database driver interface contract
- Configuration management standards
- UI and message handling patterns
- Testing requirements and coverage
- Concurrency guidelines
- Security practices
- Performance guidelines
- Documentation standards
- Commit message format

### [system-architecture.md](system-architecture.md)
**Detailed architecture with ASCII diagrams**
- High-level component architecture
- State machine (AppState transitions)
- Configuration lifecycle and security
- Database layer (Driver interface)
- History persistence model
- Message flow and state management
- Autocomplete engine design
- Terminal rendering pipeline
- Data flow examples (3 detailed scenarios)
- Concurrency model
- Performance characteristics
- Error recovery
- Security architecture
- Extension points for new drivers/components
- Testing strategy

### [project-roadmap.md](project-roadmap.md)
**Current status and future development roadmap**
- Current status (v0.1.0 - Beta)
- Known limitations
- Phase 2: Multi-connection & UX (Q2 2026)
- Phase 3: Analytics & Visualization (Q4 2026)
- Phase 4: Enterprise & Automation (2027)
- Deferred features and out-of-scope items
- Release timeline
- Development priorities
- Risk mitigation
- Success criteria

### [tech-stack.md](tech-stack.md)
**Technology stack overview (reference)**
- Go 1.25.4
- Bubble Tea + Lipgloss (TUI)
- SQLite (history, config storage)
- PostgreSQL, MySQL drivers
- Chroma (syntax highlighting)
- System keyring (credential storage)

## Getting Started

### For Users
1. Start with [README.md](../README.md) for installation and usage
2. Check [Keybindings](#keybindings) section below for commands
3. See Configuration section for advanced setup
4. Refer to [project-overview-pdr.md](project-overview-pdr.md) for feature overview

### For Developers
1. Read [README.md](../README.md) Development section
2. Study [codebase-summary.md](codebase-summary.md) for structure
3. Follow [code-standards.md](code-standards.md) when writing code
4. Review [system-architecture.md](system-architecture.md) for design patterns
5. Check [project-roadmap.md](project-roadmap.md) for development areas

### For Contributors
1. Review [code-standards.md](code-standards.md) for conventions
2. Understand [system-architecture.md](system-architecture.md) design patterns
3. Ensure tests follow [Testing Requirements](#testing-requirements)
4. Use conventional commit messages (see [code-standards.md](code-standards.md#commit-messages))
5. Open PR against develop branch with clear description

## Key Sections

### Architecture Overview

**Layers**:
1. **Terminal User Input** ← → **Bubble Tea TUI**
2. **UI Layer** (app.go, components) ← → **Message/Command System**
3. **Config Layer** (TOML, XDG) ← → **Database Layer** (Driver interface)
4. **History Layer** (SQLite) ← → **Database Connections** (Postgres/MySQL/SQLite)

**Key Pattern**: Elm-inspired Model-Update-View with async operations via tea.Cmd

### Keybindings Reference

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

### Configuration Files

**Location**: `$XDG_CONFIG_HOME/ezdb/config.toml` (default: `~/.config/ezdb/config.toml`)

**Key Sections**:
- Database profiles (Postgres, MySQL, SQLite)
- Theme colors (Nord palette)
- Keybindings
- Paging and history settings

See [README.md](../README.md#configuration) for example.

### State Machine

```
SelectingProfile → Connecting → Ready
                      ↓
                 (Error → SelectingProfile)
```

### Message Types

| Message | Triggered By | Handler |
|---------|--------------|---------|
| `KeyMsg` | Keyboard input | Model.handleKeyMsg() |
| `QueryExecutedMsg` | Query completion | Display results in table |
| `QueryErrorMsg` | Query execution failed | Show error message |
| `TablesLoadedMsg` | Schema introspection done | Cache for autocomplete |
| `HistoryLoadedMsg` | History retrieval done | Display history browser |

## Development Workflow

### Building
```bash
make build      # Build to ./bin/ezdb
make run        # Build and run
make test       # Run tests
make clean      # Clean artifacts
```

### Code Organization
- Keep files under 200 lines
- Use descriptive filenames in kebab-case
- Follow naming conventions (see [code-standards.md](code-standards.md))
- Add godoc comments for public APIs

### Testing
- Write tests for business logic
- Test error paths, not just happy path
- Use SQLite in-memory for database tests
- Run `make test` before committing

### Committing
```
feat(scope): description
fix(scope): description
refactor(scope): description
docs(scope): description
test(scope): description
```

See [code-standards.md](code-standards.md#commit-messages) for details.

## Testing Requirements

### Coverage Goals
- Critical paths: > 80%
- Error handling: All error branches
- Edge cases: Empty results, large datasets
- Special characters: SQL injection patterns

### Test Naming
- `Test<Function><Scenario>`: `TestConfigLoad_FileNotFound`
- `Benchmark<Function>`: `BenchmarkExecuteQuery`
- Test helpers: `TestHelper_<Name>`

### Running Tests
```bash
go test ./...              # Run all tests
go test -cover ./...       # Show coverage
go test -v ./...           # Verbose output
go test -timeout 10s ./... # With timeout
```

## Common Tasks

### Add a New Database Driver
1. Implement `Driver` interface (db/driver.go)
2. Register in `NewDriver()` factory
3. Add tests in new file
4. Update documentation

### Add a New UI Component
1. Create `internal/ui/components/<name>/`
2. Implement BubbleTea Model interface
3. Add routing to app.go Update()
4. Render in app.go View()
5. Add tests

### Change Configuration Format
1. Update `Config` struct in config/config.go
2. Handle migration in `Load()` function
3. Update example in README.md
4. Document in code-standards.md

### Optimize Performance
1. Profile with `pprof` or `go test -bench`
2. Identify bottleneck (see Performance Characteristics in architecture)
3. Apply optimization (caching, pagination, etc.)
4. Measure improvement with benchmarks
5. Document decision in code comment

## File Size Reference

| File | Lines | Purpose |
|------|-------|---------|
| cmd/ezdb/main.go | ~64 | Entry point, startup |
| internal/ui/app.go | ~500+ | Root BubbleTea Model |
| internal/db/driver.go | ~100 | Database abstraction |
| internal/config/config.go | ~202 | Configuration loading |
| internal/ui/autocomplete.go | ~175+ | SQL suggestions |

**Guideline**: Keep new files under 200 lines; split if larger.

## Performance Targets

| Operation | Target | Measured |
|-----------|--------|----------|
| Query execution | < 5s typical | (DB dependent) |
| Autocomplete suggestions | < 500ms | (Schema cached) |
| UI render frame | 60 FPS | (Bubble Tea) |
| Memory per session | < 50MB | (W/o large results) |
| Config load | < 100ms | (First run: keyring init) |

## Security Checklist

- [ ] Passwords encrypted with AES-256-GCM
- [ ] Master key in system keyring (not plaintext)
- [ ] Config file permissions 0600
- [ ] No hardcoded credentials
- [ ] No password logging
- [ ] SQL injection prevention
- [ ] Secure file cleanup on exit

## Troubleshooting

### Build Fails
- Ensure `CGO_ENABLED=1` and SQLite3 dev libs installed
- Check Go version >= 1.25.4

### Tests Fail
- Verify `make test` output for specific failures
- Check database connectivity for integration tests
- Review code-standards.md for test patterns

### Runtime Issues
- Check `~/.config/ezdb/config.toml` for syntax errors
- Verify database credentials
- Review error message in UI
- Check system keyring for master key issues

## Documentation Maintenance

### Update When
- [ ] New feature added (update roadmap, architecture)
- [ ] API changes (update code-standards)
- [ ] New component created (update codebase-summary)
- [ ] Build process changes (update README)
- [ ] Dependencies updated (update tech-stack)

### Review Checklist
- [ ] All code examples are current
- [ ] Architecture diagrams are accurate
- [ ] API documentation matches implementation
- [ ] Links to files are valid
- [ ] Examples can be copy-pasted and run

## Archive

Old session reports and theme-related documentation have been archived in `docs/archive/` for reference:
- FINAL-REPORT-260131.md
- FINAL-THEME-STATE.md
- SESSION-SUMMARY-260131.md
- THEME-FIXES.md
- THEME.md

These were development notes from previous sessions; refer to current documentation for definitive information.

## Quick Links

- **GitHub**: https://github.com/nhath/ezdb
- **Issues**: https://github.com/nhath/ezdb/issues
- **Build**: `make build`
- **Run**: `make run`
- **Test**: `make test`

## Questions?

- Architecture questions: See [system-architecture.md](system-architecture.md)
- Code standards: See [code-standards.md](code-standards.md)
- Feature requests: See [project-roadmap.md](project-roadmap.md)
- Implementation help: See [codebase-summary.md](codebase-summary.md)
