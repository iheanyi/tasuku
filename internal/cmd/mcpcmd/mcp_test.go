package mcpcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
)

func TestMCPCmdStructure(t *testing.T) {
	if Cmd.Use != "mcp" {
		t.Errorf("expected Use to be 'mcp', got %s", Cmd.Use)
	}

	// Check subcommands exist
	subcommands := make(map[string]bool)
	for _, sub := range Cmd.Commands() {
		subcommands[sub.Use] = true
	}

	expected := []string{"serve", "install", "uninstall", "config"}
	for _, exp := range expected {
		if !subcommands[exp] {
			t.Errorf("expected subcommand '%s' not found", exp)
		}
	}
}

func TestInstallCmdFlags(t *testing.T) {
	// Find install subcommand
	var installCmd *struct{}
	for _, sub := range Cmd.Commands() {
		if sub.Use == "install" {
			forceFlag := sub.Flags().Lookup("force")
			if forceFlag == nil {
				t.Error("expected --force flag on install command")
			}
			break
		}
	}
	_ = installCmd
}

func TestConfigCmd(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "config")
	h.AssertNoError(err)

	// Should output JSON config snippet
	h.AssertOutputContains("tasuku")
	h.AssertOutputContains("command")
	h.AssertOutputContains("args")
}

func TestGetSupportedAITools(t *testing.T) {
	// Test global tools
	tools := getSupportedAITools(false)

	if len(tools) == 0 {
		t.Error("expected at least one supported AI tool")
	}

	// Check that Claude Code is supported
	foundClaude := false
	for _, tool := range tools {
		if tool.Name == "Claude Code" {
			foundClaude = true
			if tool.MCPKey != "mcpServers" {
				t.Errorf("expected MCPKey 'mcpServers', got %s", tool.MCPKey)
			}
			break
		}
	}
	if !foundClaude {
		t.Error("expected Claude Code in supported tools")
	}

	// Test local tools
	localTools := getSupportedAITools(true)
	if len(localTools) != 1 {
		t.Errorf("expected 1 local tool, got %d", len(localTools))
	}
	if localTools[0].Name != "Claude Code (project)" {
		t.Errorf("expected 'Claude Code (project)', got %s", localTools[0].Name)
	}
	if localTools[0].SettingsPath != ".claude.json" {
		t.Errorf("expected '.claude.json', got %s", localTools[0].SettingsPath)
	}
}

func TestUninstallNoConfig(t *testing.T) {
	h := testutil.New(t)

	// Create a temp home dir without any AI tool configs
	tempHome := filepath.Join(h.TempDir(), "home")
	os.MkdirAll(tempHome, 0755)
	os.Setenv("HOME", tempHome)
	defer os.Unsetenv("HOME")

	err := h.Execute(Cmd, "uninstall")
	h.AssertNoError(err)
	h.AssertOutputContains("was not configured")
}
