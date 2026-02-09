// Package serve provides CLI commands for starting Tasuku servers.
package serve

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	tkhttp "github.com/iheanyi/tasuku/internal/http"
	"github.com/iheanyi/tasuku/internal/mcp"
	"github.com/iheanyi/tasuku/internal/store"
)

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start a Tasuku server",
		Long: `Start a Tasuku server for AI tool integration or HTTP API access.

Modes:
  mcp    Start MCP server (stdio mode for AI tools)
  http   Start HTTP REST API server

Examples:
  tk serve mcp              # Start MCP server (for Claude Code, Cursor)
  tk serve http             # Start HTTP server on :3000
  tk serve http --port 8080 # Start HTTP server on custom port`,
	}

	cmd.AddCommand(newMCPCmd())
	cmd.AddCommand(newHTTPCmd())

	return cmd
}

// Cmd is the parent command for all serve operations
var Cmd = newServeCmd()

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server (stdio mode)",
		Long: `Start the MCP (Model Context Protocol) server in stdio mode.

This is the mode used by AI tools like Claude Code and Cursor.
The server communicates via stdin/stdout using the MCP protocol.

You typically don't run this directly - instead use 'tk mcp install'
to configure your AI tool to run it automatically.

MCP Tools Exposed:
  tk_list, tk_add, tk_start, tk_done, tk_block, tk_unblock,
  tk_learn, tk_decide, tk_context, and more.

Examples:
  tk serve mcp                        # Start MCP server
  tk serve mcp --dir /path/to/project # Start with explicit project dir`,
		RunE: runMCP,
	}

	cmd.Flags().String("dir", "", "Project directory (overrides cwd for storage detection)")

	return cmd
}

func newHTTPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "http",
		Short: "Start HTTP REST API server",
		Long: `Start a REST API server for programmatic access.

Useful for integration with other tools, custom scripts, or
accessing Tasuku from web applications.

Includes a web dashboard at the root URL (/).

Endpoints:
  GET    /tasks          List all tasks
  POST   /tasks          Create a task
  GET    /tasks/{id}     Get task details
  PUT    /tasks/{id}     Update task status
  DELETE /tasks/{id}     Delete a task
  GET    /ready          List ready tasks
  GET    /context        Get full context
  POST   /learnings      Add a learning
  POST   /decisions      Record a decision

Examples:
  tk serve http              # Start on default port :3000
  tk serve http --port 8080  # Start on custom port
  tk serve http --addr :8080 # Alternative syntax`,
		RunE: runHTTP,
	}

	cmd.Flags().Int("port", 3000, "Port to listen on")
	cmd.Flags().String("addr", "", "Address to listen on (e.g., :8080, localhost:3000)")

	return cmd
}

func runMCP(cmd *cobra.Command, args []string) error {
	if dir, _ := cmd.Flags().GetString("dir"); dir != "" {
		if err := os.Chdir(dir); err != nil {
			return fmt.Errorf("failed to change to project directory %s: %w", dir, err)
		}
	}
	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		return err
	}
	mcpServer := mcp.New(s)
	return mcpServer.Run()
}

func runHTTP(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	addr, _ := cmd.Flags().GetString("addr")

	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		return err
	}
	httpServer := tkhttp.New(s)

	// --addr takes precedence over --port
	if addr != "" {
		fmt.Printf("Starting HTTP server on %s\n", addr)
		return httpServer.Run(addr)
	}

	listenAddr := fmt.Sprintf(":%d", port)
	fmt.Printf("Starting HTTP server on %s\n", listenAddr)
	return httpServer.Run(listenAddr)
}
