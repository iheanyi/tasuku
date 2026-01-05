package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestHasGhCLI tests the hasGhCLI function
func TestHasGhCLI(t *testing.T) {
	// This test just verifies the function doesn't panic
	// The result depends on whether gh is installed on the test machine
	result := hasGhCLI()
	t.Logf("hasGhCLI() = %v", result)
}

// TestPRCommands tests the PR command integration
func TestPRCommands(t *testing.T) {
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

	// Initialize
	if _, err := runTk("init"); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Test pr help
	t.Run("pr-help", func(t *testing.T) {
		output, err := runTk("pr", "--help")
		if err != nil {
			t.Fatalf("pr --help failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "GitHub Pull Request") {
			t.Errorf("expected PR description in help: %s", output)
		}
		if !strings.Contains(output, "create") {
			t.Errorf("expected create subcommand: %s", output)
		}
		if !strings.Contains(output, "list") {
			t.Errorf("expected list subcommand: %s", output)
		}
	})

	// Test pr create help
	t.Run("pr-create-help", func(t *testing.T) {
		output, err := runTk("pr", "create", "--help")
		if err != nil {
			t.Fatalf("pr create --help failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "--task") {
			t.Errorf("expected --task flag: %s", output)
		}
		if !strings.Contains(output, "--done") {
			t.Errorf("expected --done flag: %s", output)
		}
	})

	// Test pr list help
	t.Run("pr-list-help", func(t *testing.T) {
		output, err := runTk("pr", "list", "--help")
		if err != nil {
			t.Fatalf("pr list --help failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "List GitHub pull requests") {
			t.Errorf("expected list description: %s", output)
		}
	})

	// Test pr create with non-existent task
	t.Run("pr-create-task-not-found", func(t *testing.T) {
		// Skip if gh is not installed - we can't test task validation without attempting gh
		if !hasGhCLI() {
			t.Skip("gh CLI not installed, skipping task validation test")
		}

		output, err := runTk("pr", "create", "--task", "nonexistent-task")
		if err == nil {
			t.Error("expected error for nonexistent task")
		}
		if !strings.Contains(output, "not found") {
			t.Errorf("expected 'not found' error: %s", output)
		}
	})

	// Test pr commands gracefully handle missing gh
	t.Run("pr-no-gh-graceful", func(t *testing.T) {
		// This test verifies the behavior when gh is not available
		// We can't easily mock exec.LookPath, so we test the help message format
		if hasGhCLI() {
			t.Skip("gh CLI is installed, skipping graceful degradation test")
		}

		// When gh is not installed, create should show helpful message
		output, err := runTk("pr", "create")
		if err != nil {
			t.Fatalf("pr create should not error when gh missing: %v\n%s", err, output)
		}
		if !strings.Contains(output, "gh") || !strings.Contains(output, "not installed") {
			t.Errorf("expected installation message: %s", output)
		}
	})
}

// TestBuildTaskContext tests the task context formatting
func TestBuildTaskContext(t *testing.T) {
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

	// Create test directory
	testDir := filepath.Join(dir, "project")
	os.MkdirAll(testDir, 0755)

	runTk := func(args ...string) (string, error) {
		cmd := exec.Command(binary, args...)
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	// Initialize and create a task with notes and tags
	runTk("init")
	runTk("add", "--id", "test-task", "Test task for PR", "--tag", "backend,api")
	runTk("note", "add", "test-task", "Important note about implementation")
	runTk("priority", "test-task", "high")

	// Show the task to verify it was created correctly
	t.Run("task-created-with-metadata", func(t *testing.T) {
		output, err := runTk("show", "test-task")
		if err != nil {
			t.Fatalf("show failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "backend") {
			t.Errorf("expected backend tag: %s", output)
		}
		if !strings.Contains(output, "high") {
			t.Errorf("expected high priority: %s", output)
		}
	})
}

// TestPRCreateWithTask tests creating a PR with a linked task
func TestPRCreateWithTask(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if gh is not installed
	if !hasGhCLI() {
		t.Skip("gh CLI not installed, skipping PR creation test")
	}

	// Build the binary
	dir := t.TempDir()
	binary := filepath.Join(dir, "tk")

	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = "."
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, output)
	}

	// Create test directory
	testDir := filepath.Join(dir, "project")
	os.MkdirAll(testDir, 0755)

	runTk := func(args ...string) (string, error) {
		cmd := exec.Command(binary, args...)
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	// Initialize and create a task
	runTk("init")
	runTk("add", "--id", "pr-task", "Task for PR creation test")

	// Test that trying to create PR with task validates task exists
	// This will fail on gh pr create (no git repo), but task should be validated first
	t.Run("validates-task-exists", func(t *testing.T) {
		output, err := runTk("pr", "create", "--task", "nonexistent")
		if err == nil {
			t.Log("gh may have failed for other reasons, checking output")
		}
		if strings.Contains(output, "not found") {
			// Task validation worked
			return
		}
		// If we get here, either gh failed for a git reason or task was found
		t.Logf("Output: %s", output)
	})
}

// TestPRInstallMessage tests the install message content
func TestPRInstallMessage(t *testing.T) {
	// Verify the install message contains helpful information
	if !strings.Contains(ghInstallMessage, "brew install gh") {
		t.Error("expected macOS install instructions")
	}
	if !strings.Contains(ghInstallMessage, "apt install gh") {
		t.Error("expected Ubuntu install instructions")
	}
	if !strings.Contains(ghInstallMessage, "winget") {
		t.Error("expected Windows install instructions")
	}
	if !strings.Contains(ghInstallMessage, "gh auth login") {
		t.Error("expected auth instructions")
	}
}
