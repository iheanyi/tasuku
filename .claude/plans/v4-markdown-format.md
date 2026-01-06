# V4 Markdown Storage Format

## Overview

Tasuku V4 introduces Markdown-based storage with YAML frontmatter, replacing the JSON-only V3 format. This provides:

- **Human readability**: Browse and edit tasks in any editor, GitHub renders beautifully
- **Rich content**: Code blocks, checklists, formatted text natively supported
- **Clean git diffs**: Content changes without JSON structural noise
- **Fast agent queries**: Auto-generated `index.json` for structured access

## Storage Structure

```
.tasuku/
├── tasks/
│   ├── auth-jwt.md           # Individual task files
│   └── api-refactor.md
├── archive/
│   └── completed-task.md     # Archived tasks (also Markdown)
├── context/
│   ├── learnings.md          # Project learnings
│   └── decisions.md          # Architectural decisions
└── index.json                # Auto-generated index for queries
```

## Task File Format

```markdown
---
status: in_progress
priority: 2
tags: [backend, api]
blocked_by: [auth-setup]
parent_id: epic-123
owner: claude
claimed_by: agent-1
time_spent: 3600
fields:
  estimate: 2h
  pr: https://github.com/org/repo/pull/123
created_at: 2024-01-05T10:00:00Z
updated_at: 2024-01-05T11:00:00Z
---

# Implement JWT authentication

Add token-based authentication to protect API endpoints.

Support **rich formatting**, `inline code`, and code blocks:

```go
func ValidateToken(token string) (*Claims, error) {
    // Implementation
}
```

## Acceptance Criteria

- [ ] Login endpoint returns JWT
- [ ] Middleware validates tokens
- [ ] Refresh token flow works

## Notes

### 2024-01-05 11:00
Found root cause - race condition in auth middleware. Need to add mutex.

### 2024-01-05 10:30
Started investigating the bug. Reproduced locally with concurrent requests.
```

### Frontmatter Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | enum | Yes | `ready`, `in_progress`, `blocked`, `done` |
| `priority` | int | No | 0 (critical) to 4 (backlog), default 2 |
| `tags` | string[] | No | Categorization tags |
| `blocked_by` | string[] | No | Task IDs that block this task |
| `parent_id` | string | No | Parent task ID for subtasks |
| `owner` | string | No | Assigned owner |
| `claimed_by` | string | No | Agent currently working on task |
| `time_spent` | int | No | Seconds spent on task |
| `fields` | map | No | Custom key-value fields |
| `created_at` | timestamp | Yes | ISO 8601 creation time |
| `updated_at` | timestamp | Yes | ISO 8601 last modified time |

### Notes Section

- **Format**: H3 headers with timestamp `### YYYY-MM-DD HH:MM`
- **Order**: Reverse chronological (newest first)
- **Rationale**: Latest context immediately visible when agent picks up task

### Content Sections

The Markdown body supports these conventional sections:

1. **Title/Description** (required): First H1 or paragraph after frontmatter
2. **Acceptance Criteria** (optional): H2 section with checklist
3. **Notes** (optional): H2 section with timestamped H3 entries

## Learnings Format

```markdown
# Learnings

## abc123 - 2024-01-05
Never use O(n²) algorithms when O(n log n) alternatives exist.

```go
// Bad: O(n²)
for _, a := range items {
    for _, b := range items { ... }
}

// Good: O(n)
lookup := make(map[string]Item)
for _, item := range items {
    lookup[item.ID] = item
}
```

## def456 - 2024-01-05
Always ensure switch cases return early in TUI apps to prevent fall-through.
```

### Learning Structure

- **H2 header**: `## <id> - <date>`
- **Content**: Learning text, can include code blocks, links, formatting
- **Order**: Chronological (oldest first, append new at bottom)

## Decisions Format

```markdown
# Decisions

## auth-strategy - 2024-01-05
**Chose**: JWT tokens
**Over**: Session cookies, OAuth2
**Because**: Stateless, scalable, no session storage overhead. Works across microservices.

## database-choice - 2024-01-05
**Chose**: PostgreSQL
**Over**: MySQL, MongoDB
**Because**: Better JSON support, complex query performance, mature ecosystem.
```

### Decision Structure

- **H2 header**: `## <id> - <date>`
- **Chose**: Bold label with chosen option
- **Over**: Bold label with alternatives (comma-separated)
- **Because**: Bold label with reasoning (can be multi-paragraph with lists, code)

## Index.json

Auto-generated index for fast agent queries:

```json
{
  "version": "v4",
  "tasks": {
    "auth-jwt": {
      "status": "in_progress",
      "priority": 2,
      "tags": ["backend", "api"],
      "blocked_by": ["auth-setup"],
      "parent_id": "epic-123",
      "owner": "claude",
      "claimed_by": "agent-1",
      "file": "tasks/auth-jwt.md",
      "updated_at": "2024-01-05T11:00:00Z"
    }
  },
  "archived_count": 42,
  "learnings_count": 5,
  "decisions_count": 3,
  "updated_at": "2024-01-05T11:00:00Z"
}
```

### Index Regeneration Strategy

- **Trigger**: Regenerate on every write operation
- **Scope**: Parse frontmatter only (not full Markdown content)
- **Performance**: <50ms for 100 tasks, <200ms for 500 tasks
- **Consistency**: Always in sync with source .md files

## Migration

### Command

```bash
tk migrate v4
```

### Process

1. Detect V3 `.tasuku/` directory with JSON files
2. For each `tasks/<id>.json`:
   - Extract fields to YAML frontmatter
   - Convert description to Markdown body
   - Convert notes array to H3 sections (reverse chronological)
   - Write as `tasks/<id>.md`
   - Delete original JSON file
3. Convert `context/learnings.json` to `context/learnings.md`
4. Convert `context/decisions.json` to `context/decisions.md`
5. Generate `index.json`
6. Print summary of migrated files

### Backwards Compatibility

- V2 (single `.tasuku.json`) → Use `tk migrate v3` first, then `tk migrate v4`
- V3 (directory JSON) → Direct `tk migrate v4`
- Auto-detection: Read `version` field or file extension to determine format

## Default Format

- `tk init` creates V4 Markdown format immediately
- No opt-in flag needed for new projects

## Error Handling

### Tiered Approach

1. **Single-task operations** (`tk task show`, `tk task start`): **Strict**
   - Fail with clear error if task file is malformed
   - Error message points to specific issue (line number, field)

2. **Bulk operations** (`tk task list`, `tk context show`): **Lenient**
   - Warn about malformed files
   - Continue with valid files
   - Summary: "3 tasks loaded, 1 skipped (malformed)"

3. **Explicit validation** (`tk validate`): **Comprehensive**
   - Check all .md files
   - Report all issues
   - Exit code 1 if any errors
   - Suitable for CI/pre-commit hooks

### No Auto-Repair

- Malformed files are never silently modified
- Clear error messages guide user to fix issues
- Prevents data loss from unexpected mutations

## TUI Rendering

- Use `glamour` library for styled Markdown rendering
- Code blocks: Syntax highlighting
- Headers: Bold/colored
- Lists: Proper indentation
- Links: Clickable (terminal permitting)
- Checklists: Visual checkboxes

## Implementation Tasks

### Phase 1: Core Format
- [ ] Define YAML frontmatter schema
- [ ] Implement Markdown parser for task files
- [ ] Implement Markdown writer for task files
- [ ] Create `index.json` generator

### Phase 2: Storage Layer
- [ ] Add V4 storage backend alongside V3
- [ ] Implement format auto-detection
- [ ] Update all store operations for Markdown

### Phase 3: Migration
- [ ] Implement `tk migrate v4` command
- [ ] Handle edge cases (empty notes, special characters)
- [ ] Add migration tests

### Phase 4: Validation
- [ ] Implement `tk validate` command
- [ ] Add frontmatter validation
- [ ] Add tiered error handling

### Phase 5: TUI
- [ ] Integrate glamour for Markdown rendering
- [ ] Update task detail view
- [ ] Update notes display

### Phase 6: Polish
- [ ] Update documentation
- [ ] Update MCP tool descriptions
- [ ] Performance testing with large task sets

## Open Questions

1. **Archive format**: Should archived tasks also get `index.json`?
   - Recommendation: Separate `archive/index.json` for archived task queries

2. **File naming**: Should task IDs allow special characters?
   - Recommendation: Restrict to alphanumeric, hyphens, underscores

3. **Concurrent writes**: File locking strategy for Markdown files?
   - Recommendation: Same flock approach as V3, lock individual .md files
