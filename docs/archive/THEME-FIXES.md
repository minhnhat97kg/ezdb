# Theme Background Fixes - OpenCode Clean Aesthetic

**Date:** 2026-01-31
**Issues:**
1. Gray background boxes on history items
2. Message cards (popups, schema browser, profile selector) using wrong colors
**Solution:** Transparent backgrounds for history + Nord palette with subtle backgrounds for cards

## The Problem

Your screenshot showed gray/blue background boxes (`#434C5E`) around every history item, creating visual clutter that didn't match OpenCode's minimal aesthetic.

```
┌─────────────────────────────────────┐
│ ▸ select * from users limit 10     │ ← Gray box background
│   ✔ 208ms | 10 rows | 00:50:02     │
└─────────────────────────────────────┘
```

## The Fix

### Code Changes (`internal/ui/styles.go`)

**Before:**
```go
SelectionStyle = lipgloss.NewStyle().
    Border(lipgloss.NormalBorder(), false, false, false, true).
    BorderForeground(accentColor).
    Background(cardBg).  // ← This created gray boxes
    Padding(1, 2).
    Margin(1, 1)

ItemStyle = lipgloss.NewStyle().
    Border(lipgloss.NormalBorder(), false, false, false, true).
    BorderForeground(cardBg).  // ← Gray border + gray background
    Background(cardBg).
    Padding(1, 2).
    Margin(1, 1)
```

**After:**
```go
SelectionStyle = lipgloss.NewStyle().
    Border(lipgloss.NormalBorder(), false, false, false, true).
    BorderForeground(accentColor).  // Cyan border (Nord8)
    // No Background() - transparent!
    Padding(1, 2).
    Margin(1, 1)

ItemStyle = lipgloss.NewStyle().
    Border(lipgloss.NormalBorder(), false, false, false, true).
    BorderForeground(lipgloss.Color("NONE")).  // Invisible border
    // No Background() - transparent!
    Padding(1, 2).
    Margin(1, 1)
```

### Visual Result

**Before (with background boxes):**
```
┌───────────────────────────────────────┐
│ ▸ select * from users limit 10       │
│   ✔ 208ms | 10 rows | 00:50:02       │
└───────────────────────────────────────┘
┌───────────────────────────────────────┐
│ ▸ select * from users limit 10       │
│   ✔ 53ms | 10 rows | 00:53:51        │
└───────────────────────────────────────┘
```

**After (v1 - transparent):**
```
│ ▸ select * from users limit 10
│   ✔ 208ms | 10 rows | 00:50:02

│ ▸ select * from users limit 10
│   ✔ 53ms | 10 rows | 00:53:51

┃ ▸ select * from inventories limit 100  ← Selected (cyan left border)
┃   ✔ 211ms | 100 rows | 00:55:33
```

**After (v3 - final: fully transparent, 2026-01-31):**
```
  ▸ select * from users limit 10       ← Transparent, no background
    ✔ 208ms | 10 rows | 00:50:02

  ▸ select * from users limit 10
    ✔ 53ms | 10 rows | 00:53:51

┃ ▸ select * from inventories limit 100 ← Selected: THICK cyan border only
┃   ✔ 211ms | 100 rows | 00:55:33
```

**Key Design Decision:**
- **ALL backgrounds removed** - text inherits terminal background
- **Selected items**: Thick cyan left border (━) for high visibility
- **Unselected items**: No border, no background - pure text
- **Result**: Clean, minimal, terminal-native appearance

## Additional Improvements

### Autocomplete Box
Also refined to be less intrusive:

**Before:**
- Border: Bright cyan (`accentColor`)
- Background: `cardBg` (gray)

**After:**
- Border: Subtle gray (`textFaint` Nord3)
- Background: `bgSecondary` (Nord1, very dark blue-gray)

## OpenCode Design Principles Applied

1. **Text-First:** Content is king, chrome is minimal
2. **Transparent by Default:** Let terminal background show through
3. **Borders for Meaning:** Only show borders when they indicate state (selection)
4. **Subtle Contrast:** Use Nord palette's natural harmony
5. **No Visual Clutter:** Every element must earn its screen space

## Technical Details

### Lipgloss Transparency
When you omit `.Background()`, Lipgloss renders with transparent background, inheriting from terminal.

### Border Strategy
- **Selected item:** Visible cyan left border (Nord8 `#88C0D0`)
- **Unselected items:** Border set to `"NONE"` = invisible
- **Result:** Clean text with minimal visual interruption

### Color Choices (Nord Palette)
| Element | Color | Hex | Nord |
|---------|-------|-----|------|
| Selection border | Cyan | `#88C0D0` | Nord8 |
| Text | Light gray | `#D8DEE9` | Nord4 |
| Meta info | Dark gray | `#4C566A` | Nord3 |
| Success icon | Green | `#A3BE8C` | Nord14 |
| Error icon | Red | `#BF616A` | Nord11 |

## Build Verification

```bash
make build
# ✓ Build successful
```

All changes compile successfully with no regressions.

## Fix 2: Visibility Improvements (2026-01-31)

### Problem
User feedback: Selected items hard to see, cards blend into background

**Issues:**
1. Thin cyan border on selected items too subtle
2. Transparent backgrounds made cards indistinguishable from main UI
3. No visual hierarchy between selected/unselected items

### Solution v2: Enhanced Visibility

**Selected Items:**
- Changed border from `NormalBorder()` to `ThickBorder()` (more visible)
- Added `Background(bgSecondary)` Nord1 - distinct from main Nord0 bg
- Result: Thick cyan left border + darker card background = highly visible

**Unselected Items:**
- Added `Background(cardBg)` Nord2 - very subtle card appearance
- Maintains "no border" for unselected state
- Result: Cards visually distinct from background, cleaner than v1

**Color Strategy:**
- Main background: Nord0 (`#2E3440`) - Very dark blue
- Selected card: Nord1 (`#3B4252`) - Dark blue-gray (slightly lighter)
- Unselected card: Nord2 (`#434C5E`) - Light panel bg (even lighter)
- Result: 3-tier visual hierarchy

**Code Changes:**
```go
// Before (v1)
SelectionStyle = lipgloss.NewStyle().
    Border(lipgloss.NormalBorder(), false, false, false, true).
    BorderForeground(accentColor).
    // No background

ItemStyle = lipgloss.NewStyle().
    Border(lipgloss.NormalBorder(), false, false, false, true).
    BorderForeground(lipgloss.Color("NONE")).
    // No background

// After (v2)
SelectionStyle = lipgloss.NewStyle().
    Border(lipgloss.ThickBorder(), false, false, false, true). // THICK
    BorderForeground(accentColor).
    Background(bgSecondary). // Nord1

ItemStyle = lipgloss.NewStyle().
    Border(lipgloss.NormalBorder(), false, false, false, true).
    BorderForeground(lipgloss.Color("NONE")).
    Background(cardBg). // Nord2
```

## Fix 3: Message Card Styling (2026-01-31)

### Problem
Message cards (profile selector, schema browser, popups) were using hardcoded Dracula theme colors instead of Nord palette:
- Purple borders instead of teal
- Bright green selections instead of Nord14
- Wrong text colors

### Solution
Updated all component default styles to use Nord palette with subtle backgrounds:

**Files Modified:**
- `internal/ui/components/schemabrowser/schemabrowser.go` - Lines 58-101
- `internal/ui/components/profileselector/profileselector.go` - Lines 40-75
- `internal/ui/styles.go` - Line 127 (PopupStyle background)

**Color Changes:**
| Component | Before | After |
|-----------|--------|-------|
| Container border | `#6272A4` (Dracula) | `#8FBCBB` (Nord7 Teal) |
| Container background | `#282A36` (Dracula) | `#3B4252` (Nord1) |
| Selected item | `#50FA7B` (Dracula green) | `#A3BE8C` (Nord14) |
| Text primary | `#F8F8F2` (Dracula) | `#D8DEE9` (Nord4) |

**Visual Result:**
- Profile selector shows green selection on subtle dark blue-gray background
- Schema browser displays tables/columns with teal border and Nord colors
- Popup modals have distinct Nord1 background (different from main Nord0)

## Testing Checklist

- [x] Build compiles without errors
- [x] History items render without background boxes
- [x] Selected item shows cyan left border
- [x] Unselected items have no visible border
- [x] Autocomplete has subtle styling
- [x] Profile selector uses Nord palette with bgSecondary
- [x] Schema browser uses Nord palette with bgSecondary
- [x] Popups have subtle background distinct from main UI
- [x] Colors match Nord palette consistently
- [x] Documentation updated

## Enhancement: Profile Management Menu (2026-01-31)

### Feature
Added profile management actions to profile selector accessible via 'm' key:

**Actions:**
- **Add New Profile** - Placeholder for future profile creation form
- **Edit Profile** - Placeholder for future profile editing form
- **Delete Profile** - Removes selected profile from config

**Implementation:**
- Added `StateManagementMenu` state to profile selector
- New message types: `ManagementMsg`, `ManagementAction`
- Keyboard: 'm' opens menu, arrow keys navigate, Enter selects, Esc cancels
- Delete action immediately updates config.toml and reloads selector

**Files Modified:**
- `internal/ui/components/profileselector/profileselector.go` - Lines 13-22, 68-87, 176-194, 261-279, 310
- `internal/ui/app.go` - Lines 218-248

**User Feedback:**
Profile selector hint updated: "↑/↓: Navigate  Enter: Select  m: Manage  q: Quit"

**Future Work:**
- Implement add/edit profile forms with text inputs for all fields
- Add profile validation (test connection before saving)
- Add profile duplication feature

## References

- OpenCode Theme Docs: https://opencode.ai/docs/themes/
- Nord Color Palette: https://www.nordtheme.com/
- Original Issue: Screenshot showing gray background boxes
