---
name: stats
description: Show project statistics and progress. Use when user wants metrics, completion status, or progress overview.
---

# Task Statistics

Display project statistics including task counts, completion rates, and progress.

## Usage

```bash
tk task stats                   # Show statistics
tk task stats --format json     # Output as JSON
```

## Metrics Displayed

- Total task count
- Tasks by status (ready, in_progress, blocked, done)
- Completion percentage
- Blocked task count

## When to Use

- Getting a quick project health check
- Reporting progress to stakeholders
- Understanding project velocity
- Identifying bottlenecks (high blocked count)

## Interpreting Results

- High blocked count: Dependencies need resolution
- Many in_progress: Consider focusing on completion
- Low ready count: May need to plan next tasks
