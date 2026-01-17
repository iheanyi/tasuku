---
status: done
priority: 1
tags: [refactor, cli]
created_at: 2026-01-17T15:09:09.025233Z
updated_at: 2026-01-17T15:17:22.711116Z
---

# Add `tk task restore <id>` as a top-level task verb. Move logic from `tk task...

Add `tk task restore <id>` as a top-level task verb. Move logic from `tk task archive restore`. This makes restore a sibling to archive, matching the verb pattern of other task lifecycle commands.
