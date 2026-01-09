---
name: done
description: Mark a task as completed. Use when finishing work, completing a feature, or closing out a task.
---

# Mark Task Done

Quick command to mark a task as complete.

**For a guided completion workflow with learning capture, use `/tasuku:complete` instead.**

## Usage

```bash
tk task done <task-id>          # Mark task as complete
tk task done <id1> <id2>        # Complete multiple tasks
```

## Automatic Behavior

- If a timer is running on the task, it will be automatically stopped
- The elapsed time is added to the task's total duration

## When to Use

- Quick completion when learnings are already documented
- Batch completing multiple small tasks
- Simple tasks that don't warrant reflection

## When to Use `/tasuku:complete` Instead

Use the guided workflow `/tasuku:complete` when:
- You just fixed a bug (learning capture is critical)
- You completed significant work
- You want to see what tasks are unblocked
- You want suggestions for what to do next

## After Completing

Always consider:
1. **Record learnings:** `/tasuku:learn "what you discovered"`
2. **Check impact:** What tasks are now unblocked?
3. **Archive if done:** `tk task archive add <id>`
4. **Pick up next:** `/tasuku:pickup`

## Related Skills

- `/tasuku:complete` - Guided completion workflow (recommended)
- `/tasuku:learn` - Record learnings
- `/tasuku:pickup` - Start next task
- `/tasuku:ready` - See what's available
