package learning

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
)

func TestLearningListEmpty(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputContains("No learnings recorded")
}

func TestLearningListWithLearnings(t *testing.T) {
	h := testutil.New(t)

	h.AddLearning("Redis connection pooling improves latency")
	h.AddLearning("Auth middleware must run before rate limiting")

	err := h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputContains("Redis connection pooling")
	h.AssertOutputContains("Auth middleware")
}

func TestLearningListJSON(t *testing.T) {
	h := testutil.New(t)

	h.AddLearning("Test learning")

	err := h.ExecuteWithFormat(Cmd, "json", "list")
	h.AssertNoError(err)

	output := h.Stdout()
	if !strings.Contains(output, `"text"`) {
		t.Errorf("expected JSON output, got:\n%s", output)
	}
}

func TestLearningListYAML(t *testing.T) {
	h := testutil.New(t)

	h.AddLearning("Test learning")

	err := h.ExecuteWithFormat(Cmd, "yaml", "list")
	h.AssertNoError(err)

	output := h.Stdout()
	if !strings.Contains(output, "text:") {
		t.Errorf("expected YAML output, got:\n%s", output)
	}
}

func TestLearningAdd(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "add", "New insight about the codebase")
	h.AssertNoError(err)
	h.AssertOutputContains("Learning added")

	// Verify it was added
	err = h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputContains("New insight")
}

func TestLearningAddRule(t *testing.T) {
	h := testutil.New(t)

	// Learning starting with "Never" should be detected as a rule
	err := h.Execute(Cmd, "add", "Never use raw SQL queries")
	h.AssertNoError(err)
	h.AssertOutputContains("[RULE]")

	// Verify it shows in rules list
	err = h.Execute(Cmd, "rules")
	h.AssertNoError(err)
	h.AssertOutputContains("Never use raw SQL")
}

func TestLearningAddWithRuleFlag(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "add", "--rule", "Custom rule that should be marked")
	h.AssertNoError(err)
	h.AssertOutputContains("[RULE]")
}

func TestLearningRemoveByID(t *testing.T) {
	h := testutil.New(t)

	id, _ := h.AddLearning("Learning to remove")

	err := h.Execute(Cmd, "remove", id)
	h.AssertNoError(err)
	h.AssertOutputContains("Removed learning")

	// Verify it's gone
	err = h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputNotContains("Learning to remove")
}

func TestLearningRemoveByText(t *testing.T) {
	h := testutil.New(t)

	h.AddLearning("Unique learning about Redis")

	err := h.Execute(Cmd, "remove", "Redis")
	h.AssertNoError(err)
	h.AssertOutputContains("Removed learning")
}

func TestLearningRemoveNotFound(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "remove", "nonexistent-id")
	h.AssertError(err)
	if !strings.Contains(err.Error(), "no learning found") {
		t.Errorf("expected 'no learning found' error, got: %v", err)
	}
}

func TestLearningRulesEmpty(t *testing.T) {
	h := testutil.New(t)

	// Add non-rule learning
	h.AddLearning("Regular insight")

	err := h.Execute(Cmd, "rules")
	h.AssertNoError(err)
	h.AssertOutputContains("No rule learnings")
}

func TestLearningRulesWithRules(t *testing.T) {
	h := testutil.New(t)

	h.AddLearning("Regular learning")
	h.Execute(Cmd, "add", "Never commit secrets")
	h.Execute(Cmd, "add", "Always validate input")

	err := h.Execute(Cmd, "rules")
	h.AssertNoError(err)
	h.AssertOutputContains("Never commit secrets")
	h.AssertOutputContains("Always validate input")
	h.AssertOutputNotContains("Regular learning")
}

func TestLearningRulesJSON(t *testing.T) {
	h := testutil.New(t)

	h.Execute(Cmd, "add", "Never do this")

	err := h.ExecuteWithFormat(Cmd, "json", "rules")
	h.AssertNoError(err)

	output := h.Stdout()
	if !strings.Contains(output, `"is_rule"`) {
		t.Errorf("expected JSON output with is_rule, got:\n%s", output)
	}
}

func TestLearningAddNoArgs(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "add")
	h.AssertError(err)
}

func TestLearningRemoveNoArgs(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "remove")
	h.AssertError(err)
}

func TestLearningCmdStructure(t *testing.T) {
	if Cmd.Use != "learning" {
		t.Errorf("expected Use to be 'learning', got %s", Cmd.Use)
	}

	// Check subcommands exist - extract command name (first word) from Use
	subcommands := make(map[string]bool)
	for _, sub := range Cmd.Commands() {
		// Use field may be "add \"insight\"" so extract just the command name
		name := strings.Fields(sub.Use)[0]
		subcommands[name] = true
	}

	expected := []string{"list", "add", "remove", "rules", "promote"}
	for _, name := range expected {
		if !subcommands[name] {
			t.Errorf("expected '%s' subcommand", name)
		}
	}
}

func TestLearningAddAlways(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "add", "Always test your code")
	h.AssertNoError(err)
	h.AssertOutputContains("[RULE]")
}

func TestLearningAddAvoid(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "add", "Avoid using global variables")
	h.AssertNoError(err)
	h.AssertOutputContains("[RULE]")
}

func TestLearningAddPrefer(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "add", "Prefer composition over inheritance")
	h.AssertNoError(err)
	h.AssertOutputContains("[RULE]")
}

func TestLearningAddRegular(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "add", "Redis caches data effectively")
	h.AssertNoError(err)
	h.AssertOutputNotContains("[RULE]")
}

func TestLearningListShowsRuleIndicator(t *testing.T) {
	h := testutil.New(t)

	h.Execute(Cmd, "add", "Never skip tests")
	h.Execute(Cmd, "add", "Regular observation")

	err := h.Execute(Cmd, "list")
	h.AssertNoError(err)
	// Rule learnings should have indicator
	h.AssertOutputContains("[RULE]")
}

func TestLearningAddWithScope(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "add", "--scope", "src/api/**", "API error handling pattern")
	h.AssertNoError(err)
	h.AssertOutputContains("Learning added")
}

func TestLearningPromoteByID(t *testing.T) {
	h := testutil.New(t)

	// Add a learning
	id, _ := h.AddLearning("Important insight to promote")

	// Create CLAUDE.md in the test directory
	claudePath := h.TempDir() + "/CLAUDE.md"
	if err := writeTestFile(claudePath, "# Project\n\n## Learnings\n\n"); err != nil {
		t.Fatal(err)
	}

	err := h.Execute(Cmd, "promote", "--to", claudePath, id)
	h.AssertNoError(err)
	h.AssertOutputContains("Promoted to")
	h.AssertOutputContains("Important insight")

	// Verify learning was removed (default behavior)
	err = h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputNotContains("Important insight")
}

func TestLearningPromoteByText(t *testing.T) {
	h := testutil.New(t)

	// Add a learning
	h.AddLearning("Unique learning about caching")

	// Create CLAUDE.md in the test directory
	claudePath := h.TempDir() + "/CLAUDE.md"
	if err := writeTestFile(claudePath, "# Project\n\n## Learnings\n\n"); err != nil {
		t.Fatal(err)
	}

	err := h.Execute(Cmd, "promote", "--to", claudePath, "caching")
	h.AssertNoError(err)
	h.AssertOutputContains("Promoted to")
}

func TestLearningPromoteWithKeep(t *testing.T) {
	h := testutil.New(t)

	// Add a learning
	id, _ := h.AddLearning("Keep this learning after promote")

	// Create CLAUDE.md in the test directory
	claudePath := h.TempDir() + "/CLAUDE.md"
	if err := writeTestFile(claudePath, "# Project\n\n## Learnings\n\n"); err != nil {
		t.Fatal(err)
	}

	err := h.Execute(Cmd, "promote", "--to", claudePath, "--keep", id)
	h.AssertNoError(err)
	h.AssertOutputContains("kept in learnings")

	// Verify learning still exists
	err = h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputContains("Keep this learning")
}

func TestLearningPromoteNotFound(t *testing.T) {
	h := testutil.New(t)

	claudePath := h.TempDir() + "/CLAUDE.md"
	if err := writeTestFile(claudePath, "# Project\n"); err != nil {
		t.Fatal(err)
	}

	err := h.Execute(Cmd, "promote", "--to", claudePath, "nonexistent")
	h.AssertError(err)
	if !strings.Contains(err.Error(), "no learning found") {
		t.Errorf("expected 'no learning found' error, got: %v", err)
	}
}

func TestLearningPromoteNoArgs(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "promote")
	h.AssertError(err)
}

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"just now", 30 * time.Second, "just now"},
		{"minutes", 5 * time.Minute, "5m ago"},
		{"hours", 3 * time.Hour, "3h ago"},
		{"days", 48 * time.Hour, "2d ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testTime := time.Now().Add(-tt.duration)
			result := formatAge(testTime)
			if result != tt.expected {
				t.Errorf("formatAge(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestFormatAgeZero(t *testing.T) {
	result := formatAge(time.Time{})
	if result != "" {
		t.Errorf("formatAge(zero) = %q, want empty string", result)
	}
}

func TestDetectContextFile(t *testing.T) {
	// Save original directory
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	// Create temp directory
	dir := t.TempDir()
	os.Chdir(dir)

	// With no context files, should default to CLAUDE.md
	result := detectContextFile()
	if result != "CLAUDE.md" {
		t.Errorf("expected CLAUDE.md as default, got %s", result)
	}
}

func TestDetectContextFileClaude(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	dir := t.TempDir()
	os.Chdir(dir)

	// Create CLAUDE.md
	os.WriteFile("CLAUDE.md", []byte("# Project\n"), 0644)

	result := detectContextFile()
	if result != "CLAUDE.md" {
		t.Errorf("expected CLAUDE.md, got %s", result)
	}
}

func TestDetectContextFileGemini(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	dir := t.TempDir()
	os.Chdir(dir)

	// Create GEMINI.md (no CLAUDE.md)
	os.WriteFile("GEMINI.md", []byte("# Project\n"), 0644)

	result := detectContextFile()
	if result != "GEMINI.md" {
		t.Errorf("expected GEMINI.md, got %s", result)
	}
}

func TestDetectContextFileCursor(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	dir := t.TempDir()
	os.Chdir(dir)

	// Create .cursorrules (no CLAUDE.md or GEMINI.md)
	os.WriteFile(".cursorrules", []byte("# Rules\n"), 0644)

	result := detectContextFile()
	if result != ".cursorrules" {
		t.Errorf("expected .cursorrules, got %s", result)
	}
}

func TestAppendToContextFileWithLearningsSection(t *testing.T) {
	dir := t.TempDir()
	filePath := dir + "/CLAUDE.md"

	// Create file with existing Learnings section
	os.WriteFile(filePath, []byte("# Project\n\n## Learnings\n\n- Existing learning\n"), 0644)

	err := appendToContextFile(filePath, "New learning to append")
	if err != nil {
		t.Fatalf("appendToContextFile error: %v", err)
	}

	content, _ := os.ReadFile(filePath)
	if !strings.Contains(string(content), "New learning to append") {
		t.Error("expected new learning in file")
	}
	if !strings.Contains(string(content), "Existing learning") {
		t.Error("expected existing learning to be preserved")
	}
}

func TestAppendToContextFileWithoutLearningsSection(t *testing.T) {
	dir := t.TempDir()
	filePath := dir + "/CLAUDE.md"

	// Create file without Learnings section
	os.WriteFile(filePath, []byte("# Project\n\nSome content.\n"), 0644)

	err := appendToContextFile(filePath, "First learning")
	if err != nil {
		t.Fatalf("appendToContextFile error: %v", err)
	}

	content, _ := os.ReadFile(filePath)
	if !strings.Contains(string(content), "## Learnings") {
		t.Error("expected Learnings section to be created")
	}
	if !strings.Contains(string(content), "First learning") {
		t.Error("expected learning in file")
	}
}

func TestAppendToContextFileNewFile(t *testing.T) {
	dir := t.TempDir()
	filePath := dir + "/NEW_FILE.md"

	// File doesn't exist yet
	err := appendToContextFile(filePath, "Learning for new file")
	if err != nil {
		t.Fatalf("appendToContextFile error: %v", err)
	}

	content, _ := os.ReadFile(filePath)
	if !strings.Contains(string(content), "Learning for new file") {
		t.Error("expected learning in new file")
	}
}

// Helper function to write test files
func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
