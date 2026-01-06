package v4

import (
	"testing"
	"time"

	"github.com/iheanyi/tasuku/internal/task"
)

func TestParseLearningsFile(t *testing.T) {
	content := `# Learnings

## abc123 - 2024-01-05
Never use O(n²) algorithms when O(n log n) alternatives exist.

` + "```go" + `
// Bad: O(n²)
for _, a := range items {
    for _, b := range items { ... }
}

// Good: O(n)
lookup := make(map[string]Item)
for _, item := range items {
    lookup[item.ID] = item
}
` + "```" + `

## def456 - 2024-01-05
Always ensure switch cases return early in TUI apps to prevent fall-through.
`

	result, err := ParseLearningsFile([]byte(content))
	if err != nil {
		t.Fatalf("ParseLearningsFile() error = %v", err)
	}

	if len(result.Learnings) != 2 {
		t.Fatalf("Learnings count = %d, want 2", len(result.Learnings))
	}

	// First learning
	l1 := result.Learnings[0]
	if l1.ID != "abc123" {
		t.Errorf("First learning ID = %q, want abc123", l1.ID)
	}
	if !l1.CreatedAt.Equal(time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("First learning date = %v, want 2024-01-05", l1.CreatedAt)
	}
	if !l1.IsRule {
		t.Error("First learning should be detected as rule (starts with 'Never')")
	}

	// Second learning
	l2 := result.Learnings[1]
	if l2.ID != "def456" {
		t.Errorf("Second learning ID = %q, want def456", l2.ID)
	}
	if !l2.IsRule {
		t.Error("Second learning should be detected as rule (starts with 'Always')")
	}
}

func TestWriteLearningsFile(t *testing.T) {
	learnings := []task.Learning{
		{
			ID:        "abc123",
			Text:      "Never do this.",
			IsRule:    true,
			CreatedAt: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        "def456",
			Text:      "Always do that.",
			IsRule:    true,
			CreatedAt: time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC),
		},
	}

	content := WriteLearningsFile(learnings)

	// Parse it back
	result, err := ParseLearningsFile(content)
	if err != nil {
		t.Fatalf("Failed to parse written content: %v\nContent:\n%s", err, string(content))
	}

	if len(result.Learnings) != 2 {
		t.Fatalf("Roundtrip learnings count = %d, want 2", len(result.Learnings))
	}

	if result.Learnings[0].ID != "abc123" {
		t.Errorf("Roundtrip first ID = %q, want abc123", result.Learnings[0].ID)
	}
	if result.Learnings[0].Text != "Never do this." {
		t.Errorf("Roundtrip first text = %q, want 'Never do this.'", result.Learnings[0].Text)
	}
}

func TestParseDecisionsFile(t *testing.T) {
	content := `# Decisions

## auth-strategy - 2024-01-05
**Chose**: JWT tokens
**Over**: Session cookies, OAuth2
**Because**: Stateless, scalable, no session storage overhead. Works across microservices.

## database-choice - 2024-01-05
**Chose**: PostgreSQL
**Over**: MySQL, MongoDB
**Because**: Better JSON support, complex query performance, mature ecosystem.
`

	result, err := ParseDecisionsFile([]byte(content))
	if err != nil {
		t.Fatalf("ParseDecisionsFile() error = %v", err)
	}

	if len(result.Decisions) != 2 {
		t.Fatalf("Decisions count = %d, want 2", len(result.Decisions))
	}

	// First decision
	d1 := result.Decisions[0]
	if d1.ID != "auth-strategy" {
		t.Errorf("First decision ID = %q, want auth-strategy", d1.ID)
	}
	if d1.Chose != "JWT tokens" {
		t.Errorf("First decision Chose = %q, want 'JWT tokens'", d1.Chose)
	}
	if len(d1.Over) != 2 {
		t.Errorf("First decision Over count = %d, want 2", len(d1.Over))
	} else {
		if d1.Over[0] != "Session cookies" {
			t.Errorf("First decision Over[0] = %q, want 'Session cookies'", d1.Over[0])
		}
		if d1.Over[1] != "OAuth2" {
			t.Errorf("First decision Over[1] = %q, want 'OAuth2'", d1.Over[1])
		}
	}
	if d1.Because != "Stateless, scalable, no session storage overhead. Works across microservices." {
		t.Errorf("First decision Because = %q", d1.Because)
	}

	// Second decision
	d2 := result.Decisions[1]
	if d2.ID != "database-choice" {
		t.Errorf("Second decision ID = %q, want database-choice", d2.ID)
	}
	if d2.Chose != "PostgreSQL" {
		t.Errorf("Second decision Chose = %q, want 'PostgreSQL'", d2.Chose)
	}
}

func TestWriteDecisionsFile(t *testing.T) {
	decisions := []task.Decision{
		{
			ID:        "auth-strategy",
			Chose:     "JWT tokens",
			Over:      []string{"Session cookies", "OAuth2"},
			Because:   "Stateless and scalable.",
			CreatedAt: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        "db-choice",
			Chose:     "PostgreSQL",
			Over:      []string{"MySQL"},
			Because:   "Better JSON support.",
			CreatedAt: time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC),
		},
	}

	content := WriteDecisionsFile(decisions)

	// Parse it back
	result, err := ParseDecisionsFile(content)
	if err != nil {
		t.Fatalf("Failed to parse written content: %v\nContent:\n%s", err, string(content))
	}

	if len(result.Decisions) != 2 {
		t.Fatalf("Roundtrip decisions count = %d, want 2", len(result.Decisions))
	}

	if result.Decisions[0].ID != "auth-strategy" {
		t.Errorf("Roundtrip first ID = %q, want auth-strategy", result.Decisions[0].ID)
	}
	if result.Decisions[0].Chose != "JWT tokens" {
		t.Errorf("Roundtrip first Chose = %q, want 'JWT tokens'", result.Decisions[0].Chose)
	}
	if len(result.Decisions[0].Over) != 2 {
		t.Errorf("Roundtrip first Over count = %d, want 2", len(result.Decisions[0].Over))
	}
}

func TestEmptyFiles(t *testing.T) {
	// Empty learnings file
	learnings, err := ParseLearningsFile([]byte("# Learnings\n"))
	if err != nil {
		t.Errorf("ParseLearningsFile empty error = %v", err)
	}
	if len(learnings.Learnings) != 0 {
		t.Errorf("Empty learnings count = %d, want 0", len(learnings.Learnings))
	}

	// Empty decisions file
	decisions, err := ParseDecisionsFile([]byte("# Decisions\n"))
	if err != nil {
		t.Errorf("ParseDecisionsFile empty error = %v", err)
	}
	if len(decisions.Decisions) != 0 {
		t.Errorf("Empty decisions count = %d, want 0", len(decisions.Decisions))
	}
}

func TestRFC3339Timestamps(t *testing.T) {
	// Test that RFC3339 format parses correctly (new format)
	learningsContent := `# Learnings

## abc123 - 2024-01-05T10:30:00Z
Learning with full timestamp.

## def456 - 2024-01-05T14:45:30Z
Another learning later the same day.
`

	learnings, err := ParseLearningsFile([]byte(learningsContent))
	if err != nil {
		t.Fatalf("ParseLearningsFile() error = %v", err)
	}

	if len(learnings.Learnings) != 2 {
		t.Fatalf("Learnings count = %d, want 2", len(learnings.Learnings))
	}

	// Check first learning timestamp
	expected1 := time.Date(2024, 1, 5, 10, 30, 0, 0, time.UTC)
	if !learnings.Learnings[0].CreatedAt.Equal(expected1) {
		t.Errorf("First learning timestamp = %v, want %v", learnings.Learnings[0].CreatedAt, expected1)
	}

	// Check second learning timestamp
	expected2 := time.Date(2024, 1, 5, 14, 45, 30, 0, time.UTC)
	if !learnings.Learnings[1].CreatedAt.Equal(expected2) {
		t.Errorf("Second learning timestamp = %v, want %v", learnings.Learnings[1].CreatedAt, expected2)
	}

	// Test decisions with RFC3339
	decisionsContent := `# Decisions

## auth-v1 - 2024-01-05T09:00:00Z
**Chose**: JWT
**Over**: Sessions
**Because**: Stateless.

## auth-v2 - 2024-01-05T16:30:00Z
**Chose**: Sessions
**Over**: JWT
**Because**: Changed our mind after performance testing.
`

	decisions, err := ParseDecisionsFile([]byte(decisionsContent))
	if err != nil {
		t.Fatalf("ParseDecisionsFile() error = %v", err)
	}

	if len(decisions.Decisions) != 2 {
		t.Fatalf("Decisions count = %d, want 2", len(decisions.Decisions))
	}

	// Verify timestamps show ordering
	if !decisions.Decisions[0].CreatedAt.Before(decisions.Decisions[1].CreatedAt) {
		t.Error("Expected auth-v1 to be before auth-v2 based on timestamps")
	}

	// Verify the decision was overridden later the same day
	if decisions.Decisions[0].CreatedAt.Day() != decisions.Decisions[1].CreatedAt.Day() {
		t.Error("Both decisions should be on the same day")
	}
}

func TestMultiLineBecause(t *testing.T) {
	content := `# Decisions

## complex-decision - 2024-01-05
**Chose**: Option A
**Over**: Option B
**Because**: This is the first line.
This is the second line.
And a third line with reasoning.
`

	result, err := ParseDecisionsFile([]byte(content))
	if err != nil {
		t.Fatalf("ParseDecisionsFile() error = %v", err)
	}

	if len(result.Decisions) != 1 {
		t.Fatalf("Decisions count = %d, want 1", len(result.Decisions))
	}

	expected := "This is the first line.\nThis is the second line.\nAnd a third line with reasoning."
	if result.Decisions[0].Because != expected {
		t.Errorf("Multi-line Because = %q, want %q", result.Decisions[0].Because, expected)
	}
}
