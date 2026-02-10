---
paths: internal/cmd/testutil/**
---

# Tasuku Learnings

_Auto-synced from .tasuku/context/learnings.md_

## Rules

- Always ensure the test harness storage backend matches the production default. When migrating storage formats (V3→V4), the test harness must also be updated — otherwise ALL command tests pass but exercise the wrong backend. Added TestHarness_UsesV4Storage as a regression guard that detects if harness and AutoDetect() ever diverge.

