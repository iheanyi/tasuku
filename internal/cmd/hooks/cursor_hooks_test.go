package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
)

func TestInstallCursorHooks(t *testing.T) {
	h := testutil.New(t)

	// Initialize a git repo in the temp dir (needed for general hooks install)
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Install Cursor hooks only (use --local so hooks go to temp dir, not real ~/.cursor/)
	err := h.Execute(Cmd, "install", "--cursor", "--local")
	h.AssertNoError(err)
	h.AssertOutputContains("Cursor")

	// Check hooks file was created
	hooksFile := filepath.Join(h.TempDir(), ".cursor", "hooks.json")
	if _, err := os.Stat(hooksFile); os.IsNotExist(err) {
		t.Error("Cursor hooks file should exist")
	}

	// Verify the content
	data, err := os.ReadFile(hooksFile)
	if err != nil {
		t.Fatalf("failed to read hooks file: %v", err)
	}

	var config CursorHooksConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse hooks file: %v", err)
	}

	if config.Version != 1 {
		t.Errorf("expected version 1, got %d", config.Version)
	}

	// Check that hooks were added
	if len(config.Hooks["sessionStart"]) != 1 {
		t.Errorf("expected 1 sessionStart hook, got %d", len(config.Hooks["sessionStart"]))
	}
	if len(config.Hooks["stop"]) != 1 {
		t.Errorf("expected 1 stop hook, got %d", len(config.Hooks["stop"]))
	}
	if len(config.Hooks["preCompact"]) != 1 {
		t.Errorf("expected 1 preCompact hook, got %d", len(config.Hooks["preCompact"]))
	}
	if len(config.Hooks["postToolUse"]) != 1 {
		t.Errorf("expected 1 postToolUse hook, got %d", len(config.Hooks["postToolUse"]))
	}
	if len(config.Hooks["beforeSubmitPrompt"]) != 1 {
		t.Errorf("expected 1 beforeSubmitPrompt hook, got %d", len(config.Hooks["beforeSubmitPrompt"]))
	}

	// Verify hook commands contain the tasuku marker
	for hookType, hooks := range config.Hooks {
		for _, hook := range hooks {
			if !containsTasukuMarker(hook.Command) {
				t.Errorf("hook %s command should contain tasuku marker: %s", hookType, hook.Command)
			}
		}
	}
}

func TestUninstallCursorHooks(t *testing.T) {
	h := testutil.New(t)

	// Initialize git repo
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Install first (local)
	err := h.Execute(Cmd, "install", "--cursor", "--local")
	h.AssertNoError(err)

	// Then uninstall (local)
	err = h.Execute(Cmd, "uninstall", "--cursor", "--local")
	h.AssertNoError(err)
	h.AssertOutputContains("Removed")

	// Check that file was deleted (since it only had Tasuku hooks)
	hooksFile := filepath.Join(h.TempDir(), ".cursor", "hooks.json")
	if _, err := os.Stat(hooksFile); !os.IsNotExist(err) {
		t.Error("Cursor hooks file should be deleted when empty")
	}
}

func TestCursorHooksIdempotent(t *testing.T) {
	h := testutil.New(t)

	// Initialize git repo
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Install twice (local)
	err := h.Execute(Cmd, "install", "--cursor", "--local")
	h.AssertNoError(err)

	err = h.Execute(Cmd, "install", "--cursor", "--local")
	h.AssertNoError(err)
	h.AssertOutputContains("up to date")

	// Should still have only one hook per type
	hooksFile := filepath.Join(h.TempDir(), ".cursor", "hooks.json")
	data, _ := os.ReadFile(hooksFile)

	var config CursorHooksConfig
	json.Unmarshal(data, &config)

	if len(config.Hooks["sessionStart"]) != 1 {
		t.Errorf("expected 1 sessionStart hook after double install, got %d", len(config.Hooks["sessionStart"]))
	}
}

func TestCursorHooksForceUpdate(t *testing.T) {
	h := testutil.New(t)

	// Initialize git repo
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Create hooks dir and file with old content
	hooksDir := filepath.Join(h.TempDir(), ".cursor")
	os.MkdirAll(hooksDir, 0755)
	oldConfig := CursorHooksConfig{
		Version: 1,
		Hooks: map[string][]CursorHook{
			"sessionStart": {
				{
					Command: "tk hooks session # tasuku-hook-old-version",
					Timeout: 30,
				},
			},
		},
	}
	data, _ := json.Marshal(oldConfig)
	os.WriteFile(filepath.Join(hooksDir, "hooks.json"), data, 0644)

	// Force reinstall (local)
	err := h.Execute(Cmd, "install", "--cursor", "--local", "--force")
	h.AssertNoError(err)

	// Should have updated all hooks
	newData, _ := os.ReadFile(filepath.Join(hooksDir, "hooks.json"))
	var newConfig CursorHooksConfig
	json.Unmarshal(newData, &newConfig)

	// Force should add all hooks, not just update existing
	if len(newConfig.Hooks["stop"]) != 1 {
		t.Error("force install should add all hooks including stop")
	}
	if len(newConfig.Hooks["preCompact"]) != 1 {
		t.Error("force install should add all hooks including preCompact")
	}
}

func TestCursorHooksPreservesOther(t *testing.T) {
	h := testutil.New(t)

	// Initialize git repo
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Create hooks dir and file with a non-Tasuku hook
	hooksDir := filepath.Join(h.TempDir(), ".cursor")
	os.MkdirAll(hooksDir, 0755)
	existingConfig := CursorHooksConfig{
		Version: 1,
		Hooks: map[string][]CursorHook{
			"sessionStart": {
				{
					Command: "echo 'other hook'",
					Timeout: 10,
				},
			},
		},
	}
	data, _ := json.Marshal(existingConfig)
	os.WriteFile(filepath.Join(hooksDir, "hooks.json"), data, 0644)

	// Install Tasuku hooks (local)
	err := h.Execute(Cmd, "install", "--cursor", "--local")
	h.AssertNoError(err)

	// Should have both hooks
	newData, _ := os.ReadFile(filepath.Join(hooksDir, "hooks.json"))
	var newConfig CursorHooksConfig
	json.Unmarshal(newData, &newConfig)

	if len(newConfig.Hooks["sessionStart"]) != 2 {
		t.Errorf("expected 2 sessionStart hooks (original + Tasuku), got %d", len(newConfig.Hooks["sessionStart"]))
	}
}

func TestUninstallCursorHooksPreservesOther(t *testing.T) {
	h := testutil.New(t)

	// Initialize git repo
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Create hooks dir and file with both Tasuku and non-Tasuku hooks
	hooksDir := filepath.Join(h.TempDir(), ".cursor")
	os.MkdirAll(hooksDir, 0755)
	existingConfig := CursorHooksConfig{
		Version: 1,
		Hooks: map[string][]CursorHook{
			"sessionStart": {
				{
					Command: "echo 'other hook'",
					Timeout: 10,
				},
				{
					Command: "tk hooks session # tasuku-hook",
					Timeout: 30,
				},
			},
		},
	}
	data, _ := json.Marshal(existingConfig)
	os.WriteFile(filepath.Join(hooksDir, "hooks.json"), data, 0644)

	// Uninstall Tasuku hooks (local)
	err := h.Execute(Cmd, "uninstall", "--cursor", "--local")
	h.AssertNoError(err)

	// Should still have the non-Tasuku hook
	newData, _ := os.ReadFile(filepath.Join(hooksDir, "hooks.json"))
	var newConfig CursorHooksConfig
	json.Unmarshal(newData, &newConfig)

	if len(newConfig.Hooks["sessionStart"]) != 1 {
		t.Errorf("expected 1 sessionStart hook after uninstall, got %d", len(newConfig.Hooks["sessionStart"]))
	}
	if newConfig.Hooks["sessionStart"][0].Command != "echo 'other hook'" {
		t.Error("non-Tasuku hook should be preserved")
	}
}

func TestCursorHooksLocalPath(t *testing.T) {
	h := testutil.New(t)

	// Initialize git repo
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Install with --local flag
	err := h.Execute(Cmd, "install", "--cursor", "--local")
	h.AssertNoError(err)

	// Hooks should be at .cursor/hooks.json (local)
	localHooksFile := filepath.Join(h.TempDir(), ".cursor", "hooks.json")
	if _, err := os.Stat(localHooksFile); os.IsNotExist(err) {
		t.Error("local Cursor hooks file should exist at .cursor/hooks.json")
	}
}

func TestCursorHooksNoCursorDir(t *testing.T) {
	h := testutil.New(t)

	// Initialize git repo but don't create .cursor directory
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Should create .cursor directory automatically (local)
	err := h.Execute(Cmd, "install", "--cursor", "--local")
	h.AssertNoError(err)

	// Check .cursor was created
	cursorDir := filepath.Join(h.TempDir(), ".cursor")
	if _, err := os.Stat(cursorDir); os.IsNotExist(err) {
		t.Error(".cursor directory should be created")
	}
}

func TestUninstallCursorHooksNoFile(t *testing.T) {
	h := testutil.New(t)

	// Initialize git repo
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Uninstall when no hooks file exists should not error (local)
	err := h.Execute(Cmd, "uninstall", "--cursor", "--local")
	h.AssertNoError(err)
	h.AssertOutputContains("No Cursor hooks file found")
}

func TestCursorHookUsesCommandNotBash(t *testing.T) {
	h := testutil.New(t)

	// Initialize git repo
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Install Cursor hooks (local)
	err := h.Execute(Cmd, "install", "--cursor", "--local")
	h.AssertNoError(err)

	// Read and verify the JSON structure uses "command" not "bash"
	hooksFile := filepath.Join(h.TempDir(), ".cursor", "hooks.json")
	data, err := os.ReadFile(hooksFile)
	if err != nil {
		t.Fatalf("failed to read hooks file: %v", err)
	}

	// Verify raw JSON contains "command" key, not "bash"
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	hooks := raw["hooks"].(map[string]interface{})
	for hookType, hookList := range hooks {
		entries := hookList.([]interface{})
		for _, entry := range entries {
			hookMap := entry.(map[string]interface{})
			if _, ok := hookMap["command"]; !ok {
				t.Errorf("hook %s should have 'command' field", hookType)
			}
			if _, ok := hookMap["bash"]; ok {
				t.Errorf("hook %s should NOT have 'bash' field (Cursor uses 'command')", hookType)
			}
		}
	}
}

func TestCursorHooksGlobalPath(t *testing.T) {
	// Test that global path resolves to ~/.cursor/hooks.json
	globalPath := getCursorHooksPath(false)
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".cursor", "hooks.json")
	if globalPath != expected {
		t.Errorf("global path = %q, want %q", globalPath, expected)
	}

	// Test that local path resolves to .cursor/hooks.json
	localPath := getCursorHooksPath(true)
	expectedLocal := filepath.Join(".cursor", "hooks.json")
	if localPath != expectedLocal {
		t.Errorf("local path = %q, want %q", localPath, expectedLocal)
	}
}
