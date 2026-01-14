package mcpcmd

import (
	"encoding/json"
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
	if len(localTools) != 4 {
		t.Errorf("expected 4 local tools, got %d", len(localTools))
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
	// Check OpenCode (project) - should detect via opencode.json
	if localTools[2].Name != "OpenCode (project)" {
		t.Errorf("expected 'OpenCode (project)', got %s", localTools[2].Name)
	}
	if localTools[2].MCPKey != "mcp" {
		t.Errorf("expected MCPKey 'mcp', got %s", localTools[2].MCPKey)
	}
	// Check Gemini (project)
	if localTools[3].Name != "Gemini (project)" {
		t.Errorf("expected 'Gemini (project)', got %s", localTools[3].Name)
	}
	if len(localTools[3].DetectPaths) != 2 {
		t.Errorf("expected 2 DetectPaths for Gemini, got %d", len(localTools[3].DetectPaths))
	}
	if localTools[3].DetectPaths[0] != ".gemini" || localTools[3].DetectPaths[1] != "GEMINI.md" {
		t.Errorf("expected DetectPaths ['.gemini', 'GEMINI.md'], got %v", localTools[3].DetectPaths)
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

func TestBuildMCPEntry(t *testing.T) {
	executable := "/usr/bin/tk"

	// Test Claude Code / Cursor entry (stdio type)
	claudeTool := AITool{Name: "Claude Code", MCPKey: "mcpServers"}
	entry := buildMCPEntry(claudeTool, executable)
	if entry["type"] != "stdio" {
		t.Errorf("expected type 'stdio', got %v", entry["type"])
	}
	if entry["command"] != executable {
		t.Errorf("expected command '%s', got %v", executable, entry["command"])
	}
	args, ok := entry["args"].([]string)
	if !ok || len(args) != 2 || args[0] != "serve" || args[1] != "mcp" {
		t.Errorf("expected args ['serve', 'mcp'], got %v", entry["args"])
	}

	// Test OpenCode entry (local type with command array)
	openCodeTool := AITool{Name: "OpenCode", MCPKey: "mcp"}
	entry = buildMCPEntry(openCodeTool, executable)
	if entry["type"] != "local" {
		t.Errorf("expected type 'local' for OpenCode, got %v", entry["type"])
	}
	cmdArray, ok := entry["command"].([]string)
	if !ok || len(cmdArray) != 3 || cmdArray[0] != executable || cmdArray[1] != "serve" || cmdArray[2] != "mcp" {
		t.Errorf("expected command ['%s', 'serve', 'mcp'] for OpenCode, got %v", executable, entry["command"])
	}
}

func TestInstallToJSON(t *testing.T) {
	dir := t.TempDir()

	// Test creating new JSON config
	tool := AITool{
		Name:         "Test Tool",
		SettingsPath: filepath.Join(dir, "config.json"),
		MCPKey:       "mcpServers",
		MCPEntryKey:  "tasuku",
	}

	installed, wasReinstall, err := installToJSON(tool, "/usr/bin/tk", false, false)
	if err != nil {
		t.Fatalf("failed to install: %v", err)
	}
	if !installed {
		t.Error("expected installed=true")
	}
	if wasReinstall {
		t.Error("expected wasReinstall=false for new install")
	}

	// Verify file was created
	data, err := os.ReadFile(tool.SettingsPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if !filepath.IsAbs(string(data)) && len(data) == 0 {
		t.Error("expected non-empty config file")
	}

	// Test re-install without force (should skip)
	installed, wasReinstall, err = installToJSON(tool, "/usr/bin/tk", false, true)
	if err != nil {
		t.Fatalf("failed on re-install check: %v", err)
	}
	if installed {
		t.Error("expected installed=false when already installed")
	}

	// Test re-install with force
	installed, wasReinstall, err = installToJSON(tool, "/usr/bin/tk", true, true)
	if err != nil {
		t.Fatalf("failed on force reinstall: %v", err)
	}
	if !installed {
		t.Error("expected installed=true with force")
	}
	if !wasReinstall {
		t.Error("expected wasReinstall=true with force")
	}
}

func TestInstallToTOML(t *testing.T) {
	dir := t.TempDir()

	// Test creating new TOML config (Codex format)
	tool := AITool{
		Name:         "Codex",
		SettingsPath: filepath.Join(dir, "config.toml"),
		MCPKey:       "mcp_servers",
		MCPEntryKey:  "tasuku",
		Format:       FormatTOML,
	}

	installed, wasReinstall, err := installToTOML(tool, "/usr/bin/tk", false, false)
	if err != nil {
		t.Fatalf("failed to install TOML: %v", err)
	}
	if !installed {
		t.Error("expected installed=true")
	}
	if wasReinstall {
		t.Error("expected wasReinstall=false for new install")
	}

	// Verify file was created with TOML content
	data, err := os.ReadFile(tool.SettingsPath)
	if err != nil {
		t.Fatalf("failed to read TOML config: %v", err)
	}
	content := string(data)
	if !contains(content, "[mcp_servers.tasuku]") {
		t.Errorf("expected TOML to contain [mcp_servers.tasuku], got:\n%s", content)
	}
	if !contains(content, "command") {
		t.Errorf("expected TOML to contain 'command', got:\n%s", content)
	}

	// Test re-install without force (should skip)
	installed, wasReinstall, err = installToTOML(tool, "/usr/bin/tk", false, true)
	if err != nil {
		t.Fatalf("failed on re-install check: %v", err)
	}
	if installed {
		t.Error("expected installed=false when already installed")
	}

	// Test re-install with force
	installed, wasReinstall, err = installToTOML(tool, "/usr/bin/tk", true, true)
	if err != nil {
		t.Fatalf("failed on force reinstall: %v", err)
	}
	if !installed {
		t.Error("expected installed=true with force")
	}
	if !wasReinstall {
		t.Error("expected wasReinstall=true with force")
	}
}

func TestUninstallFromJSON(t *testing.T) {
	dir := t.TempDir()

	// Create a JSON config with tasuku installed
	configPath := filepath.Join(dir, "config.json")
	initialConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"tasuku": map[string]interface{}{
				"command": "/usr/bin/tk",
				"args":    []string{"serve", "mcp"},
			},
			"other-server": map[string]interface{}{
				"command": "/usr/bin/other",
			},
		},
	}
	data, _ := json.MarshalIndent(initialConfig, "", "  ")
	os.WriteFile(configPath, data, 0644)

	tool := AITool{
		Name:         "Test Tool",
		SettingsPath: configPath,
		MCPKey:       "mcpServers",
		MCPEntryKey:  "tasuku",
	}

	removed, err := uninstallFromJSON(tool)
	if err != nil {
		t.Fatalf("failed to uninstall: %v", err)
	}
	if !removed {
		t.Error("expected removed=true")
	}

	// Verify tasuku was removed but other-server preserved
	data, _ = os.ReadFile(configPath)
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	mcpServers := result["mcpServers"].(map[string]interface{})
	if _, exists := mcpServers["tasuku"]; exists {
		t.Error("expected tasuku to be removed")
	}
	if _, exists := mcpServers["other-server"]; !exists {
		t.Error("expected other-server to be preserved")
	}
}

func TestUninstallFromJSONNotInstalled(t *testing.T) {
	dir := t.TempDir()

	// Create a JSON config without tasuku
	configPath := filepath.Join(dir, "config.json")
	initialConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"other-server": map[string]interface{}{
				"command": "/usr/bin/other",
			},
		},
	}
	data, _ := json.MarshalIndent(initialConfig, "", "  ")
	os.WriteFile(configPath, data, 0644)

	tool := AITool{
		Name:         "Test Tool",
		SettingsPath: configPath,
		MCPKey:       "mcpServers",
		MCPEntryKey:  "tasuku",
	}

	removed, err := uninstallFromJSON(tool)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed {
		t.Error("expected removed=false when tasuku not installed")
	}
}

func TestUninstallFromTOML(t *testing.T) {
	dir := t.TempDir()

	// Create a TOML config with tasuku installed
	configPath := filepath.Join(dir, "config.toml")
	tomlContent := `[mcp_servers.tasuku]
command = "/usr/bin/tk"
args = ["serve", "mcp"]

[mcp_servers.other]
command = "/usr/bin/other"
`
	os.WriteFile(configPath, []byte(tomlContent), 0644)

	tool := AITool{
		Name:         "Codex",
		SettingsPath: configPath,
		MCPKey:       "mcp_servers",
		MCPEntryKey:  "tasuku",
		Format:       FormatTOML,
	}

	removed, err := uninstallFromTOML(tool)
	if err != nil {
		t.Fatalf("failed to uninstall TOML: %v", err)
	}
	if !removed {
		t.Error("expected removed=true")
	}

	// Verify tasuku was removed but other preserved
	data, _ := os.ReadFile(configPath)
	content := string(data)
	if contains(content, "tasuku") {
		t.Error("expected tasuku to be removed from TOML")
	}
	if !contains(content, "other") {
		t.Error("expected other server to be preserved in TOML")
	}
}

func TestUninstallFromTOMLNotInstalled(t *testing.T) {
	dir := t.TempDir()

	// Create a TOML config without tasuku
	configPath := filepath.Join(dir, "config.toml")
	tomlContent := `[mcp_servers.other]
command = "/usr/bin/other"
`
	os.WriteFile(configPath, []byte(tomlContent), 0644)

	tool := AITool{
		Name:         "Codex",
		SettingsPath: configPath,
		MCPKey:       "mcp_servers",
		MCPEntryKey:  "tasuku",
		Format:       FormatTOML,
	}

	removed, err := uninstallFromTOML(tool)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed {
		t.Error("expected removed=false when tasuku not installed")
	}
}

func TestInstallWithExistingJSONConfig(t *testing.T) {
	dir := t.TempDir()

	// Create existing config with another MCP server
	configPath := filepath.Join(dir, "config.json")
	initialConfig := map[string]interface{}{
		"other_setting": "value",
		"mcpServers": map[string]interface{}{
			"other-server": map[string]interface{}{
				"command": "/usr/bin/other",
			},
		},
	}
	data, _ := json.MarshalIndent(initialConfig, "", "  ")
	os.WriteFile(configPath, data, 0644)

	tool := AITool{
		Name:         "Test Tool",
		SettingsPath: configPath,
		MCPKey:       "mcpServers",
		MCPEntryKey:  "tasuku",
	}

	installed, _, err := installToJSON(tool, "/usr/bin/tk", false, true)
	if err != nil {
		t.Fatalf("failed to install: %v", err)
	}
	if !installed {
		t.Error("expected installed=true")
	}

	// Verify other settings preserved
	data, _ = os.ReadFile(configPath)
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if result["other_setting"] != "value" {
		t.Error("expected other_setting to be preserved")
	}
	mcpServers := result["mcpServers"].(map[string]interface{})
	if _, exists := mcpServers["other-server"]; !exists {
		t.Error("expected other-server to be preserved")
	}
	if _, exists := mcpServers["tasuku"]; !exists {
		t.Error("expected tasuku to be added")
	}
}

func TestInstallLocalWithCursorrules(t *testing.T) {
	h := testutil.New(t)
	dir := h.TempDir()

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Create only .cursorrules (no .cursor/ directory)
	os.WriteFile(filepath.Join(dir, ".cursorrules"), []byte("rules"), 0644)

	err := h.Execute(Cmd, "install", "--local", "--tool", "cursor")
	h.AssertNoError(err)

	// Should detect Cursor and create .cursor/mcp.json
	h.AssertOutputContains("Cursor")

	// Verify .cursor/mcp.json was created
	if _, err := os.Stat(filepath.Join(dir, ".cursor", "mcp.json")); os.IsNotExist(err) {
		t.Error("expected .cursor/mcp.json to be created when .cursorrules exists")
	}
}

func TestInstallForceReinstall(t *testing.T) {
	h := testutil.New(t)
	dir := h.TempDir()

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Create CLAUDE.md
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Test"), 0644)

	// First install
	err := h.Execute(Cmd, "install", "--local")
	h.AssertNoError(err)
	h.AssertOutputContains("installed")

	// Second install without force (should say already configured)
	err = h.Execute(Cmd, "install", "--local")
	h.AssertNoError(err)
	h.AssertOutputContains("Already configured")

	// Third install with force (should reinstall)
	err = h.Execute(Cmd, "install", "--local", "--force")
	h.AssertNoError(err)
	h.AssertOutputContains("reinstalled")
}

func TestUninstallLocal(t *testing.T) {
	h := testutil.New(t)
	dir := h.TempDir()

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Setup: Create CLAUDE.md and install
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Test"), 0644)
	h.Execute(Cmd, "install", "--local")

	// Verify .claude.json exists
	if _, err := os.Stat(filepath.Join(dir, ".claude.json")); os.IsNotExist(err) {
		t.Fatal("expected .claude.json to exist after install")
	}

	// Uninstall
	err := h.Execute(Cmd, "uninstall", "--local")
	h.AssertNoError(err)
	h.AssertOutputContains("removed")

	// Verify tasuku removed from config
	data, _ := os.ReadFile(filepath.Join(dir, ".claude.json"))
	var config map[string]interface{}
	json.Unmarshal(data, &config)
	mcpServers, _ := config["mcpServers"].(map[string]interface{})
	if _, exists := mcpServers["tasuku"]; exists {
		t.Error("expected tasuku to be removed from .claude.json")
	}
}

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
