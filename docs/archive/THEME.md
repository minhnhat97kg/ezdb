# EzDB Theme - OpenCode Nord Palette

**Last Updated:** 2026-01-31

## Overview

EzDB uses a Nord-inspired color palette matching OpenCode's aesthetic for a professional, muted terminal UI experience.

**Reference:** [OpenCode Theme Documentation](https://opencode.ai/docs/themes/)

## Color Palette

### Text Colors
| Variable | Hex | Nord | Usage |
|----------|-----|------|-------|
| `textPrimary` | `#D8DEE9` | Nord4 | Primary text, headings |
| `textSecondary` | `#81A1C1` | Nord9 | Secondary text, labels |
| `textFaint` | `#4C566A` | Nord3 | Muted text, hints |

### Accent Colors
| Variable | Hex | Nord | Usage |
|----------|-----|------|-------|
| `accentColor` | `#88C0D0` | Nord8 | Primary accent, links, highlights |
| `successColor` | `#A3BE8C` | Nord14 | Success messages, VISUAL mode |
| `errorColor` | `#BF616A` | Nord11 | Error messages, warnings |
| `highlightColor` | `#8FBCBB` | Nord7 | Selection highlights, popups |
| `warningColor` | `#D08770` | Nord12 | Warning badges, strict mode |

### Background Colors
| Variable | Hex | Nord | Usage |
|----------|-----|------|-------|
| `bgPrimary` | `#2E3440` | Nord0 | Main background, popup bg |
| `bgSecondary` | `#3B4252` | Nord1 | Status bar, panels |
| `cardBg` | `#434C5E` | Nord2 | Cards, connection info |

## Component Styles

### Status Bar
- **Background:** `bgSecondary` (Nord1 `#3B4252`)
- **Foreground:** `textPrimary` (Nord4 `#D8DEE9`)

### Mode Indicators
- **VISUAL Mode:** Green background (`successColor` Nord14 `#A3BE8C`)
- **INSERT Mode:** Cyan background (`accentColor` Nord8 `#88C0D0`)
- **Foreground:** `bgPrimary` for contrast

### History Items
- **Query Text:** Bold, highlighted SQL syntax
- **Meta Info:** Italic, `textFaint`
- **Selected:** Left border `accentColor` (Nord8), **transparent background**
- **Unselected:** No border, **transparent background**
- **Philosophy:** Clean, minimal—no background boxes (OpenCode-style)

### Input Area
- **Border:** Top only, `textFaint`
- **Prompt:** Bold, `accentColor`

### Autocomplete
- **Box Border:** Rounded, `textFaint` (subtle)
- **Background:** `bgSecondary` (Nord1: subtle dark, not intrusive)
- **Selected Item:** Bold, black text on `highlightColor` (Nord7)

### Messages
- **Success:** `successColor` (Nord14 Green)
- **Error:** `errorColor` (Nord11 Red), bold
- **Warning:** Orange background (`warningColor` Nord12)
- **System:** `highlightColor` (Nord7 Teal), bold

### Popup Modal
- **Border:** Rounded, `highlightColor` (Nord7)
- **Background:** `bgPrimary` (Nord0)
- **Padding:** 1-2 spaces

## Design Philosophy

1. **Muted Aesthetic:** Avoid harsh whites and bright colors
2. **Transparent Backgrounds:** No background boxes on history items—clean OpenCode style
3. **Minimal Borders:** Only show borders when meaningful (selection indicator)
4. **Consistent Contrast:** Maintain readability across all terminal emulators
5. **Professional Look:** Clean, modern interface inspired by terminal.shop
6. **Nord Harmony:** Colors work together cohesively

## Terminal Compatibility

- **Truecolor (24-bit):** Full Nord palette support
- **256-color:** Graceful degradation to nearest colors
- **16-color (ANSI):** Basic color fallback

## Future Enhancements

- [ ] Theme switching support (light/dark/custom)
- [ ] Dynamic theme adaptation based on terminal background
- [ ] User-defined color overrides via config
- [ ] Syntax highlighting for SQL queries

## References

- [OpenCode Theme System](https://opencode.ai/docs/themes/)
- [Nord Color Palette](https://www.nordtheme.com/)
- [Lipgloss Styling Library](https://github.com/charmbracelet/lipgloss)
