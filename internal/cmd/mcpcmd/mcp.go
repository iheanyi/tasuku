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
  install    Auto-configure Tasuku MCP in Claude Code, Cursor, Codex, Copilot CLI, OpenCode, Amp
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

	cmd.AddCommand(newServeCmd())
	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newUninstallCmd())
	cmd.AddCommand(newConfigCmd())

	return cmd
}

// Cmd is the parent command for all MCP operations
var Cmd = newMCPCmd()

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
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

	cmd.Flags().String("dir", "", "Project directory (overrides cwd for storage detection)")

	return cmd
}

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Auto-configure MCP in AI tools",
		Long: `Automatically configure the Tasuku MCP server in supported AI tools.

Supported tools:
  - Claude Code (~/.claude.json or ./.claude.json with --local)
  - Copilot CLI (~/.copilot/mcp-config.json or ./.copilot/mcp-config.json with --local)
  - Cursor (project-level ./.cursor/mcp.json; --tool cursor auto-uses --local)
  - Codex (~/.codex/config.toml)
  - OpenCode (~/.config/opencode/opencode.json or ./opencode.json with --local)
  - Gemini (~/.gemini/mcp.json or .gemini/mcp.json with --local)
  - Amp (~/.config/amp/settings.json or ./.amp/settings.json with --local)

Detection signals (local install):
  - Claude Code: .claude/ directory OR CLAUDE.md file
  - Copilot CLI: .copilot/ directory
  - Cursor: .cursorrules file OR .cursor/ directory
  - OpenCode: opencode.json file
  - Gemini: .gemini/ directory OR GEMINI.md file
  - Amp: .amp/ directory OR AGENTS.md file

The configuration will be added to existing settings without
overwriting other MCP servers or configurations.

Use --local to install to project-level config instead of global.
Use --force to reinstall even if already configured.
Use --tool to target a specific tool (claude, copilot, cursor, codex, opencode, gemini, amp).`,
		RunE: runInstall,
	}

	cmd.Flags().Bool("force", false, "Force reinstall even if already configured")
	cmd.Flags().Bool("local", false, "Install to project-level config")
	cmd.Flags().String("tool", "", "Target specific tool: claude, cursor, codex, opencode, gemini, amp")

	return cmd
}

func newUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove MCP configuration from AI tools",
		Long: `Remove the Tasuku MCP server configuration from AI tools.

This removes only the Tasuku configuration; other MCP servers
will be preserved.

Use --local to remove from project-level .claude.json.`,
		RunE: runUninstall,
	}

	cmd.Flags().Bool("local", false, "Remove from project-level .claude.json")

	return cmd
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show MCP configuration snippet",
		Long: `Display the MCP configuration snippet for manual setup.

Use this if automatic installation doesn't work or you want
to configure MCP manually in your AI tool settings.`,
		RunE: runConfig,
	}

	return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
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

func runInstall(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	local, _ := cmd.Flags().GetBool("local")
	toolFilter, _ := cmd.Flags().GetString("tool")
	explicitToolSelection := toolFilter != ""

	// Cursor requires project-level config (global MCP has no project context).
	// Auto-promote to --local when --tool cursor is used without --local.
	if !local && strings.ToLower(toolFilter) == "cursor" {
		local = true
		fmt.Println("Note: Cursor requires project-level MCP config (using --local automatically).")
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// For local installs, capture the project directory so we can embed
	// --dir in the generated config. This ensures the MCP server can find
	// .tasuku/ even when the AI tool spawns it from a different cwd.
	var projectDir string
	if local {
		projectDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get project directory: %w", err)
		}
		// Ensure it's absolute
		projectDir, err = filepath.Abs(projectDir)
		if err != nil {
			return fmt.Errorf("failed to resolve project directory: %w", err)
		}
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
			return fmt.Errorf("unknown tool: %s (valid: claude, copilot, cursor, codex, opencode, gemini, amp)", toolFilter)
		}
		tools = filtered
	}

	installedTo := []string{}
	alreadyInstalled := []string{}
	reinstalled := []string{}
	cursorLocalConfigured := false

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

		// When a tool is explicitly selected (via --tool), install even if
		// detection markers don't exist yet (we'll create config directories/files).
		if !explicitToolSelection && !settingsExist && !toolInstalled {
			// Skip if neither settings file nor tool directory exists
			continue
		}

		// Handle based on config format
		var installed, wasReinstall bool
		var err error

		switch tool.Format {
		case FormatTOML:
			installed, wasReinstall, err = installToTOML(tool, executable, projectDir, force, settingsExist)
		default: // FormatJSON
			installed, wasReinstall, err = installToJSON(tool, executable, projectDir, force, settingsExist)
		}

		if err != nil {
			continue
		}

		if local && strings.Contains(strings.ToLower(tool.Name), "cursor") {
			cursorLocalConfigured = true
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

	if cursorLocalConfigured {
		removedLegacy, cleanupWarnings := removeLegacyCursorGlobalConfigs()
		if len(removedLegacy) > 0 {
			fmt.Printf("Removed legacy global Cursor configs: %s\n", strings.Join(removedLegacy, ", "))
			fmt.Println("This prevents duplicate Tasuku MCP servers in Cursor.")
		}
		for _, warning := range cleanupWarnings {
			fmt.Printf("Warning: %s\n", warning)
		}
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
	if !local {
		// Mirror install behavior: --tool cursor auto-installs to project config.
		// Default uninstall should also try removing project-level Cursor config.
		for _, localTool := range getSupportedAITools(true) {
			if strings.Contains(strings.ToLower(localTool.Name), "cursor") {
				tools = append(tools, localTool)
				break
			}
		}

		// Also clean up legacy Cursor global locations from older installs.
		tools = append(tools, getLegacyCursorGlobalTools()...)
	}
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

	// Include --dir with the current directory so the MCP server
	// can find .tasuku/ regardless of where the AI tool spawns it.
	projectDir, _ := filepath.Abs(".")

	config := map[string]interface{}{
		"tasuku": map[string]interface{}{
			"command": executable,
			"args":    []string{"serve", "mcp", "--dir", projectDir},
			"type":    "stdio",
		},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	fmt.Println("Add this to your AI tool's MCP config (e.g. ~/.claude.json under 'mcpServers'):")
	fmt.Println()
	fmt.Println(string(data))
	fmt.Println()
	fmt.Printf("The --dir flag ensures the MCP server can find .tasuku/ in %s.\n", projectDir)
	fmt.Println("You can also set TASUKU_PROJECT_DIR environment variable instead.")
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
// projectDir is included as --dir in args when non-empty (local installs).
// Returns (installed, wasReinstall, error)
func installToJSON(tool AITool, executable, projectDir string, force, settingsExist bool) (bool, bool, error) {
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
	mcpEntry := buildMCPEntry(tool, executable, projectDir)
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
// projectDir is included as --dir in args when non-empty (local installs).
// Returns (installed, wasReinstall, error)
func installToTOML(tool AITool, executable, projectDir string, force, settingsExist bool) (bool, bool, error) {
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
	args := []interface{}{"serve", "mcp"}
	if projectDir != "" {
		args = append(args, "--dir", projectDir)
	}
	mcpServers[tool.MCPEntryKey] = map[string]interface{}{
		"command": executable,
		"args":    args,
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

// buildMCPEntry creates the appropriate MCP entry structure for a tool.
// If projectDir is non-empty, --dir is added to args so the MCP server
// can find .tasuku/ even when the AI tool spawns it from a different cwd.
func buildMCPEntry(tool AITool, executable, projectDir string) map[string]interface{} {
	nameLower := strings.ToLower(tool.Name)

	args := []string{"serve", "mcp"}
	if projectDir != "" {
		args = append(args, "--dir", projectDir)
	}

	// OpenCode uses "type": "local" and "command" as array
	if strings.Contains(nameLower, "opencode") {
		cmdArray := append([]string{executable}, args...)
		return map[string]interface{}{
			"type":    "local",
			"command": cmdArray,
		}
	}
	// Copilot CLI uses "type": "local" with separate command and args
	if strings.Contains(nameLower, "copilot") {
		return map[string]interface{}{
			"type":    "local",
			"command": executable,
			"args":    args,
		}
	}
	// Amp uses command/args without type field
	if strings.Contains(nameLower, "amp") {
		return map[string]interface{}{
			"command": executable,
			"args":    args,
		}
	}
	// Cursor uses "type": "stdio" with "cwd" for reliable project directory
	if strings.Contains(nameLower, "cursor") && projectDir != "" {
		return map[string]interface{}{
			"command": executable,
			"args":    args,
			"type":    "stdio",
			"cwd":     projectDir,
		}
	}
	// Claude Code, Gemini use "type": "stdio"
	return map[string]interface{}{
		"command": executable,
		"args":    args,
		"type":    "stdio",
	}
}

func getSupportedAITools(local bool) []AITool {
	if local {
		// Local installation targets project-level config files
		// Detect Claude Code by .claude/ directory OR CLAUDE.md file
		// Detect Copilot CLI by .copilot/ directory
		// Detect Cursor by .cursorrules file OR .cursor/ directory
		// Detect OpenCode by opencode.json file
		return []AITool{
			{"Claude Code (project)", ".claude.json", "mcpServers", []string{".claude", "CLAUDE.md"}, FormatJSON, "tasuku"},
			{"Copilot CLI (project)", ".copilot/mcp-config.json", "mcpServers", []string{".copilot"}, FormatJSON, "tasuku"},
			{"Cursor (project)", ".cursor/mcp.json", "mcpServers", []string{".cursorrules", ".cursor"}, FormatJSON, "tasuku"},
			{"OpenCode (project)", "opencode.json", "mcp", []string{"opencode.json"}, FormatJSON, "tasuku"},
			{"Gemini (project)", ".gemini/mcp.json", "mcpServers", []string{".gemini", "GEMINI.md"}, FormatJSON, "tasuku"},
			{"Amp (project)", ".amp/settings.json", "amp.mcpServers", []string{".amp", "AGENTS.md"}, FormatJSON, "tasuku"},
		}
	}

	// Global installation targets user-level config files
	home, _ := os.UserHomeDir()
	configDir := home + "/.config"
	// XDG_CONFIG_HOME support
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		configDir = xdg
	}

	// Copilot CLI respects XDG_CONFIG_HOME for its config location
	copilotConfigDir := home + "/.copilot"
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		copilotConfigDir = xdg + "/copilot"
	}

	return []AITool{
		// Claude Code: detect by ~/.claude/ directory, config at ~/.claude.json
		{"Claude Code", home + "/.claude.json", "mcpServers", []string{home + "/.claude"}, FormatJSON, "tasuku"},
		// Copilot CLI: detect by ~/.copilot/ directory, config at ~/.copilot/mcp-config.json
		{"Copilot CLI", copilotConfigDir + "/mcp-config.json", "mcpServers", []string{copilotConfigDir}, FormatJSON, "tasuku"},
		// Cursor: excluded from global installs — requires project-level config
		// because Cursor doesn't set cwd to the workspace. Use --local instead.
		// Codex: config at ~/.codex/config.toml (TOML format)
		{"Codex", home + "/.codex/config.toml", "mcp_servers", []string{home + "/.codex"}, FormatTOML, "tasuku"},
		// OpenCode: config at ~/.config/opencode/opencode.json
		{"OpenCode", configDir + "/opencode/opencode.json", "mcp", []string{configDir + "/opencode"}, FormatJSON, "tasuku"},
		// Gemini: config at ~/.gemini/mcp.json
		{"Gemini", home + "/.gemini/mcp.json", "mcpServers", []string{home + "/.gemini"}, FormatJSON, "tasuku"},
		// Amp: config at ~/.config/amp/settings.json
		{"Amp", configDir + "/amp/settings.json", "amp.mcpServers", []string{configDir + "/amp"}, FormatJSON, "tasuku"},
	}
}

func getLegacyCursorGlobalTools() []AITool {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	return []AITool{
		// Historical/global locations from older Cursor integrations.
		{"Cursor (legacy global)", filepath.Join(home, ".cursor", "mcp.json"), "mcpServers", []string{filepath.Join(home, ".cursor")}, FormatJSON, "tasuku"},
		{"Cursor (legacy alt)", filepath.Join(home, "Library", "Application Support", "Cursor", "User", "globalStorage", "mcp.json"), "mcpServers", []string{filepath.Join(home, "Library", "Application Support", "Cursor", "User", "globalStorage")}, FormatJSON, "tasuku"},
	}
}

func removeLegacyCursorGlobalConfigs() ([]string, []string) {
	removed := []string{}
	warnings := []string{}

	for _, tool := range getLegacyCursorGlobalTools() {
		if _, err := os.Stat(tool.SettingsPath); os.IsNotExist(err) {
			continue
		}

		didRemove, err := uninstallFromJSON(tool)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("failed cleaning %s at %s: %v", tool.Name, tool.SettingsPath, err))
			continue
		}
		if didRemove {
			removed = append(removed, tool.Name)
		}
	}

	return removed, warnings
}
