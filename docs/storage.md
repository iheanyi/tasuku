# Storage Format

Tasuku stores tasks as Markdown files with YAML frontmatter.

## Directory Structure

```
.tasuku/
├── tasks/
│   └── task-id.md        # Individual task files
├── archive/
│   └── old-task.md       # Archived completed tasks
├── context/
│   ├── learnings.md      # Recorded insights
│   └── decisions.md      # Architectural decisions
├── config.json           # Version marker {"version": 4}
└── index.json            # Auto-generated for fast queries
```

## Task File Format

`.tasuku/tasks/task-id.md`:

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

# Task title from first line

Description supports **rich Markdown** formatting,
including `code`, lists, and code blocks.

## Notes

### 2024-01-05 11:00 [abc123]
Note content with full Markdown support.
```

## Learnings Format

`.tasuku/context/learnings.md`:

```markdown
# Learnings

## learning-id - 2024-01-04T10:30:00Z
Things discovered while working.
```

## Decisions Format

`.tasuku/context/decisions.md`:

```markdown
# Decisions

## decision-id - 2024-01-04T10:30:00Z
**Chose**: Option A
**Over**: Option B, Option C
**Because**: Reasoning for the decision.
```

## Status Values

| Status | Description |
|--------|-------------|
| `ready` | Available to work on |
| `in_progress` | Currently being worked on |
| `blocked` | Waiting on other tasks |
| `done` | Completed |

## Priority Values

| Priority | Value | Description |
|----------|-------|-------------|
| critical | 0 | Blocking/urgent |
| high | 1 | Important |
| normal | 2 | Default |
| low | 3 | Can wait |
| backlog | 4 | Future ideas |

## File Locking

For parallel agent safety, Tasuku uses `flock` on individual task files:

```go
f, _ := os.OpenFile(".tasuku/tasks/my-task.md", os.O_RDWR, 0644)
syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
```

This allows multiple agents to work on different tasks simultaneously.
