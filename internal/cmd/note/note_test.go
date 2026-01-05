package note

import (
	"strings"
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
)

func TestNoteListEmpty(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputContains("No notes recorded")
}

func TestNoteListAll(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("task-1", "First task")
	h.AddTask("task-2", "Second task")
	h.AddNote("task-1", "Note for task 1")
	h.AddNote("task-2", "Note for task 2")

	err := h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputContains("task-1")
	h.AssertOutputContains("Note for task 1")
	h.AssertOutputContains("task-2")
	h.AssertOutputContains("Note for task 2")
}

func TestNoteListForTask(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("task-1", "First task")
	h.AddTask("task-2", "Second task")
	h.AddNote("task-1", "Note A for task 1")
	h.AddNote("task-1", "Note B for task 1")
	h.AddNote("task-2", "Note for task 2")

	err := h.Execute(Cmd, "list", "task-1")
	h.AssertNoError(err)
	h.AssertOutputContains("Note A for task 1")
	h.AssertOutputContains("Note B for task 1")
	h.AssertOutputNotContains("Note for task 2")
}

func TestNoteListForTaskNotFound(t *testing.T) {
	h := testutil.New(t)

	// Add a task but no notes for it, then query a different task
	h.AddTask("task-1", "First task")
	h.AddNote("task-1", "Note for task 1")

	err := h.Execute(Cmd, "list", "nonexistent")
	h.AssertError(err)
	if !strings.Contains(err.Error(), "no notes found") {
		t.Errorf("expected 'no notes found' error, got: %v", err)
	}
}

func TestNoteListJSON(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("task-1", "First task")
	h.AddNote("task-1", "Test note")

	err := h.ExecuteWithFormat(Cmd, "json", "list")
	h.AssertNoError(err)

	output := h.Stdout()
	if !strings.Contains(output, `"task-1"`) {
		t.Errorf("expected JSON output with task ID, got:\n%s", output)
	}
}

func TestNoteListYAML(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("task-1", "First task")
	h.AddNote("task-1", "Test note")

	err := h.ExecuteWithFormat(Cmd, "yaml", "list")
	h.AssertNoError(err)

	output := h.Stdout()
	if !strings.Contains(output, "task-1:") {
		t.Errorf("expected YAML output, got:\n%s", output)
	}
}

func TestNoteAdd(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	err := h.Execute(Cmd, "add", "my-task", "Started implementation")
	h.AssertNoError(err)
	h.AssertOutputContains("Note")
	h.AssertOutputContains("added to: my-task")

	// Verify it was added
	err = h.Execute(Cmd, "list", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("Started implementation")
}

func TestNoteAddMultiple(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	h.Execute(Cmd, "add", "my-task", "First note")
	h.Execute(Cmd, "add", "my-task", "Second note")
	h.Execute(Cmd, "add", "my-task", "Third note")

	err := h.Execute(Cmd, "list", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("First note")
	h.AssertOutputContains("Second note")
	h.AssertOutputContains("Third note")
}

func TestNoteRemove(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")
	h.AddNote("my-task", "Note to keep")
	noteID, _ := h.AddNote("my-task", "Note to remove")

	err := h.Execute(Cmd, "remove", "my-task", noteID)
	h.AssertNoError(err)
	h.AssertOutputContains("Removed note")

	// Verify the removed note is gone but other note remains
	err = h.Execute(Cmd, "list", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("Note to keep")
	h.AssertOutputNotContains("Note to remove")
}

func TestNoteRemoveNotFound(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")
	h.AddNote("my-task", "Existing note")

	err := h.Execute(Cmd, "remove", "my-task", "nonexistent-id")
	h.AssertError(err)
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestNoteRemoveWrongTask(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("task-1", "Task 1")
	h.AddTask("task-2", "Task 2")
	noteID, _ := h.AddNote("task-1", "Note on task 1")

	// Try to remove from wrong task
	err := h.Execute(Cmd, "remove", "task-2", noteID)
	h.AssertError(err)
}
