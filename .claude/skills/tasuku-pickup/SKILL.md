---
name: pickup
description: Pick up the next task to work on. Shows ready tasks, helps select one, and starts it with full context.
---

# Pick Up Next Task

A guided workflow for selecting and starting your next piece of work. Shows what's available, provides context, and gets you started efficiently.

## Usage

```
/tasuku-pickup [task-id]
```

If no task-id provided, shows ready tasks and helps you choose.

## Workflow Steps

### Step 1: Show What's Ready

```bash
tk task ready
```

This shows tasks sorted by priority (critical → high → normal → low → backlog).

### Step 2: Select a Task

If task-id was provided, use that. Otherwise, recommend based on:
- Priority level
- Whether it unblocks other tasks
- Time since creation

### Step 3: Show Task Context

Before starting, surface relevant context:

```bash
# Show full task details
tk task show <task-id>

# Surface related learnings
tk find "<keywords from task>"
```

### Step 4: Start the Task

```bash
tk task start <task-id>
```

Optionally with timer:
```bash
tk task start <task-id> --timer
```

### Step 5: Surface Related Knowledge

Show any learnings or decisions related to this work:
- Search learnings for relevant keywords
- Show related decisions if any exist

## Example Flow

```
User: /tasuku-pickup

Agent:
1. Ready tasks (by priority):

   CRITICAL:
   - fix-auth-regression: "Fix authentication bypass in admin panel"

   HIGH:
   - implement-rate-limiting: "Add rate limiting to API endpoints"
   - update-docs: "Update API documentation for v2"

   NORMAL:
   - refactor-logger: "Clean up logging utilities"

2. Recommendation: fix-auth-regression (critical priority, security issue)

3. Task details:
   - Description: Fix authentication bypass in admin panel
   - Tags: security, bug
   - Created: 2 hours ago

4. Related learnings:
   - "Always validate session tokens on every admin request"
   - "Never trust client-side auth state"

5. Starting task...
   ✓ tk task start fix-auth-regression

   Ready to work on: fix-auth-regression
```

## With Specific Task

```
User: /tasuku-pickup implement-rate-limiting

Agent:
1. Task details:
   - implement-rate-limiting: "Add rate limiting to API endpoints"
   - Priority: high
   - Tags: api, performance

2. Related decisions:
   - rate-limit-strategy: Chose "token bucket" over "fixed window"

3. Related learnings:
   - "Use Redis for distributed rate limit counters"

4. Starting task...
   ✓ tk task start implement-rate-limiting
```

## Why Use This Skill

- **Informed selection** - See all options with priorities
- **Context loading** - Relevant learnings and decisions surfaced automatically
- **Quick start** - Goes from "what should I do?" to working in one command
- **Knowledge continuity** - Previous learnings about this area are shown

## Related Skills

- `/tasuku-ready` - Just list ready tasks (no workflow)
- `/tasuku-start` - Just start a task (no context loading)
- `/tasuku-show` - Show task details
- `/tasuku-complete` - When you finish the task
