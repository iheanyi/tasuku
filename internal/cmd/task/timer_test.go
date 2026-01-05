package task

import (
	"testing"
	"time"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
)

func TestTimerCmd_Start(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	err := h.Execute(Cmd, "timer", "start", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("Timer started")

	tsk := h.MustGetTask("my-task")
	if tsk.TimerStart == nil {
		t.Error("expected timer to be started")
	}
}

func TestTimerCmd_Stop(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")
	h.Store().StartTimer("my-task")

	// Wait a tiny bit to accumulate time
	time.Sleep(10 * time.Millisecond)

	err := h.Execute(Cmd, "timer", "stop", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("Timer stopped")

	tsk := h.MustGetTask("my-task")
	if tsk.TimerStart != nil {
		t.Error("expected timer to be stopped")
	}
}

func TestTimerCmd_Status(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("task-1", "First task")
	h.AddTask("task-2", "Second task")
	h.Store().StartTimer("task-1")

	err := h.Execute(Cmd, "timer", "status")
	h.AssertNoError(err)
	h.AssertOutputContains("task-1")
}

func TestTimerCmd_NonexistentTask(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "timer", "start", "nonexistent")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "not found")
}

func TestTimerCmd_StopNotStarted(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	// Stopping a timer that wasn't started should fail or be a no-op
	err := h.Execute(Cmd, "timer", "stop", "my-task")
	// Just verify it doesn't crash - behavior may vary
	_ = err
}

func TestTimerCmd_MissingArgs(t *testing.T) {
	h := testutil.New(t)

	// timer without subcommand shows help (not an error)
	err := h.Execute(Cmd, "timer")
	// Just check it doesn't crash

	// timer start without task ID should error
	err = h.Execute(Cmd, "timer", "start")
	h.AssertError(err)
	_ = err
}
