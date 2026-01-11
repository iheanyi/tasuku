---
description: "Get full project context including tasks, learnings, and decisions"
argument-hint: "[--format FORMAT]"
---

# Project Context

```!
tk context show $ARGUMENTS
```

Load the complete project context including all tasks, learnings, and decisions.

## Usage

```bash
tk context show                 # Full context as JSON
tk context show --format yaml   # Full context as YAML
```

## What's Included

The context contains:

### Tasks
- All active tasks with status, priority, and blockers
- Subtask relationships
- Owner and claim information

### Learnings
- Insights discovered while working
- "Never do X" and "Always use Y" patterns
- Codebase-specific knowledge

### Decisions
- Architectural choices made
- Alternatives that were considered
- Reasoning behind each decision

## When to Use

- Starting a new session to understand project state
- Onboarding to an unfamiliar project
- Before making decisions that might duplicate past work
- Understanding why things were built a certain way
