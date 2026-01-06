# Decisions

## language - 2026-01-06
**Chose**: Go
**Over**: TypeScript, Rust
**Because**: Single binary, good concurrency, fast startup for CLI

## data-format - 2026-01-06
**Chose**: JSON
**Over**: YAML, TOML, SQLite
**Because**: Faster parsing, unambiguous, better for programmatic access

## agent-model - 2026-01-06
**Chose**: Hierarchical with parallel workers
**Over**: Pure parallel, Sequential handoff
**Because**: Best for token efficiency - workers get scoped context

## grove-over-native - 2026-01-06
**Chose**: Grove integration
**Over**: Native worktree support
**Because**: Single responsibility - Tasuku manages tasks, Grove manages worktrees. Avoids duplication and leverages existing Grove infrastructure.

## v2-scope - 2026-01-06
**Chose**: Local-first features
**Over**: Webhooks, GitHub Actions, remote sync
**Because**: Tasuku is local-first with JSON file. Server-dependent features don't fit the philosophy. Focus on agent coordination, tags, time tracking, custom fields.

## cli-mcp-parity - 2026-01-05
**Chose**: Maintain 1:1 parity between CLI commands and MCP tools
**Over**: MCP as subset of CLI, MCP with different capabilities, CLI-only features
**Because**: Agent-first design means agents must be able to do everything humans can. If CLI has a capability that MCP lacks, agents are second-class citizens. This defeats the purpose of Tasuku.

## todowrite-vs-tasuku - 2026-01-06
**Chose**: Different tools for different scopes
**Over**: Auto-sync between tools, Always use tk only, Always use TodoWrite only
**Because**: TodoWrite is session-scoped (ephemeral implementation steps), Tasuku is project-scoped (persistent features/bugs). Avoids infinite loops while ensuring important tasks are tracked.

