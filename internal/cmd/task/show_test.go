package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
	itask "github.com/iheanyi/tasuku/internal/task"
)

func TestShowCmd_Basic(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task description")

	err := h.Execute(Cmd, "show", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("ID:")
	h.AssertOutputContains("my-task")
	h.AssertOutputContains("Description:")
	h.AssertOutputContains("My task description")
	h.AssertOutputContains("Status:")
	h.AssertOutputContains("ready")
}

func TestShowCmd_WithOwner(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "Task with owner")
	h.Store().SetOwner("my-task", "agent-1")

	err := h.Execute(Cmd, "show", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("Owner:")
	h.AssertOutputContains("agent-1")
}

func TestShowCmd_WithBlockers(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("blocker-1", "First blocker")
	h.AddTask("blocker-2", "Second blocker")
	h.AddTask("blocked-task", "Blocked task")
	h.Store().BlockTask("blocked-task", []string{"blocker-1", "blocker-2"})

	err := h.Execute(Cmd, "show", "blocked-task")
	h.AssertNoError(err)
	h.AssertOutputContains("Blocked by:")
	h.AssertOutputContains("blocker-1")
	h.AssertOutputContains("blocker-2")
}

func TestShowCmd_WithTags(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("tagged-task", "Task with tags")
	h.Store().AddTag("tagged-task", "frontend")
	h.Store().AddTag("tagged-task", "urgent")

	err := h.Execute(Cmd, "show", "tagged-task")
	h.AssertNoError(err)
	h.AssertOutputContains("Tags:")
	h.AssertOutputContains("frontend")
	h.AssertOutputContains("urgent")
}

func TestShowCmd_WithNotes(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "Task with notes")
	h.AddNote("my-task", "First note")
	h.AddNote("my-task", "Second note")

	err := h.Execute(Cmd, "show", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("Notes:")
	h.AssertOutputContains("First note")
	h.AssertOutputContains("Second note")
}

func TestShowCmd_WithPriority(t *testing.T) {
	h := testutil.New(t)

	h.AddTaskWithPriority("critical-task", "Critical task", itask.PriorityCritical)

	err := h.Execute(Cmd, "show", "critical-task")
	h.AssertNoError(err)
	h.AssertOutputContains("Priority:")
	h.AssertOutputContains("critical")
}

func TestShowCmd_WithSubtasks(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("parent-task", "Parent task")
	h.Store().AddSubtask("child-1", "First child", "parent-task")
	h.Store().AddSubtask("child-2", "Second child", "parent-task")

	err := h.Execute(Cmd, "show", "parent-task")
	h.AssertNoError(err)
	h.AssertOutputContains("Subtasks:")
	h.AssertOutputContains("child-1")
	h.AssertOutputContains("child-2")
}

func TestShowCmd_JSONFormat(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	err := h.ExecuteWithFormat(Cmd, "json", "show", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains(`"id": "my-task"`)
	h.AssertOutputContains(`"description": "My task"`)
}

func TestShowCmd_YAMLFormat(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	err := h.ExecuteWithFormat(Cmd, "yaml", "show", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("id: my-task")
	h.AssertOutputContains("description: My task")
}

func TestShowCmd_NonexistentTask(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "show", "nonexistent")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "not found")
}

func TestShowCmd_MissingArg(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "show")
	h.AssertError(err)
}
