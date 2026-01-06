---
name: start
description: Start working on a task. Use when beginning work, picking up a new task, or resuming work.
---

# Start Task

Begin working on a task by marking it as in_progress.

## Usage

```bash
tk task start <task-id>                  # Start a task
tk task start <task-id> --timer          # Start task and begin timing
tk task start <task-id> --unblock        # Clear blockers and start
tk task start <task-id> --timer --unblock  # All options combined
```

## Flags

- `--timer`: Also start a time tracking timer on the task
- `--unblock`: Clear any blockers before starting (for blocked tasks)

## When to Use

- Picking up a new task from the ready list
- Resuming work on a task
- Claiming a task to indicate active work

## Best Practices

1. Only have one task in_progress at a time for focus
2. Use `--timer` if tracking time spent
3. Use `tk task pause <id>` if you need to switch tasks

## After Starting

- Record learnings with `tk learn "insight"`
- Add notes with `tk note add <id> "note"`
- When done, use `tk task done <id>`
