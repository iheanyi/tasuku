---
description: "Record a learning or insight"
argument-hint: "\"INSIGHT\" [--scope GLOB]"
---

# Record Learning

```!
tk learn $ARGUMENTS
```

## Usage

```bash
tk learn "The API rate limits to 100 req/min"     # Add a learning
tk learn "Never use sync calls in handlers"       # Record a "never do" pattern
tk learn "Always validate input before DB write"  # Record an "always do" pattern
tk learn "Use mocks for external APIs" --scope "**/*_test.go"  # Scoped learning
```

## What to Record

- **API behaviors**: "The auth endpoint returns 401 for expired tokens, not 403"
- **Code patterns**: "Use the `withRetry` wrapper for all external API calls"
- **Gotchas**: "The config file must be loaded before initializing the logger"
- **Never/Always rules**: "Never store passwords in plain text"

## Scoped Learnings

Use `--scope` with a glob pattern to apply learnings only to specific files:

```bash
tk learn "Use React Query for data fetching" --scope "src/components/**"
```

## Promoting Learnings

For learnings that should be permanent documentation:

```bash
tk learning promote <id>                # Promote to CLAUDE.md
tk learning promote <id> --to AGENTS.md # Promote to specific file
```
