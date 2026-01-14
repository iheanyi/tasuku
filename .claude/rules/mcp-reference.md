# MCP Server Reference

## TUI/CLI/MCP/Plugin Parity Principle

**Every capability should be accessible through all interfaces.** This is critical for agent-first design:

| Interface | Users | Description |
|-----------|-------|-------------|
| **CLI** | Humans in terminal | `tk task list`, `tk learn "insight"` |
| **MCP** | AI agents (Claude Code, Cursor, Codex, OpenCode) | `tk_list`, `tk_learn` tools |
| **TUI** | Humans who prefer visual interfaces | Interactive terminal UI (`tk ui`) |
| **Plugin** | AI agents (slash commands) | `/tasuku:list`, `/tasuku:learn` |

**Parity Rules:**
1. **New CLI command → Add MCP tool** - Agents need the same capabilities as humans
2. **New MCP tool → Consider CLI equivalent** - Humans may want to use it manually
3. **Core operations → Add to TUI** - Visual interface for task management
4. **Frequent operations → Consider Plugin commands** - Slash commands for quick agent access
5. **Same behavior across all interfaces** - Identical semantics, different UX

## MCP Tools Reference

| Tool | CLI Equivalent | Description |
|------|----------------|-------------|
| **Core Task Operations** |||
| `tk_list` | `tk task list` | List tasks with optional status filter |
| `tk_add` | `tk task add` | Create a new task |
| `tk_show` | `tk task show` | Get detailed task info (notes, priority, timestamps) |
| `tk_start` | `tk task start` | Mark task as in_progress |
| `tk_done` | `tk task done` | Mark task as complete |
| `tk_pause` | `tk task pause` | Revert in_progress → ready, clear owner |
| `tk_edit` | `tk task edit` | Update task description |
| `tk_delete` | `tk task delete` | Permanently delete a task |
| `tk_priority` | `tk task priority` | Set priority (critical/high/normal/low/backlog) |
| `tk_ready` | `tk task ready` | List tasks ready to work on (sorted by priority) |
| `tk_deps` | `tk task deps` | Show task dependency tree |
| `tk_stats` | `tk task stats` | Show task statistics and progress |
| **Blocking & Dependencies** |||
| `tk_block` | `tk task block` | Mark task as blocked by others |
| `tk_unblock` | `tk task unblock` | Remove blockers (all or specific) |
| **Ownership & Coordination** |||
| `tk_owner` | `tk task owner` | Set or clear task owner |
| `tk_claim` | `tk task claim` | Claim task for exclusive agent work |
| `tk_release` | `tk task release` | Release claimed task |
| `tk_who` | `tk task who` | Show tasks claimed by each owner/agent |
| **Context & Search** |||
| `tk_context` | `tk context show` | Get full context for agent consumption |
| `tk_find` | `tk task find` | Search across tasks, notes, learnings, decisions |
| `tk_learn` | `tk learn` | Add a learning to context |
| `tk_decide` | `tk decide` | Record an architectural decision |
| `tk_note` | `tk note add` | Add a note to a task |
| **Tags & Custom Fields** |||
| `tk_tag_add` | `tk task tag add` | Add tag to a task |
| `tk_tag_remove` | `tk task tag remove` | Remove tag from a task |
| `tk_field_set` | `tk task field set` | Set custom field on task |
| `tk_field_remove` | `tk task field remove` | Remove custom field |
| **Time Tracking** |||
| `tk_timer_start` | `tk task timer start` | Start timer on task |
| `tk_timer_stop` | `tk task timer stop` | Stop timer, record elapsed time |
| `tk_timer_status` | `tk task timer status` | Get status of running timers |
| **Archiving** |||
| `tk_archive` | `tk task archive add` | Archive a done task |
| `tk_archive_restore` | `tk task archive restore` | Restore archived task |
| `tk_archive_list` | `tk task archive list` | List archived tasks |
| `tk_archive_all` | `tk task archive all` | Archive all done tasks older than duration |
| **Learning Management** |||
| `tk_learning_list` | `tk learning list` | List all learnings |
| `tk_learning_promote` | `tk learning promote` | Promote learning to permanent docs |
| `tk_learning_remove` | `tk learning remove` | Remove a learning |
| `tk_learning_rules` | `tk learning rules` | List rule learnings (never/always patterns) |
| **Decision Management** |||
| `tk_decision_list` | `tk decision list` | List all decisions |
| `tk_decision_remove` | `tk decision remove` | Remove a decision |
| **Note Management** |||
| `tk_note_list` | `tk note list` | List notes for a task or all notes |
| `tk_note_remove` | `tk note remove` | Remove a note |
| **Agent Workflow** |||
| `tk_suggest` | `tk suggest` | Analyze if a task should persist to tk or stay session-only |
| **Health & Diagnostics** |||
| `tk_health` | `tk health` | Project health check with recommendations |
| **Rules Sync** |||
| `tk_rules_sync` | `tk rules sync` | Sync learnings/decisions to editor rules directories |

## TUI Keybindings Reference

Launch the TUI with `tk ui`. The following keybindings are available:

| Key | Action | Notes |
|-----|--------|-------|
| **Task Operations** |||
| `n` | New task | Opens task creation dialog |
| `e` | Edit task | Edit selected task description |
| `s` | Start task | Mark ready task as in_progress |
| `d` | Mark done | Complete in_progress task |
| `P` | Pause task | Revert in_progress to ready |
| `b` | Block task | Mark task as blocked |
| `u` | Unblock task | Remove blockers and set to ready |
| `x` | Delete task | Delete with confirmation |
| `t` | Toggle timer | Start/stop time tracking |
| `a` | Archive task | Archive done task |
| `A` | Archive all done | Bulk archive with confirmation |
| `enter` | View details | Show full task information |
| **Navigation** |||
| `/` | Filter tasks | Text search through tasks |
| `0` | All tasks | Show all statuses |
| `1` | Ready only | Filter to ready tasks |
| `2` | In progress only | Filter to in_progress |
| `3` | Blocked only | Filter to blocked tasks |
| `4` | Done only | Filter to done tasks |
| `p` | Toggle priority sort | Switch between status/priority sort |
| `N` | View notes | Show notes for selected task |
| `L` | View learnings | Show project learnings |
| `D` | View decisions | Show architectural decisions |
| **General** |||
| `r` | Refresh | Reload data from storage |
| `?` | Help | Show keybinding help |
| `q` | Quit | Exit TUI |

## Running the Server

```bash
tk serve --port 3000
```

Or via stdio (for AI tools like Claude Code, Cursor, Codex, OpenCode):
```bash
tk serve --stdio
```
