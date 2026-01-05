package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
	itask "github.com/iheanyi/tasuku/internal/task"
)

func TestAddCmd_Basic(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "add", "My new task")
	h.AssertNoError(err)
	h.AssertOutputContains("Created task:")

	// Verify task was created
	if !h.TaskExists("my-new-task") {
		t.Error("expected task 'my-new-task' to be created")
	}

	tsk := h.MustGetTask("my-new-task")
	if tsk.Description != "My new task" {
		t.Errorf("expected description 'My new task', got %q", tsk.Description)
	}
	if tsk.Status != itask.StatusReady {
		t.Errorf("expected status 'ready', got %s", tsk.Status)
	}
}

func TestAddCmd_WithCustomID(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "add", "My task", "--id", "custom-id")
	h.AssertNoError(err)
	h.AssertOutputContains("custom-id")

	if !h.TaskExists("custom-id") {
		t.Error("expected task 'custom-id' to be created")
	}
}

func TestAddCmd_WithPriority(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "add", "Critical task", "--priority", "0")
	h.AssertNoError(err)
	h.AssertOutputContains("critical")

	tsk := h.MustGetTask("critical-task")
	if tsk.GetPriority() != itask.PriorityCritical {
		t.Errorf("expected critical priority (0), got %d", tsk.GetPriority())
	}
}

func TestAddCmd_WithTags(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "add", "Tagged task", "--tag", "frontend", "--tag", "urgent")
	h.AssertNoError(err)
	h.AssertOutputContains("frontend")
	h.AssertOutputContains("urgent")

	tsk := h.MustGetTask("tagged-task")
	if !contains(tsk.Tags, "frontend") {
		t.Error("expected task to have 'frontend' tag")
	}
	if !contains(tsk.Tags, "urgent") {
		t.Error("expected task to have 'urgent' tag")
	}
}

func TestAddCmd_WithParent(t *testing.T) {
	h := testutil.New(t)

	// Create parent first
	h.AddTask("parent-task", "Parent task")

	err := h.Execute(Cmd, "add", "Child task", "--parent", "parent-task")
	h.AssertNoError(err)
	h.AssertOutputContains("subtask of: parent-task")

	tsk := h.MustGetTask("child-task")
	if tsk.ParentID == nil || *tsk.ParentID != "parent-task" {
		t.Error("expected task to have parent-task as parent")
	}
}

func TestAddCmd_MissingDescription(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "add")
	h.AssertError(err)
}

func TestAddCmd_IDCollision(t *testing.T) {
	h := testutil.New(t)

	// Add first task
	err := h.Execute(Cmd, "add", "Test task")
	h.AssertNoError(err)

	if !h.TaskExists("test-task") {
		t.Error("expected 'test-task' to exist")
	}

	// Add task with same description - should get different ID or fail
	err = h.Execute(Cmd, "add", "Test task")
	if err != nil {
		// If it errors (duplicate ID), that's expected behavior
		return
	}

	// If it succeeded, the second task should have a different ID
	// Check if collision handling generated a unique ID
	if h.TaskExists("test-task-1") || h.TaskExists("test-task-2") {
		// Collision handling worked
		return
	}

	// Check if it just failed silently by still having only one task
	f, _ := h.Store().Read()
	if len(f.Tasks) != 2 {
		t.Errorf("expected 2 tasks after collision, got %d", len(f.Tasks))
	}
}

func TestAddCmd_AllPriorityLevels(t *testing.T) {
	h := testutil.New(t)
	priorities := []struct {
		level    string
		expected int
	}{
		{"0", itask.PriorityCritical},
		{"1", itask.PriorityHigh},
		{"2", itask.PriorityNormal},
		{"3", itask.PriorityLow},
		{"4", itask.PriorityBacklog},
	}

	for _, p := range priorities {
		err := h.Execute(Cmd, "add", "Task "+p.level, "--id", "task-"+p.level, "--priority", p.level)
		h.AssertNoError(err)

		tsk := h.MustGetTask("task-" + p.level)
		if tsk.GetPriority() != p.expected {
			t.Errorf("priority %s: expected %d, got %d", p.level, p.expected, tsk.GetPriority())
		}
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
