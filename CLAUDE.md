# Tasuku Development Guidelines

## Overview

Tasuku is an agent-first task management system. It's designed for AI agents working on codebases, prioritizing:
- **Pull over push**: Agents query when needed, no constant injections
- **Parallel-safe**: File locking for multiple agents
- **Minimal context**: Only load what's needed for the current task
- **Human-readable**: JSON files, can be edited by hand

## Architecture

```
tasuku/
├── cmd/tk/              # CLI entrypoint
├── internal/
│   ├── store/           # Storage backends (V2 file, V3 directory)
│   ├── task/            # Task domain logic
│   ├── http/            # HTTP REST API server
│   ├── mcp/             # MCP server for Claude Code integration
│   └── tui/             # Terminal UI (bubble tea)
├── .tasuku/             # V3 directory-based storage (default)
│   ├── tasks/           # Individual task JSON files
│   ├── archive/         # Archived task files
│   └── context/         # learnings.json, decisions.json
└── CLAUDE.md            # This file
```

### Storage Formats

**V3 (Default)**: Directory-based storage in `.tasuku/`
- Each task is a separate JSON file in `tasks/<id>.json`
- Archived tasks in `archive/<id>.json`
- Context stored in `context/learnings.json` and `context/decisions.json`
- Better for git diffs (one file per task = cleaner commits)
- Parallel-safe with per-file locking

**V2 (Legacy)**: Single `.tasuku.json` file
- All tasks in one JSON file
- Auto-detected and supported for backwards compatibility
- Migrate with `tk migrate v3`

## CLI Command: `tk`

```bash
# Initialization
tk init                    # Create .tasuku/ directory (V3)

# Task Management (noun-verb style)
tk task list               # List all tasks
tk task list --tree        # Show hierarchical subtask view
tk task add "description"  # Add a task
tk task add "desc" --parent <id>  # Add subtask
tk task start <id>         # Mark task in_progress
tk task done <id>          # Mark task complete
tk task block <id> --by <other>   # Mark blocked
tk task show <id>          # Show task details
tk task delete <id>        # Delete a task

# Context
tk learn "insight"         # Add a learning
tk decide <id> --chose X --over Y,Z --because "reason"
tk context show            # Dump full context (for agent consumption)

# Server
tk serve mcp               # Start MCP server (for AI tools)
tk serve http              # Start HTTP REST API on :3000
tk serve http --port 8080  # Start HTTP on custom port

# Hooks
tk hooks install           # Install git pre-commit/post-commit hooks
tk hooks uninstall         # Remove Tasuku git hooks
tk hooks plan-sync plan.md # Extract tasks from plan file (uses nudge rule)
tk hooks session           # Display context summary

# Migration
tk migrate v3              # Migrate from .tasuku.json to .tasuku/
tk migrate beads           # Migrate from Beads format
```

## Data Model

### V3 Directory Structure (Default)

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

**Task File** (`.tasuku/tasks/task-id.json`):
```json
{
  "status": "ready|in_progress|blocked|done",
  "description": "What needs to be done",
  "blocked_by": ["other-task-id"],
  "parent_id": "parent-task",
  "owner": "agent-1",
  "priority": 2,
  "tags": ["backend", "api"],
  "fields": {"estimate": "2h"},
  "time_spent": 3600,
  "notes": [{"text": "Note 1", "created_at": "..."}],
  "created_at": "2024-01-04T10:00:00Z",
  "updated_at": "2024-01-04T10:00:00Z"
}
```

**Learnings** (`.tasuku/context/learnings.json`):
```json
[
  {"text": "Things discovered while working", "created_at": "..."}
]
```

**Decisions** (`.tasuku/context/decisions.json`):
```json
[
  {
    "id": "decision-id",
    "chose": "Option A",
    "over": ["Option B", "Option C"],
    "because": "Reasoning",
    "created_at": "2024-01-04T10:00:00Z"
  }
]
```

### V2 Format (Legacy)

Single `.tasuku.json` file with all data in one place. Supported for backwards compatibility.

## Development Workflow

### We dogfood tasuku while building it

1. Tasks are tracked in `.tasuku/` at repo root (V3 format)
2. Use `tk` commands (once built) or edit JSON files directly
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

### CLI/MCP Parity Principle

**Every CLI command must have a corresponding MCP tool.** This is critical for agent-first design:

- Agents interact via MCP tools, humans via CLI
- Same capabilities, same behavior, different interfaces
- When adding a new CLI command, always add the MCP tool
- When adding a new MCP tool, consider if CLI equivalent is needed

This ensures agents can do everything humans can do, which is the whole point of Tasuku.

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
| **Blocking & Dependencies** |||
| `tk_block` | `tk task block` | Mark task as blocked by others |
| `tk_unblock` | `tk task unblock` | Remove blockers (all or specific) |
| **Ownership & Coordination** |||
| `tk_owner` | `tk task owner` | Set or clear task owner |
| `tk_claim` | `tk task claim` | Claim task for exclusive agent work |
| `tk_release` | `tk task release` | Release claimed task |
| **Context & Search** |||
| `tk_context` | `tk context show` | Get full context for agent consumption |
| `tk_find` | `tk task find` | Search across tasks, notes, learnings, decisions |
| `tk_learn` | `tk learn` | Add a learning to context |
| `tk_decide` | `tk decide` | Record an architectural decision |
| `tk_note` | `tk note` | Add a note to a task |
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
| **Agent Workflow** |||
| `tk_suggest` | `tk suggest` | Analyze if a task should persist to tk or stay session-only |

### Running the server

```bash
tk serve --port 3000
```

Or via stdio (for Claude Code):
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

Claude Code hooks automatically integrate Tasuku into your workflow:

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
tk hooks install --claude  # Adds SessionStart and Stop hooks
```

## Future Enhancements (Planned)

### Git/GitHub Integration
- **Task-branch linking**: `tk start` could auto-create feature branches
- **PR generation**: `tk pr` to create PR from task details
- **CI integration**: GitHub Actions webhook to mark tasks done on PR merge

### Grove Integration (Worktree Management)
- Integrate with [Grove](https://github.com/iheanyi/grove) for worktree management
- `tk start <id> --worktree` triggers worktree creation
- **Auto-detection with graceful fallback**:
  - If Grove available → `grove_new` for full experience (worktree + dev server + port)
  - If Grove not available → native `git worktree add` for basic isolation
  - Print tip: "Install Grove for automatic dev server management"
- `tk done <id>` can optionally clean up worktree/stop Grove server
- Detection: Check Claude/Cursor MCP settings for Grove config, or probe `grove_list`

### "Never/Always" Learning Detection
- Detect phrases like "Never do X" or "Always use Y" in agent interactions
- Auto-suggest promoting these as permanent learnings
- Hook into agent conversations to capture institutional knowledge
