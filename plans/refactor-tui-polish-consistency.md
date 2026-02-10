# Refactor: TUI Polish Consistency Pass

## Enhancement Summary

**Deepened on:** 2026-01-20
**Research agents used:** code-simplicity-reviewer, architecture-strategist, pattern-recognition-specialist, best-practices-researcher, Context7 (Lipgloss, Bubbles)

### Key Improvements from Research
1. **Simplified scope** - Only 2-3 phases are truly necessary; others are over-engineering
2. **NO_COLOR support** - Critical accessibility addition discovered
3. **Theme organization** - Three-tier hierarchy pattern from Charmbracelet ecosystem
4. **Help bar fix confirmed** - Use `SetShowHelp(false)` not just `SetShowStatusBar(false)`

### New Considerations Discovered
- Style consolidation (Phase 4) is over-engineering - inline styles are more explicit
- Semantic border colors (Phase 6) add indirection without value
- Decision IDs in TUI are unnecessary - users reference IDs via CLI, not TUI
- NO_COLOR environment variable support is a critical accessibility gap

---

## Overview

Ensure consistent visual polish throughout the Tasuku TUI by fixing the help bar duplication, standardizing spacing, and removing dead code.

## Problem Statement

The TUI has accumulated visual inconsistencies:
- Dashboard help bar shows duplicate keybindings (filter, quit, help appear twice)
- Spacing/indentation varies between similar views (4 vs 5 spaces)
- Unused styles in theme.go create confusion

## Proposed Solution

A focused polish pass with only essential fixes. **Per simplicity review, skip unnecessary abstractions.**

---

## Technical Approach

### Phase 1: Fix Dashboard Help Bar Duplication (High Priority) ✅ KEEP

**Problem:** The dashboard shows THREE help lines with duplicated keys (`/:filter`, `q:quit`, `?:help`):
1. Bubbles/list built-in help: `↑/k up • ↓/j down • / filter • q quit • ? more`
2. Custom help1: `n:new  e:edit  s:start  d:done  P:pause  b:block  u:unblock  x:delete  t:timer  a:archive`
3. Custom help2: `0-4:status  p:priority  N:notes  L:learnings  D:decisions  /:filter  r:refresh  ?:help  q:quit`

**Fix:** Disable bubbles/list built-in help since we have comprehensive custom help.

```go
// internal/tui/model.go in initTaskList()
m.taskList.SetShowHelp(false)  // Disable built-in help - we have custom help lines
```

**Files:**
- `internal/tui/model.go:445` - Add `SetShowHelp(false)` after `SetShowStatusBar(false)`

### Research Insights (Phase 1)

**From Bubbles documentation:**
The list component's help can be disabled independently with `SetShowHelp(false)`. This is separate from the status bar.

**Best Practice:**
When providing custom help text, always disable the component's built-in help to avoid duplication. The list's auto-generated help is useful for prototypes but should be replaced in production apps.

---

### Phase 2: Standardize Indentation (Medium Priority) ⚠️ SIMPLIFIED

**Problem:** Inconsistent indentation for timestamps/metadata:
- Notes view: 5-space indent (`"     %s\n"`)
- Learnings view: 4-space indent (`"    %s\n"`)

**Fix:** Standardize on 4-space indent for consistency with learnings/decisions.

```go
// internal/tui/model.go in viewNotes()
// Change from:
b.WriteString(fmt.Sprintf("     %s\n", HelpStyle.Render(task.FormatLocalTime(n.CreatedAt))))
// To:
b.WriteString(fmt.Sprintf("    %s\n", HelpStyle.Render(task.FormatLocalTime(n.CreatedAt))))
```

**Files:**
- `internal/tui/model.go:1493` - Change `"     %s\n"` to `"    %s\n"`

### Research Insights (Phase 2)

**From accessibility research:**
- Consistent indentation aids both visual scanning and screen reader navigation
- 2-4 spaces is the recommended range; just be consistent throughout

**Simplicity review:**
This is a single-character change that improves consistency. Worth doing.

---

### Phase 3: Clean Up Unused Styles (Low Priority) ✅ KEEP

**Problem:** `BaseStyle`, `SubtitleStyle`, `LearningStyle` defined but never used.

**Fix:** Remove dead code.

```go
// internal/tui/theme.go - Remove these unused definitions:
// - BaseStyle (lines 32-34)
// - SubtitleStyle (lines 43-45)
// - LearningStyle (lines 119-120)
```

**Files:**
- `internal/tui/theme.go` - Remove ~8 lines of unused style definitions

### Research Insights (Phase 3)

**From simplicity review:**
Dead code should always be removed. It creates confusion about intended usage and increases cognitive load when reading the codebase.

---

### Phase 4: ~~Consolidate Theme Styles~~ ❌ SKIP - Over-Engineering

**Original proposal:** Add `TitleStyleAccent`, `TitleStyleWarning`, `TitleStylePurple`, `SectionStyle` to theme.go.

**Why skip (from simplicity review):**
> The existing inline styles work. Each view function defines its own `titleStyle` which is **fine** - it's explicit and local to where it's used. Creating 4 new global styles adds abstraction without benefit.

**From architecture review:**
> While semantic naming is correct in principle, the current 7 inline definitions are not causing problems. The views intentionally use different colors for context (amber for warnings, purple for decisions). This is already semantic - just expressed locally rather than globally.

**Decision:** Keep inline styles. They are more explicit and easier to understand than abstract named styles.

---

### Phase 5: ~~Display Decision IDs~~ ❌ SKIP - Not Needed in TUI

**Original proposal:** Show decision IDs in the Decisions view.

**Why skip (from simplicity review):**
> Do users actually need to see IDs in the TUI? The TUI is for viewing, not CLI operations. If they want to reference a decision by ID, they'd use `tk decision list` in CLI.

**Decision:** Skip. The TUI is read-only; IDs are for CLI reference.

---

### Phase 6: ~~Add Semantic Border Colors~~ ❌ SKIP - Unnecessary Indirection

**Original proposal:** Add `BorderColorWarning`, `BorderColorInfo`, `BorderColorInput`, `BorderColorPanel`.

**Why skip (from simplicity review):**
> Currently, border colors are used directly: `BorderForeground(ColorAmber)` for confirm, `BorderForeground(ColorAccent)` for help. This is explicit and clear. Adding semantic aliases hides what color is actually used and adds indirection without benefit.

**Decision:** Direct color references are clearer than semantic aliases.

---

### NEW Phase: Add NO_COLOR Support (Accessibility) 🆕 RECOMMENDED

**Problem discovered in research:** The TUI doesn't respect the [NO_COLOR](https://no-color.org/) environment variable, a de-facto accessibility standard.

**Fix:** Check for `NO_COLOR` environment variable and disable colors when set.

```go
// internal/tui/theme.go - Add at package init
func init() {
    if os.Getenv("NO_COLOR") != "" {
        // Disable all color output
        ColorBg = lipgloss.NoColor{}
        ColorFg = lipgloss.NoColor{}
        // ... set all colors to NoColor
    }
}
```

**Why this matters (from accessibility research):**
> NO_COLOR is a de-facto accessibility standard adopted by hundreds of CLI tools. Users with color blindness, specific terminal themes, or who pipe output to files rely on this.

**Files:**
- `internal/tui/theme.go` - Add NO_COLOR check at init

**Note:** This is a valuable addition but can be done as a separate PR if desired.

---

## Acceptance Criteria

### Functional Requirements
- [x] Dashboard shows only ONE set of help keybindings (no duplicates)
- [x] All timestamp indents use 4 spaces consistently
- [x] No unused style definitions in theme.go

### Non-Functional Requirements
- [x] No visual regressions in existing views
- [x] Golden tests updated and passing

### Quality Gates
- [x] `go test ./internal/tui/...` passes
- [x] Golden files reviewed for visual correctness

---

## Implementation Checklist (Simplified)

### Phase 1: Help Bar Duplication (~5 min)
- [x] Add `m.taskList.SetShowHelp(false)` in `initTaskList()` after line 442
- [x] Run `go test ./internal/tui/... -run TestGolden -update`
- [x] Verify dashboard shows only custom help lines (2 lines, not 3)

### Phase 2: Indentation (~2 min)
- [x] Change Notes view indent from 5 to 4 spaces at line 1493
- [x] Run `go test ./internal/tui/... -run TestGolden -update`
- [x] Verify visual consistency between Notes, Learnings, Decisions views

### Phase 3: Cleanup (~5 min)
- [x] Remove `BaseStyle` from theme.go (lines 32-34)
- [x] Remove `SubtitleStyle` from theme.go (lines 43-45)
- [x] Remove `LearningStyle` from theme.go (lines 119-120)
- [x] Run `go build ./...` to verify no compilation errors

### Optional: NO_COLOR Support (~15 min)
- [ ] Add NO_COLOR environment variable check to theme.go
- [ ] Test with `NO_COLOR=1 tk ui`

---

## Files Changed

| File | Changes |
|------|---------|
| `internal/tui/model.go` | Add `SetShowHelp(false)`, fix 5→4 space indent |
| `internal/tui/theme.go` | Remove 3 unused styles (~8 lines) |
| `internal/tui/testdata/*.golden` | Update affected golden files |

**Estimated total changes:** ~10 lines modified, ~8 lines removed

---

## Risk Analysis

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Visual regression | Low | Medium | Golden tests catch regressions |
| Breaking keybindings | None | N/A | Only changing display, not functionality |

---

## What We're NOT Doing (and Why)

| Skipped Item | Reason |
|--------------|--------|
| Theme style consolidation | Over-engineering; inline styles are more explicit |
| Semantic border colors | Unnecessary indirection; direct colors are clearer |
| Decision ID display | TUI is read-only; IDs are for CLI reference |

**From simplicity review:**
> The current TUI code is already reasonably minimal. Most of the original plan was polish-for-polish's-sake that adds no user value.

---

## References

### Internal References
- Theme definitions: `internal/tui/theme.go`
- View functions: `internal/tui/model.go:1112-1610`
- Golden tests: `internal/tui/testdata/*.golden`

### External References
- [Charmbracelet Lipgloss - Style Inheritance](https://github.com/charmbracelet/lipgloss)
- [Charmbracelet Bubbles - List Component](https://github.com/charmbracelet/bubbles/tree/master/list)
- [NO_COLOR Standard](https://no-color.org/)
- [Building a More Accessible GitHub CLI](https://github.blog/engineering/user-experience/building-a-more-accessible-github-cli/)

### Research Agent Findings
- **code-simplicity-reviewer:** Phases 4-6 are YAGNI violations; only do phases 1, 2, 5
- **architecture-strategist:** Three-tier theme organization (Palette → Semantic → Component) is ideal but current inline approach is acceptable
- **pattern-recognition-specialist:** Found additional duplication (modal construction, confirmation dialogs) for future refactoring
- **best-practices-researcher:** NO_COLOR support is a critical accessibility gap; status symbols should have text labels where space permits
