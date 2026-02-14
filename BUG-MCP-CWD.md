# Bug: MCP server fails to find `.tasuku/` storage when spawned without explicit `cwd`

## Problem

When AI tools (Cursor, Claude Code) spawn `tk serve mcp` as a subprocess, the process inherits the tool's working directory — not the project root. Since `DetectStorageTypeUp()` in `internal/store/storage.go:159` uses `os.Getwd()` as the starting point for its walk-up search, it starts from the wrong directory and never finds the `.tasuku/` storage.

**Result:** Every `tk_*` MCP tool call returns `"no Tasuku storage found - run 'tk init' to create one"` even though `.tasuku/` exists at the project root.

## Root Cause

`DetectStorageTypeUp()` walks up from `os.Getwd()` looking for `.tasuku/` directories, stopping at the git root. When the MCP server's working directory is outside the project tree (e.g., `/` or `$HOME`), the walk-up never reaches the project's `.tasuku/` directory.

```go
// internal/store/storage.go:159-162
func DetectStorageTypeUp() (StorageType, string) {
    dir, err := os.Getwd()  // <-- This is wrong when spawned by AI tools
    if err != nil {
        return StorageTypeNone, ""
    }
    // ... walks up to git root, never finds .tasuku/
}
```

## Current Workaround

Users must set `cwd` in their project-level MCP config:

```json
// .cursor/mcp.json (project-level)
{
  "mcpServers": {
    "tasuku": {
      "command": "/path/to/tk",
      "args": ["serve", "mcp"],
      "type": "stdio",
      "cwd": "/absolute/path/to/project"
    }
  }
}
```

This is not obvious, not documented, and defeats the purpose of having auto-detection.

## Suggested Fix

The MCP server should accept an optional `--dir` / `--project` flag or environment variable (e.g., `TASUKU_PROJECT_DIR`) that overrides `os.Getwd()`. The `tk mcp install` command should automatically set this in the generated config.

Alternatively, the MCP protocol's `initialize` request from the client sometimes includes `rootUri` or workspace information that could be used to determine the project root.

### Option A: CLI flag

```go
// internal/cmd/serve/serve.go
func newMCPCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "mcp",
        RunE:  runMCP,
    }
    cmd.Flags().String("dir", "", "Project directory (overrides cwd for storage detection)")
    return cmd
}

func runMCP(cmd *cobra.Command, args []string) error {
    if dir, _ := cmd.Flags().GetString("dir"); dir != "" {
        os.Chdir(dir)
    }
    s, err := store.DefaultStorageWithWarning()
    // ...
}
```

Then `tk mcp install` generates configs with `--dir` pointing to the project:

```json
{
  "command": "tk",
  "args": ["serve", "mcp", "--dir", "/path/to/project"]
}
```

### Option B: Environment variable

```go
func DetectStorageTypeUp() (StorageType, string) {
    dir := os.Getenv("TASUKU_PROJECT_DIR")
    if dir == "" {
        var err error
        dir, err = os.Getwd()
        if err != nil {
            return StorageTypeNone, ""
        }
    }
    // ... rest of walk-up logic
}
```

## Cursor-Specific Quirks

### Two levels of MCP config

Cursor has **two** MCP config files, and they interact in non-obvious ways:

1. **Global:** `~/.cursor/mcp.json` — applies to all workspaces
2. **Project-level:** `<project>/.cursor/mcp.json` — per-workspace overrides

If the same server name (e.g., `"tasuku"`) appears in both, Cursor loads **both** as separate MCP server instances. They show up as two entries in the MCP settings UI (e.g., `tasuku` and `tasuku-1`), and the tool names get namespaced differently (`user-tasuku-tk_context` vs `project-0-scanarr-tasuku-tk_context`). This means:

- The global one (wrong `cwd`) still runs and fails
- The project one (correct `cwd`) works but has different tool names
- The agent has to figure out which tool name variant to use
- If both are enabled, there's ambiguity about which server handles calls

### No implicit `cwd` from workspace

Unlike Claude Code (which passes `rootUri` in MCP initialization), Cursor does **not** automatically set the MCP server's working directory to the open workspace. The spawned process gets whatever `cwd` the Electron app has, which is unpredictable — could be `/`, `$HOME`, or wherever Cursor was launched from. There is no way to infer the workspace root from inside the MCP server without the client telling you.

### `cwd` is supported but undocumented

Cursor's MCP config **does** support a `cwd` field, but it's not well-documented. The schema is:

```json
{
  "mcpServers": {
    "server-name": {
      "command": "path/to/binary",
      "args": ["arg1", "arg2"],
      "type": "stdio",
      "cwd": "/absolute/path/to/working/directory"
    }
  }
}
```

### Reload required after config changes

After modifying `.cursor/mcp.json`, the MCP server does **not** hot-reload. You must either:
- Reload the Cursor window (Cmd+Shift+P → "Developer: Reload Window")
- Or restart Cursor entirely

The MCP settings UI (Cmd+Shift+P → "Cursor: MCP Settings") shows server connection status but doesn't provide a per-server restart button.

### `tk mcp install` should handle this

The `tk mcp install` command currently generates a global config without `cwd`. For Cursor, it should either:
1. Detect the current project and generate a **project-level** `.cursor/mcp.json` with `cwd` set
2. Or use `--dir` flag in the args so the server can `os.Chdir()` on startup
3. Or set `TASUKU_PROJECT_DIR` env var in the config

Option 1 is probably best since it's per-project anyway and keeps the global config clean.

## Affected Tools

- **Cursor** (confirmed) — global MCP config has no `cwd`, spawns from unpredictable directory. Dual config files add confusion.
- **Claude Code** — may be less affected if it passes `rootUri` in MCP init, but should still be tested.
- Any MCP client that spawns the process from a non-project directory.

## Discovered

2026-02-09, while setting up Scanarr project. The global `~/.cursor/mcp.json` config for Tasuku had no `cwd`, causing every `tk_context`, `tk_learn`, etc. call to fail with "no Tasuku storage found." Fixed by adding a project-level `.cursor/mcp.json` with explicit `cwd`. Took multiple debug cycles to figure out — the error message gives no hint about the working directory being wrong.
