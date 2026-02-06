package testutil

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

func TestHarness_Basic(t *testing.T) {
	h := New(t)

	// Test that store is initialized
	if h.Store() == nil {
		t.Fatal("store should not be nil")
	}

	// Test adding a task
	if err := h.AddTask("test-task", "Test description"); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Test getting task
	tsk := h.MustGetTask("test-task")
	if tsk.Description != "Test description" {
		t.Errorf("expected description %q, got %q", "Test description", tsk.Description)
	}
	if tsk.Status != task.StatusReady {
		t.Errorf("expected status %s, got %s", task.StatusReady, tsk.Status)
	}
}

func TestHarness_AddTaskWithStatus(t *testing.T) {
	h := New(t)

	if err := h.AddTaskWithStatus("in-progress-task", "Working on it", task.StatusInProgress); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	h.AssertTaskStatus("in-progress-task", task.StatusInProgress)
}

func TestHarness_Execute(t *testing.T) {
	h := New(t)

	// Create a simple test command
	cmd := &cobra.Command{
		Use: "test",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("hello world")
		},
	}

	err := h.Execute(cmd)
	h.AssertNoError(err)
	h.AssertOutputContains("hello world")
}

func TestHarness_TaskExists(t *testing.T) {
	h := New(t)

	if h.TaskExists("nonexistent") {
		t.Error("task should not exist")
	}

	h.AddTask("exists", "I exist")

	if !h.TaskExists("exists") {
		t.Error("task should exist")
	}
}

func TestHarness_AssertTaskDescription(t *testing.T) {
	h := New(t)
	h.AddTask("my-task", "Original description")
	h.AssertTaskDescription("my-task", "Original description")
}

func TestHarness_Learning(t *testing.T) {
	h := New(t)

	id, err := h.AddLearning("Test learning")
	if err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}
	if id == "" {
		t.Error("learning ID should not be empty")
	}
}

// TestHarness_UsesLatestStorage is a regression guard ensuring the test harness
// uses the same storage format that AutoDetect resolves to in production.
// If this test fails, it means the harness and production storage backends
// have diverged — all command tests would be testing the wrong backend.
func TestHarness_UsesLatestStorage(t *testing.T) {
	h := New(t)

	storageType := store.DetectStorageType(h.TempDir())
	if storageType != store.LatestStorageType {
		t.Fatalf("test harness uses storage type %d, but LatestStorageType is %d — update the harness", storageType, store.LatestStorageType)
	}
}

func TestHarness_Note(t *testing.T) {
	h := New(t)

	h.AddTask("task-with-note", "A task")
	id, err := h.AddNote("task-with-note", "A note")
	if err != nil {
		t.Fatalf("failed to add note: %v", err)
	}
	if id == "" {
		t.Error("note ID should not be empty")
	}
}
