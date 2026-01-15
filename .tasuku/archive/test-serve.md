---
status: done
priority: 3
tags: [testing, coverage]
created_at: 2026-01-14T22:30:56.316588Z
updated_at: 2026-01-15T01:52:36.900625Z
---

# Add tests for internal/cmd/serve package (currently 43.5% coverage)

## Summary
Blocking server functions not practical to unit test

## Notes

### 2026-01-15T01:52:36Z [f12780]
Coverage at 43.5%. Only untested functions are runMCP and runHTTP - blocking server starters that call external packages. Command construction is fully covered.

