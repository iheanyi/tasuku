---
name: tasuku
description: Task management for AI agents. Use /tasuku for an overview or specific skills like /tasuku:list, /tasuku:add, /tasuku:start, /tasuku:done.
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

## Workflow Skills (Recommended)

Use these for guided workflows:

| Skill | When to Use |
|-------|-------------|
| `/tasuku:pickup` | Starting work - shows options, loads context, starts task |
| `/tasuku:complete` | Finishing work - marks done, captures learnings, shows next |
| `/tasuku:reflect` | After discoveries - guided learning extraction |
| `/tasuku:help` | See all available skills |

## Basic Skills

| Skill | Purpose |
|-------|---------|
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

## Key Principles

1. **Capture learnings immediately** - Don't wait until session end
2. **Use "Never/Always" for rules** - They get special treatment
3. **Document decisions when made** - Future you will thank you
4. **Complete tasks properly** - Use `/tasuku:complete` for the full workflow

## Time Tracking

```bash
tk task start <id> --timer    # Start with timer
tk task timer status          # Check running timers
tk task done <id>             # Auto-stops timer
```
