package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iheanyi/tasuku/internal/task"
)

func TestScopeToSlug(t *testing.T) {
	tests := []struct {
		scope    string
		expected string
	}{
		{"src/api/**", "api"},
		{"src/components/**/*.tsx", "components"},
		{"src/api/**/*.ts", "api"},
		{"internal/**", "internal"},
		{"**/*.go", "scoped"},
		{"src/frontend/components/**", "components"},
		{"lib/utils/**", "utils"},
	}

	for _, tt := range tests {
		t.Run(tt.scope, func(t *testing.T) {
			result := scopeToSlug(tt.scope)
			if result != tt.expected {
				t.Errorf("scopeToSlug(%q) = %q, want %q", tt.scope, result, tt.expected)
			}
		})
	}
}

func TestGetTargets(t *testing.T) {
	// Create temp directory
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// No targets initially
	targets := GetTargets()
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}

	// Create .claude directory
	os.Mkdir(".claude", 0755)
	targets = GetTargets()
	if len(targets) != 1 {
		t.Errorf("expected 1 target after .claude, got %d", len(targets))
	}
	if targets[0].Name != "Claude Code" {
		t.Errorf("expected 'Claude Code', got %q", targets[0].Name)
	}

	// Create CLAUDE.md (already have .claude, should still be 1)
	os.WriteFile("CLAUDE.md", []byte("# Test"), 0644)
	targets = GetTargets()
	if len(targets) != 1 {
		t.Errorf("expected 1 target with both signals, got %d", len(targets))
	}

	// Create .cursor directory
	os.Mkdir(".cursor", 0755)
	targets = GetTargets()
	if len(targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(targets))
	}
}

func TestGetTargetsClaudeMDOnly(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Only CLAUDE.md, no .claude directory
	os.WriteFile("CLAUDE.md", []byte("# Test"), 0644)
	targets := GetTargets()
	if len(targets) != 1 {
		t.Errorf("expected 1 target from CLAUDE.md alone, got %d", len(targets))
	}
}

func TestSync(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Create .claude directory
	os.Mkdir(".claude", 0755)

	learnings := []task.Learning{
		{ID: "abc123", Text: "Never use eval()", IsRule: true, CreatedAt: time.Now()},
		{ID: "def456", Text: "Redis improves latency", IsRule: false, CreatedAt: time.Now()},
		{ID: "ghi789", Text: "Always validate API input", IsRule: true, Scope: "src/api/**", CreatedAt: time.Now()},
	}

	decisions := []task.Decision{
		{ID: "auth-strategy", Chose: "JWT", Over: []string{"Sessions"}, Because: "Stateless", CreatedAt: time.Now()},
	}

	results, err := Sync(learnings, decisions)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Editor != "Claude Code" {
		t.Errorf("expected 'Claude Code', got %q", result.Editor)
	}

	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}

	// Should have 3 files: learnings.md, learnings-api.md, decisions.md
	if len(result.FilesWritten) != 3 {
		t.Errorf("expected 3 files written, got %d: %v", len(result.FilesWritten), result.FilesWritten)
	}

	// Verify learnings.md exists and contains expected content
	learningsPath := filepath.Join(".claude/rules/tasuku", "learnings.md")
	content, err := os.ReadFile(learningsPath)
	if err != nil {
		t.Fatalf("failed to read learnings.md: %v", err)
	}
	if !strings.Contains(string(content), "Never use eval()") {
		t.Error("learnings.md should contain 'Never use eval()'")
	}
	if strings.Contains(string(content), "Always validate API input") {
		t.Error("learnings.md should NOT contain scoped learning")
	}

	// Verify learnings-api.md exists and has frontmatter
	apiPath := filepath.Join(".claude/rules/tasuku", "learnings-api.md")
	apiContent, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("failed to read learnings-api.md: %v", err)
	}
	if !strings.Contains(string(apiContent), "paths: src/api/**") {
		t.Error("learnings-api.md should contain paths frontmatter")
	}
	if !strings.Contains(string(apiContent), "Always validate API input") {
		t.Error("learnings-api.md should contain the scoped learning")
	}

	// Verify decisions.md exists
	decisionsPath := filepath.Join(".claude/rules/tasuku", "decisions.md")
	decContent, err := os.ReadFile(decisionsPath)
	if err != nil {
		t.Fatalf("failed to read decisions.md: %v", err)
	}
	if !strings.Contains(string(decContent), "JWT") {
		t.Error("decisions.md should contain 'JWT'")
	}
}

func TestSyncNoTargets(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// No editor signals
	_, err := Sync([]task.Learning{}, []task.Decision{})
	if err == nil {
		t.Error("expected error when no targets found")
	}
}

func TestClean(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Create .claude and rules
	os.MkdirAll(".claude/rules/tasuku", 0755)
	os.WriteFile(".claude/rules/tasuku/learnings.md", []byte("test"), 0644)
	os.WriteFile(".claude/rules/tasuku/decisions.md", []byte("test"), 0644)

	removed, err := Clean()
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	if len(removed) != 2 {
		t.Errorf("expected 2 files removed, got %d", len(removed))
	}

	// Verify files are gone
	if _, err := os.Stat(".claude/rules/tasuku/learnings.md"); !os.IsNotExist(err) {
		t.Error("learnings.md should be removed")
	}
}

func TestGenerateLearningsMarkdown(t *testing.T) {
	learnings := []task.Learning{
		{ID: "abc", Text: "Never do X", IsRule: true},
		{ID: "def", Text: "Y is interesting", IsRule: false},
	}

	// Unscoped
	content := generateLearningsMarkdown(learnings, "")
	if strings.Contains(string(content), "---") {
		t.Error("unscoped content should not have frontmatter")
	}
	if !strings.Contains(string(content), "## Rules") {
		t.Error("should have Rules section")
	}
	if !strings.Contains(string(content), "## Insights") {
		t.Error("should have Insights section")
	}

	// Scoped
	scopedContent := generateLearningsMarkdown(learnings, "src/api/**")
	if !strings.Contains(string(scopedContent), "---\npaths: src/api/**\n---") {
		t.Error("scoped content should have paths frontmatter")
	}
}

func TestGetTargetsCopilot(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// No targets initially
	targets := GetTargets()
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}

	// Create .github/hooks directory (indicates Copilot CLI hooks installed)
	os.MkdirAll(".github/hooks", 0755)
	targets = GetTargets()
	if len(targets) != 1 {
		t.Errorf("expected 1 target after .github/hooks, got %d", len(targets))
	}
	if targets[0].Name != "Copilot CLI" {
		t.Errorf("expected 'Copilot CLI', got %q", targets[0].Name)
	}
}

func TestGetTargetsCopilotDir(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Create .copilot directory (MCP config installed)
	os.Mkdir(".copilot", 0755)
	targets := GetTargets()
	if len(targets) != 1 {
		t.Errorf("expected 1 target from .copilot alone, got %d", len(targets))
	}
	if targets[0].Name != "Copilot CLI" {
		t.Errorf("expected 'Copilot CLI', got %q", targets[0].Name)
	}
}

func TestSyncToCopilot(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Create .github/hooks directory
	os.MkdirAll(".github/hooks", 0755)

	learnings := []task.Learning{
		{ID: "abc123", Text: "Never use eval()", IsRule: true, CreatedAt: time.Now()},
	}

	decisions := []task.Decision{
		{ID: "test-dec", Chose: "A", Over: []string{"B"}, Because: "Reason", CreatedAt: time.Now()},
	}

	results, err := Sync(learnings, decisions)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Editor != "Copilot CLI" {
		t.Errorf("expected 'Copilot CLI', got %q", result.Editor)
	}

	// Verify files were written to .github/rules/tasuku
	learningsPath := filepath.Join(".github/rules/tasuku", "learnings.md")
	if _, err := os.Stat(learningsPath); os.IsNotExist(err) {
		t.Error("learnings.md should exist in .github/rules/tasuku")
	}

	decisionsPath := filepath.Join(".github/rules/tasuku", "decisions.md")
	if _, err := os.Stat(decisionsPath); os.IsNotExist(err) {
		t.Error("decisions.md should exist in .github/rules/tasuku")
	}
}

func TestSyncToToolCopilot(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Create .copilot directory
	os.Mkdir(".copilot", 0755)

	learnings := []task.Learning{
		{ID: "test", Text: "Test learning", IsRule: false, CreatedAt: time.Now()},
	}

	// Test various aliases
	for _, alias := range []string{"copilot", "copilot-cli", "copilotcli", "github"} {
		results, err := SyncToTool(learnings, []task.Decision{}, alias)
		if err != nil {
			t.Errorf("SyncToTool(%q) failed: %v", alias, err)
			continue
		}
		if len(results) != 1 {
			t.Errorf("SyncToTool(%q) expected 1 result, got %d", alias, len(results))
		}
		if results[0].Editor != "Copilot CLI" {
			t.Errorf("SyncToTool(%q) expected 'Copilot CLI', got %q", alias, results[0].Editor)
		}
	}
}

func TestCleanCopilot(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Create .github/hooks and rules
	os.MkdirAll(".github/hooks", 0755)
	os.MkdirAll(".github/rules/tasuku", 0755)
	os.WriteFile(".github/rules/tasuku/learnings.md", []byte("test"), 0644)
	os.WriteFile(".github/rules/tasuku/decisions.md", []byte("test"), 0644)

	removed, err := Clean()
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	if len(removed) != 2 {
		t.Errorf("expected 2 files removed, got %d", len(removed))
	}

	// Verify files are gone
	if _, err := os.Stat(".github/rules/tasuku/learnings.md"); !os.IsNotExist(err) {
		t.Error("learnings.md should be removed")
	}
}
