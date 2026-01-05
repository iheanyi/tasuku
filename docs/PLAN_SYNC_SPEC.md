# Plan Sync Specification

## Overview

Automatically extract project-level tasks from Claude Code plan files and sync them to Tasuku, applying the nudge rule to filter out session-level implementation steps.

## Problem

When Claude creates a plan (via EnterPlanMode), it writes to a plan file like `plan.md`. These plans contain valuable project-level tasks mixed with granular implementation steps. Currently, users must manually create Tasuku tasks from plans.

## Solution

Add `tk hooks plan-sync <file>` command that:
1. Parses markdown plan files
2. Extracts actionable items (bullets, numbered lists, checkboxes)
3. Applies nudge rule to filter project vs session-level items
4. Creates Tasuku tasks for project-level items only
5. Reports what was synced vs skipped

## Command Interface

```bash
# Sync a plan file
tk hooks plan-sync plan.md

# Dry run - show what would be created
tk hooks plan-sync plan.md --dry-run

# Force sync all items (skip nudge rule)
tk hooks plan-sync plan.md --all
```

## Output Example

```
Scanning plan.md...

Creating tasks:
  + implement-user-auth: Implement user authentication system
  + add-dark-mode: Add dark mode toggle to settings
  + refactor-db-layer: Refactor database connection pooling

Skipped (session-level):
  - Update auth.go with login handler
  - Fix type errors in user service
  - Run test suite

Created 3 tasks, skipped 3 items.
```

## Parsing Rules

### Extractable Items

1. **Markdown checkboxes**: `- [ ] Task description`
2. **Bullet points**: `- Task description` or `* Task description`
3. **Numbered lists**: `1. Task description`
4. **Headers as parents**: `## Phase 1` items become children of phase

### Item Structure

```
- [ ] Implement user authentication
  - [ ] Add login endpoint        <- child/subtask
  - [ ] Add logout endpoint       <- child/subtask
```

Parent items become Tasuku tasks. Child items can optionally become subtasks (via `--with-subtasks` flag) or be ignored.

## Nudge Rule (Existing)

From `internal/cmd/hooks/hooks.go`:

**Project-level keywords** (persist):
- implement, add feature, build, create, develop
- fix bug, bugfix, hotfix, patch
- refactor, rewrite, redesign, rearchitect
- migrate, upgrade, integrate, setup, configure
- api endpoint, database, schema, authentication
- performance, optimize, deploy, release, ship

**Session-level keywords** (skip):
- fix type error, fix typo, fix lint
- update file, edit file, modify file
- run test, run build, verify, check
- add comment, add import, format, cleanup
- rename variable, rename function

## Claude Code Hook Integration

Add to Claude Code settings or `.claude/settings.json`:

```json
{
  "hooks": {
    "postToolUse": [
      {
        "matcher": "ExitPlanMode",
        "command": "tk hooks plan-sync $PLAN_FILE --dry-run"
      }
    ]
  }
}
```

Or trigger on plan file writes:

```json
{
  "hooks": {
    "postToolUse": [
      {
        "matcher": "Write",
        "pattern": "*.plan.md",
        "command": "tk hooks plan-sync $FILE --dry-run"
      }
    ]
  }
}
```

## Implementation Details

### Markdown Parsing

Use a simple line-by-line parser (no external deps):

```go
type PlanItem struct {
    Description string
    Level       int      // Indentation level
    IsCheckbox  bool
    IsChecked   bool
    Children    []*PlanItem
}

func ParsePlanFile(path string) ([]*PlanItem, error)
```

### Task ID Generation

Reuse existing `generateID()` from hooks.go:
- Kebab-case from description
- Max 50 chars
- Strip special characters

### Deduplication

Before creating, check if task ID already exists:
- If exists with same description → skip
- If exists with different description → warn, skip
- If not exists → create

## Edge Cases

1. **Empty plan file**: No-op, print "No items found"
2. **All items filtered**: Print "All items were session-level"
3. **Malformed markdown**: Best-effort parsing, warn on unparseable lines
4. **Nested items**: Only sync top-level by default
5. **Already synced**: Skip duplicates, report "already exists"

## Future Enhancements

1. **Bidirectional sync**: Update plan checkboxes when Tasuku tasks complete
2. **Epic creation**: `## Phase` headers become epics
3. **Priority detection**: "critical", "urgent" → high priority
4. **Dependency detection**: "after X", "blocked by Y" → blocked_by

## Files to Create/Modify

1. `internal/cmd/hooks/plan_sync.go` - New command implementation
2. `internal/cmd/hooks/plan_sync_test.go` - Tests
3. `internal/cmd/hooks/hooks.go` - Add subcommand
4. `README.md` - Document the feature
5. `CLAUDE.md` - Add to CLI reference
