# MCP Tools Reference

Once `tk mcp install` is run, AI agents have access to these tools.

## Overview

Tasuku exposes **17 MCP tools** organized into three tiers:
- **Core tools** (12): Frequently used operations kept as individual tools
- **Consolidated tools** (3): Less frequent operations grouped by action type
- **Utility tools** (2): Health checks and statistics

## Core Tools (Tier 1)

### Task Lifecycle

| Tool | Description |
|------|-------------|
| `tk_list` | List tasks with optional status/tag/owner filter. Use `status: "ready"` for ready tasks only. |
| `tk_add` | Create a new task |
| `tk_start` | Mark task as in_progress |
| `tk_done` | Mark task as complete |
| `tk_block` | Mark task as blocked by others |
| `tk_show` | Get detailed task info (notes, priority, timestamps, dependencies) |

### Knowledge Capture

| Tool | Description |
|------|-------------|
| `tk_context` | Get full context for agent consumption |
| `tk_find` | Search across tasks, notes, learnings, decisions |
| `tk_learn` | Add a learning to context |
| `tk_decide` | Record an architectural decision |
| `tk_note` | Add a note to a task |

### Help & Discovery

| Tool | Description |
|------|-------------|
| `tk_help` | Get help on tools and workflows. Topics: overview, tasks, metadata, knowledge, multiagent, archive, install |

## Consolidated Tools (Tier 2)

### tk_task

Handles task operations that modify task state. **Required parameter: `action`**

| Action | Description | Additional Parameters |
|--------|-------------|-----------------------|
| `edit` | Update task description | `id`, `description` |
| `delete` | Permanently delete a task | `id` |
| `pause` | Revert in_progress → ready | `id` |
| `unblock` | Remove blockers (all or specific) | `id`, `from` (optional) |
| `priority` | Set priority (critical/high/normal/low/backlog) | `id`, `priority` |
| `owner` | Set or clear task owner | `id`, `owner` (optional) |
| `archive` | Archive a done task | `id`, `summary` (optional) |
| `restore` | Restore archived task | `id` |
| `claim` | Claim task for exclusive agent work | `id`, `agent` |
| `release` | Release claimed task | `id` |
| `who` | Show tasks claimed by each owner/agent | (none) |

**Example:**
```json
{"name": "tk_task", "arguments": {"action": "archive", "id": "task-id", "summary": "Completed successfully"}}
```

### tk_metadata

Handles tags, custom fields, and note management. **Required parameter: `action`**

| Action | Description | Additional Parameters |
|--------|-------------|-----------------------|
| `tag_add` | Add tag to a task | `id`, `tag` |
| `tag_remove` | Remove tag from a task | `id`, `tag` |
| `field_set` | Set custom field on task | `id`, `key`, `value` |
| `field_remove` | Remove custom field | `id`, `key` |
| `note_list` | List notes for a task or all notes | `task_id` (optional) |
| `note_remove` | Remove a note | `task_id`, `note_id` |

**Example:**
```json
{"name": "tk_metadata", "arguments": {"action": "tag_add", "id": "task-id", "tag": "urgent"}}
```

### tk_manage

Handles learning/decision/archive management. **Required parameter: `action`**

| Action | Description | Additional Parameters |
|--------|-------------|-----------------------|
| `learning_list` | List all learnings | (none) |
| `learning_promote` | Promote learning to permanent docs | `id`, `keep` (optional), `to` (optional) |
| `learning_remove` | Remove a learning | `id` |
| `learning_rules` | List rule learnings (never/always patterns) | (none) |
| `decision_list` | List all decisions | (none) |
| `decision_remove` | Remove a decision | `id` |
| `archive_list` | List archived tasks | (none) |
| `archive_all` | Archive all done tasks older than duration | `older_than` (e.g., "7d", "24h") |

**Example:**
```json
{"name": "tk_manage", "arguments": {"action": "archive_all", "older_than": "7d"}}
```

## Utility Tools (Tier 3)

| Tool | Description |
|------|-------------|
| `tk_stats` | Show task statistics and progress |
| `tk_health` | Project health check with recommendations |

## CLI-Only Features

The following features are available via CLI only (not exposed via MCP):

- **Time tracking**: `tk timer start/stop/status`
- **Rules sync**: `tk rules sync`
- **CLAUDE.md management**: `tk claudemd lint/stats`
- **Installation**: `tk mcp install/uninstall`, `tk plugin install/uninstall`, `tk hooks install/uninstall`

## Running the Server

```bash
tk serve mcp               # Start MCP server via stdio (for AI tools)
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
