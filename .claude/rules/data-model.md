# Data Model

## V4 Directory Structure (Default)

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

## V3 Directory Structure (Legacy JSON)

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

## V2 Format (Legacy)

Single `.tasuku.json` file with all data in one place. Supported for backwards compatibility.

## File Locking

For parallel agent safety, we use `flock`:

**V3/V4 (Directory)**: Lock individual task files for granular locking
```go
// Lock specific task file
f, _ := os.OpenFile(".tasuku/tasks/my-task.md", os.O_RDWR, 0644)
syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
```

**V2 (Single file)**: Lock the entire file
```go
f, _ := os.OpenFile(".tasuku.json", os.O_RDWR, 0644)
syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
```
