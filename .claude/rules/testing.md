# Testing Strategy

## Unit Tests

```bash
go test ./...
```

Every package should have `*_test.go` files:
- `internal/store/store_test.go` - File operations, locking
- `internal/task/task_test.go` - Domain logic
- `internal/mcp/server_test.go` - MCP protocol
- `internal/http/server_test.go` - HTTP REST API
- `internal/tui/model_test.go` - Terminal UI

**CLI command tests** (using `internal/cmd/testutil/harness.go`):
- `internal/cmd/task/task_test.go` - Task subcommands
- `internal/cmd/context/context_test.go` - Context commands
- `internal/cmd/decision/decision_test.go` - Decision commands
- `internal/cmd/learning/learning_test.go` - Learning commands
- `internal/cmd/note/note_test.go` - Note commands
- `internal/cmd/hooks/hooks_test.go` - Git hooks integration
- `internal/cmd/pr/pr_test.go` - PR creation
- `internal/cmd/server/server_test.go` - Server commands
- `internal/cmd/mcpcmd/mcp_test.go` - MCP management
- `internal/cmd/migrate/migrate_test.go` - Migration commands

## Integration Tests

```bash
go test ./... -tags=integration
```

Located in `testdata/` scenarios:
- `testdata/parallel_agents/` - Test file locking
- `testdata/corrupt_json/` - Error handling

## Manual Verification Checklist

Before merging any PR:

- [ ] `tk init` creates valid `.tasuku/` directory
- [ ] `tk task add/start/done` cycle works
- [ ] `tk task list` shows correct statuses
- [ ] `tk context show` outputs valid JSON
- [ ] Parallel writes don't corrupt files
- [ ] MCP server responds to tool calls
- [ ] V2 format auto-detection works

## Test Commands

```bash
# Run all tests
go test ./...

# Run with race detector (important for parallel safety)
go test -race ./...

# Run specific package
go test ./internal/store/...

# Verbose output
go test -v ./...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```
