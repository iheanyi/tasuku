# Hooks Configuration

Hooks automatically integrate Tasuku into your AI tool workflow.

## Supported AI Tools

| Tool | Support |
|------|---------|
| Claude Code | Full hooks (SessionStart, Stop, PreCompact, etc.) |
| Cursor | MCP server + rules sync |
| Codex | MCP server + notify hook |
| OpenCode | MCP server + plugin hooks |

## Installation

```bash
tk hooks install              # Install all hooks
tk hooks install --claude     # Claude Code only
tk hooks install --codex      # Codex only
tk hooks install --opencode   # OpenCode only
tk hooks install --local      # Project-local instead of global
tk hooks install --force      # Update to latest version
tk hooks uninstall            # Remove all hooks
```

## MCP Server Installation

```bash
tk mcp install                # Auto-detect and install to all AI tools
tk mcp install --tool claude  # Claude Code only
tk mcp install --tool cursor  # Cursor only
tk mcp install --local        # Project-local config
```

**Configuration file locations:**
- Claude Code: `~/.claude.json` or `./.claude.json`
- Cursor: `~/.cursor/mcp.json` or `./.cursor/mcp.json`
- Codex: `~/.codex/config.toml`
- OpenCode: `~/.config/opencode/opencode.json` or `./opencode.json`

## What Hooks Do

| Hook | Trigger | Action |
|------|---------|--------|
| SessionStart | Session begins | Shows project context and suggested task |
| Stop | Session ends | Reminds about timers and in-progress tasks |
| PreCompact | Before context compaction | Prompts to capture learnings |
| UserPromptSubmit | User sends message | Surfaces related context |

## Hook Features

### prompt-check Features

Configurable with `--quiet` and `--disable` flags.

**Context Surfacing:**
| Feature | Description |
|---------|-------------|
| `session_continuity` | Shows in-progress tasks on "continue"/"resume" |
| `decision_lookup` | Surfaces related decisions |
| `learning_lookup` | Surfaces related learnings |
| `task_reference` | Shows task context when ID mentioned |
| `task_surfacing` | Finds related tasks by keyword |

**Nudges:**
| Feature | Description |
|---------|-------------|
| `rule_detection` | Suggests `tk learn` for "Never X"/"Always Y" |
| `bug_detection` | Prompts task creation for bug reports |
| `work_detection` | Suggests task for significant work |
| `learning_capture` | Captures "TIL"/"I learned" |
| `decision_capture` | Prompts for "X or Y" decisions |

```bash
tk hooks prompt-check --list-features
tk hooks prompt-check --quiet              # Context only, no nudges
tk hooks prompt-check --disable=shipping_check,scope_warning
```

### todo-check Features

| Feature | Description |
|---------|-------------|
| `bugfix_learning` | Prompts for learnings after bug fixes |
| `project_task` | Suggests persisting project-level tasks |
| `test_failure` | Detects test failures |
| `git_commit` | Links commits to in-progress tasks |

```bash
tk hooks todo-check --list-features
tk hooks todo-check --quiet
tk hooks todo-check --disable=test_failure
```

## Version Tracking

Hooks include version tracking:
- Version written to `.claude/.tasuku-hooks-version` on install
- SessionStart checks for updates
- Update with `tk hooks install --force`
