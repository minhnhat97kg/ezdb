# EZDB Project Roadmap

## Current Status (v0.1.0)

**Release Date**: January 2026
**Stability**: Beta / Early Adoption

### Core Features (Complete)
- Multi-database support (PostgreSQL, MySQL, SQLite)
- SQL query execution with results pagination
- Query history with SQLite persistence
- Secure credential storage (keyring + AES-256 encryption)
- Intelligent SQL autocomplete with context awareness
- TOML-based configuration with multiple profiles
- Nord theme with customizable colors
- XDG Base Directory compliance
- Result export to CSV
- Schema browser for table/column exploration
- Syntax highlighting for SQL code
- Keybinding customization

### Known Limitations
- Single database connection per session (no multi-tab)
- No transaction support (auto-commit safety mode)
- Large result sets (10k+ rows) may be slow
- No real-time query cancellation (terminal signal only)
- Limited to 90-day query history
- Binary data shown as hex only
- No query result caching between sessions

---

## Phase 2: Multi-Connection & UX Enhancements

**Target**: Q2 2026
**Focus**: Simultaneous connections, improved navigation, query snippets

### Features

#### 2.1 Multi-Tab Connection Management
- **Tabs for multiple databases**: Switch between connections instantly
- **Tab indicators**: Show connection status, profile name
- **Quick switch keybinding**: Jump to tab 1-9 with Alt+number
- **Connection pooling**: Reuse connections within tab
- **Task**: Refactor app.go to support connection stack

#### 2.2 Advanced Result Filtering
- **Column visibility toggle**: Show/hide columns
- **Row filtering**: In-memory filter with preview
- **Column sorting**: Multi-column sort support
- **Result search**: Find value in current results
- **Task**: Extend bubble-table with filter UI

#### 2.3 Query Snippets & Saved Queries
- **Snippet storage**: TOML file with named queries
- **Snippet execution**: /snippet <name> or Ctrl+Shift+Q
- **Variables in snippets**: {{table}}, {{date}}, etc.
- **Snippet management UI**: Browse, edit, delete
- **Task**: Create snippets.go module, update commands.go

#### 2.4 Improved Error Messages
- **Contextual help**: Suggest fixes for common errors
- **Query parsing**: Highlight syntax errors in SQL
- **Connection debugging**: Detailed error logs
- **Task**: Enhance error types in db/errors.go

#### 2.5 Vim Command Mode
- **Command palette**: `:q` to quit, `:e` to export, etc.
- **Navigation commands**: `:bnext`, `:bprev` for tabs
- **Settings commands**: `:set pagesize 200`
- **Task**: Add command mode to app.go Update()

### Metrics
- 95% query execution success rate
- Autocomplete accuracy > 85%
- Sub-300ms tab switching
- Memory < 50MB for typical session

---

## Phase 3: Advanced Features & Analytics

**Target**: Q4 2026
**Focus**: Performance insights, schema analysis, visual exploration

### Features

#### 3.1 Query Execution Profiling
- **Execution plans**: EXPLAIN output (PostgreSQL/MySQL)
- **Query timing breakdown**: Network vs execution vs rendering
- **Slow query detection**: Flag queries > 1s
- **Resource usage**: Show memory and CPU during execution
- **Task**: Add profiling module, extend Driver interface

#### 3.2 Schema Analysis & Relationships
- **Foreign key visualization**: Show table relationships
- **Index information**: List indexes, coverage, unused
- **Statistics**: Row count, table size, last modified
- **Constraint browser**: View all constraints with definitions
- **Task**: Enhance schemabrowser component, add schema queries

#### 3.3 Time-Series Result Visualization
- **Simple charts**: Line, bar, pie (via ascii-charts library)
- **Aggregate visualization**: SUM, AVG, COUNT graphs
- **Trend analysis**: Show data over time
- **Task**: Add visualization component, integrate chart library

#### 3.4 Advanced Schema Browser
- **Search tables**: Quick find by name pattern
- **Column relationships**: Show references to this column
- **Dependency graph**: Visual table dependency tree
- **Data preview**: Show sample rows for any table
- **Task**: Extend schemabrowser.go with search, preview

#### 3.5 Query Comparison Tool
- **Compare two results**: Diff view
- **Result history**: Compare against previous runs
- **Performance comparison**: Show timing differences
- **Task**: Create comparison module, add UI

### Metrics
- Query plans loaded in < 500ms
- Schema analysis for 1000+ tables
- Chart rendering sub-100ms
- No performance regression from Phase 1

---

## Phase 4: Enterprise & Automation

**Target**: 2027
**Focus**: Plugin system, API server, cloud support, automation

### Features

#### 4.1 Plugin System
- **Driver plugins**: Load custom database drivers
- **UI plugins**: Custom components and themes
- **Export plugins**: Additional export formats (JSON, XML, Parquet)
- **Plugin discovery**: Search plugin registry
- **Task**: Create plugin interface, loader, security model

#### 4.2 REST API Server
- **Query execution**: POST /api/query
- **Result streaming**: Server-sent events for live results
- **Authentication**: Token-based access
- **Result caching**: Cache results by query hash
- **Task**: Create api/ package with server setup

#### 4.3 Cloud Database Support
- **AWS RDS** (PostgreSQL, MySQL) with IAM auth
- **Google Cloud SQL** with Cloud IAM
- **Azure Database** with AD authentication
- **SSH tunneling**: Connect to remote databases
- **Task**: Add tunnel.go module, update Driver interface

#### 4.4 Scheduled Queries
- **Cron-like scheduling**: Run queries on schedule
- **Email notifications**: Send results by email
- **Webhook triggers**: POST results to HTTP endpoint
- **Data export**: Automated daily/weekly exports
- **Task**: Create scheduler module, event system

#### 4.5 Data Transformation Pipeline
- **ETL operations**: Transform query results
- **Data validation**: Type checking, constraints
- **Batch operations**: Execute multiple queries in transaction
- **Data migration**: Safe bulk operations with rollback
- **Task**: Create transformation package

#### 4.6 Team Collaboration
- **Shared profiles**: Team database credentials (encrypted)
- **Audit logging**: Track who ran what queries
- **Role-based access**: Team member permissions
- **Comment queries**: Add annotations to queries
- **Task**: Add team management, audit logging

### Metrics
- Plugin API stable and documented
- API server handles 1000 req/s
- Cloud connection sub-5s latency
- Team mode: 10+ simultaneous users

---

## Deferred / Out of Scope

### Features Not Planned
- **GUI Client**: Dedicated to TUI only
- **Data migration tool**: Out of scope
- **Database administration**: No create/drop/alter
- **Batch import**: No CSV/JSON import
- **Real-time collaboration**: No live cursor sharing
- **Mobile app**: Terminal only
- **Query optimization suggestions**: ML not included
- **Change tracking**: No diff tools
- **Backup/restore**: Not in scope

---

## Release Timeline

```
v0.1.0 ........... January 2026 (Current: Beta)
         ↓
v0.2.0 ........... Q2 2026 (Multi-tab, Snippets)
         ↓
v0.3.0 ........... Q4 2026 (Analytics, Visualization)
         ↓
v1.0.0 ........... Q1 2027 (Stable Release)
         ↓
v1.1.0 ........... 2027 (Plugin System)
         ↓
v2.0.0 ........... 2027+ (API Server, Enterprise)
```

---

## Development Priorities

### Immediate (Next 30 days)
1. Stability: Fix edge cases and error handling
2. Testing: Improve test coverage to 70%+
3. Documentation: User guide and troubleshooting
4. Performance: Profile and optimize query execution

### Short-term (3 months)
1. Multi-tab support (Phase 2.1)
2. Saved snippets (Phase 2.3)
3. Better error messages (Phase 2.4)
4. User feedback integration

### Medium-term (6 months)
1. Query profiling (Phase 3.1)
2. Schema visualization (Phase 3.2)
3. Advanced filtering (Phase 2.2)
4. Community feedback

### Long-term (12+ months)
1. Plugin system (Phase 4.1)
2. Cloud support (Phase 4.3)
3. API server (Phase 4.2)
4. Enterprise features

---

## Dependency Decisions

### Current Stack (Locked)
- Go 1.25+ (required for features)
- Bubble Tea 1.3+ (TUI framework)
- SQLite3 (history store)
- System keyring (credential storage)

### Phase 2 Additions (Planned)
- Snippet storage: TOML (no new deps)
- Command mode: In-house implementation

### Phase 3 Additions (Planned)
- Visualization: Consider `pterm` or `go-echarts`
- Schema analysis: SQL introspection (no new deps)

### Phase 4 Additions (Planned)
- API server: Possibly `chi` or `gin`
- Cloud SDKs: AWS, Google, Azure SDKs
- Plugin system: `hashicorp/go-plugin`

---

## Community & Contributions

### Open to Community
- Bug reports and fixes
- Theme improvements
- Driver implementations
- Documentation translations
- Driver plugins (Phase 4)

### Contribution Guidelines
- Follow code standards (docs/code-standards.md)
- Tests required for new features
- Semantic commit messages
- No hardcoded credentials or secrets

### Maintainer Commitment
- Weekly issue triage
- Monthly release cycle
- Security patch SLA: 48 hours
- Long-term support (3 years minimum)

---

## Metrics & Success Criteria

### Adoption
- 100+ GitHub stars by end of 2026
- 50+ closed issues/PRs
- 10+ active contributors
- 1000+ monthly users (estimated)

### Code Quality
- 70%+ test coverage
- Zero critical security issues
- <100ms query latency (p95)
- <50MB memory for typical session

### User Satisfaction
- NPS > 30
- <5% crash rate
- 90%+ issue resolution rate
- Community-driven features

---

## Risk Mitigation

### Technical Risks
- **Large result sets**: Pagination prevents OOM
- **Connection pooling**: Limits to N connections
- **Credential theft**: Keyring + encryption
- **SQL injection**: Read-only mode (future)

### Business Risks
- **Competing tools**: Differentiate with ease of use
- **Market adoption**: Target developers first
- **Maintenance burden**: Automated testing
- **Breaking changes**: Semantic versioning

---

## Notes for Maintainers

### Regular Maintenance
- Update dependencies monthly
- Run security audits quarterly
- Review and triage issues weekly
- Release stable versions monthly

### Decision Log Entries
- Record all major architectural decisions here
- Include rationale and date
- Keep for future reference

### Backlog Items
- [ ] Multi-connection support (partial work in feature/tabs)
- [ ] Query profiling (design phase)
- [ ] Schema visualization (research phase)
- [ ] Plugin system (not started)
- [ ] API server (not started)
- [ ] Cloud support (research phase)
- [ ] Team collaboration (not started)
- [ ] Web UI (deprioritized)
