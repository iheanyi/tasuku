// Package server provides CLI commands for managing Tasuku servers.
package server

import (
	"fmt"

	"github.com/spf13/cobra"

	tkhttp "github.com/iheanyi/tasuku/internal/http"
	"github.com/iheanyi/tasuku/internal/mcp"
	"github.com/iheanyi/tasuku/internal/store"
)

// Cmd is the parent command for all server operations
var Cmd = &cobra.Command{
	Use:     "server",
	Aliases: []string{"srv"},
	Short:   "Manage Tasuku server",
	Long: `Manage the Tasuku server for AI tool integration or HTTP API access.

Subcommands:
  start     Start the MCP or HTTP server

Examples:
  tk server start              # Start MCP server (stdio mode)
  tk server start --http :3000 # Start HTTP server`,
}

func init() {
	Cmd.AddCommand(startCmd)

	startCmd.Flags().Int("port", 0, "HTTP port (deprecated, use --http)")
	startCmd.Flags().String("http", "", "HTTP address (e.g., :3000 or localhost:8080)")
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Tasuku server (MCP or HTTP)",
	Long: `Start a server for AI tool integration or HTTP API access.

Server Modes:

  MCP stdio (default):
    Used by Claude Code and other MCP-compatible AI tools.
    Communicates via stdin/stdout using the MCP protocol.
    This is the mode used when configured in AI tool settings.

  HTTP server (--http or --port):
    Runs a REST API server for programmatic access.
    Useful for integration with other tools or custom scripts.
    Includes a web dashboard at the root URL (/).

MCP Tools Exposed:
  tk_list, tk_add, tk_start, tk_done, tk_block, tk_unblock,
  tk_learn, tk_decide, tk_context, and more.

Examples:
  tk server start                     # Start MCP server (stdio mode)
  tk server start --http :3000        # Start HTTP server on port 3000
  tk server start --http localhost:8080  # HTTP on specific address

Web Dashboard:
  When running in HTTP mode, open the root URL in your browser
  (e.g., http://localhost:3000) to view the interactive dashboard.
  The dashboard supports:
  - Real-time task status with HTMX
  - Click to start/done/archive tasks
  - Filter by status
  - Progress visualization

See also:
  tk mcp install               # Auto-configure MCP in your AI tools
  tk mcp config                # Show MCP configuration JSON`,
	RunE: runStart,
}

func runStart(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	httpAddr, _ := cmd.Flags().GetString("http")

	s := store.DefaultStorageWithWarning()

	// HTTP mode via --http
	if httpAddr != "" {
		httpServer := tkhttp.New(s)
		return httpServer.Run(httpAddr)
	}

	// HTTP mode via --port (legacy)
	if port > 0 {
		httpServer := tkhttp.New(s)
		return httpServer.Run(fmt.Sprintf(":%d", port))
	}

	// Default: MCP stdio mode
	mcpServer := mcp.New(s)
	return mcpServer.Run()
}
