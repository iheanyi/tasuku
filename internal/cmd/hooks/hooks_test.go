package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
	"github.com/iheanyi/tasuku/internal/task"
)

func TestSessionCmd(t *testing.T) {
	h := testutil.New(t)

	// Add some tasks
	h.AddTaskWithStatus("task-1", "Ready task", task.StatusReady)
	h.AddTaskWithStatus("task-2", "In progress task", task.StatusInProgress)
	h.AddTaskWithStatus("task-3", "Done task", task.StatusDone)
	h.AddLearning("Test learning")
	h.AddDecision("test-dec", "A", []string{"B"}, "reason")

	err := h.Execute(Cmd, "session")
	h.AssertNoError(err)
	h.AssertOutputContains("Tasuku Context")
	h.AssertOutputContains("Tasks:")
	h.AssertOutputContains("ready")
	h.AssertOutputContains("in_progress")
	h.AssertOutputContains("done")
	h.AssertOutputContains("Learnings:")
	h.AssertOutputContains("Decisions:")
}

func TestSessionCmdStaleTimerWarning(t *testing.T) {
	h := testutil.New(t)

	// Add a task with a stale timer (started 5 hours ago)
	h.AddTaskWithStatus("stale-timer-task", "Task with stale timer", task.StatusInProgress)
	staleTime := time.Now().Add(-5 * time.Hour)
	h.StartTimerAt("stale-timer-task", staleTime)

	err := h.Execute(Cmd, "session")
	h.AssertNoError(err)
	h.AssertOutputContains("Stale timers")
	h.AssertOutputContains("stale-timer-task")
	h.AssertOutputContains("tk task timer stop")
}

func TestSessionCmdNoStaleTimerWarning(t *testing.T) {
	h := testutil.New(t)

	// Add a task with a recent timer (started 1 hour ago - not stale)
	h.AddTaskWithStatus("recent-timer-task", "Task with recent timer", task.StatusInProgress)
	recentTime := time.Now().Add(-1 * time.Hour)
	h.StartTimerAt("recent-timer-task", recentTime)

	err := h.Execute(Cmd, "session")
	h.AssertNoError(err)
	h.AssertOutputNotContains("Stale timers")
}

func TestSessionCmdNoStorage(t *testing.T) {
	// Create temp dir without tasuku storage
	tempDir, _ := os.MkdirTemp("", "tasuku-test-nostorage-*")
	defer os.RemoveAll(tempDir)

	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Session should succeed silently when no storage exists
	err := sessionCmd.RunE(sessionCmd, []string{})
	if err != nil {
		t.Errorf("session should not error when no storage, got: %v", err)
	}
}

func TestSyncCmdNoStorage(t *testing.T) {
	// Create temp dir without tasuku storage
	tempDir, _ := os.MkdirTemp("", "tasuku-test-nostorage-*")
	defer os.RemoveAll(tempDir)

	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Sync should error when no storage exists
	err := syncCmd.RunE(syncCmd, []string{})
	if err == nil || !strings.Contains(err.Error(), "no Tasuku storage") {
		t.Errorf("expected 'no Tasuku storage' error, got: %v", err)
	}
}

func TestInstallHooksNoGit(t *testing.T) {
	h := testutil.New(t)

	// The temp dir is not a git repo
	err := h.Execute(Cmd, "install")
	h.AssertError(err)
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("expected 'not a git repository' error, got: %v", err)
	}
}

func TestInstallAndUninstallHooks(t *testing.T) {
	h := testutil.New(t)

	// Initialize a git repo in the temp dir
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	err := h.Execute(Cmd, "install")
	h.AssertNoError(err)
	h.AssertOutputContains("Git hooks installed")

	// Check hooks were created
	preCommit := filepath.Join(h.TempDir(), ".git", "hooks", "pre-commit")
	if _, err := os.Stat(preCommit); os.IsNotExist(err) {
		t.Error("pre-commit hook should exist")
	}
	postCommit := filepath.Join(h.TempDir(), ".git", "hooks", "post-commit")
	if _, err := os.Stat(postCommit); os.IsNotExist(err) {
		t.Error("post-commit hook should exist")
	}

	// Uninstall
	err = h.Execute(Cmd, "uninstall")
	h.AssertNoError(err)
	h.AssertOutputContains("uninstalled")
}

func TestInstallHooksPreservesExisting(t *testing.T) {
	h := testutil.New(t)

	// Initialize git hooks dir
	hooksDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(hooksDir, 0755)

	// Create existing pre-commit hook
	existingContent := "#!/bin/bash\necho 'existing hook'"
	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	os.WriteFile(preCommitPath, []byte(existingContent), 0755)

	// Install tasuku hooks
	err := h.Execute(Cmd, "install")
	h.AssertNoError(err)

	// Check that existing content is preserved
	content, _ := os.ReadFile(preCommitPath)
	if !strings.Contains(string(content), "existing hook") {
		t.Error("existing hook content should be preserved")
	}
	if !strings.Contains(string(content), "TASUKU HOOK") {
		t.Error("tasuku hook should be added")
	}
}

func TestGenerateID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple Task", "simple-task"},
		{"Task with CAPS", "task-with-caps"},
		{"Task with special chars!@#", "task-with-special-chars"},
		{"  Leading and trailing spaces  ", "leading-and-trailing-spaces"},
		{"Multiple   spaces", "multiple-spaces"},
	}

	for _, tt := range tests {
		got := generateID(tt.input)
		if got != tt.expected {
			t.Errorf("generateID(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestShouldPersistTask(t *testing.T) {
	tests := []struct {
		description   string
		shouldPersist bool
	}{
		{"Implement user authentication", true},
		{"Add feature for dark mode", true},
		{"Fix bug in login flow", true},
		{"Refactor database layer", true},
		{"fix type error in UserService", false},
		{"Update file with new config", false},
		{"Run tests for coverage", false},
		{"Add comment explaining logic", false},
		{"Rename variable to be clearer", false},
	}

	for _, tt := range tests {
		got := shouldPersistTask(tt.description)
		if got != tt.shouldPersist {
			t.Errorf("shouldPersistTask(%q) = %v, want %v", tt.description, got, tt.shouldPersist)
		}
	}
}

func TestTestFailureStateTracking(t *testing.T) {
	// Save original directory and change to temp dir
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Create .tasuku directory for local state storage
	os.MkdirAll(".tasuku", 0755)

	// Clean up any existing state
	clearTestFailureState()

	// Initially, no failure should be detected
	if isRecentTestFailure(30 * time.Minute) {
		t.Error("expected no recent failure initially")
	}

	// Save a test failure
	if err := saveTestFailureState("go test ./..."); err != nil {
		t.Fatalf("failed to save test failure state: %v", err)
	}

	// Now should detect recent failure
	if !isRecentTestFailure(30 * time.Minute) {
		t.Error("expected recent failure after saving state")
	}

	// Read the state
	state, err := getLastTestFailure()
	if err != nil {
		t.Fatalf("failed to get last test failure: %v", err)
	}
	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.Command != "go test ./..." {
		t.Errorf("expected command 'go test ./...', got %q", state.Command)
	}

	// Clear the state
	clearTestFailureState()

	// Should no longer detect failure
	if isRecentTestFailure(30 * time.Minute) {
		t.Error("expected no recent failure after clearing state")
	}
}

func TestDetectTestSuccess(t *testing.T) {
	tests := []struct {
		output  string
		success bool
	}{
		{"PASS\nok  github.com/example/pkg 0.123s", true},
		{"Tests passed!", true},
		{"All tests passed", true},
		{"exit status 0", true},
		{"0 failures, 10 successes", true},
		{"FAIL\nFailed: TestSomething", false},
		{"Error: assertion failed", false},
		{"Some random output", false},
	}

	for _, tt := range tests {
		got := detectTestSuccess(strings.ToLower(tt.output))
		if got != tt.success {
			t.Errorf("detectTestSuccess(%q) = %v, want %v", tt.output, got, tt.success)
		}
	}
}

