---
description: Show all available Tasuku skills organized by workflow. Use to discover what skills exist and when to use them.
---

# Tasuku Skills Reference

Complete guide to all Tasuku skills, organized by workflow stage.

## Quick Reference

| When... | Use... |
|---------|--------|
| Starting a session | `/tasuku:context` |
| Need to pick up work | `/tasuku:pickup` or `/tasuku:ready` |
| Creating a task | `/tasuku:add` |
| Starting work | `/tasuku:start` |
| Adding notes | `/tasuku:note` |
| Task is blocked | `/tasuku:block` |
| Finishing work | `/tasuku:complete` or `/tasuku:done` |
| Recording insights | `/tasuku:learn` or `/tasuku:reflect` |
| Making decisions | `/tasuku:decide` |
| Checking progress | `/tasuku:stats` |

## Skills by Workflow Stage

### Starting a Session

| Skill | Purpose |
|-------|---------|
| `/tasuku:context` | Get full project context (tasks, learnings, decisions) |
| `/tasuku:ready` | List tasks ready to work on, sorted by priority |
| `/tasuku:pickup` | **Guided workflow** - select and start a task with full context |
| `/tasuku:stats` | See project statistics and progress |

### Creating & Planning Work

| Skill | Purpose |
|-------|---------|
| `/tasuku:add` | Create a new task |
| `/tasuku:block` | Mark a task as blocked by others |
| `/tasuku:list` | List all tasks with optional filtering |
| `/tasuku:show` | Show detailed task information |

### During Work

| Skill | Purpose |
|-------|---------|
| `/tasuku:start` | Mark a task as in-progress |
| `/tasuku:note` | Add notes to track progress or context |
| `/tasuku:learn` | Record learnings as you discover them |
| `/tasuku:decide` | Document architectural decisions |

### Completing Work

| Skill | Purpose |
|-------|---------|
| `/tasuku:done` | Mark a task as complete |
| `/tasuku:complete` | **Guided workflow** - done + learning capture + next steps |
| `/tasuku:reflect` | **Guided workflow** - extract learnings from recent work |

### Knowledge Management

| Skill | Purpose |
|-------|---------|
| `/tasuku:learn` | Record a learning or insight |
| `/tasuku:decide` | Record an architectural decision |
| `/tasuku:promote` | Promote learnings to permanent documentation |
| `/tasuku:reflect` | Guided learning extraction process |

## Workflow Skills (Recommended)

These skills guide you through complete workflows:

### `/tasuku:pickup`
**Use when:** Starting work, need to choose what to do next

Does: Shows ready tasks → helps select → loads context → starts task

### `/tasuku:complete`
**Use when:** Finishing a task

Does: Marks done → prompts for learnings → shows what's unblocked → suggests next task

### `/tasuku:reflect`
**Use when:** After bug fixes, features, or debugging sessions

Does: Guides you through reflection questions → extracts learnings → records them

## Basic Skills

For quick operations without the guided workflow:

| Skill | Command | Notes |
|-------|---------|-------|
| `/tasuku:add "desc"` | `tk task add "desc"` | Add `--priority high` for important tasks |
| `/tasuku:start <id>` | `tk task start <id>` | Add `--timer` to track time |
| `/tasuku:done <id>` | `tk task done <id>` | Auto-stops timer |
| `/tasuku:learn "..."` | `tk learn "..."` | Use "Never/Always" for rules |
| `/tasuku:note <id> "..."` | `tk note add <id> "..."` | Track progress |

## Pro Tips

1. **Start sessions with context:**
   ```
   /tasuku:context
   ```

2. **Use workflow skills for important transitions:**
   - `/tasuku:pickup` when starting work
   - `/tasuku:complete` when finishing
   - `/tasuku:reflect` after discoveries

3. **Capture learnings immediately:**
   Don't wait until the end of a session. Use `/tasuku:learn` as soon as you discover something.

4. **Use "Never/Always" format for rules:**
   ```
   /tasuku:learn "Never manually truncate ANSI strings"
   /tasuku:learn "Always validate URLs before redirect"
   ```

5. **Document decisions when you make them:**
   ```
   /tasuku:decide
   ```

## Related Commands

For operations not covered by skills:

```bash
tk task ready              # Tasks ready to work on
tk task list --tree        # Hierarchical view
tk task find "query"       # Search everything
tk task timer status       # Check running timers
tk task archive <id>       # Archive completed task
tk learning rules          # List rule-type learnings
tk rules sync              # Sync to editor rules
```

## Getting Help

- `/tasuku` - Overview and quick reference
- `/tasuku:help` - This detailed guide
- `tk --help` - CLI help
- `tk task --help` - Task subcommand help
