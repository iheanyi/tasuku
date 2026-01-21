package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHandlePluginList(t *testing.T) {
	server, _ := setupTestServer(t)

	result, err := server.HandleToolCall("tk_plugin_list", map[string]interface{}{})
	if err != nil {
		t.Fatalf("handlePluginList failed: %v", err)
	}

	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}

	// Check total_commands exists and is positive
	total, ok := m["total_commands"].(int)
	if !ok {
		t.Error("expected total_commands to be int")
	}
	if total == 0 {
		t.Error("expected at least one command")
	}

	// Check workflow_commands exists
	if _, ok := m["workflow_commands"]; !ok {
		t.Error("expected workflow_commands key")
	}

	// Check basic_commands exists
	if _, ok := m["basic_commands"]; !ok {
		t.Error("expected basic_commands key")
	}
}

func TestHandlePluginStatus_NoToolsDetected(t *testing.T) {
	// Create a temp directory with no AI tool markers
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	server, _ := setupTestServer(t)

	result, err := server.HandleToolCall("tk_plugin_status", map[string]interface{}{})
	if err != nil {
		t.Fatalf("handlePluginStatus failed: %v", err)
	}

	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}

	status, _ := m["status"].(string)
	if status != "no_tools_detected" {
		t.Errorf("expected status 'no_tools_detected', got %q", status)
	}

	// Should have supported_tools list
	if _, ok := m["supported_tools"]; !ok {
		t.Error("expected supported_tools key when no tools detected")
	}
}

func TestHandlePluginInstall_UnknownTool(t *testing.T) {
	server, _ := setupTestServer(t)

	_, err := server.HandleToolCall("tk_plugin_install", map[string]interface{}{
		"tool": "unknown-tool",
	})

	if err == nil {
		t.Error("expected error for unknown tool")
	}
}

func TestHandlePluginUninstall_UnknownTool(t *testing.T) {
	server, _ := setupTestServer(t)

	_, err := server.HandleToolCall("tk_plugin_uninstall", map[string]interface{}{
		"tool": "unknown-tool",
	})

	if err == nil {
		t.Error("expected error for unknown tool")
	}
}

func TestHandleMCPInstall(t *testing.T) {
	server, _ := setupTestServer(t)

	tests := []struct {
		name     string
		args     map[string]interface{}
		contains string
	}{
		{
			name:     "basic install",
			args:     map[string]interface{}{},
			contains: "tk mcp install",
		},
		{
			name:     "with tool",
			args:     map[string]interface{}{"tool": "claude"},
			contains: "--tool claude",
		},
		{
			name:     "with local",
			args:     map[string]interface{}{"local": true},
			contains: "--local",
		},
		{
			name:     "with force",
			args:     map[string]interface{}{"force": true},
			contains: "--force",
		},
		{
			name:     "all flags",
			args:     map[string]interface{}{"tool": "cursor", "local": true, "force": true},
			contains: "--tool cursor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := server.HandleToolCall("tk_mcp_install", tt.args)
			if err != nil {
				t.Fatalf("handleMCPInstall failed: %v", err)
			}

			m, ok := result.(map[string]interface{})
			if !ok {
				t.Fatalf("expected map result, got %T", result)
			}

			if m["status"] != "guidance" {
				t.Errorf("expected status 'guidance', got %q", m["status"])
			}

			cmd, _ := m["command"].(string)
			if cmd == "" {
				t.Error("expected command in result")
			}

			if tt.contains != "" && !contains(cmd, tt.contains) {
				t.Errorf("command %q should contain %q", cmd, tt.contains)
			}
		})
	}
}

func TestHandleMCPUninstall(t *testing.T) {
	server, _ := setupTestServer(t)

	tests := []struct {
		name     string
		args     map[string]interface{}
		contains string
	}{
		{
			name:     "basic uninstall",
			args:     map[string]interface{}{},
			contains: "tk mcp uninstall",
		},
		{
			name:     "with local",
			args:     map[string]interface{}{"local": true},
			contains: "--local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := server.HandleToolCall("tk_mcp_uninstall", tt.args)
			if err != nil {
				t.Fatalf("handleMCPUninstall failed: %v", err)
			}

			m, ok := result.(map[string]interface{})
			if !ok {
				t.Fatalf("expected map result, got %T", result)
			}

			cmd, _ := m["command"].(string)
			if !contains(cmd, tt.contains) {
				t.Errorf("command %q should contain %q", cmd, tt.contains)
			}
		})
	}
}

func TestHandleHooksInstall(t *testing.T) {
	server, _ := setupTestServer(t)

	tests := []struct {
		name     string
		args     map[string]interface{}
		contains []string
	}{
		{
			name:     "basic install",
			args:     map[string]interface{}{},
			contains: []string{"tk hooks install"},
		},
		{
			name:     "git only",
			args:     map[string]interface{}{"git": true},
			contains: []string{"--git"},
		},
		{
			name:     "claude only",
			args:     map[string]interface{}{"claude": true},
			contains: []string{"--claude"},
		},
		{
			name:     "codex only",
			args:     map[string]interface{}{"codex": true},
			contains: []string{"--codex"},
		},
		{
			name:     "opencode only",
			args:     map[string]interface{}{"opencode": true},
			contains: []string{"--opencode"},
		},
		{
			name:     "copilot only",
			args:     map[string]interface{}{"copilot": true},
			contains: []string{"--copilot"},
		},
		{
			name:     "with local and force",
			args:     map[string]interface{}{"local": true, "force": true},
			contains: []string{"--local", "--force"},
		},
		{
			name:     "multiple flags",
			args:     map[string]interface{}{"git": true, "claude": true, "local": true},
			contains: []string{"--git", "--claude", "--local"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := server.HandleToolCall("tk_hooks_install", tt.args)
			if err != nil {
				t.Fatalf("handleHooksInstall failed: %v", err)
			}

			m, ok := result.(map[string]interface{})
			if !ok {
				t.Fatalf("expected map result, got %T", result)
			}

			cmd, _ := m["command"].(string)
			for _, expected := range tt.contains {
				if !contains(cmd, expected) {
					t.Errorf("command %q should contain %q", cmd, expected)
				}
			}
		})
	}
}

func TestHandleHooksUninstall(t *testing.T) {
	server, _ := setupTestServer(t)

	tests := []struct {
		name     string
		args     map[string]interface{}
		contains []string
	}{
		{
			name:     "basic uninstall",
			args:     map[string]interface{}{},
			contains: []string{"tk hooks uninstall"},
		},
		{
			name:     "git only",
			args:     map[string]interface{}{"git": true},
			contains: []string{"--git"},
		},
		{
			name:     "with local",
			args:     map[string]interface{}{"local": true},
			contains: []string{"--local"},
		},
		{
			name:     "multiple flags",
			args:     map[string]interface{}{"claude": true, "codex": true, "local": true},
			contains: []string{"--claude", "--codex", "--local"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := server.HandleToolCall("tk_hooks_uninstall", tt.args)
			if err != nil {
				t.Fatalf("handleHooksUninstall failed: %v", err)
			}

			m, ok := result.(map[string]interface{})
			if !ok {
				t.Fatalf("expected map result, got %T", result)
			}

			cmd, _ := m["command"].(string)
			for _, expected := range tt.contains {
				if !contains(cmd, expected) {
					t.Errorf("command %q should contain %q", cmd, expected)
				}
			}
		})
	}
}

func TestBuildMCPInstallCommand(t *testing.T) {
	tests := []struct {
		tool     string
		local    bool
		force    bool
		expected string
	}{
		{"", false, false, "tk mcp install"},
		{"claude", false, false, "tk mcp install --tool claude"},
		{"", true, false, "tk mcp install --local"},
		{"", false, true, "tk mcp install --force"},
		{"cursor", true, true, "tk mcp install --tool cursor --local --force"},
	}

	for _, tt := range tests {
		got := buildMCPInstallCommand(tt.tool, tt.local, tt.force)
		if got != tt.expected {
			t.Errorf("buildMCPInstallCommand(%q, %v, %v) = %q, want %q",
				tt.tool, tt.local, tt.force, got, tt.expected)
		}
	}
}

func TestCheckPluginInstalled(t *testing.T) {
	// Test with non-existent directory
	if checkPluginInstalled("/nonexistent/path") {
		t.Error("expected false for non-existent directory")
	}

	// Test with empty directory
	emptyDir := t.TempDir()
	if checkPluginInstalled(emptyDir) {
		t.Error("expected false for empty directory")
	}

	// Test with directory containing non-md files
	nonMdDir := t.TempDir()
	os.WriteFile(filepath.Join(nonMdDir, "test.txt"), []byte("test"), 0644)
	if checkPluginInstalled(nonMdDir) {
		t.Error("expected false for directory with no .md files")
	}

	// Test with directory containing .md files
	mdDir := t.TempDir()
	os.WriteFile(filepath.Join(mdDir, "skill.md"), []byte("test"), 0644)
	if !checkPluginInstalled(mdDir) {
		t.Error("expected true for directory with .md files")
	}

	// Test with file (not directory)
	file := filepath.Join(t.TempDir(), "file.txt")
	os.WriteFile(file, []byte("test"), 0644)
	if checkPluginInstalled(file) {
		t.Error("expected false for file (not directory)")
	}
}

// contains checks if s contains substr (helper for tests)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
