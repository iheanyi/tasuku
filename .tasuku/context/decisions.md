# Decisions

## language - 2026-01-06T00:00:00Z
**Chose**: Go
**Over**: TypeScript, Rust
**Because**: Single binary, good concurrency, fast startup for CLI

## data-format - 2026-01-06T00:00:00Z
**Chose**: JSON
**Over**: YAML, TOML, SQLite
**Because**: Faster parsing, unambiguous, better for programmatic access

## agent-model - 2026-01-06T00:00:00Z
**Chose**: Hierarchical with parallel workers
**Over**: Pure parallel, Sequential handoff
**Because**: Best for token efficiency - workers get scoped context

## grove-over-native - 2026-01-06T00:00:00Z
**Chose**: Grove integration
**Over**: Native worktree support
**Because**: Single responsibility - Tasuku manages tasks, Grove manages worktrees. Avoids duplication and leverages existing Grove infrastructure.

## v2-scope - 2026-01-06T00:00:00Z
**Chose**: Local-first features
**Over**: Webhooks, GitHub Actions, remote sync
**Because**: Tasuku is local-first with JSON file. Server-dependent features don't fit the philosophy. Focus on agent coordination, tags, time tracking, custom fields.

## cli-mcp-parity - 2026-01-05T00:00:00Z
**Chose**: Maintain 1:1 parity between CLI commands and MCP tools
**Over**: MCP as subset of CLI, MCP with different capabilities, CLI-only features
**Because**: Agent-first design means agents must be able to do everything humans can. If CLI has a capability that MCP lacks, agents are second-class citizens. This defeats the purpose of Tasuku.

## todowrite-vs-tasuku - 2026-01-06T00:00:00Z
**Chose**: Different tools for different scopes
**Over**: Auto-sync between tools, Always use tk only, Always use TodoWrite only
**Because**: TodoWrite is session-scoped (ephemeral implementation steps), Tasuku is project-scoped (persistent features/bugs). Avoids infinite loops while ensuring important tasks are tracked.

## timestamp-storage - 2026-01-06T00:00:00Z
**Chose**: RFC3339 UTC for all timestamps
**Over**: Date-only strings (YYYY-MM-DD), Unix timestamps, Local timezone storage
**Because**: RFC3339 is human-readable, unambiguous, sortable, and timezone-aware. UTC storage ensures consistency across systems while local display provides good UX. Date-only was insufficient for same-day decision ordering.

## timezone-display - 2026-01-06T00:00:00Z
**Chose**: Local timezone for display, UTC for storage
**Over**: Configurable timezone setting, Always UTC display, Store in local timezone
**Because**: Sensible defaults without configuration. Humans see familiar times, machines get unambiguous UTC. Go's time.Local handles detection automatically. JSON/YAML output stays UTC for machine readability.

## proactive-skills - 2026-01-09T20:10:25Z
**Chose**: Workflow skills with guided processes
**Over**: Basic CRUD skills only, Auto-invocation without guidance, Skills as thin wrappers
**Because**: Workflow skills (complete, pickup, reflect) guide agents through multi-step processes, ensuring learnings are captured and context is loaded. Hooks suggest skills instead of raw commands, making the right action discoverable at the right moment.

