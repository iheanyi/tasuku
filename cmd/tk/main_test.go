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
