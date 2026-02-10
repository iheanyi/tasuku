# Fix Codex CLI MCP Compatibility

## Overview

Investigate and fix Codex CLI compatibility issues with Tasuku MCP server. The user reports that "Codex CLI is not working with Tasuku MCP" even after implementing framed protocol support.

## Problem Statement

Codex CLI is unable to communicate with Tasuku's MCP server. Previous attempts to add "framed protocol support" were unsuccessful. This investigation aims to identify the root cause and provide a fix.

## Research Findings

### 1. MCP Protocol Specification

The [official MCP specification](https://modelcontextprotocol.io/specification/2025-03-26/basic/transports) defines stdio transport as:

- **JSONL (Newline-delimited JSON)**: Messages are delimited by newlines (`\n`) and MUST NOT contain embedded newlines
- **No Content-Length headers required** for stdio transport

However, some clients (including potentially Codex with the rmcp client) may use **LSP-style Content-Length framing**:
- `Content-Length: <n>\r\n\r\n<json body>`

### 2. Codex CLI MCP Client

Based on [OpenAI Codex MCP Documentation](https://developers.openai.com/codex/mcp/) and [community discussions](https://community.openai.com/t/mcp-servers-all-time-out-narrowed-it-down-to-stdio-bug/1363658):

- Codex supports STDIO and Streamable HTTP servers
- The `experimental_use_rmcp_client` flag enables a new MCP client using the official Rust SDK
- Previous versions had stdio-handshake issues that caused timeouts
- Community reports suggest Content-Length framed responses may be expected

### 3. Current Tasuku Implementation

The MCP server (`internal/mcp/server.go:3282-3523`) already supports **dual protocol**:

1. **Auto-detection logic** (line 3326-3340):
   - If first non-empty line starts with `content-` (case-insensitive), switch to framed mode
   - Otherwise, use JSONL mode

2. **Framed response format** (line 3517-3519):
   ```go
   fmt.Fprintf(s.out, "Content-Type: application/json\r\nContent-Length: %d\r\n\r\n", len(data))
   s.out.Write(data)
   ```

### 4. Issues Found

#### Issue 1: Test Bug in `TestRun_FramedProtocol`

**File**: `internal/mcp/server_test.go:298`

**Problem**: The test incorrectly parses the response headers. It splits by `\r\n\r\n` which gives:
- `parts[0]` = `Content-Type: application/json\r\nContent-Length: 36`
- `parts[1]` = JSON body

Then it tries to extract Content-Length with:
```go
lengthStr := strings.TrimSpace(strings.TrimPrefix(parts[0], "Content-Length:"))
```

This fails because `parts[0]` starts with `Content-Type:`, not `Content-Length:`.

**Evidence**:
```
=== RUN   TestRun_FramedProtocol
    server_test.go:301: invalid content-length in response: strconv.Atoi:
    parsing "Content-Type: application/json\r\nContent-Length: 36": invalid syntax
--- FAIL: TestRun_FramedProtocol (0.00s)
```

#### Issue 2: Protocol Verification

Tested the actual `tk serve mcp` command:

1. **JSONL works correctly**:
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"ping"}' | tk serve mcp
   # Returns: {"jsonrpc":"2.0","id":1,"result":{}}
   ```

2. **Content-Length framing works when correct**:
   ```bash
   printf 'Content-Length: 151\r\n\r\n<json>' | tk serve mcp
   # Returns proper framed response
   ```

3. **Doctor command passes**:
   - `tk doctor` reports "MCP framed protocol (Content-Length) supported"
   - This works because `root.go:readFramedResponse` is implemented correctly

#### Issue 3: Potential Codex-Specific Requirements

Based on research, potential issues with Codex may include:

1. **Startup timeout**: Default 10 seconds may be too short if initialization is slow
2. **experimental_use_rmcp_client flag**: May need to be enabled in user's config
3. **Windows-specific stdio bugs**: If on Windows, there were known issues (now fixed)

## Proposed Solution

### Phase 1: Fix Test Bug

Update `internal/mcp/server_test.go` to correctly parse framed response headers:

**Current (broken)**:
```go
parts := strings.SplitN(resp, "\r\n\r\n", 2)
lengthStr := strings.TrimSpace(strings.TrimPrefix(parts[0], "Content-Length:"))
```

**Fixed**:
```go
parts := strings.SplitN(resp, "\r\n\r\n", 2)
if len(parts) != 2 {
    t.Fatalf("invalid framed response: %s", resp)
}

// Parse headers properly
headers := strings.Split(parts[0], "\r\n")
contentLength := -1
for _, header := range headers {
    if idx := strings.Index(header, ":"); idx != -1 {
        key := strings.TrimSpace(header[:idx])
        val := strings.TrimSpace(header[idx+1:])
        if strings.EqualFold(key, "Content-Length") {
            var err error
            contentLength, err = strconv.Atoi(val)
            if err != nil {
                t.Fatalf("invalid Content-Length: %v", err)
            }
        }
    }
}
if contentLength < 0 {
    t.Fatalf("missing Content-Length in response: %s", resp)
}
```

### Phase 2: Verify Codex Configuration

Ensure user's Codex configuration is correct:

1. **Check `~/.codex/config.toml`**:
   ```toml
   [mcp_servers.tasuku]
   command = "/path/to/tk"
   args = ["serve", "mcp"]
   startup_timeout_sec = 20  # Increase from default 10
   ```

2. **Enable rmcp client if needed**:
   ```toml
   [features]
   experimental_use_rmcp_client = true
   ```

### Phase 3: Add Diagnostic Logging

Add optional stderr logging for debugging:

```go
func (s *Server) readMessage(reader *bufio.Reader) ([]byte, error) {
    // Add debug logging to stderr (allowed by MCP spec)
    if os.Getenv("TASUKU_MCP_DEBUG") == "1" {
        fmt.Fprintf(os.Stderr, "[tasuku-mcp] reading message...\n")
    }
    // ... existing code
}
```

## Acceptance Criteria

- [ ] `TestRun_FramedProtocol` passes
- [ ] `TestRun_JSONLProtocol` passes
- [ ] `tk doctor` shows all checks passing
- [ ] Manual test with Codex CLI works

## Testing Approach

1. Run unit tests: `go test ./internal/mcp/... -v`
2. Run `tk doctor` to verify MCP server functionality
3. Test with actual Codex CLI (requires user verification)

## Technical Considerations

### What's Working

- JSONL protocol (standard MCP spec)
- Content-Length framed protocol (LSP-style)
- Auto-detection between protocols
- Doctor command diagnostic

### What Needs Investigation

- Whether Codex actually uses JSONL or Content-Length framing
- Whether the `experimental_use_rmcp_client` flag is required
- Whether there are additional handshake requirements

## References

- [MCP Transports Specification](https://modelcontextprotocol.io/specification/2025-03-26/basic/transports)
- [OpenAI Codex MCP Documentation](https://developers.openai.com/codex/mcp/)
- [Codex MCP Timeout Discussion](https://community.openai.com/t/mcp-servers-all-time-out-narrowed-it-down-to-stdio-bug/1363658)
- [rmcp Rust SDK PR #4252](https://github.com/openai/codex/pull/4252)
- Code files:
  - `internal/mcp/server.go:3282-3523` (Run function and protocol handling)
  - `internal/mcp/server_test.go:272-314` (Protocol tests)
  - `internal/cmd/root.go:552-750` (Doctor command and framed test)

## Open Questions (Critical)

Based on spec flow analysis, these questions need answers before proceeding:

### Q1: What is the actual Codex error message?

Without the specific error, we cannot confirm the root cause. Possible errors include:
- "Connection timeout" - suggests startup timeout
- "Protocol error" - suggests framing mismatch
- "Tool not found" - suggests registration issue
- Something else entirely

**Current assumption**: Proceeding with test fix first, as it's a confirmed bug.

### Q2: Does Codex use JSONL or Content-Length framing?

The official MCP spec uses JSONL, but community reports suggest Codex may use Content-Length. The Tasuku server supports both via auto-detection.

**Current assumption**: Both protocols are supported; the issue lies elsewhere.

### Q3: Is `experimental_use_rmcp_client` required?

This flag enables a new MCP client in Codex using the official Rust SDK.

**Current assumption**: Document as optional but recommended.

## MVP Implementation

### server_test.go

Fix the `TestRun_FramedProtocol` test to properly parse multi-line headers:

```go
func TestRun_FramedProtocol(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, ".tasuku.json")
    s := store.New(path)
    if err := s.Init(); err != nil {
        t.Fatalf("failed to init store: %v", err)
    }

    body := `{"jsonrpc":"2.0","id":1,"method":"ping"}`
    request := fmt.Sprintf("Content-Type: application/json\r\nContent-Length: %d\r\n\r\n%s", len(body), body)
    in := bytes.NewBufferString(request)
    out := &bytes.Buffer{}
    server := NewWithIO(s, in, out)

    if err := server.Run(); err != nil {
        t.Fatalf("server run failed: %v", err)
    }

    resp := out.String()
    if !strings.HasPrefix(resp, "Content-Type: application/json\r\nContent-Length:") {
        t.Fatalf("expected framed response, got: %s", resp)
    }

    // Split headers from body
    parts := strings.SplitN(resp, "\r\n\r\n", 2)
    if len(parts) != 2 {
        t.Fatalf("invalid framed response format: %s", resp)
    }

    // Parse headers to find Content-Length
    var contentLength int
    for _, header := range strings.Split(parts[0], "\r\n") {
        if idx := strings.Index(header, ":"); idx != -1 {
            key := strings.TrimSpace(header[:idx])
            val := strings.TrimSpace(header[idx+1:])
            if strings.EqualFold(key, "Content-Length") {
                n, err := strconv.Atoi(val)
                if err != nil {
                    t.Fatalf("invalid content-length: %v", err)
                }
                contentLength = n
            }
        }
    }

    if contentLength == 0 {
        t.Fatalf("missing Content-Length header in response")
    }
    if len(parts[1]) != contentLength {
        t.Fatalf("content-length mismatch: expected %d, got %d", contentLength, len(parts[1]))
    }

    var rpcResp Response
    if err := json.Unmarshal([]byte(parts[1]), &rpcResp); err != nil {
        t.Fatalf("invalid JSON response: %v", err)
    }
    if rpcResp.Error != nil {
        t.Fatalf("unexpected error response: %+v", rpcResp.Error)
    }
}
```
