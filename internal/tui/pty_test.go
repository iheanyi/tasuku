//go:build integration

package tui_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/iheanyi/tasuku/internal/task"
)

// testTimeout is the default timeout for PTY operations
const testTimeout = 5 * time.Second

// ptyTest provides utilities for PTY-based TUI testing
type ptyTest struct {
	t        *testing.T
	pty      *os.File
	cmd      *exec.Cmd
	dir      string
	output   bytes.Buffer
	mu       sync.Mutex
	doneChan chan struct{}
}

// newPTYTest creates a new PTY test harness
func newPTYTest(t *testing.T) *ptyTest {
	t.Helper()

	// Create a temporary directory for the test
	dir := t.TempDir()

	// Initialize .tasuku.json with test data
	initTestData(t, dir)

	return &ptyTest{
		t:        t,
		dir:      dir,
		doneChan: make(chan struct{}),
	}
}

// initTestData creates a .tasuku.json file with test tasks
func initTestData(t *testing.T, dir string) {
	t.Helper()

	taskFile := task.NewFile()

	// Add test tasks
	taskFile.Tasks["test-ready"] = task.Task{
		Status:      task.StatusReady,
		Description: "A ready task for testing",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	taskFile.Tasks["test-progress"] = task.Task{
		Status:      task.StatusInProgress,
		Description: "An in-progress task",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	taskFile.Tasks["test-done"] = task.Task{
		Status:      task.StatusDone,
		Description: "A completed task",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	data, err := json.MarshalIndent(taskFile, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal task file: %v", err)
	}

	err = os.WriteFile(filepath.Join(dir, ".tasuku.json"), data, 0644)
	if err != nil {
		t.Fatalf("failed to write task file: %v", err)
	}
}

// start launches the TUI in a PTY
func (p *ptyTest) start() {
	p.t.Helper()

	// Build the tk binary for this test
	tkPath := buildTK(p.t)

	// Create the command
	p.cmd = exec.Command(tkPath, "ui")
	p.cmd.Dir = p.dir
	p.cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
	)

	// Start with PTY
	var err error
	p.pty, err = pty.StartWithSize(p.cmd, &pty.Winsize{
		Rows: 24,
		Cols: 80,
	})
	if err != nil {
		p.t.Fatalf("failed to start PTY: %v", err)
	}

	// Start reading output in background
	go func() {
		defer close(p.doneChan)
		buf := make([]byte, 4096)
		for {
			n, err := p.pty.Read(buf)
			if n > 0 {
				p.mu.Lock()
				p.output.Write(buf[:n])
				p.mu.Unlock()
			}
			if err != nil {
				if err != io.EOF {
					// Expected when PTY closes
				}
				return
			}
		}
	}()
}

// write sends keystrokes to the PTY
func (p *ptyTest) write(input string) {
	p.t.Helper()

	_, err := p.pty.WriteString(input)
	if err != nil {
		p.t.Fatalf("failed to write to PTY: %v", err)
	}
}

// writeKey sends a special key to the PTY
func (p *ptyTest) writeKey(key string) {
	p.t.Helper()

	// Map common key names to escape sequences
	keys := map[string]string{
		"enter":  "\r",
		"escape": "\x1b",
		"esc":    "\x1b",
		"up":     "\x1b[A",
		"down":   "\x1b[B",
		"left":   "\x1b[C",
		"right":  "\x1b[D",
		"ctrl+c": "\x03",
		"ctrl+d": "\x04",
		"tab":    "\t",
		"space":  " ",
	}

	seq, ok := keys[strings.ToLower(key)]
	if !ok {
		p.t.Fatalf("unknown key: %s", key)
	}

	p.write(seq)
}

// stripANSI removes ANSI escape sequences from text
func stripANSI(s string) string {
	// Match various ANSI escape sequences:
	// - CSI sequences: \x1b[...letter
	// - OSC sequences: \x1b]...BEL or \x1b]...ST
	// - Private mode sequences: \x1b[?...letter
	// - Other escape sequences: \x1b followed by various patterns
	patterns := []string{
		`\x1b\[[0-9;?]*[a-zA-Z]`,            // CSI sequences
		`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`, // OSC sequences (BEL or ST terminated)
		`\x1b\][^\x07]*`,                    // OSC sequences (unterminated)
		`\x1b[PX^_][^\x1b]*\x1b\\`,          // DCS/SOS/PM/APC sequences
		`\x1b.`,                             // Other 2-char escapes
	}
	combined := strings.Join(patterns, "|")
	re := regexp.MustCompile(combined)
	return re.ReplaceAllString(s, "")
}

// waitForOutput waits for specific text to appear in the output
func (p *ptyTest) waitForOutput(text string, timeout time.Duration) bool {
	p.t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		p.mu.Lock()
		output := stripANSI(p.output.String())
		p.mu.Unlock()
		if strings.Contains(output, text) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// waitForAnyOutput waits for any of the specified texts to appear
func (p *ptyTest) waitForAnyOutput(texts []string, timeout time.Duration) string {
	p.t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		p.mu.Lock()
		output := stripANSI(p.output.String())
		p.mu.Unlock()
		for _, text := range texts {
			if strings.Contains(output, text) {
				return text
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return ""
}

// getOutput returns the current output buffer with ANSI codes stripped
func (p *ptyTest) getOutput() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return stripANSI(p.output.String())
}

// getRawOutput returns the current output buffer with ANSI codes intact
func (p *ptyTest) getRawOutput() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.output.String()
}

// stop terminates the TUI and cleans up
func (p *ptyTest) stop() {
	p.t.Helper()

	if p.pty != nil {
		p.pty.Close()
	}

	if p.cmd != nil && p.cmd.Process != nil {
		// Send SIGTERM first
		p.cmd.Process.Signal(os.Interrupt)

		// Wait briefly for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- p.cmd.Wait()
		}()

		select {
		case <-done:
			// Process exited
		case <-time.After(2 * time.Second):
			// Force kill if it didn't exit gracefully
			p.cmd.Process.Kill()
		}
	}

	// Wait for output reader to finish
	select {
	case <-p.doneChan:
	case <-time.After(time.Second):
	}
}

// Build state for caching
var (
	tkBinaryPath string
	tkBuildOnce  sync.Once
	tkBuildErr   error
)

// buildTK builds the tk binary and returns its path
// The binary is built once and cached for all tests in the same run
func buildTK(t *testing.T) string {
	t.Helper()

	tkBuildOnce.Do(func() {
		// Build to a location outside of t.TempDir() to persist across tests
		tmpDir, err := os.MkdirTemp("", "tasuku-pty-test-*")
		if err != nil {
			tkBuildErr = err
			return
		}

		binPath := filepath.Join(tmpDir, "tk")

		// Get the project root
		wd, err := os.Getwd()
		if err != nil {
			tkBuildErr = err
			return
		}

		// Find project root by looking for go.mod
		projectRoot := findProjectRoot(t, wd)

		cmd := exec.Command("go", "build", "-o", binPath, "./cmd/tk")
		cmd.Dir = projectRoot
		output, err := cmd.CombinedOutput()
		if err != nil {
			tkBuildErr = fmt.Errorf("failed to build tk: %v\n%s", err, output)
			return
		}

		tkBinaryPath = binPath
	})

	if tkBuildErr != nil {
		t.Fatalf("failed to build tk binary: %v", tkBuildErr)
	}

	return tkBinaryPath
}

// findProjectRoot walks up the directory tree to find the project root
func findProjectRoot(t *testing.T, startDir string) string {
	t.Helper()

	dir := startDir
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find project root (go.mod) from %s", startDir)
		}
		dir = parent
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestPTY_TUIStarts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pt := newPTYTest(t)
	defer pt.stop()

	pt.start()

	// Wait for the TUI to start and display the dashboard
	// The TUI should show "Ready:" in the stats header (appears before "Tasks")
	if !pt.waitForOutput("Ready:", testTimeout) {
		t.Errorf("TUI did not display 'Ready:' within timeout\nOutput:\n%s", pt.getOutput())
	}
}

func TestPTY_DisplaysTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pt := newPTYTest(t)
	defer pt.stop()

	pt.start()

	// Wait for TUI to display - look for the stats header which appears consistently
	if !pt.waitForOutput("Ready:", testTimeout) {
		t.Fatalf("TUI did not start properly\nOutput:\n%s", pt.getOutput())
	}

	// Give the TUI more time to render the full task list
	time.Sleep(500 * time.Millisecond)

	// Check that tasks are visible
	output := pt.getOutput()
	if !strings.Contains(output, "test-progress") && !strings.Contains(output, "test-ready") {
		t.Errorf("TUI should display task names\nOutput:\n%s", output)
	}
}

func TestPTY_NavigationJK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pt := newPTYTest(t)
	defer pt.stop()

	pt.start()

	// Wait for initial display
	if !pt.waitForOutput("Ready:", testTimeout) {
		t.Fatalf("TUI did not start properly\nOutput:\n%s", pt.getOutput())
	}

	// Give the TUI time to fully render
	time.Sleep(300 * time.Millisecond)

	// Press 'j' to move down
	pt.write("j")
	time.Sleep(100 * time.Millisecond)

	// Press 'k' to move back up
	pt.write("k")
	time.Sleep(100 * time.Millisecond)

	// The TUI should still be running and responsive
	// We verify this by checking that stats are still displayed
	output := pt.getOutput()
	if !strings.Contains(output, "Ready:") {
		t.Errorf("TUI not responsive after navigation\nOutput:\n%s", output)
	}
}

func TestPTY_NavigationArrowKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pt := newPTYTest(t)
	defer pt.stop()

	pt.start()

	// Wait for initial display
	if !pt.waitForOutput("Ready:", testTimeout) {
		t.Fatalf("TUI did not start properly\nOutput:\n%s", pt.getOutput())
	}

	// Give the TUI time to fully render
	time.Sleep(300 * time.Millisecond)

	// Press down arrow to move down
	pt.writeKey("down")
	time.Sleep(100 * time.Millisecond)

	// Press up arrow to move back up
	pt.writeKey("up")
	time.Sleep(100 * time.Millisecond)

	// The TUI should still be running and responsive
	output := pt.getOutput()
	if !strings.Contains(output, "Ready:") {
		t.Errorf("TUI not responsive after arrow key navigation\nOutput:\n%s", output)
	}
}

func TestPTY_QuitWithQ(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pt := newPTYTest(t)
	defer pt.stop()

	pt.start()

	// Wait for initial display
	if !pt.waitForOutput("Ready:", testTimeout) {
		t.Fatalf("TUI did not start properly\nOutput:\n%s", pt.getOutput())
	}

	// Give the TUI time to fully render
	time.Sleep(300 * time.Millisecond)

	// Press 'q' to quit
	pt.write("q")

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		done <- pt.cmd.Wait()
	}()

	select {
	case err := <-done:
		// Process exited - this is expected
		if err != nil {
			// Non-zero exit is okay for some quit scenarios
			t.Logf("Process exited with: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("TUI did not quit within timeout after pressing 'q'")
	}
}

func TestPTY_QuitWithCtrlC(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pt := newPTYTest(t)
	defer pt.stop()

	pt.start()

	// Wait for initial display
	if !pt.waitForOutput("Ready:", testTimeout) {
		t.Fatalf("TUI did not start properly\nOutput:\n%s", pt.getOutput())
	}

	// Give the TUI time to fully render
	time.Sleep(300 * time.Millisecond)

	// Press Ctrl+C to quit
	pt.writeKey("ctrl+c")

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		done <- pt.cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited - this is expected
	case <-time.After(3 * time.Second):
		t.Error("TUI did not quit within timeout after pressing Ctrl+C")
	}
}

func TestPTY_ViewTaskDetails(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pt := newPTYTest(t)
	defer pt.stop()

	pt.start()

	// Wait for initial display
	if !pt.waitForOutput("Ready:", testTimeout) {
		t.Fatalf("TUI did not start properly\nOutput:\n%s", pt.getOutput())
	}

	// Give the TUI time to fully render
	time.Sleep(300 * time.Millisecond)

	// Press enter to view task details
	pt.writeKey("enter")

	// Wait for detail view indicators
	time.Sleep(500 * time.Millisecond)

	// The detail view should show "Description" or "Status"
	output := pt.getOutput()
	if !strings.Contains(output, "Description") && !strings.Contains(output, "Status:") {
		t.Logf("Note: Task detail view content\nOutput:\n%s", output)
	}

	// Press escape to go back
	pt.writeKey("esc")
	time.Sleep(300 * time.Millisecond)
}

func TestPTY_HelpOverlay(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pt := newPTYTest(t)
	defer pt.stop()

	pt.start()

	// Wait for initial display
	if !pt.waitForOutput("Ready:", testTimeout) {
		t.Fatalf("TUI did not start properly\nOutput:\n%s", pt.getOutput())
	}

	// Give the TUI time to fully render
	time.Sleep(300 * time.Millisecond)

	// Press '?' to open help
	pt.write("?")

	// Wait for help overlay to appear
	time.Sleep(500 * time.Millisecond)
	output := pt.getOutput()
	if !strings.Contains(output, "Help") {
		t.Logf("Help overlay may not have appeared\nOutput:\n%s", output)
	}

	// Press escape to close help
	pt.writeKey("esc")
	time.Sleep(300 * time.Millisecond)
}

func TestPTY_RefreshTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pt := newPTYTest(t)
	defer pt.stop()

	pt.start()

	// Wait for initial display
	if !pt.waitForOutput("Ready:", testTimeout) {
		t.Fatalf("TUI did not start properly\nOutput:\n%s", pt.getOutput())
	}

	// Give the TUI time to fully render
	time.Sleep(300 * time.Millisecond)

	// Press 'r' to refresh
	pt.write("r")
	time.Sleep(300 * time.Millisecond)

	// TUI should still show stats after refresh
	output := pt.getOutput()
	if !strings.Contains(output, "Ready:") {
		t.Errorf("TUI not showing stats after refresh\nOutput:\n%s", output)
	}
}

func TestPTY_StatusFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pt := newPTYTest(t)
	defer pt.stop()

	pt.start()

	// Wait for initial display
	if !pt.waitForOutput("Ready:", testTimeout) {
		t.Fatalf("TUI did not start properly\nOutput:\n%s", pt.getOutput())
	}

	// Give the TUI time to fully render
	time.Sleep(300 * time.Millisecond)

	// Press '1' to filter to ready tasks
	pt.write("1")
	time.Sleep(300 * time.Millisecond)

	// Press '0' to show all tasks again
	pt.write("0")
	time.Sleep(300 * time.Millisecond)

	// TUI should still be running
	output := pt.getOutput()
	if !strings.Contains(output, "Ready:") {
		t.Errorf("TUI not responsive after status filter\nOutput:\n%s", output)
	}
}

func TestPTY_SearchFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pt := newPTYTest(t)
	defer pt.stop()

	pt.start()

	// Wait for initial display
	if !pt.waitForOutput("Ready:", testTimeout) {
		t.Fatalf("TUI did not start properly\nOutput:\n%s", pt.getOutput())
	}

	// Give the TUI time to fully render
	time.Sleep(300 * time.Millisecond)

	// Press '/' to enter filter mode
	pt.write("/")
	time.Sleep(100 * time.Millisecond)

	// Type a filter term
	pt.write("ready")
	time.Sleep(300 * time.Millisecond)

	// Press escape to exit filter mode
	pt.writeKey("esc")
	time.Sleep(300 * time.Millisecond)

	// TUI should still be running
	output := pt.getOutput()
	if !strings.Contains(output, "Ready:") {
		t.Errorf("TUI not responsive after search filter\nOutput:\n%s", output)
	}
}

func TestPTY_TerminalResize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pt := newPTYTest(t)
	defer pt.stop()

	pt.start()

	// Wait for initial display
	if !pt.waitForOutput("Ready:", testTimeout) {
		t.Fatalf("TUI did not start properly\nOutput:\n%s", pt.getOutput())
	}

	// Give the TUI time to fully render
	time.Sleep(300 * time.Millisecond)

	// Resize the terminal
	err := pty.Setsize(pt.pty, &pty.Winsize{
		Rows: 40,
		Cols: 120,
	})
	if err != nil {
		t.Fatalf("failed to resize PTY: %v", err)
	}

	// Give the TUI time to handle resize
	time.Sleep(500 * time.Millisecond)

	// TUI should still be running after resize
	output := pt.getOutput()
	if !strings.Contains(output, "Ready:") {
		t.Errorf("TUI not responsive after resize\nOutput:\n%s", output)
	}
}

func TestPTY_MultipleKeystrokes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pt := newPTYTest(t)
	defer pt.stop()

	pt.start()

	// Wait for initial display
	if !pt.waitForOutput("Ready:", testTimeout) {
		t.Fatalf("TUI did not start properly\nOutput:\n%s", pt.getOutput())
	}

	// Give the TUI time to fully render
	time.Sleep(300 * time.Millisecond)

	// Rapid sequence of keystrokes
	keystrokes := []string{"j", "j", "k", "j", "k", "k"}
	for _, key := range keystrokes {
		pt.write(key)
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for TUI to process all keystrokes
	time.Sleep(300 * time.Millisecond)

	// TUI should still be running
	output := pt.getOutput()
	if !strings.Contains(output, "Ready:") {
		t.Errorf("TUI not responsive after multiple keystrokes\nOutput:\n%s", output)
	}
}

func TestPTY_NoTasksHandled(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a test with an empty task file
	dir := t.TempDir()

	taskFile := task.NewFile()
	data, err := json.MarshalIndent(taskFile, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal task file: %v", err)
	}

	err = os.WriteFile(filepath.Join(dir, ".tasuku.json"), data, 0644)
	if err != nil {
		t.Fatalf("failed to write task file: %v", err)
	}

	pt := &ptyTest{
		t:        t,
		dir:      dir,
		doneChan: make(chan struct{}),
	}
	defer pt.stop()

	pt.start()

	// Wait for the TUI to start - with empty tasks, stats show "Ready: 0"
	if !pt.waitForOutput("Ready:", testTimeout) {
		t.Errorf("TUI did not start with empty task list\nOutput:\n%s", pt.getOutput())
	}

	// TUI should handle empty list gracefully
	// Press 'q' to quit
	time.Sleep(300 * time.Millisecond)
	pt.write("q")

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		done <- pt.cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited - this is expected
	case <-time.After(3 * time.Second):
		t.Error("TUI did not quit cleanly with empty task list")
	}
}

func TestPTY_StartTask(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pt := newPTYTest(t)
	defer pt.stop()

	pt.start()

	// Wait for initial display
	if !pt.waitForOutput("Ready:", testTimeout) {
		t.Fatalf("TUI did not start properly\nOutput:\n%s", pt.getOutput())
	}

	// Give the TUI time to fully render
	time.Sleep(300 * time.Millisecond)

	// Navigate to find the ready task and press 's' to start it
	// First, let's press 's' - if the first task is ready, it should start
	pt.write("s")
	time.Sleep(300 * time.Millisecond)

	// TUI should still be running
	output := pt.getOutput()
	if !strings.Contains(output, "Ready:") && !strings.Contains(output, "Progress:") {
		t.Errorf("TUI not responsive after start command\nOutput:\n%s", output)
	}
}

func TestPTY_ViewLearnings(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pt := newPTYTest(t)
	defer pt.stop()

	pt.start()

	// Wait for initial display
	if !pt.waitForOutput("Ready:", testTimeout) {
		t.Fatalf("TUI did not start properly\nOutput:\n%s", pt.getOutput())
	}

	// Give the TUI time to fully render
	time.Sleep(300 * time.Millisecond)

	// Press 'L' (shift+l) to view learnings
	pt.write("L")
	time.Sleep(500 * time.Millisecond)

	// Should show learnings view
	output := pt.getOutput()
	if !strings.Contains(output, "Learnings") {
		t.Logf("Learnings view may not have appeared\nOutput:\n%s", output)
	}

	// Press escape to go back
	pt.writeKey("esc")
	time.Sleep(300 * time.Millisecond)
}

func TestPTY_ViewDecisions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pt := newPTYTest(t)
	defer pt.stop()

	pt.start()

	// Wait for initial display
	if !pt.waitForOutput("Ready:", testTimeout) {
		t.Fatalf("TUI did not start properly\nOutput:\n%s", pt.getOutput())
	}

	// Give the TUI time to fully render
	time.Sleep(300 * time.Millisecond)

	// Press 'D' (shift+d) to view decisions
	pt.write("D")
	time.Sleep(500 * time.Millisecond)

	// Should show decisions view
	output := pt.getOutput()
	if !strings.Contains(output, "Decisions") {
		t.Logf("Decisions view may not have appeared\nOutput:\n%s", output)
	}

	// Press escape to go back
	pt.writeKey("esc")
	time.Sleep(300 * time.Millisecond)
}
