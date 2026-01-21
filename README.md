# Tasuku

Agent-first task management for codebases. Designed for AI agents working alongside humans.

![Tasuku TUI Demo](assets/demo.gif)

## Why Tasuku?

- **Pull over push**: Agents query when needed, no constant context injections
- **Parallel-safe**: Per-file locking for multiple agents working simultaneously
- **Minimal context**: Only load what's needed for the current task
- **Human-readable**: Markdown files with YAML frontmatter, editable by hand
- **Git-friendly**: One file per task means clean diffs and fewer merge conflicts

## Installation

```bash
go install github.com/iheanyi/tasuku/cmd/tk@latest
```

Or build from source:
```bash
git clone https://github.com/iheanyi/tasuku.git
cd tasuku && go build -o tk ./cmd/tk
sudo mv tk /usr/local/bin/
```

## Quick Start

### 1. Initialize and integrate

```bash
cd your-project
tk init                    # Create .tasuku/ directory
tk mcp install             # Enable tk_* tools in your AI editor
tk hooks install           # Context at session start, reminders at end
git add .tasuku/           # Tasks travel with your code
```

Works with Claude Code, Cursor, Copilot CLI, Codex, and OpenCode. Restart your editor after installing.

### 2. Your AI agent now has access to

**MCP Tools** (40+ tools, see `docs/mcp.md`):
- `tk_add` / `tk_list` / `tk_done` - Task management
- `tk_learn` / `tk_decide` - Knowledge capture
- `tk_context` - Load full project state

**Slash Commands** (Claude Code with plugin):
- `/tasuku:pickup` - Select and start a task
- `/tasuku:complete` - Mark done, capture learnings
- `/tasuku:reflect` - Extract learnings from recent work

### 3. CLI and TUI for humans

```bash
tk task list                 # See all tasks
tk task add "New feature"    # Add a task
tk task start <id>           # Start working
tk task done <id>            # Mark complete
tk learn "insight"           # Record a learning
tk ui                        # Terminal UI (j/k to navigate, ? for help)
```

## CLI Reference

| Command | Description |
|---------|-------------|
| `tk task list` | List tasks (`--status`, `--tag`, `--tree`) |
| `tk task add "desc"` | Add task (`--id`, `--parent`, `--priority`) |
| `tk task start/done/block <id>` | Lifecycle management |
| `tk task archive <id>` | Archive completed task |
| `tk task restore <id>` | Restore from archive |
| `tk learn "insight"` | Record a learning |
| `tk decide --id X --chose Y --over Z --because "..."` | Record decision |
| `tk task claim/release <id>` | Multi-agent coordination |
| `tk task timer start/stop <id>` | Time tracking |
| `tk plugin install` | Install slash commands to AI tools |
| `tk rules sync` | Sync learnings to editor rules |

Full reference: `docs/cli.md`

## Data Format

Tasks are Markdown files with YAML frontmatter:

```
.tasuku/
├── tasks/task-id.md      # One file per task
├── archive/              # Archived tasks
├── context/
│   ├── learnings.md      # Recorded insights
│   └── decisions.md      # Architectural decisions
└── index.json            # Auto-generated for fast queries
```

## Migration from Beads

```bash
tk migrate beads           # Convert .beads/ to .tasuku/
tk migrate beads --dry-run # Preview first
```

## Documentation

Detailed docs in `docs/`:
- `cli.md` - Complete CLI commands
- `mcp.md` - MCP tools and TUI keybindings
- `hooks.md` - Hook configuration
- `plugins.md` - Slash commands for AI tools
- `storage.md` - Storage format

## Development

```bash
go test ./...           # Run tests
go test -race ./...     # With race detector
go build -o tk ./cmd/tk # Build
```

## License

MIT
