# CLI Reference

The main CLI command is `tk`. Here's the complete reference:

## Initialization

```bash
tk init                    # Create .tasuku/ directory (V4 Markdown)
```

## Task Management (noun-verb style)

```bash
tk task list               # List all tasks (aliases: tk t ls, tk tasks)
tk task list --tree        # Show hierarchical subtask view
tk task list --status ready  # Filter by status
tk task list --tag backend   # Filter by tag
tk task add "description"  # Add a task
tk task add "desc" --parent <id>  # Add subtask
tk task add "desc" --id my-id     # Custom task ID
tk task add "desc" --priority high --tag feature  # With priority/tags
tk task start <id>         # Mark task in_progress
tk task start <id> --unblock      # Clear blockers and start
tk task done <id>          # Mark task complete
tk task done <id1> <id2>   # Complete multiple tasks
tk task pause <id>         # Revert in_progress → ready
tk task block <id> --by <other>   # Mark blocked
tk task unblock <id>       # Remove all blockers
tk task unblock <id> --from <blocker>  # Remove specific blocker
tk task show <id>          # Show task details
tk task edit <id> "new desc"      # Update description
tk task delete <id>        # Delete a task
tk task priority <id> high # Set priority (critical/high/normal/low/backlog)
tk task ready              # List tasks ready to work on (sorted by priority)
tk task find "query"       # Search across tasks, notes, learnings, decisions
tk task deps <id>          # Show task dependency tree
tk task stats              # Show task statistics and progress
```

## Ownership & Claims (multi-agent coordination)

```bash
tk task owner <id> "name"  # Set task owner
tk task owner <id>         # Clear owner
tk task claim <id> agent-1 # Claim for exclusive work
tk task release <id>       # Release claimed task
tk task who                # Show who has claimed what
```

## Tags & Custom Fields

```bash
tk task tag add <id> bug   # Add tag
tk task tag remove <id> bug  # Remove tag
tk task field set <id> estimate 2h   # Set custom field
tk task field remove <id> estimate   # Remove field
```

## Time Tracking

```bash
tk task timer start <id>   # Start timer
tk task timer stop <id>    # Stop timer, record elapsed
tk task timer status       # Show running timers
```

## Archiving

```bash
tk task archive add <id>   # Archive a done task
tk task archive list       # List archived tasks
tk task archive restore <id>  # Restore archived task
tk task archive all --older-than 7d  # Bulk archive old done tasks
```

## Context & Learnings

```bash
tk learn "insight"         # Add a learning (shortcut)
tk learning list           # List all learnings
tk learning promote <id>   # Promote to permanent docs
tk learning remove <id>    # Remove a learning
tk learning rules          # List never/always rule learnings
tk decide --id auth --chose JWT --over "sessions,OAuth" --because "reason"
tk decision list           # List all decisions
tk decision remove <id>    # Remove a decision
tk note add <task-id> "note"  # Add note to task
tk note list               # List all notes
tk note list --task <id>   # List notes for task
tk note remove <task-id> <note-id>  # Remove a note
tk context show            # Dump full context (for agent consumption)
tk suggest "description"   # Check if task should persist to tk
```

## Rules Sync (for Claude Code, Cursor)

```bash
tk rules sync              # Sync learnings/decisions to editor rules
tk rules sync --tool claude  # Sync to specific tool only
tk rules status            # Show sync status
tk rules clean             # Remove Tasuku-generated rules
```

## Server

```bash
tk serve mcp               # Start MCP server (for AI tools)
tk serve http              # Start HTTP REST API on :3000
tk serve http --port 8080  # Start HTTP on custom port
```

## MCP Configuration

```bash
tk mcp install             # Auto-detect and install to all AI tools
tk mcp install --tool claude  # Install to Claude Code only
tk mcp install --tool cursor  # Install to Cursor only
tk mcp install --local     # Project-local config
tk mcp uninstall           # Remove MCP configuration
```

## Hooks

```bash
tk hooks install              # Install all hooks (git + AI tools)
tk hooks install --claude     # Install Claude Code hooks only
tk hooks install --codex      # Install Codex hooks only
tk hooks install --opencode   # Install OpenCode hooks only
tk hooks install --local      # Install to project instead of global
tk hooks install --force      # Reinstall/update hooks
tk hooks uninstall            # Remove all Tasuku hooks
tk hooks session              # Display context summary
tk hooks stop-reminder        # Check for running timers/in-progress
tk hooks plan-sync plan.md    # Extract tasks from plan file
tk hooks prompt-check         # Detect task intent in prompts
tk hooks todo-check           # Check if TodoWrite items should persist
tk hooks pre-compact          # Capture insights before compaction
tk hooks subagent-done        # Capture insights from subagent
```

## UI & Diagnostics

```bash
tk ui                      # Launch terminal user interface
tk health                  # Project health check with recommendations
tk doctor                  # Diagnose MCP and CLI setup
tk validate                # Validate storage for correctness
```

## Migration

```bash
tk migrate v3              # Migrate from .tasuku.json to .tasuku/ (JSON)
tk migrate v4              # Migrate from V3 JSON to V4 Markdown
tk migrate beads           # Migrate from Beads format
```
