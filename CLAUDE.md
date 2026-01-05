# Tasuku Development Guidelines

## Overview

Tasuku is an agent-first task management system. It's designed for AI agents working on codebases, prioritizing:
- **Pull over push**: Agents query when needed, no constant injections
- **Parallel-safe**: File locking for multiple agents
- **Minimal context**: Only load what's needed for the current task
- **Human-readable**: JSON file, can be edited by hand

## Architecture

```
tasuku/
├── cmd/tk/              # CLI entrypoint
├── internal/
│   ├── store/           # JSON file operations with flock
│   ├── task/            # Task domain logic
│   └── mcp/             # MCP server for Claude Code integration
├── .tasuku.json         # Task file (lives in user's repo)
└── CLAUDE.md            # This file
```

## CLI Command: `tk`

```bash
tk init                    # Create .tasuku.json
tk list                    # List all tasks
tk add "description"       # Add a task
tk start <id>              # Mark task in_progress
tk done <id>               # Mark task complete
tk block <id> --by <other> # Mark blocked
tk learn "insight"         # Add a learning
tk decide <id> --chose X --over Y,Z --because "reason"
tk context                 # Dump full context (for agent consumption)
tk serve                   # Start MCP server
tk hooks install           # Install git hooks (optional)
```

## Data Model

```json
{
  "version": 1,
  "tasks": {
    "task-id": {
      "status": "ready|in_progress|blocked|done",
      "description": "What needs to be done",
      "blocked_by": ["other-task-id"],
      "owner": "agent-1",
      "created_at": "2024-01-04T10:00:00Z",
      "updated_at": "2024-01-04T10:00:00Z"
    }
  },
  "context": {
    "learnings": ["Things discovered while working"],
    "decisions": [
      {
        "id": "decision-id",
        "chose": "Option A",
        "over": ["Option B", "Option C"],
        "because": "Reasoning",
        "created_at": "2024-01-04T10:00:00Z"
      }
    ],
    "notes": {
      "task-id": [
        {
          "text": "Note 1",
          "created_at": "2024-01-04T10:00:00Z"
        }
      ]
    }
  }
}
```

## Development Workflow

### We dogfood tasuku while building it

1. Tasks are tracked in `.tasuku.json` at repo root
2. Use `tk` commands (once built) or edit JSON directly
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

### Integration Tests

```bash
go test ./... -tags=integration
```

Located in `testdata/` scenarios:
- `testdata/parallel_agents/` - Test file locking
- `testdata/corrupt_json/` - Error handling

### Manual Verification Checklist

Before merging any PR:

- [ ] `tk init` creates valid `.tasuku.json`
- [ ] `tk add/start/done` cycle works
- [ ] `tk list` shows correct statuses
- [ ] `tk context` outputs valid JSON
- [ ] Parallel writes don't corrupt file
- [ ] MCP server responds to tool calls

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

```go
// Acquire exclusive lock
f, _ := os.OpenFile(".tasuku.json", os.O_RDWR, 0644)
syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

// Now safe to read-modify-write
```

## Code Style

- `go fmt` on save
- `golint` clean
- Errors are wrapped with context: `fmt.Errorf("store: failed to read: %w", err)`
- No magic - explicit is better than implicit

## Key Decisions

Record architectural decisions here as we make them:

1. **JSON over YAML** - Faster parsing, no ambiguity, better for agents
2. **flock for locking** - Simple, works on macOS/Linux, sufficient for local
3. **MCP over REST** - Native Claude Code integration, no HTTP overhead
4. **Single file** - `.tasuku.json` in repo root, travels with code
5. **User-specified task IDs over directory namespacing** - Task IDs are either user-provided via `--id` flag or auto-generated from description (kebab-case). We chose this over automatic directory-namespacing because:
   - **Simplicity**: Agents and humans can use short, memorable IDs like `fix-auth-bug`
   - **Flexibility**: Users can choose their own naming conventions
   - **No path coupling**: Task IDs shouldn't change if files move
   - **Cross-project clarity**: IDs like `api-v2-migration` are clearer than `src/api/migration`
   - If namespacing is needed, users can include it in the ID: `tk add "Fix bug" --id auth/login-timeout`
6. **Cobra CLI framework** - Industry standard, proper `--flag` syntax, shell completion, Viper config integration
7. **Grove integration over native worktree support** - Tasuku manages tasks, Grove manages worktrees. Each tool does one thing well. Avoids duplication and leverages existing Grove infrastructure.

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
