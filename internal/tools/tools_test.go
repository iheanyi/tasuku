package tools

import (
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		input    string
		expected Tool
		found    bool
	}{
		// Claude Code aliases
		{"claude", ToolClaude, true},
		{"Claude", ToolClaude, true},
		{"CLAUDE", ToolClaude, true},
		{"claude-code", ToolClaude, true},
		{"Claude-Code", ToolClaude, true},
		{"claudecode", ToolClaude, true},

		// Cursor aliases
		{"cursor", ToolCursor, true},
		{"Cursor", ToolCursor, true},
		{"CURSOR", ToolCursor, true},

		// Copilot CLI aliases
		{"copilot", ToolCopilot, true},
		{"Copilot", ToolCopilot, true},
		{"copilot-cli", ToolCopilot, true},
		{"copilotcli", ToolCopilot, true},
		{"github", ToolCopilot, true},
		{"GitHub", ToolCopilot, true},

		// Codex aliases
		{"codex", ToolCodex, true},
		{"Codex", ToolCodex, true},
		{"CODEX", ToolCodex, true},

		// OpenCode aliases
		{"opencode", ToolOpenCode, true},
		{"OpenCode", ToolOpenCode, true},
		{"open-code", ToolOpenCode, true},

		// Gemini aliases
		{"gemini", ToolGemini, true},
		{"Gemini", ToolGemini, true},
		{"google", ToolGemini, true},
		{"Google", ToolGemini, true},

		// Unknown tools
		{"unknown", "", false},
		{"vscode", "", false},
		{"vim", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, found := Resolve(tt.input)
			if found != tt.found {
				t.Errorf("Resolve(%q) found = %v, want %v", tt.input, found, tt.found)
			}
			if got != tt.expected {
				t.Errorf("Resolve(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTool_String(t *testing.T) {
	tests := []struct {
		tool     Tool
		expected string
	}{
		{ToolClaude, "Claude Code"},
		{ToolCursor, "Cursor"},
		{ToolCopilot, "Copilot CLI"},
		{ToolCodex, "Codex"},
		{ToolOpenCode, "OpenCode"},
		{ToolGemini, "Gemini"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.tool.String(); got != tt.expected {
				t.Errorf("Tool.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAll(t *testing.T) {
	all := All()

	// Should have all 6 tools
	if len(all) != 6 {
		t.Errorf("All() returned %d tools, want 6", len(all))
	}

	// Check all expected tools are present
	expected := map[Tool]bool{
		ToolClaude:   false,
		ToolCursor:   false,
		ToolCopilot:  false,
		ToolCodex:    false,
		ToolOpenCode: false,
		ToolGemini:   false,
	}

	for _, tool := range all {
		if _, ok := expected[tool]; !ok {
			t.Errorf("All() contains unexpected tool: %q", tool)
		}
		expected[tool] = true
	}

	for tool, found := range expected {
		if !found {
			t.Errorf("All() missing tool: %q", tool)
		}
	}
}

func TestValidNames(t *testing.T) {
	names := ValidNames()

	// Should contain all tool name keywords
	expectedKeywords := []string{"claude", "cursor", "copilot", "codex", "opencode", "gemini"}
	for _, keyword := range expectedKeywords {
		if !strings.Contains(names, keyword) {
			t.Errorf("ValidNames() = %q, missing %q", names, keyword)
		}
	}
}

func TestResolve_AllAliasesMapToValidTools(t *testing.T) {
	// Verify all aliases resolve to one of the defined tool constants
	validTools := map[Tool]bool{
		ToolClaude:   true,
		ToolCursor:   true,
		ToolCopilot:  true,
		ToolCodex:    true,
		ToolOpenCode: true,
		ToolGemini:   true,
	}

	for alias, tool := range aliases {
		if !validTools[tool] {
			t.Errorf("alias %q maps to invalid tool %q", alias, tool)
		}
	}
}
