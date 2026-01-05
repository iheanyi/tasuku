package context

import (
	"strings"
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
	"github.com/iheanyi/tasuku/internal/task"
)

func TestShowCmd(t *testing.T) {
	h := testutil.New(t)

	// Add some test data
	h.AddTask("task-1", "First task")
	h.AddTaskWithStatus("task-2", "Second task", task.StatusDone)
	h.AddLearning("Test learning")
	h.AddDecision("test-decision", "Option A", []string{"Option B"}, "Better fit")

	err := h.Execute(Cmd, "show")
	h.AssertNoError(err)

	output := h.Stdout()
	if !strings.Contains(output, "task-1") {
		t.Errorf("expected output to contain task-1, got:\n%s", output)
	}
	if !strings.Contains(output, "Test learning") || !strings.Contains(output, "learnings") {
		t.Errorf("expected output to contain learnings, got:\n%s", output)
	}
}

func TestShowCmdJSON(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("task-1", "First task")
	h.AddLearning("Test learning")

	err := h.ExecuteWithFormat(Cmd, "json", "show")
	h.AssertNoError(err)

	output := h.Stdout()
	if !strings.Contains(output, `"version"`) {
		t.Errorf("expected JSON output with version, got:\n%s", output)
	}
	if !strings.Contains(output, `"task-1"`) {
		t.Errorf("expected JSON output with task-1, got:\n%s", output)
	}
}

func TestShowCmdYAML(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("task-1", "First task")

	err := h.ExecuteWithFormat(Cmd, "yaml", "show")
	h.AssertNoError(err)

	output := h.Stdout()
	if !strings.Contains(output, "version:") {
		t.Errorf("expected YAML output with version, got:\n%s", output)
	}
}

func TestValidateCmd(t *testing.T) {
	h := testutil.New(t)

	// Add valid task
	h.AddTask("valid-task", "A valid task")

	err := h.Execute(Cmd, "validate")
	h.AssertNoError(err)
	h.AssertOutputContains("Validation passed")
}

func TestValidateCmdEmptyDescription(t *testing.T) {
	h := testutil.New(t)

	// Add task with empty description by manipulating store directly
	h.Store().Update(func(f *task.File) error {
		f.Tasks["bad-task"] = task.Task{
			Status:      task.StatusReady,
			Description: "",
		}
		return nil
	})

	err := h.Execute(Cmd, "validate")
	h.AssertError(err)
	if !strings.Contains(err.Error(), "empty description") {
		t.Errorf("expected error about empty description, got: %v", err)
	}
}

func TestValidateCmdInvalidStatus(t *testing.T) {
	h := testutil.New(t)

	// Add task with invalid status
	h.Store().Update(func(f *task.File) error {
		f.Tasks["bad-task"] = task.Task{
			Status:      task.Status("invalid"),
			Description: "Task with bad status",
		}
		return nil
	})

	err := h.Execute(Cmd, "validate")
	h.AssertError(err)
	if !strings.Contains(err.Error(), "invalid status") {
		t.Errorf("expected error about invalid status, got: %v", err)
	}
}

func TestValidateCmdCircularDependency(t *testing.T) {
	h := testutil.New(t)

	// Create circular dependency: A -> B -> A
	h.Store().Update(func(f *task.File) error {
		f.Tasks["task-a"] = task.Task{
			Status:      task.StatusBlocked,
			Description: "Task A",
			BlockedBy:   []string{"task-b"},
		}
		f.Tasks["task-b"] = task.Task{
			Status:      task.StatusBlocked,
			Description: "Task B",
			BlockedBy:   []string{"task-a"},
		}
		return nil
	})

	err := h.Execute(Cmd, "validate")
	h.AssertError(err)
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("expected error about circular dependency, got: %v", err)
	}
}

func TestSchemaCmd(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "schema")
	h.AssertNoError(err)

	output := h.Stdout()
	if !strings.Contains(output, `"$schema"`) {
		t.Errorf("expected JSON schema output, got:\n%s", output)
	}
	if !strings.Contains(output, `"tasks"`) {
		t.Errorf("expected schema to define tasks, got:\n%s", output)
	}
	if !strings.Contains(output, `"context"`) {
		t.Errorf("expected schema to define context, got:\n%s", output)
	}
}
