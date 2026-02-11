---
name: tasuku
description: Agent-first task management. Use when tracking multi-session work, recording learnings/decisions, managing task dependencies, or persisting context across sessions. Provides tk_list, tk_add, tk_done, tk_start, tk_learn, tk_decide, tk_note, tk_show, tk_find, tk_ready, tk_context, tk_stats, tk_block, tk_task, tk_metadata, tk_manage, tk_health, tk_suggest, tk_deps, and tk_help tools.
---

# Tasuku — Agent-First Task Management

Tasuku provides persistent, git-friendly task management designed for AI agents working on codebases.

## When to Use Tasuku

- **Multi-session work**: Tasks, learnings, and decisions persist across sessions
- **Task dependencies**: Block/unblock tasks, track what's ready to work on
- **Knowledge capture**: Record learnings ("Never X", "Always Y") and architectural decisions
- **Project context**: Get a summary of project state at any time

## Quick Start

### Check project state
Use `tk_context` to get a full overview of tasks, learnings, and decisions.

### Pick up work
Use `tk_ready` to see tasks ready to work on, then `tk_start` to begin.

### Track progress
- `tk_add` — Create new tasks
- `tk_note` — Add progress notes to tasks
- `tk_done` — Mark tasks complete

### Capture knowledge
- `tk_learn` — Record insights, gotchas, rules ("Never manually edit ANSI strings")
- `tk_decide` — Record architectural decisions with alternatives and reasoning

## Task vs Session Scope

Use Tasuku for **project-level** work that spans sessions:
- Features, bugs, milestones
- Architectural decisions
- Cross-cutting learnings

Use your editor's built-in task list for **session-level** steps:
- Implementation steps within a single session
- Temporary checklists

## Workflow Commands

### Starting a Session
1. `tk_context` — See project overview
2. `tk_ready` — Find next task to work on
3. `tk_start` — Begin working on a task

### During Work
- `tk_note` — Record progress, blockers, or findings
- `tk_learn` — Capture insights as you discover them
- `tk_block` — Mark tasks blocked by other tasks

### Completing Work
1. `tk_done` — Mark task complete
2. `tk_learn` — Capture what you learned (don't skip this!)
3. `tk_ready` — See what's unblocked and ready next

### Finding Information
- `tk_find` — Search across tasks, learnings, and decisions
- `tk_show` — Get full details on a specific task
- `tk_deps` — See task dependency graph
- `tk_stats` — Project health metrics
