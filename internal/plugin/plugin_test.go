package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEmbeddedCommands(t *testing.T) {
	commands, err := LoadEmbeddedCommands()
	if err != nil {
		t.Fatalf("LoadEmbeddedCommands failed: %v", err)
	}

	if len(commands) == 0 {
		t.Error("expected at least one command")
	}

	// Check that common commands exist
	foundList := false
	foundAdd := false
	foundDone := false
	for _, cmd := range commands {
		switch cmd.Name {
		case "list":
			foundList = true
		case "add":
			foundAdd = true
		case "done":
			foundDone = true
		}
	}

	if !foundList {
		t.Error("expected 'list' command")
	}
	if !foundAdd {
		t.Error("expected 'add' command")
	}
	if !foundDone {
		t.Error("expected 'done' command")
	}
}

func TestParseClaudeCommand(t *testing.T) {
	content := `---
description: Test command description
argument-hint: <arg>
---

This is the content.
`

	cmd, err := parseClaudeCommand("test.md", []byte(content))
	if err != nil {
		t.Fatalf("parseClaudeCommand failed: %v", err)
	}

	if cmd.Name != "test" {
		t.Errorf("expected name 'test', got %q", cmd.Name)
	}
	if cmd.Description != "Test command description" {
		t.Errorf("expected description 'Test command description', got %q", cmd.Description)
	}
	if cmd.ArgumentHint != "<arg>" {
		t.Errorf("expected argument-hint '<arg>', got %q", cmd.ArgumentHint)
	}
	if cmd.Content != "This is the content." {
		t.Errorf("expected content 'This is the content.', got %q", cmd.Content)
	}
}

func TestConvertToSkillMD(t *testing.T) {
	cmd := Command{
		Name:        "test",
		Description: "A test command",
		Content:     "This is the content.",
	}

	output := ConvertToSkillMD(cmd)
	expected := `---
name: test
description: A test command
---

This is the content.`

	if string(output) != expected {
		t.Errorf("ConvertToSkillMD output mismatch:\ngot:\n%s\n\nwant:\n%s", string(output), expected)
	}
}

func TestConvertToCursorCommand(t *testing.T) {
	cmd := Command{
		Name:        "add-task",
		Description: "Create a new task with the given description.",
		Content:     "Use tk_add to create the task.",
	}

	output := ConvertToCursorCommand(cmd)
	outputStr := string(output)

	// Check for title (Title Case)
	if !contains(outputStr, "# Add Task") {
		t.Errorf("expected '# Add Task' title, got:\n%s", outputStr)
	}

	// Check for description
	if !contains(outputStr, "Create a new task with the given description.") {
		t.Errorf("expected description in output, got:\n%s", outputStr)
	}

	// Check for Instructions section
	if !contains(outputStr, "## Instructions") {
		t.Errorf("expected '## Instructions' section, got:\n%s", outputStr)
	}

	// Check for content
	if !contains(outputStr, "Use tk_add to create the task.") {
		t.Errorf("expected content in output, got:\n%s", outputStr)
	}
}

func TestGetSupportedTools(t *testing.T) {
	tools := GetSupportedTools()
	if len(tools) != 4 {
		t.Errorf("expected 4 supported tools, got %d", len(tools))
	}

	// Check that all required tools are present
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
	}

	expected := []string{"Claude Code", "Cursor", "Copilot CLI", "Codex"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected tool %q not found", name)
		}
	}
}

func TestGetToolByName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"claude", "Claude Code"},
		{"claude-code", "Claude Code"},
		{"claudecode", "Claude Code"},
		{"cursor", "Cursor"},
		{"copilot", "Copilot CLI"},
		{"copilot-cli", "Copilot CLI"},
		{"copilotcli", "Copilot CLI"},
		{"github", "Copilot CLI"},
		{"codex", "Codex"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tool := GetToolByName(tt.input)
			if tt.expected == "" {
				if tool != nil {
					t.Errorf("expected nil for %q, got %v", tt.input, tool)
				}
			} else {
				if tool == nil {
					t.Errorf("expected tool for %q, got nil", tt.input)
				} else if tool.Name != tt.expected {
					t.Errorf("expected %q for %q, got %q", tt.expected, tt.input, tool.Name)
				}
			}
		})
	}
}

func TestInstallToToolSkillMD(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Create .copilot directory to trigger detection
	os.Mkdir(".copilot", 0755)

	tool := ToolTarget{
		Name:      "Test Tool",
		Format:    "skill-md",
		LocalDir:  ".test/skills/tasuku",
		GlobalDir: filepath.Join(dir, "global/skills/tasuku"),
	}

	result := InstallToTool(tool, true) // local install

	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}

	if len(result.FilesAdded) == 0 {
		t.Error("expected files to be added")
	}

	// Verify files were created
	for _, path := range result.FilesAdded {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("file %s was not created", path)
		}
	}
}

func TestInstallToToolCursorCommand(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Create .cursor directory to trigger detection
	os.Mkdir(".cursor", 0755)

	tool := ToolTarget{
		Name:      "Cursor",
		Format:    "cursor-command",
		LocalDir:  ".cursor/commands/tasuku",
		GlobalDir: filepath.Join(dir, "global/commands/tasuku"),
	}

	result := InstallToTool(tool, true) // local install

	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}

	if len(result.FilesAdded) == 0 {
		t.Error("expected files to be added")
	}

	// Verify files were created with tasuku- prefix
	for _, path := range result.FilesAdded {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("file %s was not created", path)
		}
		// Check that files have tasuku- prefix
		basename := filepath.Base(path)
		if !strings.HasPrefix(basename, "tasuku-") {
			t.Errorf("file %s should have 'tasuku-' prefix", basename)
		}
	}
}

func TestUninstallFromToolCursorCommand(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Create commands directory with files
	commandsDir := ".cursor/commands/tasuku"
	os.MkdirAll(commandsDir, 0755)
	os.WriteFile(filepath.Join(commandsDir, "tasuku-test.md"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(commandsDir, "other.md"), []byte("test"), 0644) // should not be removed

	tool := ToolTarget{
		Name:     "Cursor",
		Format:   "cursor-command",
		LocalDir: commandsDir,
	}

	result := UninstallFromTool(tool, true) // local uninstall

	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}

	// Only tasuku- prefixed file should be removed
	if len(result.FilesAdded) != 1 {
		t.Errorf("expected 1 file removed, got %d", len(result.FilesAdded))
	}

	// Verify tasuku-test.md was removed
	if _, err := os.Stat(filepath.Join(commandsDir, "tasuku-test.md")); !os.IsNotExist(err) {
		t.Error("tasuku-test.md should be removed")
	}

	// Verify other.md still exists
	if _, err := os.Stat(filepath.Join(commandsDir, "other.md")); os.IsNotExist(err) {
		t.Error("other.md should NOT be removed")
	}
}

func TestInstallToToolClaudePlugin(t *testing.T) {
	tool := ToolTarget{
		Name:   "Claude Code",
		Format: "claude-plugin",
	}

	result := InstallToTool(tool, false)

	// Claude Code should return an error message guiding user
	if len(result.Errors) == 0 {
		t.Error("expected guidance message for Claude Code")
	}
}

func TestUninstallFromToolSkillMD(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Create skills directory with files
	skillsDir := ".test/skills/tasuku"
	os.MkdirAll(skillsDir, 0755)
	os.WriteFile(filepath.Join(skillsDir, "test.md"), []byte("test"), 0644)

	tool := ToolTarget{
		Name:     "Test Tool",
		Format:   "skill-md",
		LocalDir: skillsDir,
	}

	result := UninstallFromTool(tool, true) // local uninstall

	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}

	if len(result.FilesAdded) != 1 {
		t.Errorf("expected 1 file removed, got %d", len(result.FilesAdded))
	}
}

func TestGetDetectedTools(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Initially no tools detected
	detected := GetDetectedTools()
	if len(detected) != 0 {
		t.Errorf("expected 0 detected tools, got %d", len(detected))
	}

	// Create .claude directory
	os.Mkdir(".claude", 0755)
	detected = GetDetectedTools()
	if len(detected) != 1 {
		t.Errorf("expected 1 detected tool after .claude, got %d", len(detected))
	}
	if detected[0].Name != "Claude Code" {
		t.Errorf("expected 'Claude Code', got %q", detected[0].Name)
	}

	// Create .codex directory
	os.Mkdir(".codex", 0755)
	detected = GetDetectedTools()
	if len(detected) != 2 {
		t.Errorf("expected 2 detected tools, got %d", len(detected))
	}
}

func TestGenerateSkillIndex(t *testing.T) {
	commands := []Command{
		{Name: "pickup", Description: "Pick up a task"},
		{Name: "add", Description: "Add a task"},
		{Name: "help", Description: "Show help"},
	}

	index := GenerateSkillIndex(commands)

	if len(index) == 0 {
		t.Error("expected non-empty index")
	}

	// Check for expected content
	indexStr := string(index)
	if !contains(indexStr, "# Tasuku Skills") {
		t.Error("expected '# Tasuku Skills' header")
	}
	if !contains(indexStr, "pickup") {
		t.Error("expected 'pickup' command in index")
	}
	if !contains(indexStr, "add") {
		t.Error("expected 'add' command in index")
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
