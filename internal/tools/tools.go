// Package tools provides shared AI tool definitions and aliases for Tasuku.
// This consolidates tool name resolution across plugin, MCP, and rules packages.
package tools

import "strings"

// Tool represents a supported AI tool.
type Tool string

// Supported AI tools.
const (
	ToolClaude   Tool = "Claude Code"
	ToolCursor   Tool = "Cursor"
	ToolCopilot  Tool = "Copilot CLI"
	ToolCodex    Tool = "Codex"
	ToolOpenCode Tool = "OpenCode"
	ToolGemini   Tool = "Gemini"
)

// aliases maps various user inputs to canonical tool names.
var aliases = map[string]Tool{
	// Claude Code
	"claude":      ToolClaude,
	"claude-code": ToolClaude,
	"claudecode":  ToolClaude,

	// Cursor
	"cursor": ToolCursor,

	// Copilot CLI
	"copilot":     ToolCopilot,
	"copilot-cli": ToolCopilot,
	"copilotcli":  ToolCopilot,
	"github":      ToolCopilot,

	// Codex
	"codex": ToolCodex,

	// OpenCode
	"opencode":  ToolOpenCode,
	"open-code": ToolOpenCode,

	// Gemini
	"gemini": ToolGemini,
	"google": ToolGemini,
}

// Resolve maps a user-provided tool name to its canonical Tool constant.
// Returns the Tool and true if found, or empty string and false if not.
func Resolve(name string) (Tool, bool) {
	t, ok := aliases[strings.ToLower(name)]
	return t, ok
}

// String returns the display name of the tool.
func (t Tool) String() string {
	return string(t)
}

// ValidNames returns a comma-separated string of valid tool names for error messages.
func ValidNames() string {
	return "claude, cursor, copilot, codex, opencode, gemini"
}
