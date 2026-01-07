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
	if len(localTools) != 2 {
		t.Errorf("expected 2 local tools, got %d", len(localTools))
	}
	// Check Claude Code (project) - should detect via .claude/ OR CLAUDE.md
	if localTools[0].Name != "Claude Code (project)" {
		t.Errorf("expected 'Claude Code (project)', got %s", localTools[0].Name)
	}
	if localTools[0].SettingsPath != ".claude.json" {
		t.Errorf("expected '.claude.json', got %s", localTools[0].SettingsPath)
	}
	if len(localTools[0].DetectPaths) != 2 {
		t.Errorf("expected 2 DetectPaths for Claude Code, got %d", len(localTools[0].DetectPaths))
	}
	if localTools[0].DetectPaths[0] != ".claude" || localTools[0].DetectPaths[1] != "CLAUDE.md" {
		t.Errorf("expected DetectPaths ['.claude', 'CLAUDE.md'], got %v", localTools[0].DetectPaths)
	}
	// Check Cursor (project) - should detect via .cursorrules OR .cursor/
	if localTools[1].Name != "Cursor (project)" {
		t.Errorf("expected 'Cursor (project)', got %s", localTools[1].Name)
	}
	if len(localTools[1].DetectPaths) != 2 {
		t.Errorf("expected 2 DetectPaths for Cursor, got %d", len(localTools[1].DetectPaths))
	}
	if localTools[1].DetectPaths[0] != ".cursorrules" || localTools[1].DetectPaths[1] != ".cursor" {
		t.Errorf("expected DetectPaths ['.cursorrules', '.cursor'], got %v", localTools[1].DetectPaths)
	}
}

func TestInstallCmdToolFlag(t *testing.T) {
	// Find install subcommand and check --tool flag
	for _, sub := range Cmd.Commands() {
		if sub.Use == "install" {
			toolFlag := sub.Flags().Lookup("tool")
			if toolFlag == nil {
				t.Error("expected --tool flag on install command")
			}
			if toolFlag.DefValue != "" {
				t.Error("expected --tool default to be empty")
			}
			break
		}
	}
}

func TestInstallLocalWithClaudeMD(t *testing.T) {
	h := testutil.New(t)
	dir := h.TempDir()

	// Change to temp dir
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Create only CLAUDE.md (no .claude/ directory)
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Test"), 0644)

	// Run local install
	err := h.Execute(Cmd, "install", "--local")
	h.AssertNoError(err)

	// Should detect Claude Code and create .claude.json
	h.AssertOutputContains("Claude Code")

	// Verify .claude.json was created
	if _, err := os.Stat(filepath.Join(dir, ".claude.json")); os.IsNotExist(err) {
		t.Error("expected .claude.json to be created when CLAUDE.md exists")
	}
}

func TestInstallToolFilterInvalid(t *testing.T) {
	h := testutil.New(t)
	dir := h.TempDir()

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Create CLAUDE.md so detection works
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Test"), 0644)

	err := h.Execute(Cmd, "install", "--local", "--tool", "invalid")
	if err == nil {
		t.Error("expected error for invalid tool name")
	}
}

func TestInstallToolFilterClaude(t *testing.T) {
	h := testutil.New(t)
	dir := h.TempDir()

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Create both CLAUDE.md and .cursorrules
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Test"), 0644)
	os.WriteFile(filepath.Join(dir, ".cursorrules"), []byte("rules"), 0644)

	// Run local install with --tool claude
	err := h.Execute(Cmd, "install", "--local", "--tool", "claude")
	h.AssertNoError(err)

	// Should only install to Claude Code
	h.AssertOutputContains("Claude Code")

	// Verify .claude.json was created but .cursor/mcp.json was not
	if _, err := os.Stat(filepath.Join(dir, ".claude.json")); os.IsNotExist(err) {
		t.Error("expected .claude.json to be created")
	}
	// Cursor config should NOT be created since we filtered to claude only
	if _, err := os.Stat(filepath.Join(dir, ".cursor", "mcp.json")); err == nil {
		t.Error("expected .cursor/mcp.json to NOT be created when --tool claude")
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
