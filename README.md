# Tasuku

Agent-first task management for codebases. Designed for AI agents working alongside humans.

## Why Tasuku?

Traditional task management is built for humans pushing updates. Tasuku flips this:

- **Pull over push**: Agents query when needed, no constant context injections
- **Parallel-safe**: File locking for multiple agents working simultaneously
- **Minimal context**: Only load what's needed for the current task
- **Human-readable**: Single JSON file, can be edited by hand

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
tk init

# Add some tasks
tk add "Implement user authentication"
tk add "Write API documentation" --priority 1  # high priority
tk add "Set up CI pipeline"

# Start working on a task
tk start implement-user-authentication

# Mark it done
tk done implement-user-authentication

# See all tasks
tk list
tk list --format json  # Output as JSON
tk list --format yaml  # Output as YAML
```

## CLI Commands

### Task Management

| Command | Description |
|---------|-------------|
| `tk init` | Create `.tasuku.json` in current directory |
| `tk list` | List all tasks (use `--status` to filter) |
| `tk add "description"` | Add a new task |
| `tk add "desc" --id custom-id` | Add with custom ID |
| `tk add "desc" --priority 0` | Add with priority (0=critical, 1=high, 2=normal, 3=low, 4=backlog) |
| `tk show <id>` | Show task details |
| `tk start <id>` | Mark task as in progress |
| `tk done <id>` | Mark task as complete |
| `tk block <id> --by <other>` | Mark task as blocked |
| `tk unblock <id>` | Remove all blockers from task |
| `tk ready` | List tasks ready to work on (sorted by priority) |
| `tk find <query>` | Search tasks, learnings, and decisions |
| `tk priority <id> <level>` | Set task priority |
| `tk validate` | Check `.tasuku.json` for errors |

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
| `tk note <task-id> "note"` | Add a note to a task |
| `tk context` | Output full context as JSON |

### Server & Integration

| Command | Description |
|---------|-------------|
| `tk serve` | Start MCP server (stdio mode) |
| `tk serve --http :3000` | Start HTTP REST API server |
| `tk mcp install` | Install MCP server in Claude Code |
| `tk mcp uninstall` | Remove MCP server from Claude Code |
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
tk serve --http :3000
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

If you prefer manual setup, add this to `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "tasuku": {
      "command": "/path/to/tk",
      "args": ["serve"]
    }
  }
}
```

## Data Format

Tasuku stores everything in `.tasuku.json`:

```json
{
  "version": 1,
  "tasks": {
    "task-id": {
      "status": "ready",
      "description": "What needs to be done",
      "priority": 2,
      "blocked_by": [],
      "owner": null,
      "created_at": "2024-01-04T10:00:00Z",
      "updated_at": "2024-01-04T10:00:00Z"
    }
  },
  "context": {
    "learnings": ["Things discovered while working"],
    "decisions": [
      {
        "id": "decision-id",
        "chose": "Option A",
        "over": ["Option B", "Option C"],
        "because": "Reasoning"
      }
    ],
    "notes": {
      "task-id": ["Note 1", "Note 2"]
    }
  }
}
```

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
tk start auth-task           tk start api-task
# Both acquire locks safely, no corruption
```

## Migrating from Beads

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
