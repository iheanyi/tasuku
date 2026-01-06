---
name: tasuku
description: Task management for AI agents. Use /tasuku for an overview or specific skills like /tasuku-list, /tasuku-add, /tasuku-start, /tasuku-done.
---

# Tasuku Task Management

Tasuku is an agent-first task management system. Manage tasks, track learnings, and coordinate agents.

## Quick Reference

```bash
tk task add "description" # Create a task
tk task list              # See all tasks
tk task ready             # What can I work on?
tk task start <id>        # Begin work (add --timer to track time)
tk task done <id>         # Complete task (auto-stops timer)
tk task pause <id>        # Pause work (auto-stops timer)
tk learn "insight"        # Record learning
tk context show           # Full context
tk task stats             # Project statistics
```

## Available Skills

Use specific skills for detailed guidance:

- **/tasuku-add** - Create a new task
- **/tasuku-list** - List all tasks with optional filtering
- **/tasuku-ready** - Show tasks ready to work on
- **/tasuku-start** - Start working on a task
- **/tasuku-done** - Mark a task complete
- **/tasuku-learn** - Record learnings and insights
- **/tasuku-context** - Get full project context
- **/tasuku-stats** - Show task statistics

## Task Lifecycle

1. `tk task add "description"` - Create task
2. `tk task start <id>` - Begin work
3. `tk learn "insight"` - Record learnings
4. `tk task done <id>` - Complete task

## Time Tracking

- `tk task start <id> --timer` - Start with timer
- `tk task timer status` - See running timers
- Timers auto-stop on `done` or `pause`
