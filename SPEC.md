# Tasuku Specification

**Version:** 3.0.0
**Status:** Stable
**Last Updated:** 2025-01-04

## 1. Introduction

### 1.1 Purpose

Tasuku is an agent-first task management system designed for AI agents working on codebases. Unlike traditional issue trackers designed for human coordination, Tasuku optimizes for:

- Minimal context loading (token efficiency)
- Parallel agent coordination
- Context persistence across sessions
- Pull-based information retrieval (no constant injections)

### 1.2 Name

"Tasuku" (タスク) is Japanese for "task". The CLI command is `tk` to avoid collisions with common tools.

### 1.3 Non-Goals

- Beautiful UI (agents don't need it)
- Sprint planning, burndown charts
- Email notifications
- Enterprise permission systems

## 2. Architecture

### 2.1 Components

```
┌─────────────────────────────────────────────────────────────┐
│                        Tasuku                                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   CLI (tk)   │  │  MCP Server  │  │  HTTP API    │       │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘       │
│         │                 │                 │                │
│         └────────────┬────┴────────────────┘                │
│                      │                                       │
│              ┌───────▼───────┐                              │
│              │    Storage    │                              │
│              │  (Interface)  │                              │
│              └───────┬───────┘                              │
│                      │                                       │
│         ┌────────────┴────────────┐                         │
│         │                         │                         │
│  ┌──────▼──────┐          ┌──────▼──────┐                   │
│  │  .tasuku/   │          │.tasuku.json │                   │
│  │ (V3 - Dir)  │          │ (V2 - File) │                   │
│  └─────────────┘          └─────────────┘                   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Design Decisions

| Decision | Choice | Alternatives Considered | Rationale |
|----------|--------|------------------------|-----------|
| Language | Go | TypeScript, Rust | Single binary, good concurrency, fast CLI startup |
| Data Format | JSON | YAML, TOML, SQLite | Faster parsing, unambiguous, better for agents |
| Locking | flock | Advisory locks, SQLite WAL | Simple, works on macOS/Linux, sufficient for local |
| CLI Name | `tk` | `t`, `tasuku`, `tsk` | Short, no collisions, memorable |
| Agent Model | Hierarchical + Parallel | Pure parallel, Sequential | Best token efficiency |
| Storage V3 | Directory | Single file | Cleaner git diffs, per-file locking, better archival |
| Subtasks | parent_id field | Nested directories | Flat storage with references, enables tree view |

## 3. Data Model

### 3.1 Storage Location

**V3 (Default)**: Directory-based storage at `.tasuku/`
```
.tasuku/
├── tasks/           # Active task files (one per task)
│   └── <id>.json
├── archive/         # Archived completed tasks
│   └── <id>.json
└── context/         # Shared context
    ├── learnings.json
    └── decisions.json
```

**V2 (Legacy)**: Single file at `.tasuku.json` in repository root.

### 3.2 V3 Task Schema

Individual task file (`.tasuku/tasks/<id>.json`):

```json
{
  "status": "ready|in_progress|blocked|done",
  "description": "What needs to be done",
  "priority": 2,
  "parent_id": "optional-parent-task-id",
  "blocked_by": ["other-task-id"],
  "owner": "agent-1",
  "tags": ["backend", "api"],
  "fields": {"estimate": "2h", "sprint": "3"},
  "time_spent": 3600,
  "notes": [
    {"text": "Note text", "created_at": "2025-01-04T10:00:00Z"}
  ],
  "created_at": "2025-01-04T10:00:00Z",
  "updated_at": "2025-01-04T11:30:00Z"
}
```

### 3.3 V3 Context Schema

Learnings (`.tasuku/context/learnings.json`):
```json
[
  {"text": "Discovery about the codebase", "created_at": "2025-01-04T10:00:00Z"}
]
```

Decisions (`.tasuku/context/decisions.json`):
```json
[
  {
    "id": "decision-id",
    "chose": "Option A",
    "over": ["Option B", "Option C"],
    "because": "Reasoning for the choice",
    "created_at": "2025-01-04T10:00:00Z"
  }
]
```

### 3.4 V2 Schema (Legacy)

Single `.tasuku.json` file with all data:
```json
{
  "version": 2,
  "tasks": { "<id>": { /* task object */ } },
  "archived": { "<id>": { /* archived task */ } },
  "context": {
    "learnings": [{ "text": "...", "created_at": "..." }],
    "decisions": [{ /* decision object */ }],
    "notes": { "<task-id>": [{ "text": "...", "created_at": "..." }] }
  }
}
```

### 3.5 Task States

```
     ┌─────────┐
     │  ready  │ ◄── Initial state
     └────┬────┘
          │ tk task start
          ▼
   ┌──────────────┐
   │ in_progress  │
   └──────┬───────┘
          │
    ┌─────┴─────┐
    │           │
    ▼           ▼
┌────────┐  ┌─────────┐
│  done  │  │ blocked │
└────────┘  └────┬────┘
     │           │ blocker resolved
     │           ▼
     │      ┌─────────┐
     │      │  ready  │
     │      └─────────┘
     ▼
┌──────────┐
│ archived │  (moved to .tasuku/archive/)
└──────────┘
```

### 3.6 Task ID Format

Task IDs are kebab-case strings: `auth-fix`, `add-logout-button`, `store-core`.

Generated automatically from description if not provided:
- "Fix auth token refresh" → `fix-auth-token-refresh`
- Truncated to 32 characters
- Collisions append numeric suffix: `fix-auth-1`, `fix-auth-2`

## 4. CLI Interface

### 4.1 Commands

```bash
# Initialization
tk init                          # Create .tasuku/ directory (V3)
tk init --format v2              # Create legacy .tasuku.json

# Task Management (noun-verb style)
tk task add "description"        # Add new task, returns ID
tk task add "desc" --id myid     # Add with specific ID
tk task add "desc" --parent id   # Add as subtask
tk task list                     # List all tasks
tk task list --tree              # Show hierarchical subtask view
tk task list --status ready      # Filter by status
tk task show <id>                # Show task details

# Status Changes
tk task start <id>               # Mark as in_progress, set owner
tk task done <id>                # Mark as done
tk task block <id> --by <other>  # Mark as blocked
tk task unblock <id>             # Remove all blockers, set to ready
tk task delete <id>              # Delete a task

# Tags and Custom Fields
tk task tag add <id> <tag>       # Add tag to task
tk task tag remove <id> <tag>    # Remove tag
tk task field set <id> <k> <v>   # Set custom field
tk task field remove <id> <k>    # Remove custom field

# Time Tracking
tk task timer start <id>         # Start timer on task
tk task timer stop <id>          # Stop timer, record elapsed
tk task timer status             # Show running timers

# Archiving
tk task archive add <id>         # Archive a done task
tk task archive list             # List archived tasks
tk task archive restore <id>     # Restore archived task

# Context Management
tk learn "insight"               # Add a learning
tk decide <id> --chose X --over "Y,Z" --because "reason"
tk note add <task-id> "note"     # Add note to specific task
tk context show                  # Output full context as JSON

# Agent Coordination
tk task claim <id>               # Claim task (with lock)
tk task release <id>             # Release claim

# Servers
tk serve                         # Start MCP server (stdio mode)
tk serve --http :3000            # Start HTTP REST API

# Migration
tk migrate v3                    # Migrate from V2 to V3 format
tk migrate beads                 # Migrate from Beads
tk migrate beads --dry-run       # Preview migration

# Git Hooks
tk hooks install                 # Install git hooks
tk hooks uninstall               # Remove git hooks
```

### 4.2 Output Formats

Default output is human-readable. Add `--json` for machine parsing:

```bash
$ tk list
 ready       store-core     Implement core store with JSON...
 ready       cli-init       Implement tk init command...
 in_progress spec-doc       Write formal SPEC.md...

$ tk list --json
{
  "tasks": [
    {"id": "store-core", "status": "ready", ...},
    ...
  ]
}
```

### 4.3 Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Task not found |
| 3 | File locked (try again) |
| 4 | Invalid state transition |
| 5 | Migration error |

## 5. MCP Server

### 5.1 Transport

Primary: stdio (for Claude Code integration)
Secondary: HTTP (for testing/debugging)

### 5.2 Tools

Full CLI parity - every CLI command has a corresponding MCP tool:

| Tool | Parameters | Description |
|------|------------|-------------|
| **Core Task Operations** |||
| `tk_list` | `status?: string` | List tasks, optionally filtered |
| `tk_add` | `description: string, id?: string, parent?: string` | Create new task |
| `tk_show` | `id: string` | Get task details |
| `tk_start` | `id: string` | Mark task in_progress |
| `tk_done` | `id: string` | Mark task done |
| `tk_pause` | `id: string` | Revert to ready |
| `tk_edit` | `id: string, description: string` | Update description |
| `tk_delete` | `id: string` | Delete task |
| `tk_priority` | `id: string, priority: string` | Set priority |
| **Blocking & Dependencies** |||
| `tk_block` | `id: string, by: string[]` | Mark blocked |
| `tk_unblock` | `id: string` | Remove blockers |
| **Ownership** |||
| `tk_owner` | `id: string, owner?: string` | Set/clear owner |
| `tk_claim` | `id: string, owner: string` | Claim task for agent |
| `tk_release` | `id: string` | Release task claim |
| **Tags & Fields** |||
| `tk_tag_add` | `id: string, tag: string` | Add tag |
| `tk_tag_remove` | `id: string, tag: string` | Remove tag |
| `tk_field_set` | `id: string, key: string, value: string` | Set custom field |
| `tk_field_remove` | `id: string, key: string` | Remove custom field |
| **Time Tracking** |||
| `tk_timer_start` | `id: string` | Start timer |
| `tk_timer_stop` | `id: string` | Stop timer |
| `tk_timer_status` | none | Get running timers |
| **Context** |||
| `tk_context` | none | Get full context |
| `tk_find` | `query: string` | Search tasks/notes/learnings |
| `tk_learn` | `insight: string` | Add learning |
| `tk_decide` | `id, chose, over[], because` | Record decision |
| `tk_note` | `task_id, note: string` | Add note |
| **Archiving** |||
| `tk_archive` | `task_id: string, summary?: string` | Archive done task |
| `tk_archive_restore` | `task_id: string` | Restore archived |
| `tk_archive_list` | none | List archived tasks |

### 5.3 Example MCP Config

```json
{
  "mcpServers": {
    "tasuku": {
      "command": "tk",
      "args": ["serve"]
    }
  }
}
```

## 6. File Locking

### 6.1 Mechanism

Use `flock(2)` for file-level locking:

**V3 (Directory-based)**: Lock individual task files for granular concurrency.
```go
func (s *DirStore) updateTask(id string, fn func(*task.Task) error) error {
    path := filepath.Join(s.dir, "tasks", id+".json")
    f, err := os.OpenFile(path, os.O_RDWR, 0644)
    if err != nil {
        return err
    }
    defer f.Close()

    // Acquire exclusive lock on this task file only
    if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
        return fmt.Errorf("failed to acquire lock: %w", err)
    }
    defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

    return fn()
}
```

**V2 (Single file)**: Lock entire `.tasuku.json` file.

### 6.2 Lock Timeout

Default timeout: 5 seconds. If lock cannot be acquired, exit with code 3.

### 6.3 Windows Support

Windows doesn't have `flock`. Use `LockFileEx` instead:

```go
// +build windows
func lockFile(f *os.File) error {
    return windows.LockFileEx(windows.Handle(f.Fd()),
        windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0,
        &windows.Overlapped{})
}
```

## 7. Migration

### 7.1 V2 to V3 Migration

Migrate from single-file `.tasuku.json` to directory-based `.tasuku/`:

```bash
$ tk migrate v3
Migrating from V2 to V3 format...
  Created .tasuku/tasks/
  Created .tasuku/archive/
  Created .tasuku/context/
  Migrated 15 active tasks
  Migrated 42 archived tasks
  Migrated learnings and decisions
Migration complete!
Original .tasuku.json preserved (delete manually if desired)
```

### 7.2 Beads Migration

Beads stores issues in `.beads/issues/` as individual markdown files with YAML frontmatter:

```markdown
---
id: PROJ-1
title: Fix auth bug
status: open
created: 2024-01-01T00:00:00Z
---

Description here...
```

**Status Mapping:**

| Beads | Tasuku |
|-------|--------|
| `open` | `ready` |
| `in-progress` | `in_progress` |
| `blocked` | `blocked` |
| `closed` | `done` |
| Issue body | `notes[id]` |
| Comments | Appended to `notes[id]` |

**Migration Command:**

```bash
$ tk migrate beads
Migrating from Beads...
  Found 15 issues in .beads/issues/
  Converted PROJ-1 → proj-1 (ready)
  Converted PROJ-2 → proj-2 (in_progress)
  ...
Migration complete: 15 tasks imported
Original .beads/ preserved (delete manually if desired)
```

## 8. Git Hooks

### 8.1 pre-commit

Validates `.tasuku.json` is valid JSON and matches schema.

```bash
#!/bin/bash
if [ -f .tasuku.json ]; then
    tk validate || exit 1
fi
```

### 8.2 post-commit

Parses commit message for task references and suggests status updates.

```bash
#!/bin/bash
# Parse commit message for (#task-id) patterns
# Suggest: tk done task-id
```

**Note:** Hooks do NOT auto-modify. They only validate or suggest.

## 9. Testing Strategy

### 9.1 Unit Tests

Every package has corresponding `*_test.go`:

```
internal/store/store_test.go      # JSON operations, locking
internal/task/task_test.go        # Domain logic, state transitions
internal/mcp/server_test.go       # MCP protocol handling
```

### 9.2 Integration Tests

Located in `testdata/`:

```
testdata/parallel_writes/         # Two goroutines writing simultaneously
testdata/migration/beads/         # Sample Beads repo for migration testing
testdata/corrupt/                 # Malformed JSON handling
```

### 9.3 Race Detection

All tests run with `-race`:

```bash
go test -race ./...
```

### 9.4 Manual Verification

Before each release:

- [ ] `tk init` creates valid file
- [ ] Full add/start/done cycle works
- [ ] Parallel `tk` commands don't corrupt file
- [ ] MCP server responds correctly
- [ ] Beads migration preserves all data
- [ ] Git hooks install/uninstall cleanly

## 10. Future Considerations

### 10.1 Potential Additions (Not in v1)

- **Remote sync**: Push/pull to git remote automatically
- **Conflict resolution**: Merge algorithm for task file
- **Agent analytics**: Track which agents complete which tasks
- **Priority field**: Ordering tasks by importance

### 10.2 Explicitly Excluded

- Web UI (use `tk list` or edit JSON)
- Multi-repo coordination (one file per repo)
- Real-time sync (pull on demand)
- Encryption (it's local task data)

## Appendix A: Example V3 Directory Structure

```
.tasuku/
├── tasks/
│   ├── auth-fix.json
│   └── add-logout.json
├── archive/
│   └── setup-ci.json
└── context/
    ├── learnings.json
    └── decisions.json
```

**`.tasuku/tasks/auth-fix.json`:**
```json
{
  "status": "in_progress",
  "description": "Fix token refresh race condition in auth module",
  "priority": 1,
  "blocked_by": [],
  "owner": "claude-agent-1",
  "tags": ["backend", "security"],
  "time_spent": 3600,
  "notes": [
    {"text": "Found the bug at line 42 - two concurrent requests can both see expired token", "created_at": "2025-01-04T10:30:00Z"},
    {"text": "Fix: Add async-mutex package, wrap refresh() call", "created_at": "2025-01-04T11:00:00Z"}
  ],
  "created_at": "2025-01-04T10:00:00Z",
  "updated_at": "2025-01-04T11:30:00Z"
}
```

**`.tasuku/tasks/add-logout.json`:**
```json
{
  "status": "blocked",
  "description": "Add logout button to navbar",
  "priority": 2,
  "blocked_by": ["auth-fix"],
  "parent_id": null,
  "owner": null,
  "tags": ["frontend"],
  "created_at": "2025-01-04T10:05:00Z",
  "updated_at": "2025-01-04T10:05:00Z"
}
```

**`.tasuku/context/learnings.json`:**
```json
[
  {"text": "Auth module uses JWT with 1hr expiry, refresh logic at src/auth/refresh.ts:42", "created_at": "2025-01-04T10:15:00Z"},
  {"text": "Race condition occurs when two requests trigger refresh simultaneously", "created_at": "2025-01-04T10:45:00Z"}
]
```

**`.tasuku/context/decisions.json`:**
```json
[
  {
    "id": "refresh-lock",
    "chose": "Mutex lock around refresh call",
    "over": ["Request deduplication", "Optimistic locking"],
    "because": "Simpler implementation, refresh is infrequent enough that lock contention is not a concern",
    "created_at": "2025-01-04T11:00:00Z"
  }
]
```

## Appendix B: Comparison with Beads

| Feature | Beads | Tasuku |
|---------|-------|--------|
| Storage | SQLite + markdown | JSON files (V3: directory, V2: single file) |
| CLI | `bd` | `tk` |
| Context injection | Push (constant reminders) | Pull (agent queries) |
| Locking | SQLite WAL | flock (per-file in V3) |
| Agent focus | Partial | Primary |
| Subtasks | Via labels/dependencies | Native parent_id with tree view |
| Time tracking | No | Built-in timer system |
| Tags/Custom fields | Labels only | Tags + arbitrary key-value fields |
| Dependencies | Many (SQLite, etc) | Minimal (pure Go) |
