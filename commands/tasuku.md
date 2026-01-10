---
description: "Task management for AI agents. Overview and quick reference for all /tasuku:* commands."
---

# Tasuku Task Management

Tasuku is an agent-first task management system. Manage tasks, track learnings, and coordinate work across sessions.

## Quick Start

```bash
tk task add "description"   # Create a task
tk task ready               # What can I work on?
tk task start <id>          # Begin work
tk task done <id>           # Complete task
tk learn "insight"          # Record learning
```

## Workflow Commands (Recommended)

| Command | When to Use |
|---------|-------------|
| `/tasuku:pickup` | Starting work - shows options, loads context, starts task |
| `/tasuku:complete` | Finishing work - marks done, captures learnings, shows next |
| `/tasuku:reflect` | After discoveries - guided learning extraction |
| `/tasuku:help` | See all available commands |

## Basic Commands

| Command | Purpose |
|---------|---------|
| `/tasuku:context` | Full project context at session start |
| `/tasuku:add` | Create a new task |
| `/tasuku:list` | List tasks with filtering |
| `/tasuku:ready` | Tasks ready to work on |
| `/tasuku:start` | Begin working on a task |
| `/tasuku:done` | Mark task complete |
| `/tasuku:learn` | Record learnings and insights |
| `/tasuku:decide` | Record architectural decisions |
| `/tasuku:note` | Add notes to tasks |
| `/tasuku:show` | View task details |
| `/tasuku:block` | Mark task as blocked |
| `/tasuku:stats` | Project statistics |
| `/tasuku:promote` | Promote learnings to docs |

## Task Lifecycle

1. **Start session:** `/tasuku:context` or `/tasuku:pickup`
2. **During work:** `/tasuku:note`, `/tasuku:learn` as you go
3. **Finish work:** `/tasuku:complete` (guided) or `/tasuku:done` (quick)
