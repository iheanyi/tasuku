package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Integration tests for the CLI

func TestGenerateID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Fix authentication bug", "fix-authentication-bug"},
		{"Add logout button", "add-logout-button"},
		{"UPPERCASE TEST", "uppercase-test"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"trailing space ", "trailing-space"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := generateID(tt.input)
			if result != tt.expected {
				t.Errorf("generateID(%q) = %q, expected %q",
					tt.input, result, tt.expected)
			}
		})
	}
}

func TestCLI_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build the binary
	dir := t.TempDir()
	binary := filepath.Join(dir, "tk")

	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = "."
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, output)
	}

	// Create test directory with .tasuku.json
	testDir := filepath.Join(dir, "project")
	os.MkdirAll(testDir, 0755)

	// Helper to run tk commands
	runTk := func(args ...string) (string, error) {
		cmd := exec.Command(binary, args...)
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	// Test init
	t.Run("init", func(t *testing.T) {
		output, err := runTk("init")
		if err != nil {
			t.Fatalf("init failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Created .tasuku.json") {
			t.Errorf("unexpected output: %s", output)
		}

		// Verify file exists
		if _, err := os.Stat(filepath.Join(testDir, ".tasuku.json")); err != nil {
			t.Error("expected .tasuku.json to exist")
		}
	})

	// Test add
	t.Run("add", func(t *testing.T) {
		output, err := runTk("add", "Test task")
		if err != nil {
			t.Fatalf("add failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Created task:") {
			t.Errorf("unexpected output: %s", output)
		}
	})

	// Test add with id (flag must come before positional arg)
	t.Run("add-with-id", func(t *testing.T) {
		output, err := runTk("add", "--id", "custom-task", "Another task")
		if err != nil {
			t.Fatalf("add with id failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "custom-task") {
			t.Errorf("expected custom-task in output: %s", output)
		}
	})

	// Test list
	t.Run("list", func(t *testing.T) {
		output, err := runTk("list")
		if err != nil {
			t.Fatalf("list failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "test-task") && !strings.Contains(output, "Test task") {
			t.Errorf("expected task in list: %s", output)
		}
	})

	// Test start
	t.Run("start", func(t *testing.T) {
		output, err := runTk("start", "custom-task")
		if err != nil {
			t.Fatalf("start failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Started:") {
			t.Errorf("unexpected output: %s", output)
		}
	})

	// Test done
	t.Run("done", func(t *testing.T) {
		output, err := runTk("done", "custom-task")
		if err != nil {
			t.Fatalf("done failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Completed:") {
			t.Errorf("unexpected output: %s", output)
		}
	})

	// Test learn
	t.Run("learn", func(t *testing.T) {
		output, err := runTk("learn", "Test insight")
		if err != nil {
			t.Fatalf("learn failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Learning added") {
			t.Errorf("unexpected output: %s", output)
		}
	})

	// Test decide
	t.Run("decide", func(t *testing.T) {
		output, err := runTk("decide",
			"--id", "test-decision",
			"--chose", "Option A",
			"--over", "Option B,Option C",
			"--because", "It's simpler")
		if err != nil {
			t.Fatalf("decide failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Decision recorded") {
			t.Errorf("unexpected output: %s", output)
		}
	})

	// Test context
	t.Run("context", func(t *testing.T) {
		output, err := runTk("context")
		if err != nil {
			t.Fatalf("context failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "version") {
			t.Errorf("expected JSON output: %s", output)
		}
		if !strings.Contains(output, "Test insight") {
			t.Errorf("expected learning in context: %s", output)
		}
	})

	// Test validate
	t.Run("validate", func(t *testing.T) {
		output, err := runTk("validate")
		if err != nil {
			t.Fatalf("validate failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Validation passed") {
			t.Errorf("unexpected output: %s", output)
		}
	})

	// Test help
	t.Run("help", func(t *testing.T) {
		output, err := runTk("help")
		if err != nil {
			t.Fatalf("help failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "agent-first task management") {
			t.Errorf("unexpected help output: %s", output)
		}
	})

	// Test version
	t.Run("version", func(t *testing.T) {
		output, err := runTk("--version")
		if err != nil {
			t.Fatalf("version failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "tk version") {
			t.Errorf("unexpected version output: %s", output)
		}
	})
}

func TestBeadsMigration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	binary := filepath.Join(dir, "tk")

	cmd := exec.Command("go", "build", "-o", binary, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, output)
	}

	testDir := filepath.Join(dir, "project")
	os.MkdirAll(testDir, 0755)

	runTk := func(args ...string) (string, error) {
		cmd := exec.Command(binary, args...)
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	// Create .beads directory with issues.jsonl
	beadsDir := filepath.Join(testDir, ".beads")
	os.MkdirAll(beadsDir, 0755)

	// Create sample issues.jsonl in Beads format
	issuesJSONL := `{"id":"bd-a1b2","title":"Fix authentication bug","description":"Users can't log in","status":"open","priority":1,"created_at":"2024-01-01T10:00:00Z","updated_at":"2024-01-02T12:00:00Z"}
{"id":"bd-c3d4","title":"Add logout button","status":"in_progress","priority":2,"created_at":"2024-01-03T10:00:00Z","updated_at":"2024-01-03T14:00:00Z","assignee":"agent-1"}
{"id":"bd-e5f6","title":"Completed task","status":"closed","priority":0,"created_at":"2024-01-04T10:00:00Z","updated_at":"2024-01-05T10:00:00Z","close_reason":"Fixed in commit abc123"}
{"id":"bd-g7h8","title":"Blocked task","status":"blocked","priority":3,"created_at":"2024-01-06T10:00:00Z","updated_at":"2024-01-06T10:00:00Z","dependencies":[{"type":"blocked_by","target_id":"bd-a1b2"}]}
`
	os.WriteFile(filepath.Join(beadsDir, "issues.jsonl"), []byte(issuesJSONL), 0644)

	// Test dry-run (flags must come before positional args)
	t.Run("migrate-dry-run", func(t *testing.T) {
		output, err := runTk("migrate", "--dry-run", "beads")
		if err != nil {
			t.Fatalf("migrate dry-run failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Found 4 issues") {
			t.Errorf("expected 4 issues, got: %s", output)
		}
		if !strings.Contains(output, "bd-a1b2") {
			t.Errorf("expected bd-a1b2 in output: %s", output)
		}
	})

	// Test actual migration
	t.Run("migrate", func(t *testing.T) {
		output, err := runTk("migrate", "beads")
		if err != nil {
			t.Fatalf("migrate failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Migration complete: 4 tasks imported") {
			t.Errorf("expected 4 tasks migrated: %s", output)
		}
	})

	// Verify migrated tasks
	t.Run("verify-list", func(t *testing.T) {
		output, err := runTk("list")
		if err != nil {
			t.Fatalf("list failed: %v\n%s", err, output)
		}
		// Check all tasks are present
		if !strings.Contains(output, "bd-a1b2") {
			t.Errorf("expected bd-a1b2: %s", output)
		}
		if !strings.Contains(output, "bd-c3d4") {
			t.Errorf("expected bd-c3d4: %s", output)
		}
		if !strings.Contains(output, "bd-e5f6") {
			t.Errorf("expected bd-e5f6: %s", output)
		}
		if !strings.Contains(output, "bd-g7h8") {
			t.Errorf("expected bd-g7h8: %s", output)
		}
	})

	// Verify status mapping
	t.Run("verify-statuses", func(t *testing.T) {
		output, err := runTk("list", "--format", "json")
		if err != nil {
			t.Fatalf("list --format json failed: %v\n%s", err, output)
		}
		// open -> ready (JSON has space after colon)
		if !strings.Contains(output, `"status": "ready"`) {
			t.Errorf("expected ready status: %s", output)
		}
		// in_progress -> in_progress
		if !strings.Contains(output, `"status": "in_progress"`) {
			t.Errorf("expected in_progress status: %s", output)
		}
		// closed -> done
		if !strings.Contains(output, `"status": "done"`) {
			t.Errorf("expected done status: %s", output)
		}
		// blocked -> blocked
		if !strings.Contains(output, `"status": "blocked"`) {
			t.Errorf("expected blocked status: %s", output)
		}
	})

	// Verify priority mapping
	t.Run("verify-ready-priority", func(t *testing.T) {
		output, err := runTk("ready")
		if err != nil {
			t.Fatalf("ready failed: %v\n%s", err, output)
		}
		// Should show bd-a1b2 with high priority first
		if !strings.Contains(output, "high") || !strings.Contains(output, "bd-a1b2") {
			t.Errorf("expected high priority task: %s", output)
		}
	})

	// Verify notes migration
	t.Run("verify-notes", func(t *testing.T) {
		output, err := runTk("show", "bd-e5f6")
		if err != nil {
			t.Fatalf("show failed: %v\n%s", err, output)
		}
		// Should have close reason as note
		if !strings.Contains(output, "Close reason") {
			t.Errorf("expected close reason in notes: %s", output)
		}
	})

	// Verify blocked_by migration from Beads dependencies
	t.Run("verify-blocked-by", func(t *testing.T) {
		output, err := runTk("show", "bd-g7h8")
		if err != nil {
			t.Fatalf("show bd-g7h8 failed: %v\n%s", err, output)
		}
		// bd-g7h8 was blocked by bd-a1b2 in the Beads data
		if !strings.Contains(output, "bd-a1b2") {
			t.Errorf("expected blocked_by bd-a1b2 in output: %s", output)
		}
	})
}

func TestCLI_Errors(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	binary := filepath.Join(dir, "tk")

	cmd := exec.Command("go", "build", "-o", binary, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, output)
	}

	testDir := filepath.Join(dir, "project")
	os.MkdirAll(testDir, 0755)

	runTk := func(args ...string) (string, error) {
		cmd := exec.Command(binary, args...)
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	// Test error: no .tasuku.json
	t.Run("no-tasuku-file", func(t *testing.T) {
		_, err := runTk("list")
		if err == nil {
			t.Error("expected error when no .tasuku.json exists")
		}
	})

	// Initialize for remaining tests
	runTk("init")

	// Test error: unknown command
	t.Run("unknown-command", func(t *testing.T) {
		output, err := runTk("foobar")
		if err == nil {
			t.Error("expected error for unknown command")
		}
		if !strings.Contains(output, "unknown command") {
			t.Errorf("expected 'unknown command' in output: %s", output)
		}
	})

	// Test error: task not found
	t.Run("task-not-found", func(t *testing.T) {
		output, err := runTk("start", "nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent task")
		}
		if !strings.Contains(output, "not found") {
			t.Errorf("expected 'not found' in output: %s", output)
		}
	})
}

func TestLearningsCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	binary := filepath.Join(dir, "tk")

	cmd := exec.Command("go", "build", "-o", binary, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, output)
	}

	testDir := filepath.Join(dir, "project")
	os.MkdirAll(testDir, 0755)

	runTk := func(args ...string) (string, error) {
		cmd := exec.Command(binary, args...)
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	// Initialize
	runTk("init")

	// Test learnings list (empty)
	t.Run("learnings-empty", func(t *testing.T) {
		output, err := runTk("learnings")
		if err != nil {
			t.Fatalf("learnings failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "No learnings recorded") {
			t.Errorf("expected empty message: %s", output)
		}
	})

	// Add some learnings
	t.Run("learn-add", func(t *testing.T) {
		output, err := runTk("learn", "Redis caching improves performance")
		if err != nil {
			t.Fatalf("learn failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Learning added") {
			t.Errorf("expected confirmation: %s", output)
		}

		// Add another
		runTk("learn", "JWT tokens expire after 24 hours")
		runTk("learn", "Database indexes are critical for queries")
	})

	// Test learnings list (with items)
	t.Run("learnings-list", func(t *testing.T) {
		output, err := runTk("learnings")
		if err != nil {
			t.Fatalf("learnings failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Redis") {
			t.Errorf("expected Redis learning: %s", output)
		}
		if !strings.Contains(output, "JWT") {
			t.Errorf("expected JWT learning: %s", output)
		}
		if !strings.Contains(output, "Learnings (3)") {
			t.Errorf("expected 3 learnings: %s", output)
		}
	})

	// Test learnings JSON format
	t.Run("learnings-json", func(t *testing.T) {
		output, err := runTk("learnings", "--format", "json")
		if err != nil {
			t.Fatalf("learnings --format json failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
			t.Errorf("expected JSON array: %s", output)
		}
	})

	// Test unlearn by index
	t.Run("unlearn-by-index", func(t *testing.T) {
		output, err := runTk("unlearn", "2")
		if err != nil {
			t.Fatalf("unlearn failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Removed") {
			t.Errorf("expected removed message: %s", output)
		}
		if !strings.Contains(output, "JWT") {
			t.Errorf("expected JWT in removed: %s", output)
		}

		// Verify it's gone
		output, _ = runTk("learnings")
		if strings.Contains(output, "JWT") {
			t.Errorf("JWT should be removed: %s", output)
		}
		if !strings.Contains(output, "Learnings (2)") {
			t.Errorf("expected 2 learnings: %s", output)
		}
	})

	// Test unlearn by text match
	t.Run("unlearn-by-text", func(t *testing.T) {
		output, err := runTk("unlearn", "redis")
		if err != nil {
			t.Fatalf("unlearn by text failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Redis") {
			t.Errorf("expected Redis in removed: %s", output)
		}
	})

	// Test unlearn not found
	t.Run("unlearn-not-found", func(t *testing.T) {
		_, err := runTk("unlearn", "nonexistent-learning")
		if err == nil {
			t.Error("expected error for nonexistent learning")
		}
	})

	// Test unlearn index out of range
	t.Run("unlearn-out-of-range", func(t *testing.T) {
		_, err := runTk("unlearn", "999")
		if err == nil {
			t.Error("expected error for out of range index")
		}
	})
}

func TestPromoteCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	binary := filepath.Join(dir, "tk")

	cmd := exec.Command("go", "build", "-o", binary, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, output)
	}

	testDir := filepath.Join(dir, "project")
	os.MkdirAll(testDir, 0755)

	runTk := func(args ...string) (string, error) {
		cmd := exec.Command(binary, args...)
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	// Initialize and add learning
	runTk("init")
	runTk("learn", "Test learning for promotion")

	// Test promote (auto-detects CLAUDE.md)
	t.Run("promote-default", func(t *testing.T) {
		output, err := runTk("promote", "1")
		if err != nil {
			t.Fatalf("promote failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Promoted to CLAUDE.md") {
			t.Errorf("expected promotion message: %s", output)
		}

		// Verify CLAUDE.md was created
		content, err := os.ReadFile(filepath.Join(testDir, "CLAUDE.md"))
		if err != nil {
			t.Fatalf("failed to read CLAUDE.md: %v", err)
		}
		if !strings.Contains(string(content), "Test learning") {
			t.Errorf("expected learning in CLAUDE.md: %s", string(content))
		}

		// Verify learning was removed from .tasuku.json
		output, _ = runTk("learnings")
		if strings.Contains(output, "Test learning") {
			t.Errorf("learning should be removed from .tasuku.json: %s", output)
		}
	})

	// Test promote with --keep
	t.Run("promote-with-keep", func(t *testing.T) {
		runTk("learn", "Another learning to keep")
		output, err := runTk("promote", "1", "--keep")
		if err != nil {
			t.Fatalf("promote --keep failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "kept in learnings") {
			t.Errorf("expected kept message: %s", output)
		}

		// Verify learning still exists in .tasuku.json
		output, _ = runTk("learnings")
		if !strings.Contains(output, "Another learning") {
			t.Errorf("learning should still be in .tasuku.json: %s", output)
		}
	})

	// Test promote to custom file
	t.Run("promote-to-custom", func(t *testing.T) {
		runTk("learn", "Custom file learning")
		output, err := runTk("promote", "custom", "--to", "AGENTS.md")
		if err != nil {
			t.Fatalf("promote --to failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Promoted to AGENTS.md") {
			t.Errorf("expected custom file message: %s", output)
		}

		// Verify AGENTS.md was created
		content, err := os.ReadFile(filepath.Join(testDir, "AGENTS.md"))
		if err != nil {
			t.Fatalf("failed to read AGENTS.md: %v", err)
		}
		if !strings.Contains(string(content), "Custom file learning") {
			t.Errorf("expected learning in AGENTS.md: %s", string(content))
		}
	})

	// Test promote not found
	t.Run("promote-not-found", func(t *testing.T) {
		_, err := runTk("promote", "nonexistent-learning")
		if err == nil {
			t.Error("expected error for nonexistent learning")
		}
	})
}

func TestCircularDependencyDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	binary := filepath.Join(dir, "tk")

	cmd := exec.Command("go", "build", "-o", binary, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, output)
	}

	testDir := filepath.Join(dir, "project")
	os.MkdirAll(testDir, 0755)

	runTk := func(args ...string) (string, error) {
		cmd := exec.Command(binary, args...)
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	// Initialize
	runTk("init")

	// Create tasks that form a circular dependency: A -> B -> C -> A
	runTk("add", "--id", "task-a", "Task A")
	runTk("add", "--id", "task-b", "Task B")
	runTk("add", "--id", "task-c", "Task C")

	// Set up circular dependencies: A blocked by B, B blocked by C, C blocked by A
	runTk("block", "task-a", "--by", "task-b")
	runTk("block", "task-b", "--by", "task-c")
	runTk("block", "task-c", "--by", "task-a")

	// Validate should detect circular dependency
	t.Run("circular-dependency-detected", func(t *testing.T) {
		output, err := runTk("validate")
		if err == nil {
			t.Error("expected validation to fail with circular dependency")
		}
		if !strings.Contains(output, "Circular dependencies detected") {
			t.Errorf("expected 'Circular dependencies detected' in output: %s", output)
		}
		// The cycle should contain task-a, task-b, task-c
		if !strings.Contains(output, "task-a") || !strings.Contains(output, "task-b") || !strings.Contains(output, "task-c") {
			t.Errorf("expected all tasks in cycle output: %s", output)
		}
	})
}

func TestPartialUnblock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	binary := filepath.Join(dir, "tk")

	cmd := exec.Command("go", "build", "-o", binary, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, output)
	}

	testDir := filepath.Join(dir, "project")
	os.MkdirAll(testDir, 0755)

	runTk := func(args ...string) (string, error) {
		cmd := exec.Command(binary, args...)
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	// Initialize
	runTk("init")

	// Create tasks
	runTk("add", "--id", "main-task", "Main task")
	runTk("add", "--id", "blocker-1", "Blocker 1")
	runTk("add", "--id", "blocker-2", "Blocker 2")

	// Block main-task by both blockers
	runTk("block", "main-task", "--by", "blocker-1,blocker-2")

	// Test partial unblock
	t.Run("partial-unblock", func(t *testing.T) {
		output, err := runTk("unblock", "main-task", "--from", "blocker-1")
		if err != nil {
			t.Fatalf("partial unblock failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "removed blocker: blocker-1") {
			t.Errorf("expected partial unblock message: %s", output)
		}

		// Verify blocker-2 is still blocking
		showOutput, _ := runTk("show", "main-task")
		if !strings.Contains(showOutput, "blocker-2") {
			t.Errorf("expected blocker-2 to still be blocking: %s", showOutput)
		}
		if strings.Contains(showOutput, "blocker-1") {
			t.Errorf("blocker-1 should be removed: %s", showOutput)
		}
	})

	// Test unblock all
	t.Run("unblock-all", func(t *testing.T) {
		output, err := runTk("unblock", "main-task")
		if err != nil {
			t.Fatalf("unblock all failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "removed all blockers") {
			t.Errorf("expected unblock all message: %s", output)
		}

		// Verify no more blockers
		showOutput, _ := runTk("show", "main-task")
		if strings.Contains(showOutput, "Blocked by") {
			t.Errorf("expected no blockers: %s", showOutput)
		}
	})

	// Test error when blocker doesn't exist
	t.Run("blocker-not-found", func(t *testing.T) {
		runTk("block", "main-task", "--by", "blocker-1")
		_, err := runTk("unblock", "main-task", "--from", "nonexistent")
		if err == nil {
			t.Error("expected error when blocker doesn't exist")
		}
	})
}

func TestStatsCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	binary := filepath.Join(dir, "tk")

	cmd := exec.Command("go", "build", "-o", binary, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, output)
	}

	testDir := filepath.Join(dir, "project")
	os.MkdirAll(testDir, 0755)

	runTk := func(args ...string) (string, error) {
		cmd := exec.Command(binary, args...)
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	// Test stats on empty task list
	t.Run("stats-empty", func(t *testing.T) {
		// Initialize fresh for this test
		runTk("init")

		output, err := runTk("stats")
		if err != nil {
			t.Fatalf("stats failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Total tasks:     0") {
			t.Errorf("expected 0 total tasks: %s", output)
		}
		if !strings.Contains(output, "Completion:      0%") {
			t.Errorf("expected 0%% completion: %s", output)
		}
		if !strings.Contains(output, "Learnings:     0") {
			t.Errorf("expected 0 learnings: %s", output)
		}
		if !strings.Contains(output, "Decisions:     0") {
			t.Errorf("expected 0 decisions: %s", output)
		}
	})

	// Test stats with tasks in various states
	t.Run("stats-with-tasks", func(t *testing.T) {
		// Add tasks in different states
		runTk("add", "--id", "task-ready-1", "Ready task 1")
		runTk("add", "--id", "task-ready-2", "Ready task 2")
		runTk("add", "--id", "task-in-progress", "In progress task")
		runTk("start", "task-in-progress")
		runTk("add", "--id", "task-done-1", "Done task 1")
		runTk("done", "task-done-1")
		runTk("add", "--id", "task-done-2", "Done task 2")
		runTk("done", "task-done-2")
		runTk("add", "--id", "task-blocked", "Blocked task")
		runTk("block", "task-blocked", "--by", "task-ready-1")

		output, err := runTk("stats")
		if err != nil {
			t.Fatalf("stats failed: %v\n%s", err, output)
		}

		// Total should be 6
		if !strings.Contains(output, "Total tasks:     6") {
			t.Errorf("expected 6 total tasks: %s", output)
		}
		// Ready: 2
		if !strings.Contains(output, "Ready:         2") {
			t.Errorf("expected 2 ready tasks: %s", output)
		}
		// In Progress: 1
		if !strings.Contains(output, "In Progress:   1") {
			t.Errorf("expected 1 in progress task: %s", output)
		}
		// Done: 2
		if !strings.Contains(output, "Done:          2") {
			t.Errorf("expected 2 done tasks: %s", output)
		}
		// Blocked: 1
		if !strings.Contains(output, "Blocked:       1") {
			t.Errorf("expected 1 blocked task: %s", output)
		}
	})

	// Test stats with JSON format
	t.Run("stats-json", func(t *testing.T) {
		output, err := runTk("stats", "--format", "json")
		if err != nil {
			t.Fatalf("stats --format json failed: %v\n%s", err, output)
		}

		// Verify JSON structure
		if !strings.Contains(output, `"total_tasks":`) {
			t.Errorf("expected total_tasks in JSON: %s", output)
		}
		if !strings.Contains(output, `"by_status":`) {
			t.Errorf("expected by_status in JSON: %s", output)
		}
		if !strings.Contains(output, `"completion_percent":`) {
			t.Errorf("expected completion_percent in JSON: %s", output)
		}
		if !strings.Contains(output, `"learnings_count":`) {
			t.Errorf("expected learnings_count in JSON: %s", output)
		}
		if !strings.Contains(output, `"decisions_count":`) {
			t.Errorf("expected decisions_count in JSON: %s", output)
		}
		// Check it starts and ends with braces (valid JSON object)
		trimmed := strings.TrimSpace(output)
		if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
			t.Errorf("expected valid JSON object: %s", output)
		}
	})

	// Test stats with YAML format
	t.Run("stats-yaml", func(t *testing.T) {
		output, err := runTk("stats", "--format", "yaml")
		if err != nil {
			t.Fatalf("stats --format yaml failed: %v\n%s", err, output)
		}

		// Verify YAML structure (keys without quotes, indentation)
		if !strings.Contains(output, "total_tasks:") {
			t.Errorf("expected total_tasks in YAML: %s", output)
		}
		if !strings.Contains(output, "by_status:") {
			t.Errorf("expected by_status in YAML: %s", output)
		}
		if !strings.Contains(output, "completion_percent:") {
			t.Errorf("expected completion_percent in YAML: %s", output)
		}
		if !strings.Contains(output, "learnings_count:") {
			t.Errorf("expected learnings_count in YAML: %s", output)
		}
		if !strings.Contains(output, "decisions_count:") {
			t.Errorf("expected decisions_count in YAML: %s", output)
		}
		// YAML shouldn't have braces like JSON
		if strings.Contains(output, `"total_tasks":`) {
			t.Errorf("YAML shouldn't have quoted keys like JSON: %s", output)
		}
	})

	// Test completion percentage calculation
	t.Run("stats-completion-percentage", func(t *testing.T) {
		output, err := runTk("stats")
		if err != nil {
			t.Fatalf("stats failed: %v\n%s", err, output)
		}

		// With 2 done out of 6 total: 2/6 = 33%
		if !strings.Contains(output, "Completion:      33% (2/6)") {
			t.Errorf("expected 33%% completion (2/6): %s", output)
		}
	})

	// Test stats with context (learnings and decisions)
	t.Run("stats-with-context", func(t *testing.T) {
		// Add learnings
		runTk("learn", "First insight about the project")
		runTk("learn", "Second insight about the project")
		runTk("learn", "Third insight about the project")

		// Add decisions
		runTk("decide",
			"--id", "decision-1",
			"--chose", "Option A",
			"--over", "Option B",
			"--because", "Better performance")
		runTk("decide",
			"--id", "decision-2",
			"--chose", "JSON",
			"--over", "YAML,TOML",
			"--because", "Simpler parsing")

		output, err := runTk("stats")
		if err != nil {
			t.Fatalf("stats failed: %v\n%s", err, output)
		}

		// Verify learnings count
		if !strings.Contains(output, "Learnings:     3") {
			t.Errorf("expected 3 learnings: %s", output)
		}
		// Verify decisions count
		if !strings.Contains(output, "Decisions:     2") {
			t.Errorf("expected 2 decisions: %s", output)
		}

		// Also verify in JSON format
		jsonOutput, err := runTk("stats", "--format", "json")
		if err != nil {
			t.Fatalf("stats --format json failed: %v\n%s", err, jsonOutput)
		}
		if !strings.Contains(jsonOutput, `"learnings_count": 3`) {
			t.Errorf("expected learnings_count: 3 in JSON: %s", jsonOutput)
		}
		if !strings.Contains(jsonOutput, `"decisions_count": 2`) {
			t.Errorf("expected decisions_count: 2 in JSON: %s", jsonOutput)
		}
	})
}

func TestContextCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	binary := filepath.Join(dir, "tk")

	cmd := exec.Command("go", "build", "-o", binary, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, output)
	}

	// Helper to create fresh test environment
	setupTestDir := func(t *testing.T) (string, func(args ...string) (string, error)) {
		testDir := filepath.Join(dir, t.Name())
		os.MkdirAll(testDir, 0755)

		runTk := func(args ...string) (string, error) {
			cmd := exec.Command(binary, args...)
			cmd.Dir = testDir
			output, err := cmd.CombinedOutput()
			return string(output), err
		}

		// Initialize fresh .tasuku.json
		runTk("init")

		return testDir, runTk
	}

	// ==========================================================================
	// Decision Tests
	// ==========================================================================

	t.Run("decisions-empty", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		output, err := runTk("decisions")
		if err != nil {
			t.Fatalf("decisions failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "No decisions recorded") {
			t.Errorf("expected empty message: %s", output)
		}
	})

	t.Run("decisions-list", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		// Add some decisions
		output, err := runTk("decide",
			"--id", "json-format",
			"--chose", "JSON",
			"--over", "YAML,TOML",
			"--because", "Faster parsing and no ambiguity")
		if err != nil {
			t.Fatalf("decide failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Decision recorded") {
			t.Errorf("expected confirmation: %s", output)
		}

		// Add another decision
		runTk("decide",
			"--id", "use-cobra",
			"--chose", "Cobra",
			"--over", "flag,urfave/cli",
			"--because", "Better subcommand support")

		// List decisions
		output, err = runTk("decisions")
		if err != nil {
			t.Fatalf("decisions list failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "json-format") {
			t.Errorf("expected json-format decision: %s", output)
		}
		if !strings.Contains(output, "use-cobra") {
			t.Errorf("expected use-cobra decision: %s", output)
		}
		if !strings.Contains(output, "JSON") {
			t.Errorf("expected JSON choice: %s", output)
		}
		if !strings.Contains(output, "Decisions (2)") {
			t.Errorf("expected 2 decisions: %s", output)
		}
	})

	t.Run("decisions-json", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		// Add a decision
		runTk("decide",
			"--id", "test-decision",
			"--chose", "Option A",
			"--over", "Option B",
			"--because", "Better performance")

		// List decisions as JSON
		output, err := runTk("decisions", "--format", "json")
		if err != nil {
			t.Fatalf("decisions --format json failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
			t.Errorf("expected JSON array: %s", output)
		}
		if !strings.Contains(output, `"id"`) {
			t.Errorf("expected id field in JSON: %s", output)
		}
		if !strings.Contains(output, `"chose"`) {
			t.Errorf("expected chose field in JSON: %s", output)
		}
		if !strings.Contains(output, "test-decision") {
			t.Errorf("expected test-decision in JSON: %s", output)
		}
	})

	t.Run("undecide", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		// Add decisions
		runTk("decide",
			"--id", "to-remove",
			"--chose", "Remove Me",
			"--over", "Keep Me",
			"--because", "Testing removal")
		runTk("decide",
			"--id", "to-keep",
			"--chose", "Keep This",
			"--over", "Other",
			"--because", "Should remain")

		// Remove the first decision
		output, err := runTk("undecide", "to-remove")
		if err != nil {
			t.Fatalf("undecide failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Removed decision") {
			t.Errorf("expected removed message: %s", output)
		}
		if !strings.Contains(output, "to-remove") {
			t.Errorf("expected decision id in output: %s", output)
		}

		// Verify it's gone
		output, _ = runTk("decisions")
		if strings.Contains(output, "to-remove") {
			t.Errorf("to-remove should be gone: %s", output)
		}
		if !strings.Contains(output, "to-keep") {
			t.Errorf("to-keep should still exist: %s", output)
		}
		if !strings.Contains(output, "Decisions (1)") {
			t.Errorf("expected 1 decision: %s", output)
		}
	})

	t.Run("undecide-not-found", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		_, err := runTk("undecide", "nonexistent-decision")
		if err == nil {
			t.Error("expected error for nonexistent decision")
		}
	})

	// ==========================================================================
	// Notes Tests
	// ==========================================================================

	t.Run("notes-empty", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		output, err := runTk("notes")
		if err != nil {
			t.Fatalf("notes failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "No notes recorded") {
			t.Errorf("expected empty message: %s", output)
		}
	})

	t.Run("notes-for-task", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		// Create a task first
		runTk("add", "--id", "my-task", "My test task")

		// Add notes to the task
		output, err := runTk("note", "my-task", "First note for task")
		if err != nil {
			t.Fatalf("note failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Note added") {
			t.Errorf("expected confirmation: %s", output)
		}

		runTk("note", "my-task", "Second note for task")
		runTk("note", "my-task", "Third note for task")

		// List notes for specific task
		output, err = runTk("notes", "my-task")
		if err != nil {
			t.Fatalf("notes my-task failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "First note") {
			t.Errorf("expected first note: %s", output)
		}
		if !strings.Contains(output, "Second note") {
			t.Errorf("expected second note: %s", output)
		}
		if !strings.Contains(output, "Third note") {
			t.Errorf("expected third note: %s", output)
		}
		if !strings.Contains(output, "Notes for my-task (3)") {
			t.Errorf("expected 3 notes header: %s", output)
		}
	})

	t.Run("notes-all", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		// Create tasks
		runTk("add", "--id", "task-a", "Task A")
		runTk("add", "--id", "task-b", "Task B")

		// Add notes to different tasks
		runTk("note", "task-a", "Note for task A")
		runTk("note", "task-a", "Another note for A")
		runTk("note", "task-b", "Note for task B")

		// List all notes
		output, err := runTk("notes")
		if err != nil {
			t.Fatalf("notes failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "task-a") {
			t.Errorf("expected task-a: %s", output)
		}
		if !strings.Contains(output, "task-b") {
			t.Errorf("expected task-b: %s", output)
		}
		if !strings.Contains(output, "Note for task A") {
			t.Errorf("expected note for task A: %s", output)
		}
		if !strings.Contains(output, "Note for task B") {
			t.Errorf("expected note for task B: %s", output)
		}
		if !strings.Contains(output, "3 total") {
			t.Errorf("expected 3 total notes: %s", output)
		}
		if !strings.Contains(output, "2 tasks") {
			t.Errorf("expected 2 tasks: %s", output)
		}
	})

	t.Run("notes-json", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		// Create task and add notes
		runTk("add", "--id", "json-task", "JSON test task")
		runTk("note", "json-task", "Note for JSON output")

		// List notes as JSON
		output, err := runTk("notes", "--format", "json")
		if err != nil {
			t.Fatalf("notes --format json failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
			t.Errorf("expected JSON object: %s", output)
		}
		if !strings.Contains(output, "json-task") {
			t.Errorf("expected json-task in output: %s", output)
		}
		if !strings.Contains(output, "Note for JSON output") {
			t.Errorf("expected note content in JSON: %s", output)
		}
	})

	t.Run("unnote", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		// Create task and add notes
		runTk("add", "--id", "unnote-task", "Task for unnote test")
		runTk("note", "unnote-task", "First note to keep")
		runTk("note", "unnote-task", "Second note to remove")
		runTk("note", "unnote-task", "Third note to keep")

		// Remove the second note (index 2)
		output, err := runTk("unnote", "unnote-task", "2")
		if err != nil {
			t.Fatalf("unnote failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Removed note") {
			t.Errorf("expected removed message: %s", output)
		}
		if !strings.Contains(output, "Second note to remove") {
			t.Errorf("expected removed note content: %s", output)
		}

		// Verify it's gone
		output, _ = runTk("notes", "unnote-task")
		if strings.Contains(output, "Second note to remove") {
			t.Errorf("second note should be removed: %s", output)
		}
		if !strings.Contains(output, "First note to keep") {
			t.Errorf("first note should remain: %s", output)
		}
		if !strings.Contains(output, "Third note to keep") {
			t.Errorf("third note should remain: %s", output)
		}
		if !strings.Contains(output, "Notes for unnote-task (2)") {
			t.Errorf("expected 2 notes: %s", output)
		}
	})

	t.Run("unnote-out-of-range", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		// Create task and add one note
		runTk("add", "--id", "range-task", "Task for range test")
		runTk("note", "range-task", "Only note")

		// Try to remove note at invalid index
		_, err := runTk("unnote", "range-task", "5")
		if err == nil {
			t.Error("expected error for out of range index")
		}

		// Try to remove note at index 0 (invalid, 1-based)
		output, err := runTk("unnote", "range-task", "0")
		if err == nil {
			t.Error("expected error for index 0")
		}
		if !strings.Contains(output, "invalid index") {
			t.Errorf("expected invalid index message: %s", output)
		}
	})
}

func TestTaskCRUDCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	binary := filepath.Join(dir, "tk")

	cmd := exec.Command("go", "build", "-o", binary, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, output)
	}

	// Helper to create a fresh test directory with initialized .tasuku.json
	setupTestDir := func(t *testing.T) (string, func(args ...string) (string, error)) {
		testDir := filepath.Join(dir, t.Name())
		os.MkdirAll(testDir, 0755)

		runTk := func(args ...string) (string, error) {
			cmd := exec.Command(binary, args...)
			cmd.Dir = testDir
			output, err := cmd.CombinedOutput()
			return string(output), err
		}

		// Initialize .tasuku.json
		if _, err := runTk("init"); err != nil {
			t.Fatalf("failed to init: %v", err)
		}

		return testDir, runTk
	}

	// Test delete: Create a task, delete it, verify it's gone
	t.Run("delete", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		// Create a task
		_, err := runTk("add", "--id", "task-to-delete", "Task to be deleted")
		if err != nil {
			t.Fatalf("add failed: %v", err)
		}

		// Verify task exists
		output, err := runTk("show", "task-to-delete")
		if err != nil {
			t.Fatalf("show failed: %v\n%s", err, output)
		}

		// Delete the task
		output, err = runTk("delete", "task-to-delete")
		if err != nil {
			t.Fatalf("delete failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Deleted: task-to-delete") {
			t.Errorf("expected deletion message: %s", output)
		}

		// Verify task is gone
		output, err = runTk("show", "task-to-delete")
		if err == nil {
			t.Error("expected error when showing deleted task")
		}
		if !strings.Contains(output, "not found") {
			t.Errorf("expected 'not found' error: %s", output)
		}
	})

	// Test delete-cleans-notes: Create task with notes, delete task, verify notes removed
	t.Run("delete-cleans-notes", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		// Create a task and add notes
		runTk("add", "--id", "task-with-notes", "Task with notes")
		runTk("note", "task-with-notes", "Note 1 for this task")
		runTk("note", "task-with-notes", "Note 2 for this task")

		// Create another task with notes to keep the notes map non-empty
		runTk("add", "--id", "other-task", "Other task")
		runTk("note", "other-task", "Other task note")

		// Verify notes exist
		output, err := runTk("notes", "task-with-notes")
		if err != nil {
			t.Fatalf("notes failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Note 1") {
			t.Errorf("expected notes to exist: %s", output)
		}

		// Delete the task
		output, err = runTk("delete", "task-with-notes")
		if err != nil {
			t.Fatalf("delete failed: %v\n%s", err, output)
		}

		// Verify notes for deleted task are gone (other-task notes still exist)
		output, err = runTk("notes", "task-with-notes")
		if err == nil {
			t.Error("expected error when showing notes for deleted task")
		}
		if !strings.Contains(output, "no notes found") {
			t.Errorf("expected 'no notes found' error: %s", output)
		}

		// Verify other-task notes still exist
		output, err = runTk("notes", "other-task")
		if err != nil {
			t.Fatalf("notes for other-task failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Other task note") {
			t.Errorf("expected other-task notes to remain: %s", output)
		}
	})

	// Test delete-cleans-blockers: Create task A blocked by B, delete B, verify A's blocked_by is cleared
	t.Run("delete-cleans-blockers", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		// Create tasks
		runTk("add", "--id", "task-a", "Task A")
		runTk("add", "--id", "task-b", "Task B (blocker)")

		// Block task-a by task-b
		runTk("block", "task-a", "--by", "task-b")

		// Verify task-a is blocked by task-b
		output, err := runTk("show", "task-a")
		if err != nil {
			t.Fatalf("show failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "task-b") {
			t.Errorf("expected task-a to be blocked by task-b: %s", output)
		}

		// Delete task-b
		output, err = runTk("delete", "task-b")
		if err != nil {
			t.Fatalf("delete failed: %v\n%s", err, output)
		}

		// Verify task-a's blocked_by is cleared
		output, err = runTk("show", "task-a")
		if err != nil {
			t.Fatalf("show failed: %v\n%s", err, output)
		}
		if strings.Contains(output, "Blocked by") && strings.Contains(output, "task-b") {
			t.Errorf("task-a should not be blocked by deleted task-b: %s", output)
		}
	})

	// Test edit: Create task, edit description, verify updated
	t.Run("edit", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		// Create a task
		runTk("add", "--id", "task-to-edit", "Original description")

		// Edit the task
		output, err := runTk("edit", "task-to-edit", "Updated description")
		if err != nil {
			t.Fatalf("edit failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Updated: task-to-edit") {
			t.Errorf("expected update message: %s", output)
		}

		// Verify the description was updated
		output, err = runTk("show", "task-to-edit")
		if err != nil {
			t.Fatalf("show failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Updated description") {
			t.Errorf("expected updated description: %s", output)
		}
		if strings.Contains(output, "Original description") {
			t.Errorf("should not contain original description: %s", output)
		}
	})

	// Test pause: Create task, start it, pause it, verify status is ready and owner cleared
	t.Run("pause", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		// Create and start a task
		runTk("add", "--id", "task-to-pause", "Task to pause")
		runTk("start", "task-to-pause")

		// Set an owner
		runTk("owner", "task-to-pause", "agent-1")

		// Verify task is in_progress with owner
		output, err := runTk("show", "task-to-pause")
		if err != nil {
			t.Fatalf("show failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "in_progress") {
			t.Errorf("expected in_progress status: %s", output)
		}

		// Pause the task
		output, err = runTk("pause", "task-to-pause")
		if err != nil {
			t.Fatalf("pause failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Paused: task-to-pause") {
			t.Errorf("expected pause message: %s", output)
		}
		if !strings.Contains(output, "now ready") {
			t.Errorf("expected 'now ready' in message: %s", output)
		}

		// Verify status is ready and owner cleared
		output, err = runTk("show", "task-to-pause")
		if err != nil {
			t.Fatalf("show failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "ready") {
			t.Errorf("expected ready status: %s", output)
		}
		if strings.Contains(output, "in_progress") {
			t.Errorf("should not be in_progress: %s", output)
		}
		// Owner should be cleared (not shown in output)
		if strings.Contains(output, "Owner:") && strings.Contains(output, "agent-1") {
			t.Errorf("owner should be cleared after pause: %s", output)
		}
	})

	// Test pause-error: Try to pause a ready task, expect error
	t.Run("pause-error", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		// Create a task (starts as ready)
		runTk("add", "--id", "ready-task", "A ready task")

		// Try to pause a ready task
		output, err := runTk("pause", "ready-task")
		if err == nil {
			t.Error("expected error when pausing a ready task")
		}
		if !strings.Contains(output, "not in_progress") {
			t.Errorf("expected 'not in_progress' error: %s", output)
		}
	})

	// Test owner-set: Create task, set owner, verify
	t.Run("owner-set", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		// Create a task
		runTk("add", "--id", "task-owner", "Task for owner test")

		// Set owner
		output, err := runTk("owner", "task-owner", "agent-1")
		if err != nil {
			t.Fatalf("owner set failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Set owner of task-owner to: agent-1") {
			t.Errorf("expected owner set message: %s", output)
		}

		// Verify owner in show output
		output, err = runTk("show", "task-owner")
		if err != nil {
			t.Fatalf("show failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "agent-1") {
			t.Errorf("expected owner in show output: %s", output)
		}
	})

	// Test owner-clear: Set owner, clear with --clear flag, verify
	t.Run("owner-clear", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		// Create a task and set owner
		runTk("add", "--id", "task-clear-owner", "Task to clear owner")
		runTk("owner", "task-clear-owner", "agent-1")

		// Clear owner
		output, err := runTk("owner", "task-clear-owner", "--clear")
		if err != nil {
			t.Fatalf("owner clear failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Cleared owner of: task-clear-owner") {
			t.Errorf("expected clear message: %s", output)
		}

		// Verify owner is cleared
		output, err = runTk("owner", "task-clear-owner")
		if err != nil {
			t.Fatalf("owner show failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "has no owner") {
			t.Errorf("expected 'has no owner' message: %s", output)
		}
	})

	// Test owner-show: Set owner, run owner command without args, verify output shows owner
	t.Run("owner-show", func(t *testing.T) {
		_, runTk := setupTestDir(t)

		// Create a task and set owner
		runTk("add", "--id", "task-show-owner", "Task to show owner")
		runTk("owner", "task-show-owner", "agent-2")

		// Show owner (just task ID, no owner name)
		output, err := runTk("owner", "task-show-owner")
		if err != nil {
			t.Fatalf("owner show failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Owner of task-show-owner: agent-2") {
			t.Errorf("expected owner display message: %s", output)
		}
	})
}
