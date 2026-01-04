package task

import (
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
