---
name: add
description: Create a new task. Use when user wants to add work items, create todos, or break down features into tasks.
---

# Add Task

Create a new task in Tasuku.

## Usage

```bash
tk task add "Task description"              # Create with auto-generated ID
tk task add "Task description" --id my-id   # Create with custom ID
tk task add "Subtask" --parent parent-id    # Create as subtask
tk task add "Urgent fix" --priority high    # Create with priority
```

## Options

- `--id`: Custom task ID (otherwise auto-generated from description)
- `--parent`: Parent task ID to create as subtask
- `--priority`: Priority level (critical, high, normal, low, backlog)

## Priority Levels

| Level | Name | When to Use |
|-------|------|-------------|
| 0 | critical | Blocking issues, urgent bugs |
| 1 | high | Important, do soon |
| 2 | normal | Default priority |
| 3 | low | Can wait |
| 4 | backlog | Future work, ideas |

## When to Use

- Breaking down a feature into subtasks
- Capturing new work items
- Creating follow-up tasks during implementation
- Adding bugs or issues discovered during work

## Best Practices

1. Use descriptive task names that explain the goal
2. Use `--parent` to organize related work as subtasks
3. Set priority based on urgency and importance
4. Consider using `tk task start <id>` immediately if starting work
