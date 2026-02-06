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
		{"Do this (always)", true},           // Parentheses around keyword
		{"Use [always] this approach", true}, // Brackets around keyword
		{"Apply `always` consistently", true}, // Backticks around keyword

		// New rule start phrases
		{"Avoid using global variables", true},
		{"Prefer composition over inheritance", true},
		{"Ensure all errors are handled", true},
		{"Must validate input before processing", true},
		{"Don't use magic numbers", true},
		{"Do not commit secrets to version control", true},
		{"Make sure tests pass before merging", true},
		{"Be sure to close file handles", true},
		{"Remember to update the changelog", true},

		// Rule phrases that appear in text
		{"Use batch operations when possible", true},
		{"Parallelize I/O where possible", true},
		{"Cache results whenever possible", true},
		{"Don't forget to run linter", true},
		{"Make sure to update docs", true},
		{"Be careful to handle edge cases", true},
		{"It's important to test edge cases", true},
		{"It's critical to validate input", true},
		{"It's essential to log errors", true},
		{"This is a best practice for Go", true},
		{"Using eval is an anti-pattern", true},
		{"Nested callbacks are a code smell", true},

		// Edge cases - should NOT match
		{"Redis connection pooling improves performance", false},
		{"The cache expires every 24 hours", false},
		{"Use indexes for frequent queries", false},
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

func TestGenerateTaskID_NoCollision(t *testing.T) {
	// Without existing IDs, should generate clean deterministic ID
	id := GenerateTaskID("Fix login bug", nil)
	if id != "fix-login-bug" {
		t.Errorf("expected 'fix-login-bug', got %s", id)
	}

	// Same description without collision check should produce same ID
	id1 := GenerateTaskID("Same description", nil)
	id2 := GenerateTaskID("Same description", nil)
	if id1 != id2 {
		t.Errorf("expected same IDs without collision check, got %s and %s", id1, id2)
	}
}

func TestGenerateTaskID_WithCollision(t *testing.T) {
	existingIDs := map[string]struct{}{
		"fix-login-bug": {},
	}

	// Should add suffix when collision detected
	id := GenerateTaskID("Fix login bug", existingIDs)
	if id == "fix-login-bug" {
		t.Errorf("expected ID with suffix on collision, got %s", id)
	}
	if !strings.HasPrefix(id, "fix-login-bug-") {
		t.Errorf("expected prefix 'fix-login-bug-', got %s", id)
	}

	// Suffix should be 3 chars
	parts := strings.Split(id, "-")
	suffix := parts[len(parts)-1]
	if len(suffix) != 3 {
		t.Errorf("expected 3-char suffix, got %d chars: %s", len(suffix), suffix)
	}
}

func TestGenerateTaskID_SimilarDescriptions(t *testing.T) {
	// Similar descriptions that truncate to same base should still work
	existingIDs := make(map[string]struct{})

	id1 := GenerateTaskID("Improve AddressInput component - add clear button", existingIDs)
	existingIDs[id1] = struct{}{}

	id2 := GenerateTaskID("Improve AddressInput component - add recent searches", existingIDs)

	// Both should be valid (second one gets suffix since first one took the clean ID)
	if id1 == id2 {
		t.Errorf("similar descriptions should produce different IDs, both got %s", id1)
	}
}

func TestGenerateTaskID_Truncation(t *testing.T) {
	longDesc := "This is a very long task description that should be truncated"
	id := GenerateTaskID(longDesc, nil)
	if len(id) > 32 {
		t.Errorf("expected ID length <= 32, got %d: %s", len(id), id)
	}
}

func TestGenerateTaskID_EmptyDescription(t *testing.T) {
	id := GenerateTaskID("", nil)
	if !strings.HasPrefix(id, "task-") {
		t.Errorf("expected prefix 'task-' for empty description, got %s", id)
	}
}

func TestDurationMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		duration Duration
		expected string
	}{
		{"zero", Duration(0), "null"},
		{"one hour", Duration(time.Hour), `"1h0m0s"`},
		{"30 minutes", Duration(30 * time.Minute), `"30m0s"`},
		{"90 seconds", Duration(90 * time.Second), `"1m30s"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.duration.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON failed: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(data))
			}
		})
	}
}

func TestDurationUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Duration
	}{
		{"null", "null", Duration(0)},
		{"empty string", `""`, Duration(0)},
		{"one hour", `"1h0m0s"`, Duration(time.Hour)},
		{"30 minutes", `"30m0s"`, Duration(30 * time.Minute)},
		{"2h30m", `"2h30m"`, Duration(2*time.Hour + 30*time.Minute)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Duration
			err := d.UnmarshalJSON([]byte(tt.input))
			if err != nil {
				t.Fatalf("UnmarshalJSON failed: %v", err)
			}
			if d != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, d)
			}
		})
	}
}

func TestDurationUnmarshalJSON_Invalid(t *testing.T) {
	var d Duration
	err := d.UnmarshalJSON([]byte(`"invalid"`))
	if err == nil {
		t.Error("expected error for invalid duration")
	}
}

func TestDurationString(t *testing.T) {
	d := Duration(2*time.Hour + 30*time.Minute)
	expected := "2h30m0s"
	if d.String() != expected {
		t.Errorf("expected %s, got %s", expected, d.String())
	}
}

func TestDurationFormatHumanReadable(t *testing.T) {
	tests := []struct {
		name     string
		duration Duration
		expected string
	}{
		{"zero", Duration(0), "0s"},
		{"seconds only", Duration(45 * time.Second), "45s"},
		{"minutes only", Duration(30 * time.Minute), "30m"},
		{"hours only", Duration(2 * time.Hour), "2h"},
		{"hours and minutes", Duration(2*time.Hour + 30*time.Minute), "2h30m"},
		{"full", Duration(2*time.Hour + 30*time.Minute + 15*time.Second), "2h30m15s"},
		{"minutes and seconds", Duration(5*time.Minute + 30*time.Second), "5m30s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.duration.FormatHumanReadable()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestTaskIsTimerRunning(t *testing.T) {
	now := time.Now()

	// Timer not running
	task := Task{}
	if task.IsTimerRunning() {
		t.Error("expected timer not running")
	}

	// Timer running
	task.TimerStart = &now
	if !task.IsTimerRunning() {
		t.Error("expected timer running")
	}
}

func TestTaskCurrentDuration(t *testing.T) {
	// No timer, no previous duration
	task := Task{}
	if task.CurrentDuration() != 0 {
		t.Errorf("expected 0, got %v", task.CurrentDuration())
	}

	// With previous duration, no timer
	task.Duration = Duration(time.Hour)
	if task.CurrentDuration() != time.Hour {
		t.Errorf("expected 1h, got %v", task.CurrentDuration())
	}

	// With timer running (approximate check)
	start := time.Now().Add(-10 * time.Second)
	task.TimerStart = &start
	current := task.CurrentDuration()
	// Should be at least 1h + 10s
	if current < time.Hour+10*time.Second {
		t.Errorf("expected at least 1h10s, got %v", current)
	}
}

func TestTaskIsClaimStale(t *testing.T) {
	task := Task{}

	// No claim - not stale
	if task.IsClaimStale(time.Hour) {
		t.Error("expected no claim to not be stale")
	}

	// Fresh claim - not stale
	now := time.Now()
	task.ClaimedAt = &now
	if task.IsClaimStale(time.Hour) {
		t.Error("expected fresh claim to not be stale")
	}

	// Old claim - stale
	oldTime := time.Now().Add(-2 * time.Hour)
	task.ClaimedAt = &oldTime
	if !task.IsClaimStale(time.Hour) {
		t.Error("expected old claim to be stale")
	}
}

func TestTaskHasTag(t *testing.T) {
	task := Task{Tags: []string{"bug", "urgent"}}

	if !task.HasTag("bug") {
		t.Error("expected task to have 'bug' tag")
	}
	if !task.HasTag("urgent") {
		t.Error("expected task to have 'urgent' tag")
	}
	if task.HasTag("feature") {
		t.Error("expected task to not have 'feature' tag")
	}
}

func TestTaskHasTagEmpty(t *testing.T) {
	task := Task{}
	if task.HasTag("any") {
		t.Error("expected empty tags to not match")
	}
}

func TestTaskIsSubtask(t *testing.T) {
	task := Task{}
	if task.IsSubtask() {
		t.Error("expected task without parent to not be subtask")
	}

	parentID := "parent-task"
	task.ParentID = &parentID
	if !task.IsSubtask() {
		t.Error("expected task with parent to be subtask")
	}

	emptyParent := ""
	task.ParentID = &emptyParent
	if task.IsSubtask() {
		t.Error("expected task with empty parent to not be subtask")
	}
}

func TestTaskGetParentID(t *testing.T) {
	task := Task{}
	if task.GetParentID() != "" {
		t.Error("expected empty parent ID")
	}

	parentID := "parent-task"
	task.ParentID = &parentID
	if task.GetParentID() != "parent-task" {
		t.Errorf("expected 'parent-task', got %s", task.GetParentID())
	}
}

func TestTaskGetPriority(t *testing.T) {
	task := Task{}
	if task.GetPriority() != PriorityNormal {
		t.Errorf("expected default priority %d, got %d", PriorityNormal, task.GetPriority())
	}

	priority := PriorityHigh
	task.Priority = &priority
	if task.GetPriority() != PriorityHigh {
		t.Errorf("expected priority %d, got %d", PriorityHigh, task.GetPriority())
	}
}

func TestPriorityName(t *testing.T) {
	tests := []struct {
		priority int
		expected string
	}{
		{PriorityCritical, "critical"},
		{PriorityHigh, "high"},
		{PriorityNormal, "normal"},
		{PriorityLow, "low"},
		{PriorityBacklog, "backlog"},
		{99, "unknown"},
		{-1, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := PriorityName(tt.priority)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParsePriority(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"0", PriorityCritical},
		{"critical", PriorityCritical},
		{"CRITICAL", PriorityCritical},
		{"1", PriorityHigh},
		{"high", PriorityHigh},
		{"2", PriorityNormal},
		{"normal", PriorityNormal},
		{"3", PriorityLow},
		{"low", PriorityLow},
		{"4", PriorityBacklog},
		{"backlog", PriorityBacklog},
		{"invalid", -1},
		{"", -1},
		{"5", -1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParsePriority(tt.input)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestDecisionStruct(t *testing.T) {
	decision := Decision{
		ID:        "test-decision",
		Chose:     "Option A",
		Over:      []string{"Option B", "Option C"},
		Because:   "It was the best choice",
		CreatedAt: time.Now().UTC(),
	}

	if decision.ID != "test-decision" {
		t.Errorf("expected ID 'test-decision', got %s", decision.ID)
	}
	if decision.Chose != "Option A" {
		t.Errorf("expected Chose 'Option A', got %s", decision.Chose)
	}
	if len(decision.Over) != 2 {
		t.Errorf("expected 2 alternatives, got %d", len(decision.Over))
	}
}

func TestNoteStruct(t *testing.T) {
	note := Note{
		ID:        "abc123",
		Text:      "Test note",
		CreatedAt: time.Now().UTC(),
	}

	if note.ID != "abc123" {
		t.Errorf("expected ID 'abc123', got %s", note.ID)
	}
	if note.Text != "Test note" {
		t.Errorf("expected Text 'Test note', got %s", note.Text)
	}
}

func TestArchivedTaskStruct(t *testing.T) {
	now := time.Now().UTC()
	archived := ArchivedTask{
		Task: Task{
			Status:      StatusDone,
			Description: "Completed task",
			CreatedAt:   now.Add(-24 * time.Hour),
			UpdatedAt:   now,
		},
		ArchivedAt: now,
		Summary:    "Task was completed successfully",
		TotalTime:  Duration(2 * time.Hour),
	}

	if archived.Status != StatusDone {
		t.Errorf("expected status done, got %s", archived.Status)
	}
	if archived.Summary != "Task was completed successfully" {
		t.Errorf("expected summary, got %s", archived.Summary)
	}
}

func TestFormatLocalTime(t *testing.T) {
	// Zero time
	if FormatLocalTime(time.Time{}) != "" {
		t.Error("expected empty string for zero time")
	}

	// Non-zero time (just verify it doesn't panic and returns non-empty)
	result := FormatLocalTime(time.Now())
	if result == "" {
		t.Error("expected non-empty result for non-zero time")
	}
}

func TestFormatRelativeTime(t *testing.T) {
	// Zero time
	if FormatRelativeTime(time.Time{}) != "" {
		t.Error("expected empty string for zero time")
	}

	// Just now
	result := FormatRelativeTime(time.Now())
	if result != "just now" {
		t.Errorf("expected 'just now', got %s", result)
	}

	// Minutes ago
	result = FormatRelativeTime(time.Now().Add(-5 * time.Minute))
	if !strings.Contains(result, "minutes ago") {
		t.Errorf("expected 'X minutes ago', got %s", result)
	}

	// 1 minute ago
	result = FormatRelativeTime(time.Now().Add(-1 * time.Minute))
	if result != "1 minute ago" {
		t.Errorf("expected '1 minute ago', got %s", result)
	}

	// Hours ago
	result = FormatRelativeTime(time.Now().Add(-3 * time.Hour))
	if !strings.Contains(result, "hours ago") {
		t.Errorf("expected 'X hours ago', got %s", result)
	}

	// 1 hour ago
	result = FormatRelativeTime(time.Now().Add(-1 * time.Hour))
	if result != "1 hour ago" {
		t.Errorf("expected '1 hour ago', got %s", result)
	}

	// Yesterday
	result = FormatRelativeTime(time.Now().Add(-30 * time.Hour))
	if result != "yesterday" {
		t.Errorf("expected 'yesterday', got %s", result)
	}

	// Days ago
	result = FormatRelativeTime(time.Now().Add(-4 * 24 * time.Hour))
	if !strings.Contains(result, "days ago") {
		t.Errorf("expected 'X days ago', got %s", result)
	}

	// Older than 7 days - should fall back to formatted date
	result = FormatRelativeTime(time.Now().Add(-10 * 24 * time.Hour))
	if strings.Contains(result, "ago") {
		t.Errorf("expected formatted date, got %s", result)
	}
}

func TestContextUnmarshalJSON_OldNotesFormat(t *testing.T) {
	// Old format: notes as map[string][]string
	oldJSON := `{
		"learnings": [],
		"decisions": [],
		"notes": {"task-1": ["Note 1", "Note 2"]}
	}`

	var ctx Context
	if err := json.Unmarshal([]byte(oldJSON), &ctx); err != nil {
		t.Fatalf("failed to unmarshal old notes format: %v", err)
	}

	if len(ctx.Notes["task-1"]) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(ctx.Notes["task-1"]))
	}

	if ctx.Notes["task-1"][0].Text != "Note 1" {
		t.Errorf("expected 'Note 1', got %s", ctx.Notes["task-1"][0].Text)
	}

	if ctx.Notes["task-1"][0].ID == "" {
		t.Error("expected generated ID for migrated note")
	}
}

func TestContextUnmarshalJSON_NewNotesFormat(t *testing.T) {
	// New format: notes as map[string][]Note
	newJSON := `{
		"learnings": [],
		"decisions": [],
		"notes": {"task-1": [{"id": "abc123", "text": "New note", "created_at": "2024-01-01T00:00:00Z"}]}
	}`

	var ctx Context
	if err := json.Unmarshal([]byte(newJSON), &ctx); err != nil {
		t.Fatalf("failed to unmarshal new notes format: %v", err)
	}

	if len(ctx.Notes["task-1"]) != 1 {
		t.Fatalf("expected 1 note, got %d", len(ctx.Notes["task-1"]))
	}

	if ctx.Notes["task-1"][0].ID != "abc123" {
		t.Errorf("expected ID 'abc123', got %s", ctx.Notes["task-1"][0].ID)
	}
}

func TestContextUnmarshalJSON_EmptyFields(t *testing.T) {
	emptyJSON := `{}`

	var ctx Context
	if err := json.Unmarshal([]byte(emptyJSON), &ctx); err != nil {
		t.Fatalf("failed to unmarshal empty context: %v", err)
	}

	if ctx.Learnings == nil {
		t.Error("expected empty learnings slice, not nil")
	}

	if ctx.Notes == nil {
		t.Error("expected empty notes map, not nil")
	}
}

func TestLearningScope(t *testing.T) {
	learning := Learning{
		ID:        "abc123",
		Text:      "API error handling pattern",
		IsRule:    true,
		Scope:     "src/api/**",
		CreatedAt: time.Now().UTC(),
	}

	if learning.Scope != "src/api/**" {
		t.Errorf("expected scope 'src/api/**', got %s", learning.Scope)
	}

	// Test JSON serialization preserves scope
	data, err := json.Marshal(learning)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled Learning
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.Scope != "src/api/**" {
		t.Error("Scope not preserved after JSON round-trip")
	}
}

func TestValidTransition_UnknownStatus(t *testing.T) {
	result := ValidTransition(Status("unknown"), StatusReady)
	if result {
		t.Error("expected false for unknown status")
	}
}
