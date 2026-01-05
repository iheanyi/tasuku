package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePlanFile(t *testing.T) {
	// Create temp plan file
	content := `# Test Plan

## Phase 1
- [ ] Implement user authentication
- [ ] Add database pooling
- [x] Already done task

## Phase 2
- Fix some bug
* Another bullet point

## Numbered
1. First task
2. Second task

  - [ ] Nested checkbox (should be ignored)
  - Nested bullet (should be ignored)
`
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.md")
	if err := os.WriteFile(planPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	items, err := ParsePlanFile(planPath)
	if err != nil {
		t.Fatalf("ParsePlanFile failed: %v", err)
	}

	// Should have: 2 unchecked checkboxes + 2 bullets + 2 numbered = 6 top-level items
	// (Already done task is checked, so skipped)
	// (Nested items are Level > 0, so skipped)
	expectedDescriptions := []string{
		"Implement user authentication",
		"Add database pooling",
		"Fix some bug",
		"Another bullet point",
		"First task",
		"Second task",
	}

	if len(items) != len(expectedDescriptions) {
		t.Errorf("expected %d items, got %d", len(expectedDescriptions), len(items))
		for i, item := range items {
			t.Logf("  item[%d]: %q (level=%d, checkbox=%v, checked=%v)",
				i, item.Description, item.Level, item.IsCheckbox, item.IsChecked)
		}
	}

	for i, expected := range expectedDescriptions {
		if i >= len(items) {
			break
		}
		if items[i].Description != expected {
			t.Errorf("item[%d]: expected %q, got %q", i, expected, items[i].Description)
		}
	}
}

func TestParsePlanFile_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "empty.md")
	if err := os.WriteFile(planPath, []byte("# Just a header\n\nNo items here.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	items, err := ParsePlanFile(planPath)
	if err != nil {
		t.Fatalf("ParsePlanFile failed: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("expected 0 items from empty plan, got %d", len(items))
	}
}

func TestParsePlanFile_NotFound(t *testing.T) {
	_, err := ParsePlanFile("/nonexistent/path/plan.md")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParsePlanFile_CheckboxStates(t *testing.T) {
	content := `- [ ] Unchecked
- [x] Lowercase checked
- [X] Uppercase checked
`
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "checkboxes.md")
	if err := os.WriteFile(planPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	items, err := ParsePlanFile(planPath)
	if err != nil {
		t.Fatalf("ParsePlanFile failed: %v", err)
	}

	// Only unchecked items should be included
	if len(items) != 1 {
		t.Errorf("expected 1 unchecked item, got %d", len(items))
	}

	if len(items) > 0 && items[0].Description != "Unchecked" {
		t.Errorf("expected 'Unchecked', got %q", items[0].Description)
	}
}

func TestParsePlanFile_IndentedItems(t *testing.T) {
	content := `- Top level item
  - Indented child
    - Deeply nested
- Another top level
`
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "nested.md")
	if err := os.WriteFile(planPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	items, err := ParsePlanFile(planPath)
	if err != nil {
		t.Fatalf("ParsePlanFile failed: %v", err)
	}

	// Only top-level items (Level == 0)
	if len(items) != 2 {
		t.Errorf("expected 2 top-level items, got %d", len(items))
		for i, item := range items {
			t.Logf("  item[%d]: %q level=%d", i, item.Description, item.Level)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a longer string", 10, "this is..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
		}
	}
}

func TestPlanSyncNudgeRule(t *testing.T) {
	// This tests the integration with shouldPersistTask
	tests := []struct {
		description   string
		shouldPersist bool
	}{
		{"Implement user authentication", true},
		{"Add feature for dark mode", true},
		{"Refactor database layer", true},
		{"Fix type error in UserService", false},
		{"Update config file", false},
		{"Run tests", false},
	}

	for _, tt := range tests {
		got := shouldPersistTask(tt.description)
		if got != tt.shouldPersist {
			t.Errorf("shouldPersistTask(%q) = %v, want %v", tt.description, got, tt.shouldPersist)
		}
	}
}
