# Tasuku Development Guidelines

## Overview

Tasuku is an agent-first task management system. It's designed for AI agents working on codebases, prioritizing:
- **Pull over push**: Agents query when needed, no constant injections
- **Parallel-safe**: File locking for multiple agents
- **Minimal context**: Only load what's needed for the current task
- **Human-readable**: Markdown files with YAML frontmatter (V4), can be edited by hand
- **Rich content**: Full Markdown support with code blocks, lists, and formatting

## Architecture

```
tasuku/
├── cmd/tk/              # CLI entrypoint
├── internal/
│   ├── store/           # Storage backends (V2 file, V3 JSON dir, V4 Markdown dir)
│   │   └── v4/          # V4 Markdown-based storage implementation
│   ├── task/            # Task domain logic
│   ├── http/            # HTTP REST API server
│   ├── mcp/             # MCP server for AI tools (Claude Code, Cursor, Codex, OpenCode)
│   └── tui/             # Terminal UI (bubble tea)
├── .tasuku/             # V4 directory-based storage (default)
│   ├── tasks/           # Individual task Markdown files
│   ├── archive/         # Archived task files
│   ├── context/         # learnings.md, decisions.md
│   ├── config.json      # Version marker (version: 4)
│   └── index.json       # Auto-generated index for fast queries
└── CLAUDE.md            # This file
```

### Storage Formats

**V4 (Default)**: Markdown-based storage in `.tasuku/`
- Each task is a Markdown file with YAML frontmatter in `tasks/<id>.md`
- Rich content support: code blocks, lists, bold, italic, etc.
- Notes stored inline within task files (under `## Notes` section)
- Context in `context/learnings.md` and `context/decisions.md`
- Auto-generated `index.json` for fast agent queries without parsing all files
- Version marker in `config.json` (`"version": 4`)

**V3 (Legacy JSON)**: Directory-based JSON storage in `.tasuku/`
- Each task is a JSON file in `tasks/<id>.json`
- Context in `context/learnings.json` and `context/decisions.json`
- Migrate to V4 with `tk migrate v4`

**V2 (Legacy)**: Single `.tasuku.json` file
- All tasks in one JSON file
- Auto-detected and supported for backwards compatibility
- Migrate with `tk migrate v3` then `tk migrate v4`

## CLI Command: `tk`

```bash
# Initialization
tk init                    # Create .tasuku/ directory (V4 Markdown)

# Task Management (noun-verb style)
tk task list               # List all tasks (aliases: tk t ls, tk tasks)
tk task list --tree        # Show hierarchical subtask view
tk task list --status ready  # Filter by status
tk task list --tag backend   # Filter by tag
tk task add "description"  # Add a task
tk task add "desc" --parent <id>  # Add subtask
tk task add "desc" --id my-id     # Custom task ID
tk task add "desc" --priority high --tag feature  # With priority/tags
tk task start <id>         # Mark task in_progress
tk task start <id> --unblock      # Clear blockers and start
tk task done <id>          # Mark task complete
tk task done <id1> <id2>   # Complete multiple tasks
tk task pause <id>         # Revert in_progress → ready
tk task block <id> --by <other>   # Mark blocked
tk task unblock <id>       # Remove all blockers
tk task unblock <id> --from <blocker>  # Remove specific blocker
tk task show <id>          # Show task details
tk task edit <id> "new desc"      # Update description
tk task delete <id>        # Delete a task
tk task priority <id> high # Set priority (critical/high/normal/low/backlog)
tk task ready              # List tasks ready to work on (sorted by priority)
tk task find "query"       # Search across tasks, notes, learnings, decisions
tk task deps <id>          # Show task dependency tree
tk task stats              # Show task statistics and progress

# Ownership & Claims (multi-agent coordination)
tk task owner <id> "name"  # Set task owner
tk task owner <id>         # Clear owner
tk task claim <id> agent-1 # Claim for exclusive work
tk task release <id>       # Release claimed task
tk task who                # Show who has claimed what

# Tags & Custom Fields
tk task tag add <id> bug   # Add tag
tk task tag remove <id> bug  # Remove tag
tk task field set <id> estimate 2h   # Set custom field
tk task field remove <id> estimate   # Remove field

# Time Tracking
tk task timer start <id>   # Start timer
tk task timer stop <id>    # Stop timer, record elapsed
tk task timer status       # Show running timers

# Archiving
tk task archive add <id>   # Archive a done task
tk task archive list       # List archived tasks
tk task archive restore <id>  # Restore archived task
tk task archive all --older-than 7d  # Bulk archive old done tasks

# Context & Learnings
tk learn "insight"         # Add a learning (shortcut)
tk learning list           # List all learnings
tk learning promote <id>   # Promote to permanent docs
tk learning remove <id>    # Remove a learning
tk learning rules          # List never/always rule learnings
tk decide --id auth --chose JWT --over "sessions,OAuth" --because "reason"
tk decision list           # List all decisions
tk decision remove <id>    # Remove a decision
tk note add <task-id> "note"  # Add note to task
tk note list               # List all notes
tk note list --task <id>   # List notes for task
tk note remove <task-id> <note-id>  # Remove a note
tk context show            # Dump full context (for agent consumption)
tk suggest "description"   # Check if task should persist to tk

# Rules Sync (for Claude Code, Cursor)
tk rules sync              # Sync learnings/decisions to editor rules
tk rules sync --tool claude  # Sync to specific tool only
tk rules status            # Show sync status
tk rules clean             # Remove Tasuku-generated rules

# Server
tk serve mcp               # Start MCP server (for AI tools)
tk serve http              # Start HTTP REST API on :3000
tk serve http --port 8080  # Start HTTP on custom port

# MCP Configuration
tk mcp install             # Auto-detect and install to all AI tools
tk mcp install --tool claude  # Install to Claude Code only
tk mcp install --tool cursor  # Install to Cursor only
tk mcp install --local     # Project-local config
tk mcp uninstall           # Remove MCP configuration

# Hooks
tk hooks install              # Install all hooks (git + AI tools)
tk hooks install --claude     # Install Claude Code hooks only
tk hooks install --codex      # Install Codex hooks only
tk hooks install --opencode   # Install OpenCode hooks only
tk hooks install --local      # Install to project instead of global
tk hooks install --force      # Reinstall/update hooks
tk hooks uninstall            # Remove all Tasuku hooks
tk hooks session              # Display context summary
tk hooks stop-reminder        # Check for running timers/in-progress
tk hooks plan-sync plan.md    # Extract tasks from plan file
tk hooks prompt-check         # Detect task intent in prompts
tk hooks todo-check           # Check if TodoWrite items should persist
tk hooks pre-compact          # Capture insights before compaction
tk hooks subagent-done        # Capture insights from subagent

# UI & Diagnostics
tk ui                      # Launch terminal user interface
tk health                  # Project health check with recommendations
tk doctor                  # Diagnose MCP and CLI setup
tk validate                # Validate storage for correctness

# Migration
tk migrate v3              # Migrate from .tasuku.json to .tasuku/ (JSON)
tk migrate v4              # Migrate from V3 JSON to V4 Markdown
tk migrate beads           # Migrate from Beads format
```

## Data Model

### V4 Directory Structure (Default)

```
.tasuku/
├── tasks/
│   └── task-id.md        # Individual task Markdown file
├── archive/
│   └── old-task.md       # Archived completed tasks
├── context/
│   ├── learnings.md      # Learnings in Markdown format
│   └── decisions.md      # Decisions in Markdown format
├── config.json           # Version marker {"version": 4}
└── index.json            # Auto-generated index for fast queries
```

**Task File** (`.tasuku/tasks/task-id.md`):
```markdown
---
status: ready
priority: 2
tags: [backend, api]
blocked_by: [other-task-id]
parent_id: parent-task
owner: agent-1
time_spent: 3600000000000
fields:
  estimate: 2h
created_at: 2024-01-04T10:00:00Z
updated_at: 2024-01-04T10:00:00Z
---

# Task title from first line of description

Rest of the description supports **rich Markdown** formatting,
including `code`, lists, and code blocks.

## Notes

### 2024-01-05 11:00 [abc123]
Note content goes here with full Markdown support.
```

**Learnings** (`.tasuku/context/learnings.md`):
```markdown
# Learnings

## learning-id - 2024-01-04T10:30:00Z
Things discovered while working.
```

**Decisions** (`.tasuku/context/decisions.md`):
```markdown
# Decisions

## decision-id - 2024-01-04T10:30:00Z
**Chose**: Option A
**Over**: Option B, Option C
**Because**: Reasoning for the decision.
```

### V3 Directory Structure (Legacy JSON)

```
.tasuku/
├── tasks/
│   └── task-id.json      # Individual task file
├── archive/
│   └── old-task.json     # Archived completed tasks
└── context/
    ├── learnings.json    # Array of learning entries
    └── decisions.json    # Array of decision entries
```

### V2 Format (Legacy)

Single `.tasuku.json` file with all data in one place. Supported for backwards compatibility.

## Development Workflow

### We dogfood tasuku while building it

1. Tasks are tracked in `.tasuku/` at repo root (V4 Markdown format)
2. Use `tk` commands or edit Markdown files directly
3. Every PR should update task status

### Branching

- `main` - stable, tested
- `feature/*` - new features
- `fix/*` - bug fixes

### Commits

Reference task IDs in commits:
```
feat: Add file locking to store (#store-locking)
fix: Handle empty task list (#empty-list-bug)
```

## Testing Strategy

### Unit Tests

```bash
go test ./...
```

Every package should have `*_test.go` files:
- `internal/store/store_test.go` - File operations, locking
- `internal/task/task_test.go` - Domain logic
- `internal/mcp/server_test.go` - MCP protocol
- `internal/http/server_test.go` - HTTP REST API
- `internal/tui/model_test.go` - Terminal UI

**CLI command tests** (using `internal/cmd/testutil/harness.go`):
- `internal/cmd/task/task_test.go` - Task subcommands
- `internal/cmd/context/context_test.go` - Context commands
- `internal/cmd/decision/decision_test.go` - Decision commands
- `internal/cmd/learning/learning_test.go` - Learning commands
- `internal/cmd/note/note_test.go` - Note commands
- `internal/cmd/hooks/hooks_test.go` - Git hooks integration
- `internal/cmd/pr/pr_test.go` - PR creation
- `internal/cmd/server/server_test.go` - Server commands
- `internal/cmd/mcpcmd/mcp_test.go` - MCP management
- `internal/cmd/migrate/migrate_test.go` - Migration commands

### Integration Tests

```bash
go test ./... -tags=integration
```

Located in `testdata/` scenarios:
- `testdata/parallel_agents/` - Test file locking
- `testdata/corrupt_json/` - Error handling

### Manual Verification Checklist

Before merging any PR:

- [ ] `tk init` creates valid `.tasuku/` directory
- [ ] `tk task add/start/done` cycle works
- [ ] `tk task list` shows correct statuses
- [ ] `tk context show` outputs valid JSON
- [ ] Parallel writes don't corrupt files
- [ ] MCP server responds to tool calls
- [ ] V2 format auto-detection works

### Test Commands

```bash
# Run all tests
go test ./...

# Run with race detector (important for parallel safety)
go test -race ./...

# Run specific package
go test ./internal/store/...

# Verbose output
go test -v ./...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## MCP Server

### TUI/CLI/MCP/Skills Parity Principle

**Every capability should be accessible through all interfaces.** This is critical for agent-first design:

| Interface | Users | Description |
|-----------|-------|-------------|
| **CLI** | Humans in terminal | `tk task list`, `tk learn "insight"` |
| **MCP** | AI agents (Claude Code, Cursor, Codex, OpenCode) | `tk_list`, `tk_learn` tools |
| **TUI** | Humans who prefer visual interfaces | Interactive terminal UI (`tk ui`) |
| **Skills** | AI agents (slash commands) | `/tasuku-list`, `/tasuku-learn` |

**Parity Rules:**
1. **New CLI command → Add MCP tool** - Agents need the same capabilities as humans
2. **New MCP tool → Consider CLI equivalent** - Humans may want to use it manually
3. **Core operations → Add to TUI** - Visual interface for task management
4. **Frequent operations → Consider Skills** - Slash commands for quick agent access
5. **Same behavior across all interfaces** - Identical semantics, different UX

**Shortcut Commands via Cobra:**
Use Cobra's aliasing and root-level commands for ergonomic shortcuts:
- `tk learn "insight"` is a shortcut for `tk learning add "insight"`
- `tk decide --id X --chose Y --because Z` is a shortcut for `tk decision add ...`

This pattern leverages Cobra's command structure to provide ergonomic shortcuts without duplicating logic.

This ensures agents can do everything humans can do (and vice versa), which is the whole point of Tasuku.

### MCP Tools Reference

| Tool | CLI Equivalent | Description |
|------|----------------|-------------|
| **Core Task Operations** |||
| `tk_list` | `tk task list` | List tasks with optional status filter |
| `tk_add` | `tk task add` | Create a new task |
| `tk_show` | `tk task show` | Get detailed task info (notes, priority, timestamps) |
| `tk_start` | `tk task start` | Mark task as in_progress |
| `tk_done` | `tk task done` | Mark task as complete |
| `tk_pause` | `tk task pause` | Revert in_progress → ready, clear owner |
| `tk_edit` | `tk task edit` | Update task description |
| `tk_delete` | `tk task delete` | Permanently delete a task |
| `tk_priority` | `tk task priority` | Set priority (critical/high/normal/low/backlog) |
| `tk_ready` | `tk task ready` | List tasks ready to work on (sorted by priority) |
| `tk_deps` | `tk task deps` | Show task dependency tree |
| `tk_stats` | `tk task stats` | Show task statistics and progress |
| **Blocking & Dependencies** |||
| `tk_block` | `tk task block` | Mark task as blocked by others |
| `tk_unblock` | `tk task unblock` | Remove blockers (all or specific) |
| **Ownership & Coordination** |||
| `tk_owner` | `tk task owner` | Set or clear task owner |
| `tk_claim` | `tk task claim` | Claim task for exclusive agent work |
| `tk_release` | `tk task release` | Release claimed task |
| `tk_who` | `tk task who` | Show tasks claimed by each owner/agent |
| **Context & Search** |||
| `tk_context` | `tk context show` | Get full context for agent consumption |
| `tk_find` | `tk task find` | Search across tasks, notes, learnings, decisions |
| `tk_learn` | `tk learn` | Add a learning to context |
| `tk_decide` | `tk decide` | Record an architectural decision |
| `tk_note` | `tk note add` | Add a note to a task |
| **Tags & Custom Fields** |||
| `tk_tag_add` | `tk task tag add` | Add tag to a task |
| `tk_tag_remove` | `tk task tag remove` | Remove tag from a task |
| `tk_field_set` | `tk task field set` | Set custom field on task |
| `tk_field_remove` | `tk task field remove` | Remove custom field |
| **Time Tracking** |||
| `tk_timer_start` | `tk task timer start` | Start timer on task |
| `tk_timer_stop` | `tk task timer stop` | Stop timer, record elapsed time |
| `tk_timer_status` | `tk task timer status` | Get status of running timers |
| **Archiving** |||
| `tk_archive` | `tk task archive add` | Archive a done task |
| `tk_archive_restore` | `tk task archive restore` | Restore archived task |
| `tk_archive_list` | `tk task archive list` | List archived tasks |
| `tk_archive_all` | `tk task archive all` | Archive all done tasks older than duration |
| **Learning Management** |||
| `tk_learning_list` | `tk learning list` | List all learnings |
| `tk_learning_promote` | `tk learning promote` | Promote learning to permanent docs |
| `tk_learning_remove` | `tk learning remove` | Remove a learning |
| `tk_learning_rules` | `tk learning rules` | List rule learnings (never/always patterns) |
| **Decision Management** |||
| `tk_decision_list` | `tk decision list` | List all decisions |
| `tk_decision_remove` | `tk decision remove` | Remove a decision |
| **Note Management** |||
| `tk_note_list` | `tk note list` | List notes for a task or all notes |
| `tk_note_remove` | `tk note remove` | Remove a note |
| **Agent Workflow** |||
| `tk_suggest` | `tk suggest` | Analyze if a task should persist to tk or stay session-only |
| **Health & Diagnostics** |||
| `tk_health` | `tk health` | Project health check with recommendations |
| **Rules Sync** |||
| `tk_rules_sync` | `tk rules sync` | Sync learnings/decisions to editor rules directories |

### TUI Keybindings Reference

Launch the TUI with `tk ui`. The following keybindings are available:

| Key | Action | Notes |
|-----|--------|-------|
| **Task Operations** |||
| `n` | New task | Opens task creation dialog |
| `e` | Edit task | Edit selected task description |
| `s` | Start task | Mark ready task as in_progress |
| `d` | Mark done | Complete in_progress task |
| `P` | Pause task | Revert in_progress to ready |
| `b` | Block task | Mark task as blocked |
| `u` | Unblock task | Remove blockers and set to ready |
| `x` | Delete task | Delete with confirmation |
| `t` | Toggle timer | Start/stop time tracking |
| `a` | Archive task | Archive done task |
| `A` | Archive all done | Bulk archive with confirmation |
| `enter` | View details | Show full task information |
| **Navigation** |||
| `/` | Filter tasks | Text search through tasks |
| `0` | All tasks | Show all statuses |
| `1` | Ready only | Filter to ready tasks |
| `2` | In progress only | Filter to in_progress |
| `3` | Blocked only | Filter to blocked tasks |
| `4` | Done only | Filter to done tasks |
| `p` | Toggle priority sort | Switch between status/priority sort |
| `N` | View notes | Show notes for selected task |
| `L` | View learnings | Show project learnings |
| `D` | View decisions | Show architectural decisions |
| **General** |||
| `r` | Refresh | Reload data from storage |
| `?` | Help | Show keybinding help |
| `q` | Quit | Exit TUI |

### Running the server

```bash
tk serve --port 3000
```

Or via stdio (for AI tools like Claude Code, Cursor, Codex, OpenCode):
```bash
tk serve --stdio
```

## File Locking

For parallel agent safety, we use `flock`:

**V3 (Directory)**: Lock individual task files for granular locking
```go
// Lock specific task file
f, _ := os.OpenFile(".tasuku/tasks/my-task.json", os.O_RDWR, 0644)
syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
```

**V2 (Single file)**: Lock the entire file
```go
f, _ := os.OpenFile(".tasuku.json", os.O_RDWR, 0644)
syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
```

## Code Style

- `go fmt` on save
- `golint` clean
- Errors are wrapped with context: `fmt.Errorf("store: failed to read: %w", err)`
- No magic - explicit is better than implicit

### CLI Architecture (PlanetScale Pattern)

Commands use the constructor pattern instead of `init()` functions:

```go
// Good: Constructor pattern
func newTaskCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "task",
        Short: "Manage tasks",
    }
    cmd.AddCommand(newListCmd())
    cmd.AddCommand(newAddCmd())
    return cmd
}

var Cmd = newTaskCmd()

func newAddCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "add [description]",
        RunE:  runAdd,
    }
    cmd.Flags().String("id", "", "Optional task ID")
    cmd.Flags().Int("priority", 2, "Task priority (0-4)")
    return cmd
}

// Bad: init() pattern (don't use)
var addCmd = &cobra.Command{...}
func init() {
    Cmd.AddCommand(addCmd)
    addCmd.Flags().String("id", "", "Optional task ID")
}
```

**Why constructors:**
- Explicit initialization order (predictable)
- No global side effects
- Easier to test commands in isolation
- Flags stay local to functions (no package-level vars)
- Industry standard (PlanetScale CLI, GitHub CLI)

## Key Decisions

Record architectural decisions here as we make them:

1. **JSON over YAML** - Faster parsing, no ambiguity, better for agents
2. **flock for locking** - Simple, works on macOS/Linux, sufficient for local
3. **MCP over REST** - Native Claude Code integration, no HTTP overhead
4. **V3 directory-based storage over single file** - `.tasuku/` directory with individual task files instead of monolithic `.tasuku.json`:
   - **Cleaner git diffs**: One file per task means PRs only show relevant changes
   - **Reduced merge conflicts**: Parallel agents editing different tasks don't conflict
   - **Granular locking**: Lock only the task being modified, not entire task list
   - **Better archival**: Move completed tasks to `archive/` folder
   - **Backwards compatible**: Auto-detects V2 format and prompts for migration
5. **User-specified task IDs over directory namespacing** - Task IDs are either user-provided via `--id` flag or auto-generated from description (kebab-case). We chose this over automatic directory-namespacing because:
   - **Simplicity**: Agents and humans can use short, memorable IDs like `fix-auth-bug`
   - **Flexibility**: Users can choose their own naming conventions
   - **No path coupling**: Task IDs shouldn't change if files move
   - **Cross-project clarity**: IDs like `api-v2-migration` are clearer than `src/api/migration`
   - If namespacing is needed, users can include it in the ID: `tk add "Fix bug" --id auth/login-timeout`
6. **Cobra CLI framework** - Industry standard, proper `--flag` syntax, shell completion, Viper config integration
7. **Grove integration over native worktree support** - Tasuku manages tasks, Grove manages worktrees. Each tool does one thing well. Avoids duplication and leverages existing Grove infrastructure.
8. **Subtasks via parent_id field** - Flat file storage with `parent_id` reference rather than nested directories. Enables tree view (`--tree`) while keeping storage simple.
9. **StringSlice for multi-value flags** - Flags like `--by`, `--tag`, `--over` use Cobra's `StringSlice` to support both `--flag a --flag b` and `--flag a,b` patterns.
10. **TodoWrite vs Tasuku distinction** - Two different tools for two different purposes:
    - **TodoWrite**: Session-level implementation steps (ephemeral, helps track progress within a conversation)
    - **Tasuku (tk)**: Project-level tasks (persistent, survives across sessions, visible to other agents)
    - When a task is a feature, bug, or project milestone → add to tk
    - When a task is an implementation step like "fix type error" → use TodoWrite only
11. **Constructor pattern over init() for CLI commands** - Commands use `func newCmd() *cobra.Command` constructors instead of `var cmd` + `func init()`. This follows the PlanetScale CLI pattern for explicit initialization, better testability, and avoiding package-level flag variables.
12. **UTC storage, local display for timestamps** - All timestamps (tasks, notes, learnings, decisions) are:
    - **Stored**: UTC in RFC3339 format (`2024-01-04T10:30:00Z`) for sorting, cross-timezone consistency
    - **Displayed**: Local timezone for human readability (`Jan 4, 2024 2:30 AM` in PST)
    - **JSON/MCP output**: UTC for machine parsing
    - Uses `task.FormatLocalTime()` helper for consistent display formatting

## Adding New Functionality Checklist

When adding new MCP tools, CLI commands, or features, follow this audit checklist:

### 1. TUI/CLI/MCP/Skills Parity
- [ ] New CLI command → Add corresponding MCP tool
- [ ] New MCP tool → Consider CLI equivalent
- [ ] Core operation → Add to TUI if visual interaction helps
- [ ] Frequent operation → Consider adding a Skill (slash command)
- [ ] Same capabilities, same behavior across all interfaces

### 2. Tool Descriptions (Nudges)
Every MCP tool description should include:
- **WHAT it does** (basic description)
- **WHEN to use it** ("Use PROACTIVELY when...")
- **Examples** of trigger scenarios (numbered list)
- **Follow-up hints** (what to do after using)

Good nudge example:
```
"Use PROACTIVELY when: (1) Debugging reveals undocumented behavior,
(2) Finding gotchas or edge cases, (3) Discovering patterns that
work well (or poorly). Use 'Never X' or 'Always Y' prefixes for rules."
```

### 3. Response Enhancements
Consider adding smart responses that:
- **Warn** about potential issues (e.g., multiple in_progress tasks)
- **Suggest** next actions (e.g., "Consider archiving" after tk_done)
- **List** affected items (e.g., "These tasks are now unblocked")
- **Prompt** for follow-up (e.g., "Add a note explaining why paused")

### 4. Hook Integration
Check if the new feature should trigger or be triggered by:
- **SessionStart**: Should it be included in session context?
- **Stop**: Should it be reminded about at session end?
- **PostToolUse**: Should it prompt for related actions?

### 5. Documentation
- [ ] Update CLAUDE.md if it affects agent workflow
- [ ] Update README.md MCP tools table
- [ ] Add to CLI help text

### 6. Skills (Optional)
If the feature is frequently used, consider adding a skill (slash command).

## Mandatory Learning Documentation

**CRITICAL: Document learnings IMMEDIATELY when they occur, not at session end.**

### When to Record Learnings (MANDATORY)

Record a learning using `tk_learn` or `tk learn` **IMMEDIATELY** after ANY of these events:

1. **Bug Fix Completed**: After fixing ANY bug, record:
   - What the bug was
   - Why it happened (root cause)
   - The rule to prevent it ("Never X" or "Always Y")

2. **Gotcha Discovered**: When you discover unexpected behavior:
   - API that doesn't work as documented
   - Edge case that causes failures
   - Implicit assumptions in code

3. **Pattern Identified**: When you notice a recurring issue:
   - Same type of bug appearing multiple times
   - Code smell that leads to problems
   - Anti-pattern in the codebase

4. **Workaround Required**: When standard approach doesn't work:
   - Library limitation requiring different approach
   - Framework quirk needing special handling

### Learning Format

Use "Never" or "Always" prefixes for rules that should prevent future bugs:

```bash
# Good - actionable rule
tk learn "Never manually manipulate ANSI-styled strings with rune operations. Use lipgloss.Width/Height/Place which handle escape sequences correctly."

# Good - specific gotcha
tk learn "Always collect map keys into a slice before iterating if you'll modify the map during iteration."

# Bad - too vague
tk learn "Be careful with strings"
```

### Enforcement

**This is not optional.** If you fix a bug and don't record a learning, you are:
1. Allowing the same bug to happen again
2. Wasting future debugging time
3. Failing to build institutional knowledge

The `tk_health` command will warn about sessions with bug fixes but no new learnings.

## Agent Task Management

When working on this codebase, follow these guidelines for task tracking:

### When to use `tk` (Tasuku)
- New features or enhancements (e.g., "Add dark mode support")
- Bug reports (e.g., "Fix race condition in auth")
- Project milestones (e.g., "V3 migration")
- Tasks that should persist across sessions
- Tasks that other agents might need to see

### When to use TodoWrite only
- Implementation steps within a session (e.g., "Update file X", "Fix type error in Y")
- Temporary tracking of sub-steps
- Progress tracking that doesn't need to persist

### Nudge Rule

**Before adding items to TodoWrite**, use `tk_suggest` (MCP) or `tk suggest` (CLI) to check if it should also be tracked in tk:

```bash
tk suggest "Implement user authentication"
# → ✓ PERSIST TO TK (project-level feature)

tk suggest "Fix type error in auth.ts"
# → ✗ KEEP SESSION-ONLY (implementation step)
```

If the suggestion says **PERSIST TO TK**, add it to both:
1. TodoWrite (for session progress tracking)
2. `tk task add "Description" --priority high --tag feature` (for persistence)

**Project-level indicators** (should persist):
- Keywords: implement, add feature, fix bug, refactor, migrate, deploy
- Database, API, authentication, security work
- Milestones, epics, stories

**Session-level indicators** (TodoWrite only):
- Keywords: fix type error, update file, run tests, debug, verify
- Small, temporary implementation steps

This ensures important work is tracked persistently and visible to future sessions.

## Session Start/End Behaviors

AI tool hooks automatically integrate Tasuku into your workflow:

**Supported AI Tools:**
- **Claude Code**: Full hooks support (SessionStart, Stop, PreCompact, SubagentStop, etc.)
- **Cursor**: MCP server + rules sync (no hooks, uses Claude Code-style skills)
- **Codex**: MCP server + notify hook (agent-turn-complete)
- **OpenCode**: MCP server + plugin hooks (session.created, session.idle, todo.updated)

### At Session Start (SessionStart hook)
The `tk hooks session` command runs automatically and displays:
- Task counts by status (ready, in_progress, blocked, done)
- Number of learnings and decisions recorded
- Suggested next task based on priority

**What to do at session start:**
1. Review the context summary to orient yourself
2. Check if any in_progress tasks need attention
3. Use `tk_context` if you need full task details
4. Start a timer if picking up work: `tk timer start <task-id>`

### At Session End (Stop hook)
The `tk hooks stop-reminder` command runs when you exit and reminds about:
- **Running timers**: Any timers still active that should be stopped
- **In-progress tasks**: Tasks still marked in_progress that may need status updates

**What to do before ending a session:**
1. Stop any running timers: `tk timer stop <task-id>`
2. Update task status: either `tk task done <id>` or `tk task pause <id>`
3. Record any learnings discovered: `tk learn "insight"`
4. Note any blockers for future sessions: `tk note add <task-id> "blocker info"`

### Manual Session Commands

If hooks aren't installed, use these commands manually:
```bash
tk hooks session        # Show context summary
tk hooks stop-reminder  # Check for reminders before exiting
```

Install hooks with:
```bash
tk hooks install              # Install all hooks (git + AI tools)
tk hooks install --claude     # Claude Code hooks only
tk hooks install --codex      # Codex hooks only
tk hooks install --opencode   # OpenCode hooks only
tk hooks install --local      # Project-local instead of global
```

### MCP Server Installation

Configure MCP server for AI tools:
```bash
tk mcp install                # Auto-detect and install to all AI tools
tk mcp install --tool claude  # Claude Code only
tk mcp install --tool cursor  # Cursor only
tk mcp install --tool codex   # Codex only
tk mcp install --tool opencode # OpenCode only
tk mcp install --local        # Project-local config
```

**Configuration files by tool:**
- Claude Code: `~/.claude.json` or `./.claude.json` (local)
- Cursor: `~/.cursor/mcp.json` or `./.cursor/mcp.json` (local)
- Codex: `~/.codex/config.toml` (TOML format)
- OpenCode: `~/.config/opencode/opencode.json` or `./opencode.json` (local)

**Detection signals:**
- Claude Code: `.claude/` directory or `CLAUDE.md` file
- Cursor: `.cursor/` directory or `.cursorrules` file
- Codex: `.codex/` directory or `CODEX.md` file
- OpenCode: `.opencode/` directory or `opencode.json` file

### Hook Features & Configuration

Tasuku hooks are configurable with `--quiet`, `--disable`, and `--list-features` flags.

#### prompt-check Features

The `prompt-check` hook runs on `UserPromptSubmit` and analyzes user messages.

**Context Surfacing** (surfaces relevant info):
| Feature | Description |
|---------|-------------|
| `session_continuity` | Shows in-progress tasks when user says "continue"/"resume" |
| `decision_lookup` | Surfaces related decisions when asking questions |
| `learning_lookup` | Surfaces related learnings when asking questions |
| `task_reference` | Shows task context when task ID is mentioned |
| `task_surfacing` | Finds related tasks by keyword matching |

**Nudges** (prompts for action):
| Feature | Description |
|---------|-------------|
| `rule_detection` | Detects "Never X"/"Always Y" patterns, suggests `tk learn` |
| `bug_detection` | Prompts to track bug reports with `tk task add --tag bug` |
| `work_detection` | Suggests creating task for significant work requests |
| `stuck_detection` | Offers help when user seems stuck/frustrated |
| `shipping_check` | Pre-ship checklist on deploy/release mentions |
| `learning_capture` | Captures "TIL"/"I learned" as learnings |
| `decision_capture` | Prompts to record "X or Y" decision points |
| `scope_warning` | Warns about scope expansion mid-task |

```bash
tk hooks prompt-check --list-features     # List all features
tk hooks prompt-check --quiet             # Context only, no nudges
tk hooks prompt-check --disable=shipping_check,scope_warning
```

#### todo-check Features

The `todo-check` hook runs on `PostToolUse` for TodoWrite and Bash commands.

| Feature | Description |
|---------|-------------|
| `bugfix_learning` | Prompts for learnings after bug fix tasks complete |
| `project_task` | Suggests persisting project-level tasks to tk |
| `test_failure` | Detects test failures, suggests tracking |
| `git_commit` | Links git commits to related in-progress tasks |

```bash
tk hooks todo-check --list-features       # List all features
tk hooks todo-check --quiet               # Minimal output
tk hooks todo-check --disable=test_failure,git_commit
```

#### Hook Version Tracking

Hooks include version tracking to alert when updates are available:
- Version is written to `.claude/.tasuku-hooks-version` on install
- SessionStart checks installed vs current version
- Shows update prompt: "⬆️ Hooks outdated: v0.6.0 → v0.6.1"
- Update with: `tk hooks install --force` (or `--force --local`)

## Future Enhancements (Planned)

### Git/GitHub Integration
- **Task-branch linking**: `tk start <id> --branch` could auto-create feature branches
- **CI integration**: GitHub Actions webhook to mark tasks done on PR merge

### Grove Integration (Worktree Management)
- Integrate with [Grove](https://github.com/iheanyi/grove) for worktree management
- `tk start <id> --worktree` triggers worktree creation
- **Auto-detection with graceful fallback**:
  - If Grove available → `grove_new` for full experience (worktree + dev server + port)
  - If Grove not available → native `git worktree add` for basic isolation
  - Print tip: "Install Grove for automatic dev server management"
- `tk done <id>` can optionally clean up worktree/stop Grove server
- Detection: Check Claude/Cursor/Codex/OpenCode MCP settings for Grove config, or probe `grove_list`

## Recently Completed Features

### PR Generation (v0.5.0)
- `tk pr create` - Create PRs linked to tasks
- `tk pr create --task <id>` - Auto-populate PR from task details
- `tk pr create --task <id> --done` - Create PR and mark task done
- `tk pr list` - List open PRs

### "Never/Always" Learning Detection (v0.6.0)
- `prompt-check` hook detects rule patterns in user messages
- Auto-suggests `tk learn` for "Never X" or "Always Y" statements
- Surfaces related learnings when asking questions
- Rule detection from conversation context (not just direct statements)

## Learnings

- TUI/CLI/MCP/Skills Parity Principle: Every capability should be accessible through all interfaces (TUI, CLI, MCP tools, Skills). Agents interact via MCP/Skills, humans via CLI/TUI - same capabilities, different UX. When adding new functionality, consider all four interfaces.
- Leverage Cobra's command structure for ergonomic shortcuts: Add root-level commands like `tk learn` as shortcuts for nested commands like `tk learning add`. This provides a better UX without duplicating logic. Example: `newLearnShortcutCmd()` in root.go provides `tk learn "insight"` as a direct shortcut.
- Always audit MCP tools, Claude Code hooks, and nudges when adding new functionality (CLI commands, MCP methods, or features). Check: (1) MCP/CLI parity, (2) Tool descriptions include WHEN to use and follow-up hints, (3) Response enhancements with warnings/suggestions, (4) Hook integration for SessionStart/Stop/PostToolUse. See CLAUDE.md 'Adding New Functionality Checklist' for full details.
- Whenever a tk CLI command fails or feels clunky, reflect on whether this highlights a UX gap. If it does, add the missing functionality. Example: tk task done a b c failing because it only accepts 1 arg → should support multiple task IDs.
- When adding goroutines or channels, always ensure they don't leak: use buffered channels, close channels when done, use context for cancellation, and verify goroutines exit properly.
- Never use O(n²) or worse algorithms when O(n log n) or O(n) alternatives exist. Replace bubble/selection/insertion sorts with slices.SortFunc, use maps for lookups instead of nested loops, pre-compute lookup sets. Example: Replace bubble sort with slices.SortFunc (internal/mcp/server.go:1880).
- Parallelize independent I/O operations using goroutines. When multiple reads/fetches don't depend on each other, run them concurrently with channels or errgroup. Example: Dashboard handler reads tasks + archived in parallel (internal/http/server.go:1068).
- Leverage Go's type system fully: use generics for reusable data structures, define interfaces for abstraction, use custom types for domain concepts (e.g., type Status string), and prefer compile-time safety over runtime checks. Clarity trumps cleverness.
- Always ensure switch cases that should handle a key return early to prevent fall-through to default key handlers. In TUI apps with bubbles/bubbletea, unhandled keys can pass to child components and cause unexpected state changes.
- Always ensure MCP tool schema 'properties' include all parameters that the handler accepts. When implementing a handler with optional flags (like 'unblock' in tk_start), the flag MUST be documented in the InputSchema 'properties' with description. Missing schema properties means AI agents won't know the flag exists and can't use it effectively. This is a critical UX issue - verify schema matches handler implementation.
