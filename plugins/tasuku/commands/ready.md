---
description: "Show tasks ready to work on, sorted by priority"
argument-hint: "[--format FORMAT]"
---

# Ready Tasks

```!
tk task ready $ARGUMENTS
```

Shows all tasks that are ready to be worked on, sorted by priority.

## Priority Order

Tasks are sorted by priority level:
- **Critical (0)**: Urgent, blocking issues
- **High (1)**: Important, do soon
- **Normal (2)**: Default priority
- **Low (3)**: Can wait
- **Backlog (4)**: Future work

## When to Use

- Looking for the next task to work on
- Starting a new work session
- Checking what's available after completing a task

## Next Steps

After viewing ready tasks, use `tk task start <id>` to begin work on one.
