# Tasuku Development Guidelines

## Overview

Tasuku is an agent-first task management system designed for AI agents working on codebases:
- **Pull over push**: Agents query when needed, no constant injections
- **Parallel-safe**: File locking for multiple agents
- **Minimal context**: Only load what's needed for the current task
- **Human-readable**: Markdown files with YAML frontmatter (V4)
- **Rich content**: Full Markdown support with code blocks, lists, and formatting

## Architecture

```
tasuku/
├── cmd/tk/              # CLI entrypoint
├── internal/
│   ├── store/           # Storage backends (V2 file, V3 JSON dir, V4 Markdown dir)
│   │   └── v4/          # V4 Markdown-based storage implementation
│   ├── task/            # Task domain logic
│   ├── http/            # HTTP REST API server
│   ├── mcp/             # MCP server for AI tools (Claude Code, Cursor, Codex, OpenCode)
│   └── tui/             # Terminal UI (bubble tea)
├── .tasuku/             # V4 directory-based storage (default)
│   ├── tasks/           # Individual task Markdown files
│   ├── archive/         # Archived task files
│   ├── context/         # learnings.md, decisions.md
│   ├── config.json      # Version marker (version: 4)
│   └── index.json       # Auto-generated index for fast queries
└── CLAUDE.md            # This file (kept minimal - details in .claude/rules/)
```

## Reference Documentation

Detailed documentation is split into modular files in `.claude/rules/`:

| File | Contents |
|------|----------|
| `cli-reference.md` | Complete `tk` CLI command reference |
| `mcp-reference.md` | MCP tools table, TUI keybindings, parity principle |
| `data-model.md` | V4/V3/V2 storage formats, file locking |
| `testing.md` | Test strategy, commands, verification checklist |
| `development.md` | Code style, workflow, learning documentation rules |
| `hooks-config.md` | Session hooks, MCP installation, hook features |
| `tasuku/learnings.md` | Auto-synced learnings from .tasuku |
| `tasuku/decisions.md` | Auto-synced decisions from .tasuku |

## Quick Reference

### Common Commands
```bash
tk task add "description"     # Add task
tk task start <id>            # Start working
tk task done <id>             # Complete task
tk learn "insight"            # Record learning
tk decide --id X --chose Y --over Z --because "reason"
```

### When to Use tk vs TodoWrite
- **tk (Tasuku)**: Features, bugs, milestones - persists across sessions
- **TodoWrite**: Implementation steps - session-only

Use `tk suggest "description"` to check which to use.

## Key Decisions

1. **JSON over YAML** - Faster parsing, no ambiguity, better for agents
2. **flock for locking** - Simple, works on macOS/Linux, sufficient for local
3. **MCP over REST** - Native Claude Code integration, no HTTP overhead
4. **V4 Markdown storage** - Human-readable, rich content, git-friendly diffs
5. **User-specified task IDs** - Short, memorable IDs like `fix-auth-bug`
6. **Constructor pattern** - CLI commands use `newCmd()` over `init()`
7. **UTC storage, local display** - Timestamps stored UTC, displayed local
8. **TodoWrite vs Tasuku distinction** - Session vs project scope

## Learnings (Key Rules)

- TUI/CLI/MCP/Plugin Parity: Every capability accessible through all interfaces
- Always audit MCP tools, hooks, and nudges when adding new functionality
- Never use O(n²) algorithms when O(n log n) alternatives exist
- Always ensure MCP tool schema properties match handler parameters
- Never manually manipulate ANSI-styled strings - use lipgloss helpers
- Never iterate over a map while modifying it - collect keys first
- In BubbleTea TUIs, save both item ID AND index before refresh
- Plugins use `commands/*.md` for namespaced `/plugin:name` invocation

See `.claude/rules/tasuku/learnings.md` for complete list.
