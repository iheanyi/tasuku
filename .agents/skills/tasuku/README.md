# Tasuku Agent Skill

This directory contains the Tasuku skill for AI agents that support the
`.agents/skills/` convention (Amp, and others following the open agents spec).

## Files

- **SKILL.md** — Skill definition with name, description, and usage instructions.
  The description is always visible to the agent; the full content loads on demand
  when the skill is invoked.

- **mcp.json** — Bundled MCP server configuration. The Tasuku MCP server starts
  when the agent launches, but its tools remain hidden until the skill is loaded.
  This reduces context bloat compared to always-on MCP configuration.

## Installation

### Automatic (recommended)
```bash
tk plugin install --tool amp --local    # Install to .agents/skills/tasuku/
```

### Manual
Copy this directory to your project's `.agents/skills/` or your user-wide
skills directory (`~/.config/agents/skills/`).

## MCP Server Deduplication

If you also configure Tasuku's MCP server via `tk mcp install` (which writes to
your agent's settings file), the settings-based configuration takes precedence
over this skill's `mcp.json`. This means:

- **Both configured**: The settings-based MCP server is used; `mcp.json` is ignored.
  No conflict, just redundancy.
- **Only skill**: The `mcp.json` MCP server is used, tools load on demand.
- **Only settings**: The settings-based MCP server is always available.

The skill approach (this directory) is recommended because it loads tools on
demand, keeping the agent's context window clean until Tasuku is actually needed.
