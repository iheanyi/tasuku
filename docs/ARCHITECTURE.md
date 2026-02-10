# Tasuku Architecture

This document provides an in-depth explanation of how Tasuku works internally.

## Overview

Tasuku is an agent-first task management system designed for AI agents working on codebases. It prioritizes:

- **Pull over push**: Agents query when needed rather than having context constantly injected
- **Parallel-safe**: Multiple agents can work simultaneously without data corruption
- **Minimal context**: Only load what's needed for the current task
- **Human-readable**: JSON files that can be edited by hand
- **Git-friendly**: One file per task for clean diffs and fewer merge conflicts

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         User/Agent                               │
└─────────────────────────────────────────────────────────────────┘
                               │
          ┌────────────────────┼────────────────────┐
          │                    │                    │
          ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│   CLI (tk)      │  │  HTTP REST API  │  │  MCP Server     │
│   cmd/tk/       │  │  internal/http/ │  │  internal/mcp/  │
└─────────────────┘  └─────────────────┘  └─────────────────┘
          │                    │                    │
          └────────────────────┼────────────────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │  Storage Interface  │
                    │  internal/store/    │
                    │                     │
                    │  ┌───────────────┐  │
                    │  │  File Locking │  │
                    │  │   (flock)     │  │
                    │  └───────────────┘  │
                    └─────────────────────┘
                               │
            ┌──────────────────┴──────────────────┐
            ▼                                      ▼
┌─────────────────────┐              ┌─────────────────────┐
│   V3: .tasuku/      │              │   V2: .tasuku.json  │
│   (directory)       │              │   (single file)     │
│                     │              │   [Legacy]          │
│   tasks/            │              │                     │
│   archive/          │              └─────────────────────┘
│   context/          │
└─────────────────────┘
```

## Directory Structure

```
tasuku/
├── cmd/tk/                 # CLI entrypoint and command implementations
│   ├── main.go            # Cobra command definitions
│   └── main_test.go       # Integration tests
├── internal/
│   ├── store/             # Storage backends with flock locking
│   │   ├── storage.go     # Storage interface and auto-detection
│   │   ├── store.go       # V2 single-file storage
│   │   ├── dirstore.go    # V3 directory-based storage
│   │   └── *_test.go      # Store unit tests
│   ├── task/              # Task domain logic and types
│   │   ├── task.go        # Task, Decision, File types
│   │   └── task_test.go   # Task logic tests
│   ├── http/              # HTTP REST API server
│   │   ├── server.go      # HTTP handlers and routing
│   │   └── server_test.go # HTTP endpoint tests
│   ├── mcp/               # MCP (Model Context Protocol) server
│   │   ├── server.go      # MCP tool implementations
│   │   └── server_test.go # MCP protocol tests
│   └── tui/               # Terminal UI (Bubble Tea)
│       └── model.go       # TUI model and views
├── schema.json            # JSON Schema for validation
├── openapi.yaml           # OpenAPI spec for HTTP REST API
└── docs/
    └── ARCHITECTURE.md    # This file
```

### V3 Storage Layout

```
.tasuku/                   # Created by `tk init`
├── tasks/                 # Active task files
│   ├── auth-fix.json
│   └── add-logout.json
├── archive/               # Completed/archived tasks
│   └── old-task.json
└── context/               # Shared context
    ├── learnings.json     # Array of learning entries
    └── decisions.json     # Array of decision entries
```

## Core Components

### 1. Store Layer (`internal/store/`)

The store provides atomic read-modify-write operations with file locking via a common interface.

```go
// Storage interface - both V2 and V3 implement this
type Storage interface {
    Read() (*task.File, error)
    Update(fn func(*task.File) error) error
    Init() error
}

// Auto-detection picks the right backend
func AutoDetect() Storage {
    if exists(".tasuku/") {
        return NewDirStore(".tasuku")  // V3
    }
    if exists(".tasuku.json") {
        return New(".tasuku.json")      // V2
    }
    return NewDirStore(".tasuku")       // Default to V3
}
```

**V3 DirStore** (directory-based):
- Each task stored in separate file: `.tasuku/tasks/<id>.json`
- Per-file locking for better parallelism
- Archived tasks in `.tasuku/archive/`
- Context in `.tasuku/context/learnings.json` and `decisions.json`

**V2 Store** (single file):
- All data in `.tasuku.json`
- Whole-file locking
- Supported for backwards compatibility

**File Locking Strategy:**

Both backends use POSIX `flock` for cooperative file locking:

```go
func (s *DirStore) updateTask(id string, fn func(*task.Task) error) error {
    path := filepath.Join(s.dir, "tasks", id+".json")
    f, err := os.OpenFile(path, os.O_RDWR, 0644)
    if err != nil {
        return err
    }
    defer f.Close()

    // Acquire exclusive lock - blocks until available
    if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
        return err
    }
    defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

    // Read, modify, write back atomically
    // ...
}
```

This ensures:
- Multiple agents can safely read/write simultaneously
- V3: Agents editing different tasks don't block each other
- No partial writes or corruption
- Lock acquisition is blocking (agents wait their turn)

### 2. Task Domain (`internal/task/`)

Core types that represent the data model:

```go
type Status string

const (
    StatusReady      Status = "ready"
    StatusInProgress Status = "in_progress"
    StatusBlocked    Status = "blocked"
    StatusDone       Status = "done"
)

type Task struct {
    Status      Status            `json:"status"`
    Description string            `json:"description"`
    Priority    *int              `json:"priority,omitempty"`
    ParentID    string            `json:"parent_id,omitempty"`    // V3: subtask support
    BlockedBy   []string          `json:"blocked_by"`
    Owner       *string           `json:"owner,omitempty"`
    Tags        []string          `json:"tags,omitempty"`         // V3: labels
    Fields      map[string]string `json:"fields,omitempty"`       // V3: custom fields
    TimeSpent   int64             `json:"time_spent,omitempty"`   // V3: time tracking (seconds)
    Notes       []Note            `json:"notes,omitempty"`        // V3: notes stored with task
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}

type Note struct {
    Text      string    `json:"text"`
    CreatedAt time.Time `json:"created_at"`
}

type Decision struct {
    ID        string    `json:"id"`
    Chose     string    `json:"chose"`
    Over      []string  `json:"over"`
    Because   string    `json:"because"`
    CreatedAt time.Time `json:"created_at"`
}

type Learning struct {
    Text      string    `json:"text"`
    CreatedAt time.Time `json:"created_at"`
}

type Context struct {
    Learnings []Learning          `json:"learnings"`   // V3: structured learnings
    Decisions []Decision          `json:"decisions"`
    Notes     map[string][]Note   `json:"notes"`       // V2 compatibility
}

type File struct {
    Version  int             `json:"version"`  // 1=original, 2=enhanced, 3=directory
    Tasks    map[string]Task `json:"tasks"`
    Archived map[string]Task `json:"archived,omitempty"`  // V3: archived tasks
    Context  Context         `json:"context"`
}
```

### 3. CLI Layer (`cmd/tk/`)

Built with [Cobra](https://github.com/spf13/cobra), providing:
- Proper `--flag` syntax (not Go's `-flag`)
- Shell completion (bash, zsh, fish)
- Subcommand structure
- Auto-generated help

**Command Structure:**

```
tk
├── init              # Create .tasuku/ directory
├── task              # Task management subcommand
│   ├── list          # List tasks (--format, --status, --tree)
│   ├── add           # Add task (--id, --priority, --parent)
│   ├── show          # Show task details
│   ├── start         # Mark in_progress
│   ├── done          # Mark complete
│   ├── block         # Set blocked_by
│   ├── unblock       # Clear blocked_by
│   ├── delete        # Delete task
│   ├── priority      # Set task priority
│   ├── find          # Search tasks/learnings/decisions
│   ├── tag           # Manage task tags
│   ├── field         # Manage custom fields
│   ├── timer         # Time tracking (start/stop/status)
│   └── archive       # Archive management
├── ready             # List unblocked ready tasks
├── learn             # Add learning (--permanent)
├── learnings         # List learnings
├── unlearn           # Remove learning
├── promote           # Move learning to context file
├── decide            # Record decision
├── note              # Add note to task
├── context           # Context subcommand (show)
├── validate          # Validate task files
├── serve             # Start MCP or HTTP server
├── mcp               # MCP installation commands
├── migrate           # Migration (v3, beads)
├── hooks             # Git hook management
├── ui                # Launch terminal UI
└── completion        # Shell completion scripts
```

### 4. HTTP REST API (`internal/http/`)

RESTful API for programmatic access:

```
GET    /tasks           List all tasks
GET    /tasks?status=X  Filter by status
POST   /tasks           Create task
GET    /tasks/{id}      Get task details
PUT    /tasks/{id}      Update task
DELETE /tasks/{id}      Delete task
GET    /ready           List ready tasks
GET    /context         Full context
POST   /learnings       Add learning
POST   /decisions       Record decision
GET    /schema          JSON schema
GET    /health          Health check
```

Each endpoint sets CORS headers for browser access and returns JSON responses.

### 5. MCP Server (`internal/mcp/`)

[Model Context Protocol](https://modelcontextprotocol.io/) server for Claude Code integration:

```go
type Server struct {
    store store.Storage  // Uses Storage interface for V2/V3 compatibility
}

func (s *Server) handleToolCall(name string, args json.RawMessage) (interface{}, error) {
    switch name {
    case "tk_list":
        return s.list(args)
    case "tk_add":
        return s.add(args)
    case "tk_tag_add":
        return s.tagAdd(args)
    // ... additional MCP tools
    }
}
```

The MCP server runs over stdio, communicating with Claude Code via JSON-RPC. Most core task/context workflows are exposed via MCP tools; some operations (like timer controls) remain CLI-first by design.

## Data Flow

### Adding a Task

```
User: tk add "Fix auth bug" --priority 1

1. CLI parses command with Cobra
2. Validates arguments
3. Calls store.Update() with modification function
4. Store acquires flock
5. Store reads current .tasuku.json
6. Modification function adds new task
7. Store writes updated JSON
8. Store releases flock
9. CLI outputs confirmation
```

### Concurrent Agent Access

```
Agent A: tk start task-1    Agent B: tk start task-2
    │                           │
    ▼                           ▼
store.Update()              store.Update()
    │                           │
    ▼                           ▼
flock(LOCK_EX)              flock(LOCK_EX)
    │                           │ (blocks, waiting)
    ▼                           │
Read .tasuku.json               │
    │                           │
    ▼                           │
Modify task-1                   │
    │                           │
    ▼                           │
Write .tasuku.json              │
    │                           │
    ▼                           ▼
flock(LOCK_UN)              (acquires lock)
                                │
                                ▼
                            Read .tasuku.json
                            (sees task-1 changes)
                                │
                                ▼
                            Modify task-2
                                │
                                ▼
                            Write .tasuku.json
                                │
                                ▼
                            flock(LOCK_UN)
```

## Context File Detection

The `promote` command auto-detects which AI tool context file to use:

```go
func detectContextFile() string {
    contextFiles := []struct {
        path        string
        description string
    }{
        {"CLAUDE.md", "Claude Code"},
        {"GEMINI.md", "Gemini"},
        {".cursorrules", "Cursor"},
        {".github/copilot-instructions.md", "GitHub Copilot"},
        {"AGENTS.md", "Generic AI agents"},
    }

    for _, cf := range contextFiles {
        if _, err := os.Stat(cf.path); err == nil {
            return cf.path
        }
    }
    return "CLAUDE.md" // default
}
```

## Priority System

Tasks have 5 priority levels:

| Level | Name     | Use Case |
|-------|----------|----------|
| 0     | Critical | Urgent blockers |
| 1     | High     | Important, do soon |
| 2     | Normal   | Default |
| 3     | Low      | Can wait |
| 4     | Backlog  | Future work |

The `ready` command sorts by priority (critical first).

## Schema Validation

The `validate` command checks .tasuku.json against the JSON Schema:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["version", "tasks", "context"],
  "properties": {
    "version": { "const": 1 },
    "tasks": {
      "additionalProperties": { "$ref": "#/definitions/Task" }
    }
  }
}
```

## Migration System

The `migrate beads` command imports from Beads format:

```go
type beadsIssue struct {
    ID           string `json:"id"`
    Title        string `json:"title"`
    Status       string `json:"status"`
    Priority     int    `json:"priority"`
    Dependencies []struct {
        Type     string `json:"type"`
        TargetID string `json:"target_id"`
    } `json:"dependencies"`
}
```

Status mapping:
- `open` → `ready`
- `in_progress` → `in_progress`
- `closed` → `done`
- `blocked` → `blocked`

## Testing Strategy

### Unit Tests
- `internal/store/store_test.go` - File operations, locking
- `internal/task/task_test.go` - Domain logic
- `internal/mcp/server_test.go` - MCP protocol

### Integration Tests
- `cmd/tk/main_test.go` - Full CLI workflows
- `internal/http/server_test.go` - HTTP endpoints

### Race Detection
All tests run with `-race` flag to catch concurrent access issues:

```bash
go test -race ./...
```

## Future Considerations

1. **Remote storage** - Currently file-based; could add Redis/SQLite backends
2. **Multi-repo** - Support for distributed task files
3. **Webhooks** - Notify external systems on task changes

## Recent Additions (V2.0/V3.0)

The following features have been implemented:

- ✅ **Time tracking** - `tk task timer start/stop/status`
- ✅ **Tags/labels** - `tk task tag add/remove`
- ✅ **Custom fields** - `tk task field set/remove`
- ✅ **Subtasks** - `tk task add --parent <id>` with tree view
- ✅ **Directory-based storage** - `.tasuku/` with per-file locking
- ✅ **Archival** - Move completed tasks to archive folder
- ✅ **Terminal UI** - `tk ui` for interactive task management
- ✅ **Auto-detection** - Seamlessly works with V2 or V3 format
