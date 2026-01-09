---
name: start
description: Start working on a task. Use when beginning work, picking up a new task, or resuming work.
---

# Start Task

Quick command to mark a task as in-progress.

**For a guided workflow with context loading, use `/tasuku:pickup` instead.**

## Usage

```bash
tk task start <task-id>              # Start working on task
tk task start <task-id> --timer      # Start with time tracking
tk task start <task-id> --unblock    # Clear blockers and start
```

## When to Use

- You know exactly which task to work on
- Task context is already loaded
- Quick resumption of previous work

## When to Use `/tasuku:pickup` Instead

Use the guided workflow `/tasuku:pickup` when:
- Starting a session and need to choose what to work on
- Want related learnings and decisions surfaced
- Need help prioritizing between ready tasks
- Want full task context before starting

## Flags

- `--timer` - Start time tracking automatically
- `--unblock` - Clear any blockers before starting (for blocked tasks)

## Best Practices

1. Only have one task in_progress at a time for focus
2. Use `--timer` if tracking time spent
3. Use `tk task pause <id>` if you need to switch tasks

## After Starting

- Record learnings: `/tasuku:learn "insight"`
- Add notes: `/tasuku:note <id> "progress update"`
- When done: `/tasuku:complete <id>`

## Related Skills

- `/tasuku:pickup` - Guided task selection and start (recommended)
- `/tasuku:ready` - See tasks ready to work on
- `/tasuku:show` - View task details
- `/tasuku:complete` - When you finish
