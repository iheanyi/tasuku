# MCP Tools Reference

Once `tk mcp install` is run, AI agents have access to these tools.

## Task Operations

| Tool | Description |
|------|-------------|
| `tk_list` | List tasks with optional status filter |
| `tk_add` | Create a new task |
| `tk_show` | Get detailed task info (notes, priority, timestamps) |
| `tk_start` | Mark task as in_progress |
| `tk_done` | Mark task as complete |
| `tk_pause` | Revert in_progress → ready |
| `tk_edit` | Update task description |
| `tk_delete` | Permanently delete a task |
| `tk_priority` | Set priority (critical/high/normal/low/backlog) |
| `tk_ready` | List tasks ready to work on (sorted by priority) |
| `tk_deps` | Show task dependency tree |
| `tk_stats` | Show task statistics and progress |

## Blocking & Dependencies

| Tool | Description |
|------|-------------|
| `tk_block` | Mark task as blocked by others |
| `tk_unblock` | Remove blockers (all or specific) |

## Ownership & Coordination

| Tool | Description |
|------|-------------|
| `tk_owner` | Set or clear task owner |
| `tk_claim` | Claim task for exclusive agent work |
| `tk_release` | Release claimed task |
| `tk_who` | Show tasks claimed by each owner/agent |

## Knowledge Capture

| Tool | Description |
|------|-------------|
| `tk_context` | Get full context for agent consumption |
| `tk_find` | Search across tasks, notes, learnings, decisions |
| `tk_learn` | Add a learning to context |
| `tk_decide` | Record an architectural decision |
| `tk_note` | Add a note to a task |

## Tags & Custom Fields

| Tool | Description |
|------|-------------|
| `tk_tag_add` | Add tag to a task |
| `tk_tag_remove` | Remove tag from a task |
| `tk_field_set` | Set custom field on task |
| `tk_field_remove` | Remove custom field |

## Time Tracking

| Tool | Description |
|------|-------------|
| `tk_timer_start` | Start timer on task |
| `tk_timer_stop` | Stop timer, record elapsed time |
| `tk_timer_status` | Get status of running timers |

## Archiving

| Tool | Description |
|------|-------------|
| `tk_archive` | Archive a done task |
| `tk_archive_restore` | Restore archived task |
| `tk_archive_list` | List archived tasks |
| `tk_archive_all` | Archive all done tasks older than duration |

## Learning Management

| Tool | Description |
|------|-------------|
| `tk_learning_list` | List all learnings |
| `tk_learning_promote` | Promote learning to permanent docs |
| `tk_learning_remove` | Remove a learning |
| `tk_learning_rules` | List rule learnings (never/always patterns) |

## Decision Management

| Tool | Description |
|------|-------------|
| `tk_decision_list` | List all decisions |
| `tk_decision_remove` | Remove a decision |

## Note Management

| Tool | Description |
|------|-------------|
| `tk_note_list` | List notes for a task or all notes |
| `tk_note_remove` | Remove a note |

## Other Tools

| Tool | Description |
|------|-------------|
| `tk_suggest` | Analyze if task should persist to tk or stay session-only |
| `tk_health` | Project health check with recommendations |
| `tk_rules_sync` | Sync learnings/decisions to editor rules directories |

## Running the Server

```bash
tk serve mcp               # Start via stdio (for AI tools)
tk serve http --port 3000  # Start HTTP REST API
```

## TUI Keybindings

Launch the TUI with `tk ui`. Keybindings:

| Key | Action |
|-----|--------|
| `n` | New task |
| `e` | Edit task |
| `s` | Start task |
| `d` | Mark done |
| `P` | Pause task |
| `b` | Block task |
| `u` | Unblock task |
| `x` | Delete task |
| `t` | Toggle timer |
| `a` | Archive task |
| `A` | Archive all done |
| `enter` | View details |
| `/` | Filter tasks |
| `0-4` | Filter by status |
| `p` | Toggle priority sort |
| `N` | View notes |
| `L` | View learnings |
| `D` | View decisions |
| `r` | Refresh |
| `?` | Help |
| `q` | Quit |
