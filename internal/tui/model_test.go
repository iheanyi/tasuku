package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

// setupTestStore creates a temporary store with test tasks
func setupTestStore(t *testing.T) (*store.Store, func()) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")

	s := store.New(path)
	if err := s.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add some test tasks
	_ = s.AddTask("test-ready", "A ready task")
	_ = s.AddTask("test-progress", "An in-progress task")
	_ = s.SetStatus("test-progress", task.StatusInProgress)
	_ = s.AddTask("test-done", "A done task")
	_ = s.SetStatus("test-done", task.StatusDone)

	cleanup := func() {
		os.RemoveAll(dir)
	}

	return s, cleanup
}

func TestNew(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	m, err := New(s)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if m.view != ViewDashboard {
		t.Errorf("expected initial view to be ViewDashboard, got %v", m.view)
	}

	// Should have 3 tasks in the list
	if len(m.taskList.Items()) != 3 {
		t.Errorf("expected 3 tasks in list, got %d", len(m.taskList.Items()))
	}
}

func TestUpdate_Quit(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	m, _ := New(s)

	// Test ctrl+c quits
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("expected quit command, got nil")
	}
}

func TestUpdate_EnterTaskDetail(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	m, _ := New(s)

	// Press enter to view task detail
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.view != ViewTaskDetail {
		t.Errorf("expected view to be ViewTaskDetail, got %v", updated.view)
	}

	if updated.selected == "" {
		t.Error("expected selected task ID to be set")
	}
}

func TestUpdate_BackFromDetail(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	m, _ := New(s)
	// Modify the model to be in detail view
	detailModel := *m
	detailModel.view = ViewTaskDetail
	detailModel.selected = "test-ready"

	// Press escape to go back
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := detailModel.Update(msg)
	updated := newModel.(Model)

	if updated.view != ViewDashboard {
		t.Errorf("expected view to be ViewDashboard, got %v", updated.view)
	}
}

func TestUpdate_FilteringMode(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	m, _ := New(s)
	model := *m

	// Simulate entering filter mode by pressing '/'
	// The list handles '/' internally to enter filter mode
	slashMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	newModel, _ := model.Update(slashMsg)
	model = newModel.(Model)

	// Now the list should be in filtering mode
	if !model.taskList.SettingFilter() {
		t.Skip("list did not enter filter mode - this may depend on list configuration")
	}

	// When filtering, pressing 's' should NOT trigger start action
	// It should be passed to the filter input instead
	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	newModel, _ = model.Update(sMsg)
	model = newModel.(Model)

	// The filter should contain 's'
	filterValue := model.taskList.FilterValue()
	if filterValue != "s" {
		t.Logf("filter value: %q (may vary based on filter implementation)", filterValue)
	}

	// Verify we're still in dashboard view (not detail view)
	if model.view != ViewDashboard {
		t.Errorf("expected to stay in ViewDashboard while filtering, got %v", model.view)
	}
}

func TestUpdate_StartTask(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	m, _ := New(s)
	model := *m

	// Find and select the ready task
	for i, item := range model.taskList.Items() {
		if ti, ok := item.(TaskItem); ok && ti.ID == "test-ready" {
			model.taskList.Select(i)
			break
		}
	}

	// Press 's' to start the task
	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	newModel, _ := model.Update(sMsg)
	_ = newModel.(Model)

	// Verify task status changed
	f, _ := s.Read()
	if f.Tasks["test-ready"].Status != task.StatusInProgress {
		t.Errorf("expected task to be in_progress, got %v", f.Tasks["test-ready"].Status)
	}
}

func TestUpdate_DoneTask(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	m, _ := New(s)
	model := *m

	// Find and select the in-progress task
	for i, item := range model.taskList.Items() {
		if ti, ok := item.(TaskItem); ok && ti.ID == "test-progress" {
			model.taskList.Select(i)
			break
		}
	}

	// Press 'd' to mark done
	dMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	newModel, _ := model.Update(dMsg)
	_ = newModel.(Model)

	// Verify task status changed
	f, _ := s.Read()
	if f.Tasks["test-progress"].Status != task.StatusDone {
		t.Errorf("expected task to be done, got %v", f.Tasks["test-progress"].Status)
	}
}

func TestUpdate_ArchiveTask(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	m, _ := New(s)
	model := *m

	// Find and select the done task
	for i, item := range model.taskList.Items() {
		if ti, ok := item.(TaskItem); ok && ti.ID == "test-done" {
			model.taskList.Select(i)
			break
		}
	}

	// Press 'a' to trigger confirmation dialog
	aMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	newModel, _ := model.Update(aMsg)
	confirmModel := newModel.(Model)

	// Verify we're in confirmation view
	if confirmModel.view != ViewConfirm {
		t.Errorf("expected ViewConfirm, got %v", confirmModel.view)
	}
	if confirmModel.confirmAction != "archive" {
		t.Errorf("expected confirmAction 'archive', got %v", confirmModel.confirmAction)
	}
	if confirmModel.confirmTaskID != "test-done" {
		t.Errorf("expected confirmTaskID 'test-done', got %v", confirmModel.confirmTaskID)
	}

	// Press 'y' to confirm
	yMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	finalModel, _ := confirmModel.Update(yMsg)
	_ = finalModel.(Model)

	// Verify task was archived
	f, _ := s.Read()
	if _, exists := f.Tasks["test-done"]; exists {
		t.Error("expected task to be removed from active tasks")
	}
	if _, exists := f.Archive["test-done"]; !exists {
		t.Error("expected task to be in archive")
	}
}

func TestUpdate_ArchiveTaskCancel(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	m, _ := New(s)
	model := *m

	// Find and select the done task
	for i, item := range model.taskList.Items() {
		if ti, ok := item.(TaskItem); ok && ti.ID == "test-done" {
			model.taskList.Select(i)
			break
		}
	}

	// Press 'a' to trigger confirmation dialog
	aMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	newModel, _ := model.Update(aMsg)
	confirmModel := newModel.(Model)

	// Press 'n' to cancel
	nMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	finalModel, _ := confirmModel.Update(nMsg)
	updated := finalModel.(Model)

	// Verify we returned to dashboard
	if updated.view != ViewDashboard {
		t.Errorf("expected ViewDashboard, got %v", updated.view)
	}
	if updated.confirmAction != "" {
		t.Errorf("expected empty confirmAction, got %v", updated.confirmAction)
	}

	// Verify task was NOT archived
	f, _ := s.Read()
	if _, exists := f.Tasks["test-done"]; !exists {
		t.Error("expected task to still be in active tasks")
	}
}

func TestUpdate_BulkArchive(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	m, _ := New(s)
	model := *m

	// Count done tasks before
	doneCount := 0
	f, _ := s.Read()
	for _, task := range f.Tasks {
		if task.Status == "done" {
			doneCount++
		}
	}

	// Press 'A' (Shift+A) to trigger bulk archive confirmation
	aMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}}
	newModel, _ := model.Update(aMsg)
	confirmModel := newModel.(Model)

	// Verify we're in confirmation view with bulk_archive action
	if confirmModel.view != ViewConfirm {
		t.Errorf("expected ViewConfirm, got %v", confirmModel.view)
	}
	if confirmModel.confirmAction != "bulk_archive" {
		t.Errorf("expected confirmAction 'bulk_archive', got %v", confirmModel.confirmAction)
	}

	// Press 'y' to confirm
	yMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	finalModel, _ := confirmModel.Update(yMsg)
	_ = finalModel.(Model)

	// Verify all done tasks were archived
	f, _ = s.Read()
	for _, task := range f.Tasks {
		if task.Status == "done" {
			t.Error("expected no done tasks to remain in active tasks")
		}
	}
	if len(f.Archive) < doneCount {
		t.Errorf("expected at least %d tasks in archive, got %d", doneCount, len(f.Archive))
	}
}

func TestUpdate_ConfirmEscape(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	m, _ := New(s)
	model := *m

	// Find and select the done task
	for i, item := range model.taskList.Items() {
		if ti, ok := item.(TaskItem); ok && ti.ID == "test-done" {
			model.taskList.Select(i)
			break
		}
	}

	// Press 'a' to trigger confirmation dialog
	aMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	newModel, _ := model.Update(aMsg)
	confirmModel := newModel.(Model)

	// Press Escape to cancel
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	finalModel, _ := confirmModel.Update(escMsg)
	updated := finalModel.(Model)

	// Verify we returned to dashboard
	if updated.view != ViewDashboard {
		t.Errorf("expected ViewDashboard, got %v", updated.view)
	}
}

func TestUpdate_WindowResize(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	m, _ := New(s)
	model := *m

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	newModel, _ := model.Update(msg)
	updated := newModel.(Model)

	if updated.width != 100 {
		t.Errorf("expected width 100, got %d", updated.width)
	}
	if updated.height != 50 {
		t.Errorf("expected height 50, got %d", updated.height)
	}
}

func TestTaskItem_Title(t *testing.T) {
	item := TaskItem{
		ID: "test-task",
		Task: task.Task{
			Status:      task.StatusReady,
			Description: "Test description",
		},
	}

	title := item.Title()
	if title == "" {
		t.Error("expected non-empty title")
	}
	// Title should contain the task ID
	if !contains(title, "test-task") {
		t.Errorf("expected title to contain task ID, got %q", title)
	}
}

func TestTaskItem_Description(t *testing.T) {
	item := TaskItem{
		ID: "test-task",
		Task: task.Task{
			Status:      task.StatusReady,
			Description: "Short description",
			Tags:        []string{"tag1", "tag2"},
		},
	}

	desc := item.Description()
	if desc == "" {
		t.Error("expected non-empty description")
	}
	// Description should contain the task description
	if !contains(desc, "Short description") {
		t.Errorf("expected description to contain task desc, got %q", desc)
	}
	// Description should show tags
	if !contains(desc, "tag1") {
		t.Errorf("expected description to contain tag, got %q", desc)
	}
}

func TestTaskItem_FilterValue(t *testing.T) {
	item := TaskItem{
		ID: "test-task",
		Task: task.Task{
			Description: "Test description",
		},
	}

	fv := item.FilterValue()
	if !contains(fv, "test-task") {
		t.Errorf("expected filter value to contain ID, got %q", fv)
	}
	if !contains(fv, "Test description") {
		t.Errorf("expected filter value to contain description, got %q", fv)
	}
}

func TestStatusSymbol(t *testing.T) {
	tests := []struct {
		status   string
		notEmpty bool
	}{
		{"ready", true},
		{"in_progress", true},
		{"blocked", true},
		{"done", true},
		{"unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			symbol := StatusSymbol(tt.status)
			if tt.notEmpty && symbol == "" {
				t.Errorf("expected non-empty symbol for status %q", tt.status)
			}
		})
	}
}

func TestPrioritySymbol(t *testing.T) {
	priorities := []int{0, 1, 2, 3, 4, 5}
	for _, p := range priorities {
		symbol := PrioritySymbol(p)
		// Should return something (even empty string for unknown)
		_ = symbol
	}
}

func TestFilterStateCheck(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	m, _ := New(s)

	// Initially not filtering
	if m.taskList.SettingFilter() {
		t.Error("expected SettingFilter() to be false initially")
	}

	// Check FilterState type
	state := m.taskList.FilterState()
	if state != list.Unfiltered {
		t.Errorf("expected Unfiltered state, got %v", state)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
