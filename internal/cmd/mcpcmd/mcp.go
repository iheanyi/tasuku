// Package mcpcmd provides CLI commands for MCP server management.
package mcpcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/mcp"
	"github.com/iheanyi/tasuku/internal/store"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP server for AI tool integration",
		Long: `Model Context Protocol (MCP) server for AI tool integration.

Available subcommands:
  serve      Start the MCP server (stdio mode for AI tools)
  install    Auto-configure Tasuku MCP in Claude Code, Cursor, etc.
  uninstall  Remove Tasuku MCP configuration from AI tools
  config     Display MCP configuration JSON for manual setup

The MCP server enables AI tools like Claude Code and Cursor to
interact with Tasuku directly, allowing them to list, create,
and update tasks.

Examples:
  tk mcp install    # Auto-configure in Claude Code/Cursor
  tk mcp serve      # Start MCP server (used by AI tools)
  tk mcp config     # Show config for manual setup`,
	}

	cmd.AddCommand(serveCmd)
	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(uninstallCmd)
	cmd.AddCommand(configCmd)

	return cmd
}

// Cmd is the parent command for all MCP operations
var Cmd = newMCPCmd()

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	Long: `Start the MCP (Model Context Protocol) server in stdio mode.

This is the mode used by AI tools like Claude Code and Cursor.
The server communicates via stdin/stdout using the MCP protocol.

You typically don't run this directly - instead use 'tk mcp install'
to configure your AI tool to run it automatically.

MCP Tools Exposed:
  tk_list, tk_add, tk_start, tk_done, tk_block, tk_unblock,
  tk_learn, tk_decide, tk_context, and more.`,
	RunE: runServe,
}

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Auto-configure MCP in AI tools",
		Long: `Automatically configure the Tasuku MCP server in supported AI tools.

Supported tools:
  - Claude Code (~/.claude.json)
  - Cursor (~/.cursor/mcp.json)

The configuration will be added to existing settings without
overwriting other MCP servers or configurations.

Use --force to reinstall even if already configured.`,
		RunE: runInstall,
	}

	cmd.Flags().Bool("force", false, "Force reinstall even if already configured")

	return cmd
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove MCP configuration from AI tools",
	Long: `Remove the Tasuku MCP server configuration from AI tools.

This removes only the Tasuku configuration; other MCP servers
will be preserved.`,
	RunE: runUninstall,
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show MCP configuration snippet",
	Long: `Display the MCP configuration snippet for manual setup.

Use this if automatic installation doesn't work or you want
to configure MCP manually in your AI tool settings.`,
	RunE: runConfig,
}

func runServe(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
	mcpServer := mcp.New(s)
	return mcpServer.Run()
}

func runInstall(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	tools := getSupportedAITools()
	installedTo := []string{}
	alreadyInstalled := []string{}
	reinstalled := []string{}

	for _, tool := range tools {
		if _, err := os.Stat(tool.SettingsPath); os.IsNotExist(err) {
			continue
		}

		data, err := os.ReadFile(tool.SettingsPath)
		if err != nil {
			continue
		}

		var settings map[string]interface{}
		if err := json.Unmarshal(data, &settings); err != nil {
			if strings.Contains(tool.Name, "Cursor") {
				settings = make(map[string]interface{})
			} else {
				continue
			}
		}

		mcpServers, ok := settings[tool.MCPKey].(map[string]interface{})
		if !ok {
			mcpServers = make(map[string]interface{})
		}

		if _, exists := mcpServers["tasuku"]; exists {
			if !force {
				alreadyInstalled = append(alreadyInstalled, tool.Name)
				continue
			}
			reinstalled = append(reinstalled, tool.Name)
		}

		mcpServers["tasuku"] = map[string]interface{}{
			"command": executable,
			"args":    []string{"server", "start"},
			"type":    "stdio",
		}
		settings[tool.MCPKey] = mcpServers

		newData, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			continue
		}

		realPath := tool.SettingsPath
		if info, err := os.Lstat(tool.SettingsPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
			realPath, _ = os.Readlink(tool.SettingsPath)
		}

		if err := os.WriteFile(realPath, newData, 0644); err != nil {
			continue
		}

		isReinstall := false
		for _, r := range reinstalled {
			if r == tool.Name {
				isReinstall = true
				break
			}
		}
		if !isReinstall {
			installedTo = append(installedTo, tool.Name)
		}
	}

	if len(installedTo) > 0 {
		fmt.Printf("Tasuku MCP installed to: %s\n", strings.Join(installedTo, ", "))
	}
	if len(reinstalled) > 0 {
		fmt.Printf("Tasuku MCP reinstalled in: %s\n", strings.Join(reinstalled, ", "))
	}
	if len(alreadyInstalled) > 0 {
		fmt.Printf("Already configured in: %s\n", strings.Join(alreadyInstalled, ", "))
		fmt.Println("Use --force to reinstall anyway.")
	}

	if len(installedTo) == 0 && len(reinstalled) == 0 && len(alreadyInstalled) == 0 {
		fmt.Println("No supported AI tools found.")
		fmt.Println("Run 'tk mcp config' for manual setup instructions.")
	}

	return nil
}

func runUninstall(cmd *cobra.Command, args []string) error {
	tools := getSupportedAITools()
	removedFrom := []string{}

	for _, tool := range tools {
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

		if _, exists := mcpServers["tasuku"]; !exists {
			continue
		}

		delete(mcpServers, "tasuku")
		settings[tool.MCPKey] = mcpServers

		newData, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			continue
		}

		realPath := tool.SettingsPath
		if info, err := os.Lstat(tool.SettingsPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
			realPath, _ = os.Readlink(tool.SettingsPath)
		}

		if err := os.WriteFile(realPath, newData, 0644); err != nil {
			continue
		}

		removedFrom = append(removedFrom, tool.Name)
	}

	if len(removedFrom) > 0 {
		fmt.Printf("Tasuku MCP removed from: %s\n", strings.Join(removedFrom, ", "))
	} else {
		fmt.Println("Tasuku MCP was not configured in any AI tools.")
	}

	return nil
}

func runConfig(cmd *cobra.Command, args []string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	config := map[string]interface{}{
		"tasuku": map[string]interface{}{
			"command": executable,
			"args":    []string{"server", "start"},
			"type":    "stdio",
		},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	fmt.Println("Add this to ~/.claude.json under 'mcpServers':")
	fmt.Println()
	fmt.Println(string(data))
	return nil
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
