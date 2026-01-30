# Claude Code Local Settings

## Setup

Copy the example settings file:

```bash
cp .claude/settings.local.json.example .claude/settings.local.json
```

## Configuration

### Permissions

The `permissions.deny` array blocks Claude from reading certain files/directories:

| Pattern | Purpose |
|---------|---------|
| `*.log` | Log files |
| `.next/` | Next.js build output |
| `.out/` | Generic output directory |
| `build/` | Build artifacts |
| `dist/` | Distribution files |
| `node_modules/` | NPM dependencies |
| `package-lock.json` | Lock file (large, noisy) |

### Hooks

**PreToolUse > Bash**: Validates all Bash commands via `.claude/scripts/validate-bash.sh`

The validation script blocks commands containing these patterns:

| Pattern | Reason |
|---------|--------|
| `node_modules/` | Avoid large dependency trees |
| `.env` | Protect secrets |
| `build/`, `dist/`, `.next/`, `.out/` | Skip build artifacts |
| `__pycache__`, `venv/`, `.pyc` | Python artifacts |
| `.git/` | Git internals |
| `.csv`, `.log` | Data/log files |

## Customization

Edit `.claude/settings.local.json` to:
- Add/remove denied read patterns
- Modify forbidden bash patterns in `.claude/scripts/validate-bash.sh`
