// Package mcpcmd provides CLI commands for MCP server management.
package mcpcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/mcp"
	"github.com/iheanyi/tasuku/internal/store"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP configuration for AI tool integration",
		Long: `Model Context Protocol (MCP) configuration for AI tool integration.

Available subcommands:
  install    Auto-configure Tasuku MCP in Claude Code, Cursor, Codex, OpenCode
  uninstall  Remove Tasuku MCP configuration from AI tools
  config     Display MCP configuration JSON for manual setup
  serve      Alias for 'tk serve mcp' (backwards compatibility)

The MCP server enables AI tools like Claude Code, Cursor, Codex, and OpenCode
to interact with Tasuku directly, allowing them to list, create,
and update tasks.

Examples:
  tk mcp install    # Auto-configure in Claude Code/Cursor
  tk mcp config     # Show config for manual setup
  tk serve mcp      # Start MCP server (preferred)`,
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
	Short: "Start the MCP server (alias for 'tk serve mcp')",
	Long: `Start the MCP (Model Context Protocol) server in stdio mode.

This is an alias for 'tk serve mcp'. Prefer using 'tk serve mcp' directly.

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
  - Claude Code (~/.claude.json or ./.claude.json with --local)
  - Cursor (~/.cursor/mcp.json or ./.cursor/mcp.json with --local)
  - Codex (~/.codex/config.toml)
  - OpenCode (~/.config/opencode/opencode.json or ./opencode.json with --local)
  - Gemini (~/.gemini/mcp.json or .gemini/mcp.json with --local)

Detection signals (local install):
  - Claude Code: .claude/ directory OR CLAUDE.md file
  - Cursor: .cursorrules file OR .cursor/ directory
  - OpenCode: opencode.json file
  - Gemini: .gemini/ directory OR GEMINI.md file

The configuration will be added to existing settings without
overwriting other MCP servers or configurations.

Use --local to install to project-level config instead of global.
Use --force to reinstall even if already configured.
Use --tool to target a specific tool (claude, cursor, codex, opencode, gemini).`,
		RunE: runInstall,
	}

	cmd.Flags().Bool("force", false, "Force reinstall even if already configured")
	cmd.Flags().Bool("local", false, "Install to project-level config")
	cmd.Flags().String("tool", "", "Target specific tool: claude, cursor, codex, opencode, gemini")

	return cmd
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove MCP configuration from AI tools",
	Long: `Remove the Tasuku MCP server configuration from AI tools.

This removes only the Tasuku configuration; other MCP servers
will be preserved.

Use --local to remove from project-level .claude.json.`,
	RunE: runUninstall,
}

func init() {
	uninstallCmd.Flags().Bool("local", false, "Remove from project-level .claude.json")
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
	local, _ := cmd.Flags().GetBool("local")
	toolFilter, _ := cmd.Flags().GetString("tool")

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	tools := getSupportedAITools(local)

	// Filter tools if --tool is specified
	if toolFilter != "" {
		toolFilter = strings.ToLower(toolFilter)
		filtered := []AITool{}
		for _, tool := range tools {
			toolNameLower := strings.ToLower(tool.Name)
			if strings.Contains(toolNameLower, toolFilter) {
				filtered = append(filtered, tool)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("unknown tool: %s (valid: claude, cursor, codex, opencode, gemini)", toolFilter)
		}
		tools = filtered
	}

	installedTo := []string{}
	alreadyInstalled := []string{}
	reinstalled := []string{}

	for _, tool := range tools {
		// Check if tool is installed (via any DetectPath) or settings file exists
		toolInstalled := false
		for _, detectPath := range tool.DetectPaths {
			if _, err := os.Stat(detectPath); err == nil {
				toolInstalled = true
				break
			}
		}
		settingsExist := false
		if _, err := os.Stat(tool.SettingsPath); err == nil {
			settingsExist = true
		}

		if !settingsExist && !toolInstalled {
			// Skip if neither settings file nor tool directory exists
			continue
		}

		// Handle based on config format
		var installed, wasReinstall bool
		var err error

		switch tool.Format {
		case FormatTOML:
			installed, wasReinstall, err = installToTOML(tool, executable, force, settingsExist)
		default: // FormatJSON
			installed, wasReinstall, err = installToJSON(tool, executable, force, settingsExist)
		}

		if err != nil {
			continue
		}

		if installed {
			if wasReinstall {
				reinstalled = append(reinstalled, tool.Name)
			} else {
				installedTo = append(installedTo, tool.Name)
			}
		} else if !installed && !wasReinstall {
			// Already installed
			alreadyInstalled = append(alreadyInstalled, tool.Name)
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
	local, _ := cmd.Flags().GetBool("local")
	tools := getSupportedAITools(local)
	removedFrom := []string{}

	for _, tool := range tools {
		if _, err := os.Stat(tool.SettingsPath); os.IsNotExist(err) {
			continue
		}

		var removed bool
		var err error

		switch tool.Format {
		case FormatTOML:
			removed, err = uninstallFromTOML(tool)
		default: // FormatJSON
			removed, err = uninstallFromJSON(tool)
		}

		if err != nil {
			continue
		}
		if removed {
			removedFrom = append(removedFrom, tool.Name)
		}
	}

	if len(removedFrom) > 0 {
		fmt.Printf("Tasuku MCP removed from: %s\n", strings.Join(removedFrom, ", "))
	} else {
		fmt.Println("Tasuku MCP was not configured in any AI tools.")
	}

	return nil
}

// uninstallFromJSON removes tasuku from a JSON config file
func uninstallFromJSON(tool AITool) (bool, error) {
	data, err := os.ReadFile(tool.SettingsPath)
	if err != nil {
		return false, err
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return false, err
	}

	mcpServers, ok := settings[tool.MCPKey].(map[string]interface{})
	if !ok {
		return false, nil
	}

	if _, exists := mcpServers[tool.MCPEntryKey]; !exists {
		return false, nil
	}

	delete(mcpServers, tool.MCPEntryKey)
	settings[tool.MCPKey] = mcpServers

	newData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return false, err
	}

	realPath := tool.SettingsPath
	if info, err := os.Lstat(tool.SettingsPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		realPath, _ = os.Readlink(tool.SettingsPath)
	}

	if err := os.WriteFile(realPath, newData, 0644); err != nil {
		return false, err
	}

	return true, nil
}

// uninstallFromTOML removes tasuku from a TOML config file
func uninstallFromTOML(tool AITool) (bool, error) {
	data, err := os.ReadFile(tool.SettingsPath)
	if err != nil {
		return false, err
	}

	config := make(map[string]interface{})
	if _, err := toml.Decode(string(data), &config); err != nil {
		return false, err
	}

	mcpServersRaw, ok := config["mcp_servers"]
	if !ok {
		return false, nil
	}

	mcpServers, ok := mcpServersRaw.(map[string]interface{})
	if !ok {
		return false, nil
	}

	if _, exists := mcpServers[tool.MCPEntryKey]; !exists {
		return false, nil
	}

	delete(mcpServers, tool.MCPEntryKey)
	config["mcp_servers"] = mcpServers

	realPath := tool.SettingsPath
	if info, err := os.Lstat(tool.SettingsPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		realPath, _ = os.Readlink(tool.SettingsPath)
	}

	f, err := os.Create(realPath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(config); err != nil {
		return false, err
	}

	return true, nil
}

func runConfig(cmd *cobra.Command, args []string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	config := map[string]interface{}{
		"tasuku": map[string]interface{}{
			"command": executable,
			"args":    []string{"serve", "mcp"},
			"type":    "stdio",
		},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	fmt.Println("Add this to ~/.claude.json under 'mcpServers':")
	fmt.Println()
	fmt.Println(string(data))
	return nil
}

// ConfigFormat represents the configuration file format
type ConfigFormat string

const (
	FormatJSON ConfigFormat = "json"
	FormatTOML ConfigFormat = "toml"
)

// AITool represents a supported AI tool configuration
type AITool struct {
	Name         string
	SettingsPath string
	MCPKey       string       // JSON key for mcpServers (e.g., "mcpServers" or "mcp")
	DetectPaths  []string     // Paths to check if tool is installed (any match = installed)
	Format       ConfigFormat // Config file format (json or toml)
	MCPEntryKey  string       // Key for the tasuku entry (default "tasuku")
}

// installToJSON installs the MCP config to a JSON settings file.
// Returns (installed, wasReinstall, error)
func installToJSON(tool AITool, executable string, force, settingsExist bool) (bool, bool, error) {
	var settings map[string]interface{}

	if settingsExist {
		data, err := os.ReadFile(tool.SettingsPath)
		if err != nil {
			return false, false, err
		}
		if err := json.Unmarshal(data, &settings); err != nil {
			// Reset to empty if JSON is invalid
			settings = make(map[string]interface{})
		}
	} else {
		settings = make(map[string]interface{})
	}

	mcpServers, ok := settings[tool.MCPKey].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}

	wasReinstall := false
	if _, exists := mcpServers[tool.MCPEntryKey]; exists {
		if !force {
			return false, false, nil // Already installed
		}
		wasReinstall = true
	}

	// Build MCP entry based on tool type
	mcpEntry := buildMCPEntry(tool, executable)
	mcpServers[tool.MCPEntryKey] = mcpEntry
	settings[tool.MCPKey] = mcpServers

	newData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return false, false, err
	}

	realPath := tool.SettingsPath
	if info, err := os.Lstat(tool.SettingsPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		realPath, _ = os.Readlink(tool.SettingsPath)
	}

	// Create parent directory if needed
	if dir := filepath.Dir(realPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return false, false, err
		}
	}

	if err := os.WriteFile(realPath, newData, 0644); err != nil {
		return false, false, err
	}

	return true, wasReinstall, nil
}

// CodexConfig represents the TOML config structure for Codex
type CodexConfig struct {
	MCPServers map[string]CodexMCPServer `toml:"mcp_servers"`
	// Preserve other fields
	Other map[string]interface{} `toml:"-"`
}

// CodexMCPServer represents an MCP server config in Codex's TOML format
type CodexMCPServer struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args,omitempty"`
}

// installToTOML installs the MCP config to a TOML settings file (Codex).
// Returns (installed, wasReinstall, error)
func installToTOML(tool AITool, executable string, force, settingsExist bool) (bool, bool, error) {
	var config map[string]interface{}

	if settingsExist {
		data, err := os.ReadFile(tool.SettingsPath)
		if err != nil {
			return false, false, err
		}
		config = make(map[string]interface{})
		if _, err := toml.Decode(string(data), &config); err != nil {
			// Reset to empty if TOML is invalid
			config = make(map[string]interface{})
		}
	} else {
		config = make(map[string]interface{})
	}

	// Get or create mcp_servers section
	mcpServersRaw, ok := config["mcp_servers"]
	var mcpServers map[string]interface{}
	if ok {
		mcpServers, ok = mcpServersRaw.(map[string]interface{})
		if !ok {
			mcpServers = make(map[string]interface{})
		}
	} else {
		mcpServers = make(map[string]interface{})
	}

	wasReinstall := false
	if _, exists := mcpServers[tool.MCPEntryKey]; exists {
		if !force {
			return false, false, nil // Already installed
		}
		wasReinstall = true
	}

	// Add tasuku MCP server for Codex (TOML format)
	mcpServers[tool.MCPEntryKey] = map[string]interface{}{
		"command": executable,
		"args":    []interface{}{"serve", "mcp"},
	}
	config["mcp_servers"] = mcpServers

	// Write TOML
	realPath := tool.SettingsPath
	if info, err := os.Lstat(tool.SettingsPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		realPath, _ = os.Readlink(tool.SettingsPath)
	}

	// Create parent directory if needed
	if dir := filepath.Dir(realPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return false, false, err
		}
	}

	f, err := os.Create(realPath)
	if err != nil {
		return false, false, err
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(config); err != nil {
		return false, false, err
	}

	return true, wasReinstall, nil
}

// buildMCPEntry creates the appropriate MCP entry structure for a tool
func buildMCPEntry(tool AITool, executable string) map[string]interface{} {
	// OpenCode uses "type": "local" and "command" as array
	if strings.Contains(strings.ToLower(tool.Name), "opencode") {
		return map[string]interface{}{
			"type":    "local",
			"command": []string{executable, "serve", "mcp"},
		}
	}
	// Claude Code, Cursor use "type": "stdio"
	return map[string]interface{}{
		"command": executable,
		"args":    []string{"serve", "mcp"},
		"type":    "stdio",
	}
}

func getSupportedAITools(local bool) []AITool {
	if local {
		// Local installation targets project-level config files
		// Detect Claude Code by .claude/ directory OR CLAUDE.md file
		// Detect Cursor by .cursorrules file OR .cursor/ directory
		// Detect OpenCode by opencode.json file
		return []AITool{
			{"Claude Code (project)", ".claude.json", "mcpServers", []string{".claude", "CLAUDE.md"}, FormatJSON, "tasuku"},
			{"Cursor (project)", ".cursor/mcp.json", "mcpServers", []string{".cursorrules", ".cursor"}, FormatJSON, "tasuku"},
			{"OpenCode (project)", "opencode.json", "mcp", []string{"opencode.json"}, FormatJSON, "tasuku"},
			{"Gemini (project)", ".gemini/mcp.json", "mcpServers", []string{".gemini", "GEMINI.md"}, FormatJSON, "tasuku"},
		}
	}

	// Global installation targets user-level config files
	home, _ := os.UserHomeDir()
	configDir := home + "/.config"
	// XDG_CONFIG_HOME support
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		configDir = xdg
	}

	return []AITool{
		// Claude Code: detect by ~/.claude/ directory, config at ~/.claude.json
		{"Claude Code", home + "/.claude.json", "mcpServers", []string{home + "/.claude"}, FormatJSON, "tasuku"},
		// Cursor: detect by ~/.cursor/ directory
		{"Cursor", home + "/.cursor/mcp.json", "mcpServers", []string{home + "/.cursor"}, FormatJSON, "tasuku"},
		{"Cursor (alt)", home + "/Library/Application Support/Cursor/User/globalStorage/mcp.json", "mcpServers", []string{home + "/Library/Application Support/Cursor"}, FormatJSON, "tasuku"},
		// Codex: config at ~/.codex/config.toml (TOML format)
		{"Codex", home + "/.codex/config.toml", "mcp_servers", []string{home + "/.codex"}, FormatTOML, "tasuku"},
		// OpenCode: config at ~/.config/opencode/opencode.json
		{"OpenCode", configDir + "/opencode/opencode.json", "mcp", []string{configDir + "/opencode"}, FormatJSON, "tasuku"},
		// Gemini: config at ~/.gemini/mcp.json
		{"Gemini", home + "/.gemini/mcp.json", "mcpServers", []string{home + "/.gemini"}, FormatJSON, "tasuku"},
	}
}
