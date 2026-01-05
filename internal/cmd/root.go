// Package cmd provides the CLI commands for tasuku.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/cmd/config"
	contextcmd "github.com/iheanyi/tasuku/internal/cmd/context"
	"github.com/iheanyi/tasuku/internal/cmd/decision"
	"github.com/iheanyi/tasuku/internal/cmd/hooks"
	"github.com/iheanyi/tasuku/internal/cmd/learning"
	"github.com/iheanyi/tasuku/internal/cmd/mcpcmd"
	"github.com/iheanyi/tasuku/internal/cmd/migrate"
	"github.com/iheanyi/tasuku/internal/cmd/note"
	"github.com/iheanyi/tasuku/internal/cmd/pr"
	"github.com/iheanyi/tasuku/internal/cmd/server"
	"github.com/iheanyi/tasuku/internal/cmd/task"
	"github.com/iheanyi/tasuku/internal/cmd/ui"
	"github.com/iheanyi/tasuku/internal/mcp"
	"github.com/iheanyi/tasuku/internal/store"
)

const Version = "0.3.0"

// RootCmd is the base command for tk
var RootCmd = &cobra.Command{
	Use:   "tk",
	Short: "Tasuku - agent-first task management",
	Long: `tk is an agent-first task management system designed for AI agents
working on codebases.

Design Principles:
  - Pull over push: Agents query when needed, no constant injections
  - Parallel-safe: File locking for multiple simultaneous agents
  - Minimal context: Only load what's needed for the current task
  - Human-readable: JSON file that can be edited by hand

Getting Started:
  tk init                  # Create .tasuku/ directory
  tk task add "My task"    # Add a task
  tk task list             # View all tasks
  tk task start <id>       # Begin working on a task
  tk task done <id>        # Mark task complete

AI Tool Integration:
  tk mcp install           # Auto-configure MCP for Claude Code/Cursor
  tk mcp serve             # Start MCP server (for AI tools)

For full documentation: https://github.com/iheanyi/tasuku`,
	Version: Version,
}

func init() {
	// Global flags
	RootCmd.PersistentFlags().StringVarP(&config.OutputFormat, "format", "f", "table", "Output format: table, json, yaml")

	// Register all subcommands
	RootCmd.AddCommand(task.Cmd)
	RootCmd.AddCommand(learning.Cmd)
	RootCmd.AddCommand(decision.Cmd)
	RootCmd.AddCommand(note.Cmd)
	RootCmd.AddCommand(contextcmd.Cmd)
	RootCmd.AddCommand(server.Cmd)
	RootCmd.AddCommand(mcpcmd.Cmd)
	RootCmd.AddCommand(hooks.Cmd)
	RootCmd.AddCommand(migrate.Cmd)
	RootCmd.AddCommand(pr.Cmd)
	RootCmd.AddCommand(ui.Cmd)

	// Root-level commands
	RootCmd.AddCommand(initCmd)
	RootCmd.AddCommand(doctorCmd)

	// Deprecated commands for backward compatibility
	RootCmd.AddCommand(validateCmd)
	RootCmd.AddCommand(hooks.DeprecatedHookCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Tasuku in current directory",
	Long: `Initialize a new Tasuku project in the current directory.

Creates a .tasuku/ directory with:
  - tasks/     Individual task JSON files (one per task)
  - archive/   Completed tasks that have been archived
  - context/   Learnings and decisions

Benefits:
  - One file per task = cleaner git diffs, fewer merge conflicts
  - Human-readable JSON, can be edited directly
  - Safe for multiple agents working in parallel

If you have a legacy .tasuku.json file, use 'tk migrate v3' to upgrade.

Examples:
  tk init                    # Create .tasuku/ directory
  tk init && tk task add "Setup"  # Initialize and add first task`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if already initialized
		storageType := store.DetectStorageType(".")
		if storageType != store.StorageTypeNone {
			if storageType == store.StorageTypeDir {
				return fmt.Errorf(".tasuku/ directory already exists")
			}
			return fmt.Errorf(".tasuku.json already exists - run 'tk migrate v3' to upgrade")
		}

		s := store.NewDirStore(store.DirName)
		if err := s.Init(); err != nil {
			return err
		}
		fmt.Println("Created .tasuku/ directory")
		fmt.Println("  tasks/    - Your task files")
		fmt.Println("  archive/  - Archived completed tasks")
		fmt.Println("  context/  - Learnings and decisions")
		fmt.Println()
		fmt.Println("Tip: Commit .tasuku/ to git so tasks travel with your code.")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  tk task add \"Your first task\"")
		fmt.Println("  tk hooks install              # Optional: git hooks")
		fmt.Println("  tk mcp install                # Optional: AI tool integration (Claude Code, Cursor)")
		return nil
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose Tasuku setup and MCP configuration",
	Long: `Check your Tasuku installation and MCP configuration for common issues.

This command verifies:
  - tk binary is accessible and shows its location
  - Tasuku storage exists (.tasuku/ directory or .tasuku.json file)
  - MCP is configured in Claude Code, Cursor, and other AI tools
  - The configured binary path matches the current tk installation
  - The MCP server can start and respond to requests

Run this when Tasuku tools aren't appearing in your AI assistant.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
	},
}

var validateCmd = &cobra.Command{
	Use:        "validate",
	Hidden:     true,
	Deprecated: "use 'tk context validate' instead",
	Short:      "Validate Tasuku storage",
	Long: `Validate the Tasuku storage for correctness.

Checks performed:
- Version is supported
- All tasks have non-empty descriptions
- All tasks have valid statuses
- No circular dependencies in blocked_by relationships

Examples:
  tk validate`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return contextcmd.RunValidate(cmd, args)
	},
}

func runDoctor() error {
	fmt.Println("Tasuku Doctor")
	fmt.Println("=============")
	fmt.Println()

	hasErrors := false

	// 1. Check tk binary
	executable, err := os.Executable()
	if err != nil {
		fmt.Println("✗ Could not determine tk binary location")
		hasErrors = true
	} else {
		fmt.Printf("✓ tk binary: %s\n", executable)
	}

	// 2. Check Tasuku storage
	s := store.DefaultStorageWithWarning()
	tasukuPath := s.Path()
	if !s.Exists() {
		fmt.Printf("✗ No Tasuku storage found (searched from %s)\n", mustGetwd())
		fmt.Println("  Run 'tk init' to create .tasuku/ directory")
		hasErrors = true
	} else {
		fmt.Printf("✓ Tasuku storage: %s\n", tasukuPath)
	}

	fmt.Println()
	fmt.Println("MCP Configuration")
	fmt.Println("-----------------")

	// 3. Check AI tool configurations
	tools := getSupportedAITools()
	configuredTools := 0
	mismatchedPaths := []string{}

	for _, tool := range tools {
		// Check if settings file exists
		if _, err := os.Stat(tool.SettingsPath); os.IsNotExist(err) {
			continue
		}

		data, err := os.ReadFile(tool.SettingsPath)
		if err != nil {
			continue
		}

		var settings map[string]interface{}
		if err := json.Unmarshal(data, &settings); err != nil {
			continue
		}

		mcpServers, ok := settings[tool.MCPKey].(map[string]interface{})
		if !ok {
			continue
		}

		tasukuConfig, exists := mcpServers["tasuku"].(map[string]interface{})
		if !exists {
			fmt.Printf("✗ %s: MCP not configured\n", tool.Name)
			fmt.Printf("  Run 'tk mcp install' to configure\n")
			hasErrors = true
			continue
		}

		configuredPath, _ := tasukuConfig["command"].(string)
		configuredArgs, _ := tasukuConfig["args"].([]interface{})

		// Check if path matches current executable
		if configuredPath != executable {
			fmt.Printf("⚠ %s: configured but path mismatch\n", tool.Name)
			fmt.Printf("  Configured: %s\n", configuredPath)
			fmt.Printf("  Current:    %s\n", executable)
			mismatchedPaths = append(mismatchedPaths, tool.Name)
		} else {
			argsStr := ""
			for _, arg := range configuredArgs {
				if s, ok := arg.(string); ok {
					argsStr += s + " "
				}
			}
			fmt.Printf("✓ %s: configured (%s %s)\n", tool.Name, filepath.Base(configuredPath), strings.TrimSpace(argsStr))
		}
		configuredTools++
	}

	if configuredTools == 0 {
		fmt.Println("✗ No AI tools have Tasuku MCP configured")
		fmt.Println("  Run 'tk mcp install' to auto-configure")
		hasErrors = true
	}

	// 4. Test MCP server
	fmt.Println()
	fmt.Println("MCP Server Test")
	fmt.Println("---------------")

	// Quick test: can we create a server and get tools?
	if _, err := os.Stat(tasukuPath); err == nil {
		mcpServer := mcp.New(s)
		mcpTools := mcpServer.Tools()
		fmt.Printf("✓ MCP server responds with %d tools\n", len(mcpTools))

		// List a few tools
		if len(mcpTools) > 0 {
			toolNames := []string{}
			for _, t := range mcpTools {
				toolNames = append(toolNames, t.Name)
			}
			if len(toolNames) > 5 {
				toolNames = toolNames[:5]
				fmt.Printf("  Tools: %s, ... (+%d more)\n", strings.Join(toolNames, ", "), len(mcpTools)-5)
			} else {
				fmt.Printf("  Tools: %s\n", strings.Join(toolNames, ", "))
			}
		}
	} else {
		fmt.Println("⚠ Cannot test MCP server (no Tasuku storage)")
	}

	// 5. Check CLI/MCP parity
	fmt.Println()
	fmt.Println("CLI/MCP Parity")
	fmt.Println("--------------")

	if _, err := os.Stat(tasukuPath); err == nil {
		mcpServer := mcp.New(s)
		mcpTools := mcpServer.Tools()

		// Build set of MCP tool names
		mcpToolSet := make(map[string]bool)
		for _, t := range mcpTools {
			mcpToolSet[t.Name] = true
		}

		// Define expected MCP tools for CLI commands
		cliToMCP := map[string][]string{
			"task list":     {"tk_list"},
			"task add":      {"tk_add"},
			"task show":     {"tk_show"},
			"task start":    {"tk_start"},
			"task done":     {"tk_done"},
			"task block":    {"tk_block"},
			"task unblock":  {"tk_unblock"},
			"task pause":    {"tk_pause"},
			"task find":     {"tk_find"},
			"task priority": {"tk_priority"},
			"task delete":   {"tk_delete"},
			"task edit":     {"tk_edit"},
			"task owner":    {"tk_owner"},
			"task claim":    {"tk_claim"},
			"task release":  {"tk_release"},
			"task tag":      {"tk_tag_add", "tk_tag_remove"},
			"task field":    {"tk_field_set", "tk_field_remove"},
			"task timer":    {"tk_timer_start", "tk_timer_stop", "tk_timer_status"},
			"task archive":  {"tk_archive", "tk_archive_restore", "tk_archive_list"},
			"context learn": {"tk_learn"},
			"context decide":{"tk_decide"},
			"context note":  {"tk_note"},
			"context show":  {"tk_context"},
		}

		missingTools := []string{}
		for cli, expectedTools := range cliToMCP {
			for _, tool := range expectedTools {
				if !mcpToolSet[tool] {
					missingTools = append(missingTools, fmt.Sprintf("%s (missing %s)", cli, tool))
				}
			}
		}

		if len(missingTools) == 0 {
			fmt.Printf("✓ All %d CLI commands have corresponding MCP tools\n", len(cliToMCP))
		} else {
			fmt.Printf("✗ %d CLI commands missing MCP tools:\n", len(missingTools))
			for _, m := range missingTools {
				fmt.Printf("  - %s\n", m)
			}
			hasErrors = true
		}
	} else {
		fmt.Println("⚠ Cannot check parity (no Tasuku storage)")
	}

	// Summary
	fmt.Println()
	if hasErrors {
		fmt.Println("Issues found. See recommendations above.")
		if configuredTools > 0 {
			fmt.Println()
			fmt.Println("If MCP is configured but tools aren't visible:")
			fmt.Println("  1. Restart your AI tool (Claude Code, Cursor, etc.)")
			fmt.Println("  2. Run '/mcp' in Claude Code to check MCP status")
		}
		return nil
	}

	if len(mismatchedPaths) > 0 {
		fmt.Println("Configuration path mismatch detected.")
		fmt.Println("Run 'tk mcp install' to update configuration.")
		return nil
	}

	fmt.Println("Everything looks good!")
	fmt.Println()
	fmt.Println("If tools still aren't visible in your AI assistant:")
	fmt.Println("  1. Restart your AI tool (Claude Code, Cursor, etc.)")
	fmt.Println("  2. Run '/mcp' in Claude Code to check MCP status")

	return nil
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

// AITool represents a supported AI tool configuration
type AITool struct {
	Name         string
	SettingsPath string
	MCPKey       string
}

func getSupportedAITools() []AITool {
	home, _ := os.UserHomeDir()
	return []AITool{
		{"Claude Code", home + "/.claude.json", "mcpServers"},
		{"Cursor", home + "/.cursor/mcp.json", "mcpServers"},
		{"Cursor (alt)", home + "/Library/Application Support/Cursor/User/globalStorage/mcp.json", "mcpServers"},
	}
}
