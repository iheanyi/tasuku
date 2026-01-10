---
description: Complete a task with guided learning capture. Use after finishing work to mark done, document learnings, and see what's unblocked.
---

# Complete Task Workflow

A guided workflow for properly completing a task. This skill combines marking done, prompting for learnings, and showing next steps.

## Usage

```
/tasuku:complete <task-id>
```

## Workflow Steps

When you invoke this skill, follow these steps IN ORDER:

### Step 1: Mark the Task Done

```bash
tk task done <task-id>
```

This automatically stops any running timer.

### Step 2: Capture Learnings (CRITICAL)

Before moving on, reflect on what you learned:

**Ask yourself:**
- What was the root cause of any bugs fixed?
- What patterns worked well (or poorly)?
- Any "never do X" or "always do Y" rules discovered?
- Any gotchas or undocumented behaviors found?

**Record each learning:**
```bash
tk learn "Never manually manipulate ANSI strings - use lipgloss functions"
tk learn "Always process BubbleTea Init() command in tests before assertions"
```

**This is not optional.** Knowledge captured now prevents future bugs.

### Step 3: Check Impact

Show what tasks are now unblocked:
```bash
tk task deps <task-id>
```

### Step 4: Suggest Next Steps

Show the user:
1. Any tasks that were unblocked
2. Whether to archive the completed task
3. The next ready task to pick up

```bash
# If task is simple/no longer needed:
tk task archive add <task-id>

# Show what's next:
tk task ready
```

## Example Flow

```
User: /tasuku:complete fix-auth-bug

Agent:
1. Marking task done...
   ✓ tk task done fix-auth-bug

2. Learning capture - what did you discover?
   - Root cause: Missing null check in redirect handler
   - Rule: "Always validate redirect URLs before navigation"

   Recording learnings:
   ✓ tk learn "Always validate redirect URLs before navigation"

3. Impact check:
   ✓ This unblocked: deploy-to-prod, write-auth-tests

4. Next steps:
   - Archive this task? (tk task archive add fix-auth-bug)
   - Next ready task: deploy-to-prod (high priority)
```

## Why Use This Skill

- **Ensures learnings are captured** - The most common failure is forgetting to document what was learned
- **Shows downstream impact** - See what tasks are now unblocked
- **Guides next action** - Know what to work on next
- **Creates completion ritual** - Proper closure on work

## Related Skills

- `/tasuku:learn` - Record individual learnings
- `/tasuku:ready` - See tasks ready to work on
- `/tasuku:pickup` - Start working on the next task
