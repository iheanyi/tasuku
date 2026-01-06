---
name: show
description: Show detailed information about a single task. Use when you need full task details, notes, or metadata.
---

# Show Task Details

Display complete information about a specific task.

## Usage

```bash
tk task show <task-id>              # Show task details
tk task show <task-id> --format json   # Output as JSON
```

## Information Displayed

- Task ID and description
- Status (ready, in_progress, blocked, done)
- Priority level
- Blockers (if any)
- Owner/assignee
- Parent task (if subtask)
- Tags and custom fields
- Notes attached to the task
- Time tracked
- Created and updated timestamps

## When to Use

- Before starting work to understand full context
- Checking notes left by previous work sessions
- Reviewing blockers and dependencies
- Inspecting task metadata

## Related Commands

- `tk task list` - See all tasks
- `tk task deps <id>` - Show dependency tree
- `tk note list <id>` - List just the notes
