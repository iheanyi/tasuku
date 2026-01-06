---
name: done
description: Mark a task as completed. Use when finishing work, completing a feature, or closing out a task.
---

# Complete Task

Mark a task as done when work is finished.

## Usage

```bash
tk task done <task-id>          # Mark task as complete
```

## Automatic Behavior

- If a timer is running on the task, it will be automatically stopped
- The elapsed time is added to the task's total duration

## When to Use

- Finishing implementation of a feature
- Completing a bug fix
- Closing out any piece of work

## Best Practices

1. Verify the work is actually complete before marking done
2. Record any learnings discovered during the work
3. Check if completing this task unblocks others

## After Completing

Consider:
- Recording learnings: `tk learn "what you discovered"`
- Archiving if no longer needed: `tk task archive add <id>`
- Starting the next ready task: `tk task ready`
