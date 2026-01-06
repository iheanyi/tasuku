---
name: list
description: List all tasks with optional status filtering. Use when user wants to see tasks, check project status, or get an overview.
---

# List Tasks

Display all tasks in the project, optionally filtered by status.

## Usage

Run the `tk task list` command to show all tasks:

```bash
tk task list                    # All tasks
tk task list --status ready     # Only ready tasks
tk task list --status in_progress  # Only in-progress tasks
tk task list --status blocked   # Only blocked tasks
tk task list --status done      # Only completed tasks
tk task list --tree             # Hierarchical view with subtasks
tk task list --format json      # Output as JSON
```

## Output Format

Tasks are displayed with:
- Status symbol: `[-]` ready, `[*]` in_progress, `[!]` blocked, `[x]` done
- Task ID
- Description
- Blockers (if any)

## When to Use

- Starting a session to see what needs to be done
- Checking overall project progress
- Finding tasks by status
- Reviewing blocked tasks
