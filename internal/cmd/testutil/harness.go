// Package testutil provides test utilities for CLI command testing.
package testutil

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/iheanyi/tasuku/internal/cmd/config"
	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

// Harness provides test utilities for CLI commands.
type Harness struct {
	t       *testing.T
	tempDir string
	store   *store.DirStore
	stdout  *bytes.Buffer
	stderr  *bytes.Buffer
	origDir string
}

// New creates a new test harness with a temporary storage directory.
func New(t *testing.T) *Harness {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "tasuku-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Initialize storage in temp directory
	storePath := filepath.Join(tempDir, ".tasuku")
	s := store.NewDirStore(storePath)
	if err := s.Init(); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to init store: %v", err)
	}

	// Change to temp directory so commands find the store
	origDir, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	h := &Harness{
		t:       t,
		tempDir: tempDir,
		store:   s,
		stdout:  &bytes.Buffer{},
		stderr:  &bytes.Buffer{},
		origDir: origDir,
	}

	// Register cleanup
	t.Cleanup(func() {
		os.Chdir(h.origDir)
		os.RemoveAll(h.tempDir)
	})

	return h
}

// Store returns the test storage.
func (h *Harness) Store() *store.DirStore {
	return h.store
}

// TempDir returns the temporary directory path.
func (h *Harness) TempDir() string {
	return h.tempDir
}

// Stdout returns captured stdout.
func (h *Harness) Stdout() string {
	return h.stdout.String()
}

// Stderr returns captured stderr.
func (h *Harness) Stderr() string {
	return h.stderr.String()
}

// ResetOutput clears captured output buffers.
func (h *Harness) ResetOutput() {
	h.stdout.Reset()
	h.stderr.Reset()
}

// Execute runs a cobra command with the given arguments and captures output.
func (h *Harness) Execute(cmd *cobra.Command, args ...string) error {
	h.ResetOutput()

	// Reset output format to default
	config.OutputFormat = "table"

	// Capture os.Stdout and os.Stderr since commands use fmt.Print*
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Also set up command output for cobra's own messaging
	cmd.SetOut(wOut)
	cmd.SetErr(wErr)
	cmd.SetArgs(args)

	// Reset flags to defaults - visit all subcommands too
	resetFlags(cmd)

	err := cmd.Execute()

	// Restore stdout/stderr and read captured output
	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)
	h.stdout.Write(outBytes)
	h.stderr.Write(errBytes)

	return err
}

// resetFlags recursively resets all flags to their default values
func resetFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
		// Handle different flag types properly
		switch v := f.Value.(type) {
		case pflag.SliceValue:
			// For slice types, replace with empty slice
			v.Replace([]string{})
		default:
			f.Value.Set(f.DefValue)
		}
	})
	for _, sub := range cmd.Commands() {
		resetFlags(sub)
	}
}

// ExecuteWithFormat runs a command with a specific output format.
func (h *Harness) ExecuteWithFormat(cmd *cobra.Command, format string, args ...string) error {
	h.ResetOutput()

	// Set output format
	config.OutputFormat = format

	// Capture os.Stdout and os.Stderr since commands use fmt.Print*
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Also set up command output for cobra's own messaging
	cmd.SetOut(wOut)
	cmd.SetErr(wErr)
	cmd.SetArgs(args)

	// Reset flags to defaults - visit all subcommands too
	resetFlags(cmd)

	err := cmd.Execute()

	// Restore stdout/stderr and read captured output
	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)
	h.stdout.Write(outBytes)
	h.stderr.Write(errBytes)

	return err
}

// AddTask is a helper to add a task to the store.
func (h *Harness) AddTask(id, description string) error {
	return h.store.AddTask(id, description)
}

// AddTaskWithStatus adds a task and sets its status.
func (h *Harness) AddTaskWithStatus(id, description string, status task.Status) error {
	if err := h.store.AddTask(id, description); err != nil {
		return err
	}
	return h.store.SetStatus(id, status)
}

// AddTaskWithPriority adds a task with a priority.
func (h *Harness) AddTaskWithPriority(id, description string, priority int) error {
	return h.store.AddTaskWithPriority(id, description, &priority)
}

// GetTask retrieves a task from the store.
func (h *Harness) GetTask(id string) (*task.Task, error) {
	f, err := h.store.Read()
	if err != nil {
		return nil, err
	}
	t, ok := f.Tasks[id]
	if !ok {
		return nil, nil
	}
	return &t, nil
}

// MustGetTask retrieves a task or fails the test.
func (h *Harness) MustGetTask(id string) task.Task {
	h.t.Helper()
	t, err := h.GetTask(id)
	if err != nil {
		h.t.Fatalf("failed to get task %s: %v", id, err)
	}
	if t == nil {
		h.t.Fatalf("task %s not found", id)
	}
	return *t
}

// TaskExists checks if a task exists in the store.
func (h *Harness) TaskExists(id string) bool {
	t, _ := h.GetTask(id)
	return t != nil
}

// AddLearning adds a learning to the store.
func (h *Harness) AddLearning(text string) (string, error) {
	return h.store.AddLearning(text)
}

// AddDecision adds a decision to the store.
func (h *Harness) AddDecision(id, chose string, over []string, because string) error {
	return h.store.AddDecision(task.Decision{
		ID:      id,
		Chose:   chose,
		Over:    over,
		Because: because,
	})
}

// AddNote adds a note to a task.
func (h *Harness) AddNote(taskID, text string) (string, error) {
	return h.store.AddNote(taskID, text)
}

// StartTimerAt starts a timer on a task at a specific time (for testing stale timers).
func (h *Harness) StartTimerAt(taskID string, at time.Time) error {
	return h.store.StartTimerAt(taskID, at)
}

// AssertTaskStatus asserts that a task has the expected status.
func (h *Harness) AssertTaskStatus(id string, expected task.Status) {
	h.t.Helper()
	t := h.MustGetTask(id)
	if t.Status != expected {
		h.t.Errorf("task %s: expected status %s, got %s", id, expected, t.Status)
	}
}

// AssertTaskDescription asserts that a task has the expected description.
func (h *Harness) AssertTaskDescription(id string, expected string) {
	h.t.Helper()
	t := h.MustGetTask(id)
	if t.Description != expected {
		h.t.Errorf("task %s: expected description %q, got %q", id, expected, t.Description)
	}
}

// AssertOutputContains asserts that stdout contains the expected string.
func (h *Harness) AssertOutputContains(expected string) {
	h.t.Helper()
	if !bytes.Contains(h.stdout.Bytes(), []byte(expected)) {
		h.t.Errorf("expected output to contain %q, got:\n%s", expected, h.stdout.String())
	}
}

// AssertOutputNotContains asserts that stdout does not contain the string.
func (h *Harness) AssertOutputNotContains(notExpected string) {
	h.t.Helper()
	if bytes.Contains(h.stdout.Bytes(), []byte(notExpected)) {
		h.t.Errorf("expected output not to contain %q, got:\n%s", notExpected, h.stdout.String())
	}
}

// AssertErrorContains asserts that stderr contains the expected string.
func (h *Harness) AssertErrorContains(expected string) {
	h.t.Helper()
	if !bytes.Contains(h.stderr.Bytes(), []byte(expected)) {
		h.t.Errorf("expected error to contain %q, got:\n%s", expected, h.stderr.String())
	}
}

// AssertNoError asserts that the error is nil.
func (h *Harness) AssertNoError(err error) {
	h.t.Helper()
	if err != nil {
		h.t.Errorf("unexpected error: %v", err)
	}
}

// AssertError asserts that an error occurred.
func (h *Harness) AssertError(err error) {
	h.t.Helper()
	if err == nil {
		h.t.Error("expected an error, got nil")
	}
}

// AssertErrorContainsMsg asserts that the error message contains the expected string.
func (h *Harness) AssertErrorContainsMsg(err error, expected string) {
	h.t.Helper()
	if err == nil {
		h.t.Errorf("expected error containing %q, got nil", expected)
		return
	}
	if !bytes.Contains([]byte(err.Error()), []byte(expected)) {
		h.t.Errorf("expected error to contain %q, got: %v", expected, err)
	}
}
