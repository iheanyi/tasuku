---
description: "Show detailed information about a task"
argument-hint: "TASK_ID [--format FORMAT]"
---

# Show Task Details

```!
tk task show $ARGUMENTS
```

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
