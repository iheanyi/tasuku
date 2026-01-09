---
name: reflect
description: Guided reflection to extract learnings from recent work. Use after bug fixes, feature completion, or when prompted by hooks.
---

# Reflect and Extract Learnings

A guided process for extracting and recording learnings from your recent work. This skill walks you through reflection questions to ensure valuable knowledge is captured.

## Usage

```
/tasuku:reflect
```

## When to Use

- **After fixing a bug** - What was the root cause? What prevents it next time?
- **After completing a feature** - What patterns worked well?
- **After debugging session** - What did you discover about the codebase?
- **When hooks prompt you** - "Tests passing after failure - document your fix!"
- **Before ending a session** - Capture insights before context is lost

## Reflection Process

### Step 1: Identify What Happened

Look at recent activity:
- What tasks were completed?
- What bugs were fixed?
- What code was changed?

```bash
# Check recent git activity
git log --oneline -10

# Check completed tasks
tk task list --status done
```

### Step 2: Ask Reflection Questions

For each significant piece of work, answer:

**Root Cause Analysis (for bug fixes):**
- What was the actual bug?
- What was the root cause?
- Why wasn't this caught earlier?
- What rule would prevent this?

**Pattern Recognition:**
- Did you discover any codebase patterns?
- Any undocumented behaviors in libraries/APIs?
- Any "this is how things work here" insights?

**Decision Points:**
- Did you choose between approaches?
- What tradeoffs did you consider?
- Why did you choose what you chose?

### Step 3: Formulate Learnings

Good learnings are:
- **Specific** - Include enough context to be actionable
- **Rule-based** - "Never X" or "Always Y" when applicable
- **Scoped** - Can apply to specific paths if relevant

**Examples:**
```bash
# Root cause learning
tk learn "Never manually manipulate ANSI-styled strings - use lipgloss.Width/Place which handle escape sequences"

# Pattern learning
tk learn "BubbleTea async pattern: Init() returns Cmd, Update() processes result Msg, always restore selection after list refresh"

# Gotcha learning
tk learn "lipgloss.Truncate doesn't exist - use charmbracelet/x/ansi.Truncate for ANSI-aware string truncation"

# Scoped learning (applies to specific paths)
tk learn "Always force UTC timezone in golden tests" --scope "**/*_test.go"
```

### Step 4: Record Decisions

If you made architectural choices:

```bash
tk decide --id "rate-limit-algo" \
  --chose "token bucket" \
  --over "fixed window, sliding window" \
  --because "Better burst handling and distributed-system friendly"
```

### Step 5: Verify Capture

```bash
# List recent learnings
tk learning list

# Check for rule learnings that should be promoted
tk learning rules
```

## Guided Prompts

When you invoke this skill, I'll ask:

1. **What did you just finish?**
   - Bug fix? Feature? Refactor? Exploration?

2. **What broke / what was tricky?**
   - Error messages, unexpected behaviors, gotchas

3. **What was the fix / solution?**
   - The actual change that resolved it

4. **What rule prevents this in the future?**
   - "Never..." or "Always..." statement

5. **Any decisions made?**
   - Choices between approaches worth documenting

## Example Session

```
/tasuku:reflect

Agent: What did you just work on?

User: Fixed the TUI tests that were failing

Agent: What was the problem?

User: Tests were showing "Loading tasks..." instead of actual content

Agent: What was the root cause?

User: The tests weren't waiting for the async Init() command to complete.
BubbleTea's New() doesn't load data - Init() returns a Cmd that does.

Agent: What's the rule to prevent this?

User: Always process Init() command in tests before making assertions

Agent: Recording learnings...

✓ tk learn "In BubbleTea tests, always process Init() command before assertions - New() starts empty, Init() returns the load Cmd"

Any architectural decisions made?

User: No decisions, just a bug fix.

Agent: Done! Learning recorded. Run `tk learning rules` to see if it should be promoted.
```

## Why Use This Skill

- **Structured extraction** - Guided questions ensure thorough reflection
- **Prevents knowledge loss** - Capture insights while they're fresh
- **Builds project memory** - Learnings persist across sessions
- **Improves future work** - Rules prevent the same mistakes

## Related Skills

- `/tasuku:learn` - Quick learning capture (no guided process)
- `/tasuku:decide` - Record architectural decisions
- `/tasuku:complete` - Task completion workflow (includes reflection)
