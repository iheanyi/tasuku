# Session Start/End Behaviors

AI tool hooks automatically integrate Tasuku into your workflow.

## Supported AI Tools

- **Claude Code**: Full hooks support (SessionStart, Stop, PreCompact, SubagentStop, etc.)
- **Cursor**: MCP server + rules sync (no hooks, uses Claude Code-style skills)
- **Codex**: MCP server + notify hook (agent-turn-complete)
- **OpenCode**: MCP server + plugin hooks (session.created, session.idle, todo.updated)

## At Session Start (SessionStart hook)

The `tk hooks session` command runs automatically and displays:
- Task counts by status (ready, in_progress, blocked, done)
- Number of learnings and decisions recorded
- Suggested next task based on priority

**What to do at session start:**
1. Review the context summary to orient yourself
2. Check if any in_progress tasks need attention
3. Use `tk_context` if you need full task details
4. Start a timer if picking up work: `tk timer start <task-id>`

## At Session End (Stop hook)

The `tk hooks stop-reminder` command runs when you exit and reminds about:
- **Running timers**: Any timers still active that should be stopped
- **In-progress tasks**: Tasks still marked in_progress that may need status updates

**What to do before ending a session:**
1. Stop any running timers: `tk timer stop <task-id>`
2. Update task status: either `tk task done <id>` or `tk task pause <id>`
3. Record any learnings discovered: `tk learn "insight"`
4. Note any blockers for future sessions: `tk note add <task-id> "blocker info"`

## Manual Session Commands

If hooks aren't installed, use these commands manually:
```bash
tk hooks session        # Show context summary
tk hooks stop-reminder  # Check for reminders before exiting
```

Install hooks with:
```bash
tk hooks install              # Install all hooks (git + AI tools)
tk hooks install --claude     # Claude Code hooks only
tk hooks install --codex      # Codex hooks only
tk hooks install --opencode   # OpenCode hooks only
tk hooks install --local      # Project-local instead of global
```

## MCP Server Installation

Configure MCP server for AI tools:
```bash
tk mcp install                # Auto-detect and install to all AI tools
tk mcp install --tool claude  # Claude Code only
tk mcp install --tool cursor  # Cursor only
tk mcp install --tool codex   # Codex only
tk mcp install --tool opencode # OpenCode only
tk mcp install --local        # Project-local config
```

**Configuration files by tool:**
- Claude Code: `~/.claude.json` or `./.claude.json` (local)
- Cursor: `~/.cursor/mcp.json` or `./.cursor/mcp.json` (local)
- Codex: `~/.codex/config.toml` (TOML format)
- OpenCode: `~/.config/opencode/opencode.json` or `./opencode.json` (local)

## Hook Features & Configuration

Tasuku hooks are configurable with `--quiet`, `--disable`, and `--list-features` flags.

### prompt-check Features

The `prompt-check` hook runs on `UserPromptSubmit` and analyzes user messages.

**Context Surfacing** (surfaces relevant info):
| Feature | Description |
|---------|-------------|
| `session_continuity` | Shows in-progress tasks when user says "continue"/"resume" |
| `decision_lookup` | Surfaces related decisions when asking questions |
| `learning_lookup` | Surfaces related learnings when asking questions |
| `task_reference` | Shows task context when task ID is mentioned |
| `task_surfacing` | Finds related tasks by keyword matching |

**Nudges** (prompts for action):
| Feature | Description |
|---------|-------------|
| `rule_detection` | Detects "Never X"/"Always Y" patterns, suggests `tk learn` |
| `bug_detection` | Prompts to track bug reports with `tk task add --tag bug` |
| `work_detection` | Suggests creating task for significant work requests |
| `stuck_detection` | Offers help when user seems stuck/frustrated |
| `shipping_check` | Pre-ship checklist on deploy/release mentions |
| `learning_capture` | Captures "TIL"/"I learned" as learnings |
| `decision_capture` | Prompts to record "X or Y" decision points |
| `scope_warning` | Warns about scope expansion mid-task |
| `architecture_explanation` | Detects "because we"/"why" explanations, prompts `tk_decide` |
| `preference_stated` | Captures user preferences ("I prefer", "always use") |

```bash
tk hooks prompt-check --list-features     # List all features
tk hooks prompt-check --quiet             # Context only, no nudges
tk hooks prompt-check --disable=shipping_check,scope_warning
```

### todo-check Features

The `todo-check` hook runs on `PostToolUse` for TodoWrite and Bash commands.

| Feature | Description |
|---------|-------------|
| `bugfix_learning` | Prompts for learnings after bug fix tasks complete |
| `project_task` | Suggests persisting project-level tasks to tk |
| `test_failure` | Detects test failures, suggests tracking |
| `test_fix_learning` | Prompts for learning when tests pass after failure |
| `git_commit` | Links git commits to related in-progress tasks |
| `investigation_pattern` | Prompts for learning after deep file investigation + edit |

```bash
tk hooks todo-check --list-features       # List all features
tk hooks todo-check --quiet               # Minimal output
tk hooks todo-check --disable=test_failure,git_commit
```

### Hook Version Tracking

Hooks include version tracking to alert when updates are available:
- Version is written to `.claude/.tasuku-hooks-version` on install
- SessionStart checks installed vs current version
- Shows update prompt: "⬆️ Hooks outdated: v0.6.0 → v0.6.1"
- Update with: `tk hooks install --force` (or `--force --local`)
