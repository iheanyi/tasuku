---
description: Mark a task as blocked by other tasks. Use when work cannot proceed until dependencies are resolved.
---

# Block Task

Mark a task as blocked, indicating it cannot proceed until dependencies are resolved.

## Usage

```bash
tk task block <task-id> --by <blocker-id>           # Block by one task
tk task block <task-id> --by task1 --by task2       # Block by multiple tasks
tk task unblock <task-id>                           # Remove all blockers
```

## When to Use

- Task requires another task to be completed first
- Waiting on external dependencies
- Work discovered to have prerequisites during implementation
- Parallel tasks that converge

## Blocked Task Behavior

- Blocked tasks won't appear in `tk task ready`
- Shows blockers in `tk task list` output
- Can view with `tk task list --status blocked`
- Auto-unblocks when all blocking tasks are done

## Best Practices

1. Be specific about which task is blocking
2. Don't block on vague dependencies - create tasks for them
3. Check blocked tasks when completing work
4. Use `tk task deps <id>` to visualize dependency chains

## Unblocking

```bash
# When blocker is done, blocked task becomes ready automatically
tk task done <blocker-id>

# Or manually unblock
tk task unblock <blocked-id>
```
