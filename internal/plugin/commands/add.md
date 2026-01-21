---
description: "Create a new task"
argument-hint: "DESCRIPTION [--id ID] [--parent ID] [--priority LEVEL] [--tag TAG]"
---

# Add Task

```!
tk task add $ARGUMENTS
```

If no arguments provided, show usage:

## Usage

```bash
tk task add "Task description"              # Create with auto-generated ID
tk task add "Task description" --id my-id   # Create with custom ID
tk task add "Subtask" --parent parent-id    # Create as subtask
tk task add "Urgent fix" --priority high    # Create with priority
tk task add "Bug" --tag bug                 # Create with tag
```

## Priority Levels

| Level | Name | When to Use |
|-------|------|-------------|
| 0 | critical | Blocking issues, urgent bugs |
| 1 | high | Important, do soon |
| 2 | normal | Default priority |
| 3 | low | Can wait |
| 4 | backlog | Future work, ideas |
