---
name: note
description: Add a note to a task for context, progress, or insights. Use proactively when starting tasks, making progress, or discovering context.
---

# Add Task Note

Attach notes to tasks to capture context, progress, and insights.

## Usage

```bash
tk note add <task-id> "Note text"   # Add a note to a task
tk note list <task-id>              # List notes for a task
tk note remove <task-id> <index>    # Remove a specific note
```

## When to Add Notes

**PROACTIVELY** add notes when:

1. **Starting a task**: Note your planned approach
   ```bash
   tk note add auth-feature "Planning to use JWT with refresh tokens, 1hr expiry"
   ```

2. **Making progress**: Note milestones or partial work
   ```bash
   tk note add auth-feature "Login endpoint complete, working on refresh flow"
   ```

3. **Encountering issues**: Note blockers or failed approaches
   ```bash
   tk note add auth-feature "OAuth library incompatible with Node 20, trying alternative"
   ```

4. **Discovering context**: Note findings for future agents
   ```bash
   tk note add auth-feature "Found existing token validation in utils/auth.ts"
   ```

## Best Practices

1. Add notes at the start of each work session
2. Note any unexpected findings or gotchas
3. Document failed approaches so they aren't repeated
4. Include file paths and line numbers when relevant
5. Notes persist across sessions - use them for continuity
