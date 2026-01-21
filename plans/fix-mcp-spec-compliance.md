# fix: MCP Specification Compliance Gaps

## Overview

Audit and fix gaps between Tasuku's MCP server implementation and the official MCP specification. The recent `notifications/initialized` fix for Codex compatibility revealed that we should do a thorough spec review.

## Problem Statement

Our MCP server implementation has several gaps compared to the official specification:

1. **Missing notification handlers** - Will return errors for valid client messages
2. **No batch request handling** - Will fail to parse valid JSON-RPC batches
3. **Protocol version hardcoded** - No version negotiation or logging

These gaps can cause compatibility issues with different MCP clients (Claude Code, Cursor, Codex, etc.).

## Gap Analysis Summary

| Gap | Severity | Fix Complexity |
|-----|----------|----------------|
| Missing `notifications/progress` handler | Medium | Simple |
| Missing `notifications/roots/list_changed` handler | Low | Simple |
| No batch request detection/error | Medium | Simple |
| Hardcoded protocol version `2024-11-05` | Low | Optional |
| Missing `instructions` in initialize response | Low | Optional |

## Proposed Solution

### Phase 1: Add Missing Notification Handlers (Required)

Add handlers for all standard MCP notifications that a server might receive:

```go
// internal/mcp/server.go - handleRequest() switch statement

case "initialized", "notifications/initialized":
    // Notification, no response needed
case "notifications/cancelled":
    // Client cancelled a request, no response needed
case "notifications/progress":
    // Client reporting progress on a request, no response needed
case "notifications/roots/list_changed":
    // Client's filesystem roots changed, no response needed
```

**Files to modify:**
- `internal/mcp/server.go:3292-3308` - Add notification cases

### Phase 2: Add Batch Request Detection (Required)

Detect JSON-RPC batch requests (arrays) and return proper error:

```go
// internal/mcp/server.go - Run() loop

line := scanner.Text()
if line == "" {
    continue
}

// Detect batch requests (JSON arrays) - not supported
if strings.HasPrefix(strings.TrimSpace(line), "[") {
    s.sendError(nil, -32600, "Invalid Request", "Batch requests not supported")
    continue
}
```

**Files to modify:**
- `internal/mcp/server.go:3275-3290` - Add batch detection

### Phase 3: Add Instructions to Initialize Response (Optional)

Help AI agents understand Tasuku's purpose:

```go
// internal/mcp/server.go - handleInitialize()

result := InitializeResult{
    ProtocolVersion: ProtocolVersion,
    Capabilities: ServerCapability{
        Tools: &ToolsCapability{},
    },
    ServerInfo: ServerInfo{
        Name:    ServerName,
        Version: ServerVersion,
    },
    Instructions: "Tasuku is an agent-first task management system. " +
        "Use tk_context at session start to load project state. " +
        "Use tk_learn to capture insights and tk_decide for architectural decisions.",
}
```

**Files to modify:**
- `internal/mcp/server.go:69-77` - Add Instructions field to InitializeResult struct
- `internal/mcp/server.go:3311-3322` - Add Instructions to response

## Technical Considerations

### MCP Notification Method Names

Per the MCP spec, all notifications use the `notifications/` prefix:

| Notification | Direction | When Sent |
|--------------|-----------|-----------|
| `notifications/initialized` | Client → Server | After initialize response |
| `notifications/cancelled` | Bidirectional | Cancel in-progress request |
| `notifications/progress` | Bidirectional | Progress on long operations |
| `notifications/roots/list_changed` | Client → Server | Client roots changed |
| `notifications/tools/list_changed` | Server → Client | Tools changed |
| `notifications/resources/list_changed` | Server → Client | Resources changed |
| `notifications/prompts/list_changed` | Server → Client | Prompts changed |

### JSON-RPC Batch Requests

JSON-RPC 2.0 supports batch requests as arrays:
```json
[
  {"jsonrpc":"2.0","id":1,"method":"ping"},
  {"jsonrpc":"2.0","id":2,"method":"tools/list"}
]
```

Current behavior: Parse error (expects object, not array)
Correct behavior: `-32600 Invalid Request` with clear message

### Protocol Version

Current: `2024-11-05` (initial MCP release)
Latest: `2025-11-25` (async operations, tasks)

We should keep `2024-11-05` since we don't implement newer features. Consider logging a warning if client requests newer version.

## Acceptance Criteria

- [x] Server handles `notifications/progress` without error
- [x] Server handles `notifications/roots/list_changed` without error
- [x] Server returns `-32600` for batch requests (not `-32700` parse error)
- [x] All existing tests pass
- [x] New tests cover notification handling
- [x] New tests cover batch request rejection

## Test Plan

### Unit Tests

```go
// internal/mcp/server_test.go

func TestNotifications_Progress(t *testing.T) {
    // Send notifications/progress, verify no response and no error
}

func TestNotifications_RootsListChanged(t *testing.T) {
    // Send notifications/roots/list_changed, verify no response and no error
}

func TestBatchRequest_Rejected(t *testing.T) {
    // Send JSON array, verify -32600 error response
}
```

### Manual Verification

1. Start MCP server: `tk serve mcp`
2. Send batch request: `echo '[{"jsonrpc":"2.0","id":1,"method":"ping"}]' | tk serve mcp`
3. Verify error response contains "Batch requests not supported"

## Implementation Order

1. Add notification handlers (simple, low risk)
2. Add batch detection (simple, low risk)
3. Add tests for new handlers
4. Optional: Add instructions field
5. Run full test suite

## References

- [MCP Specification 2024-11-05](https://modelcontextprotocol.io/specification/2024-11-05)
- [MCP Specification 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25)
- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)
- Fix for `notifications/initialized`: commit `79c1a53`
- Learning recorded: "Codex CLI (rmcp SDK) sends MCP notifications with full method names"
