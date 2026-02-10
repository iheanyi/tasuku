---
description: "Mark a task as completed"
argument-hint: "TASK_ID [TASK_ID...]"
---

# Complete Task

```!
tk task done $ARGUMENTS
```

**For guided completion with learning capture, use `/tasuku:complete` instead.**

## Usage

```bash
tk task done <task-id>           # Mark single task complete
tk task done <id1> <id2> <id3>   # Mark multiple tasks complete
```

## Automatic Behavior

- Running timers are automatically stopped
- Elapsed time is added to task's total duration
- Blocked tasks that depended on this one may become unblocked

## After Completing

Consider:
- Recording learnings: `tk learn "what you discovered"`
- Archiving if no longer needed: `tk task archive <id>`
- Starting the next ready task: `tk task ready`
