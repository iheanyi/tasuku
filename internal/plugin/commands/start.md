---
description: "Start working on a task"
argument-hint: "TASK_ID [--timer] [--unblock]"
---

# Start Task

```!
tk task start $ARGUMENTS
```

**For guided workflow with context loading, use `/tasuku:pickup` instead.**

## Usage

```bash
tk task start <task-id>              # Start working on task
tk task start <task-id> --timer      # Start with time tracking
tk task start <task-id> --unblock    # Clear blockers and start
```

## Flags

- `--timer` - Start time tracking automatically
- `--unblock` - Clear any blockers before starting (for blocked tasks)

## Best Practices

1. Only have one task in_progress at a time for focus
2. Use `--timer` if tracking time spent
3. Use `tk task pause <id>` if you need to switch tasks

## After Starting

- Record learnings: `tk learn "insight"`
- Add notes: `tk note add <id> "progress update"`
- When done: `tk task done <id>`
