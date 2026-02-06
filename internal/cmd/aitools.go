package cmd

import "os"

// AITool represents a supported AI tool configuration
type AITool struct {
	Name         string
	SettingsPath string
	MCPKey       string
}

func getSupportedAITools() []AITool {
	home, _ := os.UserHomeDir()
	configDir := home + "/.config"
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		configDir = xdg
	}

	return []AITool{
		{"Claude Code", home + "/.claude.json", "mcpServers"},
		{"Cursor", home + "/.cursor/mcp.json", "mcpServers"},
		{"Cursor (alt)", home + "/Library/Application Support/Cursor/User/globalStorage/mcp.json", "mcpServers"},
		{"OpenCode", configDir + "/opencode/opencode.json", "mcp"},
	}
}
