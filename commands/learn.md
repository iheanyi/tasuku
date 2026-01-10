---
description: Record a learning or insight. Use when discovering important patterns, gotchas, or knowledge that should be remembered.
---

# Record Learning

Capture insights discovered while working on the project.

## Usage

```bash
tk learn "The API rate limits to 100 req/min"     # Add a learning
tk learn "Never use sync calls in handlers"       # Record a "never do" pattern
tk learn "Always validate input before DB write"  # Record an "always do" pattern
tk learnings                                      # List all learnings
```

## What to Record

Good learnings include:

- **API behaviors**: "The auth endpoint returns 401 for expired tokens, not 403"
- **Code patterns**: "Use the `withRetry` wrapper for all external API calls"
- **Gotchas**: "The config file must be loaded before initializing the logger"
- **Performance**: "Batch inserts are 10x faster than individual inserts"
- **Never/Always rules**: "Never store passwords in plain text"

## When to Use

- After debugging a tricky issue
- When discovering undocumented behavior
- After making a decision that future work should know about
- When finding a pattern that works well (or poorly)

## Promoting Learnings

For learnings that should be permanent documentation:

```bash
tk promote 1                    # Promote learning #1 to context file
tk promote 1 --to CLAUDE.md     # Promote to specific file
tk promote 1 --keep             # Keep in learnings after promoting
```

## Best Practices

1. Be specific - include context that makes the learning actionable
2. Use "Never" or "Always" prefixes for rules
3. Promote important learnings to permanent docs
4. Review learnings periodically to refresh memory
