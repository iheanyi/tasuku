---
description: "List all tasks with optional filtering"
argument-hint: "[--status STATUS] [--tag TAG] [--tree] [--format FORMAT]"
---

# List Tasks

```!
tk task list $ARGUMENTS
```

## Usage

```bash
tk task list                    # All tasks
tk task list --status ready     # Only ready tasks
tk task list --status in_progress  # Only in-progress
tk task list --status blocked   # Only blocked tasks
tk task list --status done      # Only completed tasks
tk task list --tag bug          # Filter by tag
tk task list --tree             # Hierarchical view with subtasks
tk task list --format json      # Output as JSON
```

## Output Format

Tasks are displayed with:
- Status symbol: `[-]` ready, `[*]` in_progress, `[!]` blocked, `[x]` done
- Task ID
- Description
- Blockers (if any)
