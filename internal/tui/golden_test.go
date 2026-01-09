package tui

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

func init() {
	// Force ASCII color profile for consistent golden file output across environments
	lipgloss.SetColorProfile(termenv.Ascii)

	// Force UTC timezone for consistent timestamp rendering in golden tests
	os.Setenv("TZ", "UTC")
}

// testTermWidth and testTermHeight provide consistent terminal dimensions for tests
const (
	testTermWidth  = 80
	testTermHeight = 24
)

// setupGoldenTestStore creates a temporary store with deterministic test data
// Note: Tasks are sorted by status (in_progress first) then priority.
// To ensure deterministic ordering, we assign different priorities to tasks
// with the same status.
func setupGoldenTestStore(t *testing.T) *store.Store {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")

	s := store.New(path)
	if err := s.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add tasks with various statuses and different priorities for deterministic ordering
	// Priority values: 0=critical, 1=high, 2=normal, 3=low, 4=backlog

	// Ready tasks - assign different priorities for deterministic ordering
	_ = s.AddTask("implement-auth", "Implement user authentication system")
	_ = s.SetPriority("implement-auth", 3) // low priority

	_ = s.AddTask("write-tests", "Write unit tests for auth module")
	_ = s.SetPriority("write-tests", 2) // normal priority (will show before implement-auth)

	// In-progress task
	_ = s.AddTask("fix-bug-123", "Fix login page redirect bug")
	_ = s.SetStatus("fix-bug-123", task.StatusInProgress)
	_ = s.SetPriority("fix-bug-123", 2) // normal priority

	// Done task
	_ = s.AddTask("deploy-staging", "Deploy to staging environment")
	_ = s.SetStatus("deploy-staging", task.StatusDone)
	_ = s.SetPriority("deploy-staging", 2) // normal priority

	// Blocked task
	_ = s.AddTask("prod-deploy", "Deploy to production")
	_ = s.BlockTask("prod-deploy", []string{"write-tests"})
	_ = s.SetPriority("prod-deploy", 2) // normal priority

	// Add tags to a task
	_ = s.AddTag("implement-auth", "backend")
	_ = s.AddTag("implement-auth", "security")

	// Add learnings with fixed timestamps for deterministic golden files
	_, _ = s.AddLearning("Always validate user input on both client and server side")
	_, _ = s.AddLearning("Never store passwords in plain text")

	// Fix learning timestamps
	_ = s.Update(func(f *task.File) error {
		for i := range f.Context.Learnings {
			f.Context.Learnings[i].CreatedAt = time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
		}
		return nil
	})

	// Add a decision
	_ = s.AddDecision(task.Decision{
		ID:        "auth-decision",
		Chose:     "JWT tokens",
		Over:      []string{"Session cookies", "OAuth only"},
		Because:   "Better for API-first architecture and mobile clients",
		CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	})

	// Add notes to a task with a fixed timestamp for deterministic golden files
	// We first add the note, then read and update the timestamp
	_, _ = s.AddNote("fix-bug-123", "Root cause identified: missing null check in redirect handler")

	// Fix the note timestamp for deterministic tests
	_ = s.Update(func(f *task.File) error {
		if notes, ok := f.Context.Notes["fix-bug-123"]; ok && len(notes) > 0 {
			notes[0].CreatedAt = time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
			f.Context.Notes["fix-bug-123"] = notes
		}
		return nil
	})

	return s
}

// createTestModel creates a test TUI model with the given store.
// It processes the async Init() command to load tasks before returning.
func createTestModel(t *testing.T, s *store.Store) Model {
	t.Helper()

	m, err := New(s)
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	// Set terminal size for consistent rendering
	m.width = testTermWidth
	m.height = testTermHeight
	m.taskList.SetSize(testTermWidth-4, testTermHeight-8)
	m.progress.Width = testTermWidth - 4

	// Process the Init() command to load tasks asynchronously
	// This simulates what tea.NewProgram does: call Init(), then process the resulting message
	cmd := m.Init()
	if cmd != nil {
		msg := cmd()
		newModel, _ := m.Update(msg)
		model := newModel.(Model)
		// Restore terminal size after reinitializing task list
		model.taskList.SetSize(testTermWidth-4, testTermHeight-8)
		model.progress.Width = testTermWidth - 4
		return model
	}

	return *m
}

// TestGolden_InitialDashboard tests the initial dashboard rendering
func TestGolden_InitialDashboard(t *testing.T) {
	s := setupGoldenTestStore(t)
	m := createTestModel(t, s)

	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(testTermWidth, testTermHeight),
	)

	// Wait a bit for initial render
	time.Sleep(100 * time.Millisecond)

	// Send quit to get final output
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(time.Second*2)))
	if err != nil {
		t.Fatal(err)
	}

	teatest.RequireEqualOutput(t, out)
}

// TestGolden_TaskListWithStatuses tests the task list showing different statuses
func TestGolden_TaskListWithStatuses(t *testing.T) {
	s := setupGoldenTestStore(t)
	m := createTestModel(t, s)

	// Render the view directly for golden comparison
	output := m.View()

	// Compare with golden file
	teatest.RequireEqualOutput(t, []byte(output))
}

// TestGolden_HelpOverlay tests the help overlay (? key)
func TestGolden_HelpOverlay(t *testing.T) {
	s := setupGoldenTestStore(t)
	m := createTestModel(t, s)

	// Simulate pressing '?' key to open help
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = newModel.(Model)

	// Render the view directly for golden comparison
	output := m.View()

	// Compare with golden file
	teatest.RequireEqualOutput(t, []byte(output))
}

// TestGolden_TaskDetailView tests the task detail view (enter key)
func TestGolden_TaskDetailView(t *testing.T) {
	s := setupGoldenTestStore(t)
	m := createTestModel(t, s)

	// Simulate pressing enter to view task details
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	// Render the view directly for golden comparison
	output := m.View()

	// Compare with golden file
	teatest.RequireEqualOutput(t, []byte(output))
}

// TestGolden_ConfirmationDialog tests the archive confirmation dialog
func TestGolden_ConfirmationDialog(t *testing.T) {
	s := setupGoldenTestStore(t)
	m := createTestModel(t, s)

	// First filter to done tasks
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m = newModel.(Model)

	// Then press 'a' to trigger archive confirmation
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = newModel.(Model)

	// Render the view directly for golden comparison
	output := m.View()

	// Compare with golden file
	teatest.RequireEqualOutput(t, []byte(output))
}

// TestGolden_LearningsView tests the learnings view (L key)
func TestGolden_LearningsView(t *testing.T) {
	s := setupGoldenTestStore(t)
	m := createTestModel(t, s)

	// Simulate pressing 'L' to view learnings
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	m = newModel.(Model)

	// Render the view directly for golden comparison
	output := m.View()

	// Compare with golden file
	teatest.RequireEqualOutput(t, []byte(output))
}

// TestGolden_DecisionsView tests the decisions view (D key)
func TestGolden_DecisionsView(t *testing.T) {
	s := setupGoldenTestStore(t)
	m := createTestModel(t, s)

	// Simulate pressing 'D' to view decisions
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	m = newModel.(Model)

	// Render the view directly for golden comparison
	output := m.View()

	// Compare with golden file
	teatest.RequireEqualOutput(t, []byte(output))
}

// TestGolden_StatusFilters tests filtering by status using number keys
func TestGolden_StatusFilters(t *testing.T) {
	tests := []struct {
		name   string
		key    rune
		filter string
	}{
		{"FilterAll", '0', "all"},
		{"FilterReady", '1', "ready"},
		{"FilterInProgress", '2', "in_progress"},
		{"FilterBlocked", '3', "blocked"},
		{"FilterDone", '4', "done"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := setupGoldenTestStore(t)
			m := createTestModel(t, s)

			// Apply the filter
			newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tt.key}})
			m = newModel.(Model)

			// Render the view directly for golden comparison
			output := m.View()

			// Compare with golden file
			teatest.RequireEqualOutput(t, []byte(output))
		})
	}
}

// TestGolden_PrioritySort tests toggling priority sort (p key)
func TestGolden_PrioritySort(t *testing.T) {
	s := setupGoldenTestStore(t)

	// Add tasks with different priorities
	high := 1
	critical := 0
	_ = s.SetPriority("implement-auth", critical)
	_ = s.SetPriority("fix-bug-123", high)

	m := createTestModel(t, s)

	// Toggle priority sort
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = newModel.(Model)

	// Render the view directly for golden comparison
	output := m.View()

	// Compare with golden file
	teatest.RequireEqualOutput(t, []byte(output))
}

// TestGolden_EmptyTaskList tests rendering when there are no tasks
func TestGolden_EmptyTaskList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")

	s := store.New(path)
	if err := s.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	m := createTestModel(t, s)

	// Render the view directly for golden comparison
	output := m.View()

	// Compare with golden file
	teatest.RequireEqualOutput(t, []byte(output))
}

// TestGolden_NotesView tests the notes view for a task (N key)
func TestGolden_NotesView(t *testing.T) {
	s := setupGoldenTestStore(t)
	m := createTestModel(t, s)

	// The first selected task may not have notes, but 'N' will still show notes view
	// for the currently selected task
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("N")})
	m = newModel.(Model)

	// Render the view directly for golden comparison
	output := m.View()

	// Compare with golden file
	teatest.RequireEqualOutput(t, []byte(output))
}

// TestGolden_BulkArchiveConfirmation tests the bulk archive confirmation (A key)
func TestGolden_BulkArchiveConfirmation(t *testing.T) {
	s := setupGoldenTestStore(t)
	m := createTestModel(t, s)

	// Press 'A' to trigger bulk archive confirmation
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A")})
	m = newModel.(Model)

	// Render the view directly for golden comparison
	output := m.View()

	// Compare with golden file
	teatest.RequireEqualOutput(t, []byte(output))
}

// TestGolden_NavigateAndViewDetail tests navigation and viewing details of a specific task
func TestGolden_NavigateAndViewDetail(t *testing.T) {
	s := setupGoldenTestStore(t)
	m := createTestModel(t, s)

	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(testTermWidth, testTermHeight),
	)

	// Navigate down and view details
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for task detail view
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Description")) || bytes.Contains(bts, []byte("Status:"))
	}, teatest.WithCheckInterval(time.Millisecond*50), teatest.WithDuration(time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(time.Second*2)))
	if err != nil {
		t.Fatal(err)
	}

	teatest.RequireEqualOutput(t, out)
}

// TestGolden_TaskWithTags tests rendering of a task that has tags
func TestGolden_TaskWithTags(t *testing.T) {
	s := setupGoldenTestStore(t)
	m := createTestModel(t, s)

	// Navigate to task with tags (implement-auth)
	// First, find it by navigating
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)

	// Then view its details
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	// If not the right task, we can check which one is selected
	// For this test, we'll set the selected task directly
	m.selected = "implement-auth"
	m.view = ViewTaskDetail

	// Render the view directly for golden comparison
	output := m.View()

	// Compare with golden file
	teatest.RequireEqualOutput(t, []byte(output))
}

// TestGolden_BlockedTaskView tests viewing a blocked task's details
func TestGolden_BlockedTaskView(t *testing.T) {
	s := setupGoldenTestStore(t)
	m := createTestModel(t, s)

	// Set up to view the blocked task directly
	m.selected = "prod-deploy"
	m.view = ViewTaskDetail

	// Render the view directly for golden comparison
	output := m.View()

	// Compare with golden file
	teatest.RequireEqualOutput(t, []byte(output))
}
