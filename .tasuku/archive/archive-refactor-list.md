---
status: done
priority: 1
tags: [refactor, cli]
created_at: 2026-01-17T15:09:08.881526Z
updated_at: 2026-01-17T15:17:21.956177Z
---

# Add `--status archived` flag to `tk task list` that queries `.tasuku/archive/`...

Add `--status archived` flag to `tk task list` that queries `.tasuku/archive/` directory. This abstracts the storage location from users - they just filter by status like any other state.
