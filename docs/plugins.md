# Plugins & Skills

Tasuku provides guided workflows (slash commands) that can be installed across different AI tools.

## Supported Tools

| Tool | Format | Location | Detection |
|------|--------|----------|-----------|
| Claude Code | Plugin (commands/*.md) | Via plugin marketplace | `.claude/` or `CLAUDE.md` |
| Cursor | Commands (*.md) | `.cursor/commands/tasuku/` | `.cursor/` or `.cursorrules` |
| Copilot CLI | Skills (SKILL.md) | `.github/skills/tasuku/` | `.github/hooks/` or `.copilot/` |
| Codex | Skills (SKILL.md) | `.codex/skills/tasuku/` | `.codex/` or `CODEX.md` |

## Installation

```bash
tk plugin install              # Install to all detected tools
tk plugin install --tool cursor  # Install to Cursor only
tk plugin install --tool copilot # Install to Copilot CLI only
tk plugin install --tool codex   # Install to Codex only
tk plugin install --local        # Project-local instead of global
```

### Claude Code

For Claude Code, Tasuku uses the native plugin system:

```bash
# In Claude Code:
/plugin marketplace add https://github.com/iheanyi/tasuku
/plugin install tasuku
```

This provides all `/tasuku:*` slash commands.

### Cursor

Cursor commands are installed to `.cursor/commands/tasuku/`:

```bash
tk plugin install --tool cursor
```

Commands are prefixed with `tasuku-` to namespace them (e.g., `/tasuku-pickup`).

### Copilot CLI & Codex

Skills are installed using the SKILL.md format:

```bash
tk plugin install --tool copilot  # → .github/skills/tasuku/
tk plugin install --tool codex    # → .codex/skills/tasuku/
```

## Available Commands

View available commands:

```bash
tk plugin list
```

### Workflow Commands (Recommended)

| Command | Description |
|---------|-------------|
| `pickup` | Select and start a task with full context |
| `complete` | Mark task done, capture learnings |
| `reflect` | Extract learnings from recent work |
| `help` | Show all available commands |
| `tasuku` | Overview and quick reference |

### Basic Commands

| Command | Description |
|---------|-------------|
| `add` | Create a new task |
| `list` | List tasks with filtering |
| `start` | Begin working on a task |
| `done` | Mark task as complete |
| `block` | Mark task as blocked |
| `note` | Add note to a task |
| `learn` | Record a learning |
| `decide` | Record a decision |
| `context` | Get full project context |
| `ready` | Show tasks ready to work on |
| `show` | Show task details |
| `stats` | Show project statistics |
| `promote` | Promote learning to docs |

## Uninstallation

```bash
tk plugin uninstall              # Remove from all detected tools
tk plugin uninstall --tool cursor  # Remove from Cursor only
tk plugin uninstall --local        # Remove from project-local
```

## Status

Check installation status:

```bash
tk plugin status
```

Shows detected tools and whether Tasuku is installed locally and/or globally.

## Format Details

### Claude Code Plugin Format

```markdown
---
description: Short description for command picker
argument-hint: <optional-arg>
---

Full instructions for the AI agent...
```

### Cursor Command Format

Plain Markdown without frontmatter:

```markdown
# Command Title

Description of what the command does.

## Instructions

Step-by-step instructions for the AI agent...
```

### SKILL.md Format (Copilot/Codex)

```markdown
---
name: command-name
description: Short description
---

Full instructions for the AI agent...
```
