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
└── docs/                # User documentation
```

## Parity Principle

**Every capability must be accessible through all interfaces.** This is critical for agent-first design.

| Interface | Users | Example |
|-----------|-------|---------|
| CLI | Humans in terminal | `tk task list` |
| MCP | AI agents | `tk_list` tool |
| TUI | Humans who prefer visual | `tk ui` |
| Plugin | AI agents (slash commands) | `/tasuku:list` |

**When adding functionality:**
1. New CLI command → Add MCP tool
2. New MCP tool → Consider CLI equivalent
3. Core operations → Add to TUI
4. Frequent operations → Consider Plugin command
5. Same behavior across all interfaces

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

## Documentation

### User Documentation (docs/)
| File | Contents |
|------|----------|
| `docs/cli.md` | Complete CLI command reference |
| `docs/mcp.md` | MCP tools and TUI keybindings |
| `docs/hooks.md` | Hook configuration |
| `docs/storage.md` | Storage format details |

### Development Documentation (.claude/rules/)
| File | Contents |
|------|----------|
| `development.md` | Code style, workflow, learning documentation rules |
| `testing.md` | Test strategy, commands, verification checklist |
| `tasuku/learnings.md` | Auto-synced learnings from .tasuku |
| `tasuku/decisions.md` | Auto-synced decisions from .tasuku |

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

- Always audit MCP tools, hooks, and nudges when adding new functionality
- Never use O(n²) algorithms when O(n log n) alternatives exist
- Always ensure MCP tool schema properties match handler parameters
- Never manually manipulate ANSI-styled strings - use lipgloss helpers
- Never iterate over a map while modifying it - collect keys first
- In BubbleTea TUIs, save both item ID AND index before refresh
- Plugins use `commands/*.md` for namespaced `/plugin:name` invocation

See `.claude/rules/tasuku/learnings.md` for complete list.
