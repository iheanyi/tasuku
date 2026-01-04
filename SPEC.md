# Tasuku Specification

**Version:** 1.0.0-draft
**Status:** Draft
**Last Updated:** 2024-01-04

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
│  │   CLI (tk)   │  │  MCP Server  │  │  Git Hooks   │       │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘       │
│         │                 │                 │                │
│         └────────────┬────┴────────────────┘                │
│                      │                                       │
│              ┌───────▼───────┐                              │
│              │     Store     │                              │
│              │  (JSON + flock)│                              │
│              └───────┬───────┘                              │
│                      │                                       │
│              ┌───────▼───────┐                              │
│              │ .tasuku.json  │                              │
│              └───────────────┘                              │
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

## 3. Data Model

### 3.1 File Location

The task file lives at `.tasuku.json` in the repository root. It travels with the code.

### 3.2 Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["version", "tasks", "context"],
  "properties": {
    "version": {
      "type": "integer",
      "description": "Schema version for migrations",
      "const": 1
    },
    "tasks": {
      "type": "object",
      "additionalProperties": {
        "$ref": "#/definitions/Task"
      }
    },
    "context": {
      "$ref": "#/definitions/Context"
    }
  },
  "definitions": {
    "Task": {
      "type": "object",
      "required": ["status", "description", "created_at", "updated_at"],
      "properties": {
        "status": {
          "type": "string",
          "enum": ["ready", "in_progress", "blocked", "done"]
        },
        "description": {
          "type": "string"
        },
        "blocked_by": {
          "type": "array",
          "items": { "type": "string" }
        },
        "owner": {
          "type": ["string", "null"]
        },
        "created_at": {
          "type": "string",
          "format": "date-time"
        },
        "updated_at": {
          "type": "string",
          "format": "date-time"
        }
      }
    },
    "Context": {
      "type": "object",
      "properties": {
        "learnings": {
          "type": "array",
          "items": { "type": "string" }
        },
        "decisions": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/Decision"
          }
        },
        "notes": {
          "type": "object",
          "additionalProperties": {
            "type": "array",
            "items": { "type": "string" }
          }
        }
      }
    },
    "Decision": {
      "type": "object",
      "required": ["id", "chose", "over", "because"],
      "properties": {
        "id": { "type": "string" },
        "chose": { "type": "string" },
        "over": {
          "type": "array",
          "items": { "type": "string" }
        },
        "because": { "type": "string" }
      }
    }
  }
}
```

### 3.3 Task States

```
     ┌─────────┐
     │  ready  │ ◄── Initial state
     └────┬────┘
          │ tk start
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
                 │ blocker resolved
                 ▼
            ┌─────────┐
            │  ready  │
            └─────────┘
```

### 3.4 Task ID Format

Task IDs are kebab-case strings: `auth-fix`, `add-logout-button`, `store-core`.

Generated automatically from description if not provided:
- "Fix auth token refresh" → `fix-auth-token-refresh`
- Truncated to 32 characters
- Collisions append numeric suffix: `fix-auth-1`, `fix-auth-2`

## 4. CLI Interface

### 4.1 Commands

```bash
# Initialization
tk init                         # Create .tasuku.json in current directory

# Task Management
tk add "description"            # Add new task, returns ID
tk add "description" --id myid  # Add with specific ID
tk list                         # List all tasks
tk list --status ready          # Filter by status
tk list --blocked               # Show blocked tasks only
tk show <id>                    # Show task details

# Status Changes
tk start <id>                   # Mark as in_progress, set owner
tk done <id>                    # Mark as done
tk block <id> --by <other-id>   # Mark as blocked
tk unblock <id>                 # Remove all blockers, set to ready

# Context Management
tk learn "insight"              # Add a learning
tk decide <id> --chose X --over "Y,Z" --because "reason"
tk note <task-id> "note text"   # Add note to specific task
tk context                      # Output full context as JSON

# Agent Coordination
tk claim <id>                   # Claim task (with lock)
tk release <id>                 # Release claim
tk who                          # Show current owner

# MCP Server
tk serve                        # Start MCP server (stdio mode)
tk serve --port 3000            # Start HTTP mode (for testing)

# Migration
tk migrate beads                # Migrate from Beads
tk migrate beads --dry-run      # Preview migration

# Git Hooks
tk hooks install                # Install git hooks
tk hooks uninstall              # Remove git hooks
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

| Tool | Parameters | Description |
|------|------------|-------------|
| `tk_list` | `status?: string` | List tasks, optionally filtered |
| `tk_add` | `description: string, id?: string` | Create new task |
| `tk_start` | `id: string` | Mark task in_progress |
| `tk_done` | `id: string` | Mark task done |
| `tk_block` | `id: string, by: string[]` | Mark blocked |
| `tk_learn` | `insight: string` | Add learning |
| `tk_decide` | `id: string, chose: string, over: string[], because: string` | Record decision |
| `tk_context` | none | Get full context |
| `tk_claim` | `id: string, owner: string` | Claim task for agent |
| `tk_release` | `id: string` | Release task claim |

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

```go
func (s *Store) withLock(fn func() error) error {
    f, err := os.OpenFile(s.path, os.O_RDWR, 0644)
    if err != nil {
        return err
    }
    defer f.Close()

    // Acquire exclusive lock
    if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
        return fmt.Errorf("failed to acquire lock: %w", err)
    }
    defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

    return fn()
}
```

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

## 7. Beads Migration

### 7.1 Beads File Format

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

### 7.2 Migration Mapping

| Beads | Tasuku |
|-------|--------|
| `open` | `ready` |
| `in-progress` | `in_progress` |
| `blocked` | `blocked` |
| `closed` | `done` |
| Issue body | `notes[id]` |
| Comments | Appended to `notes[id]` |

### 7.3 Migration Command

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

## Appendix A: Example .tasuku.json

```json
{
  "version": 1,
  "tasks": {
    "auth-fix": {
      "status": "in_progress",
      "description": "Fix token refresh race condition in auth module",
      "blocked_by": [],
      "owner": "claude-agent-1",
      "created_at": "2024-01-04T10:00:00Z",
      "updated_at": "2024-01-04T11:30:00Z"
    },
    "add-logout": {
      "status": "ready",
      "description": "Add logout button to navbar",
      "blocked_by": ["auth-fix"],
      "owner": null,
      "created_at": "2024-01-04T10:05:00Z",
      "updated_at": "2024-01-04T10:05:00Z"
    }
  },
  "context": {
    "learnings": [
      "Auth module uses JWT with 1hr expiry, refresh logic at src/auth/refresh.ts:42",
      "Race condition occurs when two requests trigger refresh simultaneously"
    ],
    "decisions": [
      {
        "id": "refresh-lock",
        "chose": "Mutex lock around refresh call",
        "over": ["Request deduplication", "Optimistic locking"],
        "because": "Simpler implementation, refresh is infrequent enough that lock contention is not a concern"
      }
    ],
    "notes": {
      "auth-fix": [
        "Found the bug at line 42 - two concurrent requests can both see expired token",
        "Fix: Add async-mutex package, wrap refresh() call"
      ]
    }
  }
}
```

## Appendix B: Comparison with Beads

| Feature | Beads | Tasuku |
|---------|-------|--------|
| Storage | SQLite + markdown | Single JSON file |
| CLI | `bd` | `tk` |
| Context injection | Push (constant reminders) | Pull (agent queries) |
| Locking | SQLite WAL | flock |
| Agent focus | Partial | Primary |
| Complexity | ~5000 lines | Target: ~500 lines |
| Dependencies | Many (SQLite, etc) | Minimal |
