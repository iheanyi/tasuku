# Tasuku

Agent-first task management for codebases. Designed for AI agents working alongside humans.

![Tasuku TUI Demo](assets/demo.gif)

## Why Tasuku?

Traditional task management is built for humans pushing updates. Tasuku flips this:

- **Pull over push**: Agents query when needed, no constant context injections
- **Parallel-safe**: Per-file locking for multiple agents working simultaneously
- **Minimal context**: Only load what's needed for the current task
- **Human-readable**: JSON files in `.tasuku/` directory, can be edited by hand
- **Git-friendly**: One file per task means clean diffs and fewer merge conflicts

## Installation

### From source

```bash
git clone https://github.com/iheanyi/tasuku.git
cd tasuku
go build -o tk ./cmd/tk
sudo mv tk /usr/local/bin/  # or add to your PATH
```

### With go install

```bash
go install github.com/iheanyi/tasuku/cmd/tk@latest
```

## Quick Start

```bash
# Initialize in your project
cd your-project
tk init                     # Creates .tasuku/ directory
git add .tasuku/            # Commit tasks so they travel with your code

# Add some tasks
tk task add "Implement user authentication"
tk task add "Write API documentation" --priority high
tk task add "Set up CI pipeline"

# Add subtasks
tk task add "Create login form" --parent implement-user-authentication

# Start working on a task
tk task start implement-user-authentication

# Mark it done
tk task done implement-user-authentication

# See all tasks
tk task list                  # Table view
tk task list --tree           # Hierarchical subtask view
tk task list --format json    # Output as JSON
```

## CLI Commands

### Task Management

| Command | Description |
|---------|-------------|
| `tk init` | Create `.tasuku/` directory (V3 format) |
| `tk task list` | List all tasks (use `--status` to filter) |
| `tk task list --tree` | Show hierarchical subtask view |
| `tk task add "description"` | Add a new task |
| `tk task add "desc" --id custom-id` | Add with custom ID |
| `tk task add "desc" --parent parent-id` | Add as subtask |
| `tk task add "desc" --priority high` | Add with priority (critical/high/normal/low/backlog) |
| `tk task show <id>` | Show task details |
| `tk task start <id>` | Mark task as in progress |
| `tk task done <id>` | Mark task as complete |
| `tk task block <id> --by <other>` | Mark task as blocked |
| `tk task unblock <id>` | Remove all blockers from task |
| `tk task delete <id>` | Delete a task |
| `tk task find <query>` | Search tasks, learnings, and decisions |
| `tk task priority <id> <level>` | Set task priority |
| `tk ready` | List tasks ready to work on (sorted by priority) |
| `tk validate` | Check task files for errors |

### Context & Knowledge

| Command | Description |
|---------|-------------|
| `tk learn "insight"` | Record a learning |
| `tk learn "insight" --permanent` | Record and append to context file |
| `tk learnings` | List all recorded learnings |
| `tk unlearn <index or text>` | Remove a learning |
| `tk promote <index or text>` | Move learning to permanent documentation |
| `tk promote 1 --to AGENTS.md` | Promote to specific file |
| `tk decide --id <id> --chose X --over Y,Z --because "reason"` | Record a decision |
| `tk note add <task-id> "note"` | Add a note to a task |
| `tk context show` | Output full context as JSON |

### Server & Integration

| Command | Description |
|---------|-------------|
| `tk serve mcp` | Start MCP server (stdio mode for AI tools) |
| `tk serve http` | Start HTTP REST API server on :3000 |
| `tk serve http --port 8080` | Start HTTP server on custom port |
| `tk mcp install` | Install MCP server in Claude Code |
| `tk mcp uninstall` | Remove MCP server from Claude Code |
| `tk migrate v3` | Migrate from V2 (.tasuku.json) to V3 (.tasuku/) |
| `tk migrate beads` | Migrate from Beads format |
| `tk migrate beads --dry-run` | Preview migration without changes |

### Output Formats

All list commands support the `--format` flag:

```bash
tk list --format table  # Default, human-readable
tk list --format json   # JSON output
tk list --format yaml   # YAML output
```

## Shell Completion

Tasuku supports tab completion for commands, subcommands, flags, and task IDs.

### Quick Setup

```bash
# Bash (Linux)
tk completion bash | sudo tee /etc/bash_completion.d/tk > /dev/null

# Bash (macOS with Homebrew)
tk completion bash > $(brew --prefix)/etc/bash_completion.d/tk

# Zsh
echo 'source <(tk completion zsh)' >> ~/.zshrc

# Fish
tk completion fish > ~/.config/fish/completions/tk.fish
```

### Detailed Setup

#### Bash

```bash
# Generate completion script
tk completion bash > /tmp/tk.bash

# Test in current session
source /tmp/tk.bash

# Install permanently (Linux)
sudo mv /tmp/tk.bash /etc/bash_completion.d/tk

# Install permanently (macOS)
# First install bash-completion: brew install bash-completion@2
tk completion bash > $(brew --prefix)/etc/bash_completion.d/tk
# Add to ~/.bash_profile:
# [[ -r "$(brew --prefix)/etc/profile.d/bash_completion.sh" ]] && source "$(brew --prefix)/etc/profile.d/bash_completion.sh"
```

#### Zsh

```bash
# Option 1: Source directly (add to ~/.zshrc)
source <(tk completion zsh)

# Option 2: Install to fpath
tk completion zsh > "${fpath[1]}/_tk"
# Then reload: autoload -Uz compinit && compinit

# Option 3: Custom directory
mkdir -p ~/.zsh/completions
tk completion zsh > ~/.zsh/completions/_tk
# Add to ~/.zshrc before compinit:
# fpath=(~/.zsh/completions $fpath)
```

#### Fish

```bash
tk completion fish > ~/.config/fish/completions/tk.fish
# Fish auto-loads from this directory
```

### What Gets Completed

After setup, you can tab-complete:

```bash
tk <TAB>                    # Shows: task, learning, decision, note, ...
tk task <TAB>               # Shows: list, add, show, start, done, ...
tk task list --<TAB>        # Shows: --format, --status
tk task start <TAB>         # Shows available task IDs
tk learning <TAB>           # Shows: list, add, remove, promote
```

### Troubleshooting

**Completions not working?**
- Restart your shell after installing
- Zsh: Run `compinit` or restart terminal
- Bash: Ensure bash-completion is installed

**"command not found: compdef" (Zsh)?**
```bash
autoload -Uz compinit && compinit
```

**Outdated completions?**
Regenerate with the same command after upgrading tk.

## HTTP REST API

Start the REST API server:

```bash
tk serve http              # Starts on :3000 by default
tk serve http --port 8080  # Custom port
```

### Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/tasks` | List all tasks |
| GET | `/tasks?status=ready` | Filter by status |
| POST | `/tasks` | Create a task |
| GET | `/tasks/{id}` | Get task details |
| PUT | `/tasks/{id}` | Update task status/priority |
| DELETE | `/tasks/{id}` | Delete a task |
| GET | `/ready` | List ready tasks |
| GET | `/context` | Get full context |
| POST | `/learnings` | Add a learning |
| POST | `/decisions` | Record a decision |
| GET | `/schema` | Get JSON schema |
| GET | `/health` | Health check |

### Example API Usage

```bash
# Create a task
curl -X POST http://localhost:3000/tasks \
  -H "Content-Type: application/json" \
  -d '{"description": "Build login form", "priority": 1}'

# Update status
curl -X PUT http://localhost:3000/tasks/build-login-form \
  -H "Content-Type: application/json" \
  -d '{"status": "in_progress"}'

# Add a learning
curl -X POST http://localhost:3000/learnings \
  -H "Content-Type: application/json" \
  -d '{"learning": "JWT tokens expire after 24 hours"}'
```

## Promoting Learnings to Permanent Docs

Tasuku can auto-detect your AI tool context file and promote learnings there:

```bash
# List learnings
tk learnings

# Promote learning #2 to auto-detected context file
tk promote 2

# Promote to a specific file
tk promote 2 --to .cursorrules

# Keep in .tasuku.json after promoting
tk promote 2 --keep
```

**Auto-detected context files** (in priority order):
1. `CLAUDE.md` - Claude Code
2. `.cursorrules` - Cursor
3. `.github/copilot-instructions.md` - GitHub Copilot
4. `AGENTS.md` - Generic AI agents

If none exist, defaults to creating `CLAUDE.md`.

## Claude Code Integration

Tasuku includes an MCP (Model Context Protocol) server for seamless Claude Code integration.

### Install the MCP server

```bash
tk mcp install
```

This adds Tasuku to your Claude Code settings. Restart Claude Code to activate.

### Available MCP Tools

Once installed, Claude has access to these tools:

- `tk_list` - List tasks with optional status filter
- `tk_add` - Create new tasks
- `tk_start` - Begin working on a task
- `tk_done` - Complete a task
- `tk_block` - Mark task as blocked
- `tk_learn` - Record learnings
- `tk_decide` - Record decisions
- `tk_note` - Add notes to tasks
- `tk_context` - Get full project context

### Manual Configuration

If you prefer manual setup, add this to `~/.claude.json`:

```json
{
  "mcpServers": {
    "tasuku": {
      "command": "/path/to/tk",
      "args": ["serve", "mcp"]
    }
  }
}
```

## Hooks

Tasuku provides hooks for automating task management workflows with git and Claude Code.

### Installation

```bash
# Install all hooks (git + Claude Code)
tk hooks install

# Install only git hooks
tk hooks install --git

# Install only Claude Code hooks
tk hooks install --claude

# Overwrite existing hooks
tk hooks install --force
```

### Git Hooks

- **pre-commit**: Validates task files before committing
- **post-commit**: Auto-updates task status based on commit messages

### Claude Code Hooks

- **ExitPlanMode**: Syncs tasks when Claude exits plan mode, extracting tasks from plan files

### Additional Commands

```bash
# Extract tasks from a plan file
tk hooks plan-sync

# Display session context summary
tk hooks session

# Remove all hooks
tk hooks uninstall
```

## Data Format

### V3 Format (Default)

Tasuku stores tasks in the `.tasuku/` directory:

```
.tasuku/
├── tasks/
│   └── task-id.json        # One file per task
├── archive/
│   └── old-task.json       # Completed/archived tasks
└── context/
    ├── learnings.json      # Array of learnings
    └── decisions.json      # Array of decisions
```

**Task file** (`.tasuku/tasks/task-id.json`):
```json
{
  "status": "ready",
  "description": "What needs to be done",
  "priority": 2,
  "parent_id": null,
  "blocked_by": [],
  "owner": null,
  "tags": ["backend"],
  "notes": [{"text": "Note 1", "created_at": "..."}],
  "created_at": "2024-01-04T10:00:00Z",
  "updated_at": "2024-01-04T10:00:00Z"
}
```

### V2 Format (Legacy)

Single `.tasuku.json` file with all data. Auto-detected for backwards compatibility.
Use `tk migrate v3` to upgrade.

### Task Statuses

- `ready` - Can be started
- `in_progress` - Currently being worked on
- `blocked` - Waiting on other tasks
- `done` - Completed

### Priority Levels

| Level | Name | Description |
|-------|------|-------------|
| 0 | Critical | Urgent, blocking issues |
| 1 | High | Important, do soon |
| 2 | Normal | Default priority |
| 3 | Low | Can wait |
| 4 | Backlog | Future work |

## Parallel Agent Safety

Tasuku uses `flock` for file locking, making it safe for multiple agents to work simultaneously:

```bash
# Agent 1                    # Agent 2
tk task start auth-task      tk task start api-task
# Both acquire locks safely, no corruption
```

V3's per-file locking means agents working on different tasks never block each other.

## Migration

### From V2 to V3

If you have an existing `.tasuku.json`:

```bash
# Preview the migration
tk migrate v3 --dry-run

# Run the migration
tk migrate v3
```

### From Beads

If you have an existing `.beads/` directory:

```bash
# Preview the migration
tk migrate beads --dry-run

# Run the migration
tk migrate beads
```

This converts your Beads issues to Tasuku tasks, preserving:
- Task status mapping (open->ready, in_progress->in_progress, closed->done, blocked->blocked)
- Priority levels
- Dependencies (blocked_by relationships)
- Close reasons as notes

## Development

```bash
# Run tests
go test ./...

# Run with race detector
go test -race ./...

# Build
go build -o tk ./cmd/tk
```

## License

MIT
