---
status: done
priority: 1
tags: [refactor, cli]
created_at: 2026-01-17T15:09:08.952519Z
updated_at: 2026-01-17T15:17:22.364952Z
---

# - `tk task archive --older-than <duration>` - bulk archive done tasks (no...

- `tk task archive --older-than <duration>` - bulk archive done tasks (no `--all` flag needed, the filter implies bulk)
- Error if both `<id>` and `--older-than` provided
Remove the `add` and `all` subcommands.
