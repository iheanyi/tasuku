package task

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewFile(t *testing.T) {
	f := NewFile()

	if f.Version != 1 {
		t.Errorf("expected version 1, got %d", f.Version)
	}

	if f.Tasks == nil {
		t.Error("tasks map should not be nil")
	}

	if f.Context.Learnings == nil {
		t.Error("learnings should not be nil")
	}

	if f.Context.Decisions == nil {
		t.Error("decisions should not be nil")
	}

	if f.Context.Notes == nil {
		t.Error("notes should not be nil")
	}
}

func TestNewTask(t *testing.T) {
	task := NewTask("Test description")

	if task.Status != StatusReady {
		t.Errorf("expected status ready, got %s", task.Status)
	}

	if task.Description != "Test description" {
		t.Errorf("expected description 'Test description', got %s", task.Description)
	}

	if task.BlockedBy == nil {
		t.Error("blocked_by should not be nil")
	}

	if task.Owner != nil {
		t.Error("owner should be nil for new task")
	}

	if task.CreatedAt.IsZero() {
		t.Error("created_at should be set")
	}

	if task.UpdatedAt.IsZero() {
		t.Error("updated_at should be set")
	}

	// Verify timestamps are in UTC
	if task.CreatedAt.Location() != time.UTC {
		t.Error("created_at should be in UTC")
	}
}

func TestValidTransition(t *testing.T) {
	tests := []struct {
		from     Status
		to       Status
		expected bool
	}{
		// From ready
		{StatusReady, StatusInProgress, true},
		{StatusReady, StatusBlocked, true},
		{StatusReady, StatusDone, true}, // Shortcut allowed
		{StatusReady, StatusReady, false},

		// From in_progress
		{StatusInProgress, StatusDone, true},
		{StatusInProgress, StatusBlocked, true},
		{StatusInProgress, StatusReady, true},
		{StatusInProgress, StatusInProgress, false},

		// From blocked
		{StatusBlocked, StatusReady, true},
		{StatusBlocked, StatusInProgress, false},
		{StatusBlocked, StatusDone, false},
		{StatusBlocked, StatusBlocked, false},

		// From done
		{StatusDone, StatusReady, true}, // Reopen
		{StatusDone, StatusInProgress, false},
		{StatusDone, StatusBlocked, false},
		{StatusDone, StatusDone, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			result := ValidTransition(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("ValidTransition(%s, %s) = %v, expected %v",
					tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	// Verify status string values
	if StatusReady != "ready" {
		t.Errorf("StatusReady should be 'ready', got %s", StatusReady)
	}
	if StatusInProgress != "in_progress" {
		t.Errorf("StatusInProgress should be 'in_progress', got %s", StatusInProgress)
	}
	if StatusBlocked != "blocked" {
		t.Errorf("StatusBlocked should be 'blocked', got %s", StatusBlocked)
	}
	if StatusDone != "done" {
		t.Errorf("StatusDone should be 'done', got %s", StatusDone)
	}
}

func TestGenerateShortID(t *testing.T) {
	id := GenerateShortID()
	if len(id) != 6 {
		t.Errorf("expected 6-char ID, got %d chars: %s", len(id), id)
	}

	// Generate a bunch and check uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		newID := GenerateShortID()
		if ids[newID] {
			t.Errorf("duplicate ID generated: %s", newID)
		}
		ids[newID] = true
	}
}

func TestLearningStruct(t *testing.T) {
	learning := Learning{
		ID:        "abc123",
		Text:      "Test learning",
		CreatedAt: time.Now().UTC(),
	}

	if learning.ID != "abc123" {
		t.Errorf("expected ID 'abc123', got %s", learning.ID)
	}
	if learning.Text != "Test learning" {
		t.Errorf("expected Text 'Test learning', got %s", learning.Text)
	}
	if learning.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestContextUnmarshalJSON_OldLearningsFormat(t *testing.T) {
	// Old format: learnings as []string
	oldJSON := `{
		"learnings": ["Learning 1", "Learning 2"],
		"decisions": [],
		"notes": {}
	}`

	var ctx Context
	if err := json.Unmarshal([]byte(oldJSON), &ctx); err != nil {
		t.Fatalf("failed to unmarshal old format: %v", err)
	}

	if len(ctx.Learnings) != 2 {
		t.Fatalf("expected 2 learnings, got %d", len(ctx.Learnings))
	}

	if ctx.Learnings[0].Text != "Learning 1" {
		t.Errorf("expected 'Learning 1', got %s", ctx.Learnings[0].Text)
	}

	if ctx.Learnings[0].ID == "" {
		t.Error("expected generated ID for migrated learning")
	}
}

func TestContextUnmarshalJSON_NewLearningsFormat(t *testing.T) {
	// New format: learnings as []Learning
	newJSON := `{
		"learnings": [
			{"id": "abc123", "text": "New learning", "created_at": "2024-01-01T00:00:00Z"}
		],
		"decisions": [],
		"notes": {}
	}`

	var ctx Context
	if err := json.Unmarshal([]byte(newJSON), &ctx); err != nil {
		t.Fatalf("failed to unmarshal new format: %v", err)
	}

	if len(ctx.Learnings) != 1 {
		t.Fatalf("expected 1 learning, got %d", len(ctx.Learnings))
	}

	if ctx.Learnings[0].ID != "abc123" {
		t.Errorf("expected ID 'abc123', got %s", ctx.Learnings[0].ID)
	}

	if ctx.Learnings[0].Text != "New learning" {
		t.Errorf("expected 'New learning', got %s", ctx.Learnings[0].Text)
	}
}

func TestContextMarshalJSON_LearningsFormat(t *testing.T) {
	ctx := Context{
		Learnings: []Learning{
			{ID: "test1", Text: "Test learning", CreatedAt: time.Now().UTC()},
		},
		Decisions: []Decision{},
		Notes:     make(map[string][]Note),
	}

	data, err := json.Marshal(ctx)
	if err != nil {
		t.Fatalf("failed to marshal context: %v", err)
	}

	// Verify the JSON contains the new format
	var unmarshaled Context
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(unmarshaled.Learnings) != 1 {
		t.Fatalf("expected 1 learning, got %d", len(unmarshaled.Learnings))
	}

	if unmarshaled.Learnings[0].ID != "test1" {
		t.Errorf("ID not preserved: got %s", unmarshaled.Learnings[0].ID)
	}
}

func TestIsRuleLearning(t *testing.T) {
	tests := []struct {
		text     string
		expected bool
	}{
		// Starts with Never/Always (case-insensitive)
		{"Never use raw SQL queries", true},
		{"never use raw SQL queries", true},
		{"NEVER use raw SQL queries", true},
		{"Always validate input before processing", true},
		{"always validate input before processing", true},
		{"ALWAYS validate input before processing", true},

		// Contains never/always as key words
		{"You should never use eval()", true},
		{"The system must always validate tokens", true},
		{"We should never commit secrets", true},
		{"Tests must always pass before merging", true},

		// With punctuation around the keywords
		{"You should never, ever skip tests", true},
		{"Always! Check for nil pointers", true},

		// Edge cases - should NOT match
		{"Redis connection pooling improves performance", false},
		{"The cache expires every 24 hours", false},
		{"Use indexes for frequent queries", false},
		{"Whenever possible, use batch operations", false}, // "Whenever" is not "never"
		{"This is a general learning", false},
		{"", false},
		{"   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := IsRuleLearning(tt.text)
			if result != tt.expected {
				t.Errorf("IsRuleLearning(%q) = %v, expected %v", tt.text, result, tt.expected)
			}
		})
	}
}

func TestLearningIsRuleField(t *testing.T) {
	learning := Learning{
		ID:        "abc123",
		Text:      "Never use eval()",
		IsRule:    true,
		CreatedAt: time.Now().UTC(),
	}

	if !learning.IsRule {
		t.Error("expected IsRule to be true")
	}

	// Test JSON serialization preserves IsRule
	data, err := json.Marshal(learning)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled Learning
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !unmarshaled.IsRule {
		t.Error("IsRule not preserved after JSON round-trip")
	}
}

func TestLearningIsRuleOmitEmpty(t *testing.T) {
	// When IsRule is false, it should be omitted from JSON
	learning := Learning{
		ID:        "abc123",
		Text:      "Regular learning",
		IsRule:    false,
		CreatedAt: time.Now().UTC(),
	}

	data, err := json.Marshal(learning)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, "is_rule") {
		t.Errorf("is_rule should be omitted when false, got: %s", jsonStr)
	}
}
