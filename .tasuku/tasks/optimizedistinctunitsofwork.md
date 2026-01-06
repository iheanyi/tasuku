---
status: done
created_at: 2026-01-06T04:39:11.475032Z
updated_at: 2026-01-06T04:46:52.619674Z
---

# We have something like this: internal/mcp/server.go:2308-2326, these are...

We have something like this: internal/mcp/server.go:2308-2326, these are separate units of work, therefore they can be parallelized using goroutines. Always optimize distinct units of work that are quick performance wins such as this, especially when IO is involved. Refactor the code to use goroutines for each distinct unit of work.
