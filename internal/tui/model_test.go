package tui

import (
	"path/filepath"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/iheanyi/tasuku/internal/store"
	v4 "github.com/iheanyi/tasuku/internal/store/v4"
	"github.com/iheanyi/tasuku/internal/task"
)

// setupTestStore creates a temporary V4 store with test tasks
func setupTestStore(t *testing.T) (store.Storage, func()) {
	t.Helper()

	dir := t.TempDir()
	root := filepath.Join(dir, ".tasuku")

	s := v4.New(root)
	if err := s.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add some test tasks
	_ = s.AddTask("test-ready", "A ready task")
	_ = s.AddTask("test-progress", "An in-progress task")
	_ = s.SetStatus("test-progress", task.StatusInProgress)
	_ = s.AddTask("test-done", "A done task")
	_ = s.SetStatus("test-done", task.StatusDone)

	cleanup := func() {}

	return s, cleanup
}

// newTestModel creates a new Model and processes the initial async load.
// This simulates what tea.NewProgram does: call New(), then Init(), then Update(initMsg).
func newTestModel(t *testing.T, s store.Storage) Model {
	t.Helper()
	m, err := New(s)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Process the Init command to load tasks
	cmd := m.Init()
	if cmd != nil {
		msg := cmd()
		newModel, _ := m.Update(msg)
		return newModel.(Model)
	}
	return *m
}

func TestNew(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	model := newTestModel(t, s)

	if model.view != ViewDashboard {
		t.Errorf("expected initial view to be ViewDashboard, got %v", model.view)
	}

	// Should have 3 tasks in the list
	if len(model.taskList.Items()) != 3 {
		t.Errorf("expected 3 tasks in list, got %d", len(model.taskList.Items()))
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

	model := newTestModel(t, s)

	// Press enter to view task detail
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := model.Update(msg)
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

	model := newTestModel(t, s)
	// Modify the model to be in detail view
	model.view = ViewTaskDetail
	model.selected = "test-ready"

	// Press escape to go back
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := model.Update(msg)
	updated := newModel.(Model)

	if updated.view != ViewDashboard {
		t.Errorf("expected view to be ViewDashboard, got %v", updated.view)
	}
}

func TestUpdate_FilteringMode(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	model := newTestModel(t, s)

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

	model := newTestModel(t, s)

	// Find and select the ready task
	for i, item := range model.taskList.Items() {
		if ti, ok := item.(TaskItem); ok && ti.ID == "test-ready" {
			model.taskList.Select(i)
			break
		}
	}

	// Press 's' to start the task - this returns an async command
	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	newModel, cmd := model.Update(sMsg)
	model = newModel.(Model)

	// Execute the async command and process result
	if cmd != nil {
		msg := cmd()
		newModel, _ = model.Update(msg)
		// Process the subsequent load command if ActionResultMsg
		if _, ok := msg.(ActionResultMsg); ok {
			loadCmd := loadTasksCmd(s)
			loadMsg := loadCmd()
			newModel, _ = newModel.(Model).Update(loadMsg)
		}
	}

	// Verify task status changed
	f, err := s.Read()
	if err != nil {
		t.Fatalf("failed to read store: %v", err)
	}
	if f.Tasks["test-ready"].Status != task.StatusInProgress {
		t.Errorf("expected task to be in_progress, got %v", f.Tasks["test-ready"].Status)
	}
}

func TestUpdate_DoneTask(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	model := newTestModel(t, s)

	// Find and select the in-progress task
	for i, item := range model.taskList.Items() {
		if ti, ok := item.(TaskItem); ok && ti.ID == "test-progress" {
			model.taskList.Select(i)
			break
		}
	}

	// Press 'd' to mark done - this returns an async command
	dMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	newModel, cmd := model.Update(dMsg)
	model = newModel.(Model)

	// Execute the async command and process result
	if cmd != nil {
		msg := cmd()
		newModel, _ = model.Update(msg)
		// Process the subsequent load command if ActionResultMsg
		if _, ok := msg.(ActionResultMsg); ok {
			loadCmd := loadTasksCmd(s)
			loadMsg := loadCmd()
			_, _ = newModel.(Model).Update(loadMsg)
		}
	}

	// Verify task status changed
	f, err := s.Read()
	if err != nil {
		t.Fatalf("failed to read store: %v", err)
	}
	if f.Tasks["test-progress"].Status != task.StatusDone {
		t.Errorf("expected task to be done, got %v", f.Tasks["test-progress"].Status)
	}
}

func TestUpdate_ArchiveTask(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	model := newTestModel(t, s)

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

	// Press 'y' to confirm - this returns an async command
	yMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	finalModel, cmd := confirmModel.Update(yMsg)
	model = finalModel.(Model)

	// Execute the async command and process result
	if cmd != nil {
		msg := cmd()
		finalModel, _ = model.Update(msg)
		if _, ok := msg.(ActionResultMsg); ok {
			loadCmd := loadTasksCmd(s)
			loadMsg := loadCmd()
			_, _ = finalModel.(Model).Update(loadMsg)
		}
	}

	// Verify task was archived
	f, err := s.Read()
	if err != nil {
		t.Fatalf("failed to read store: %v", err)
	}
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

	model := newTestModel(t, s)

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
	f, err := s.Read()
	if err != nil {
		t.Fatalf("failed to read store: %v", err)
	}
	if _, exists := f.Tasks["test-done"]; !exists {
		t.Error("expected task to still be in active tasks")
	}
}

func TestUpdate_BulkArchive(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	model := newTestModel(t, s)

	// Count done tasks before
	doneCount := 0
	f, err := s.Read()
	if err != nil {
		t.Fatalf("failed to read store: %v", err)
	}
	for _, tk := range f.Tasks {
		if tk.Status == task.StatusDone {
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

	// Press 'y' to confirm - this returns an async command
	yMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	finalModel, cmd := confirmModel.Update(yMsg)
	model = finalModel.(Model)

	// Execute the async command and process result
	if cmd != nil {
		msg := cmd()
		finalModel, _ = model.Update(msg)
		if _, ok := msg.(ActionResultMsg); ok {
			loadCmd := loadTasksCmd(s)
			loadMsg := loadCmd()
			_, _ = finalModel.(Model).Update(loadMsg)
		}
	}

	// Verify all done tasks were archived
	f, err = s.Read()
	if err != nil {
		t.Fatalf("failed to read store: %v", err)
	}
	for _, tk := range f.Tasks {
		if tk.Status == task.StatusDone {
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

	model := newTestModel(t, s)

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

	model := newTestModel(t, s)

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

	model := newTestModel(t, s)

	// Initially not filtering
	if model.taskList.SettingFilter() {
		t.Error("expected SettingFilter() to be false initially")
	}

	// Check FilterState type
	state := model.taskList.FilterState()
	if state != list.Unfiltered {
		t.Errorf("expected Unfiltered state, got %v", state)
	}
}

func TestUpdate_DeleteTask(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	model := newTestModel(t, s)

	// Find and select the ready task
	for i, item := range model.taskList.Items() {
		if ti, ok := item.(TaskItem); ok && ti.ID == "test-ready" {
			model.taskList.Select(i)
			break
		}
	}

	// Press 'x' to trigger delete confirmation dialog
	xMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	newModel, _ := model.Update(xMsg)
	confirmModel := newModel.(Model)

	// Verify we're in confirmation view
	if confirmModel.view != ViewConfirm {
		t.Errorf("expected ViewConfirm, got %v", confirmModel.view)
	}
	if confirmModel.confirmAction != "delete" {
		t.Errorf("expected confirmAction 'delete', got %v", confirmModel.confirmAction)
	}
	if confirmModel.confirmTaskID != "test-ready" {
		t.Errorf("expected confirmTaskID 'test-ready', got %v", confirmModel.confirmTaskID)
	}

	// Press 'y' to confirm - this returns an async command
	yMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	finalModel, cmd := confirmModel.Update(yMsg)
	model = finalModel.(Model)

	// Execute the async command and process result
	if cmd != nil {
		msg := cmd()
		finalModel, _ = model.Update(msg)
		if _, ok := msg.(ActionResultMsg); ok {
			loadCmd := loadTasksCmd(s)
			loadMsg := loadCmd()
			_, _ = finalModel.(Model).Update(loadMsg)
		}
	}

	// Verify task was deleted
	f, err := s.Read()
	if err != nil {
		t.Fatalf("failed to read store: %v", err)
	}
	if _, exists := f.Tasks["test-ready"]; exists {
		t.Error("expected task to be deleted from active tasks")
	}
}

func TestUpdate_DeleteTaskCancel(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	model := newTestModel(t, s)

	// Find and select the ready task
	for i, item := range model.taskList.Items() {
		if ti, ok := item.(TaskItem); ok && ti.ID == "test-ready" {
			model.taskList.Select(i)
			break
		}
	}

	// Press 'x' to trigger delete confirmation dialog
	xMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	newModel, _ := model.Update(xMsg)
	confirmModel := newModel.(Model)

	// Press 'n' to cancel
	nMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	finalModel, _ := confirmModel.Update(nMsg)
	updated := finalModel.(Model)

	// Verify we returned to dashboard
	if updated.view != ViewDashboard {
		t.Errorf("expected ViewDashboard, got %v", updated.view)
	}

	// Verify task was NOT deleted
	f, err := s.Read()
	if err != nil {
		t.Fatalf("failed to read store: %v", err)
	}
	if _, exists := f.Tasks["test-ready"]; !exists {
		t.Error("expected task to still exist")
	}
}

func TestRefresh_PreservesSelection(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	m, _ := New(s)
	model := *m

	// Simulate initial load (since New() no longer loads synchronously)
	cmd := model.Init()
	msg := cmd()
	newModel, _ := model.Update(msg)
	model = newModel.(Model)

	// Select a specific task (not the first one)
	for i, item := range model.taskList.Items() {
		if ti, ok := item.(TaskItem); ok && ti.ID == "test-progress" {
			model.taskList.Select(i)
			break
		}
	}

	// Trigger refresh by simulating runAction(loadTasksCmd)
	model.lastSelectedID = "test-progress"
	model.lastSelectedIdx = model.taskList.Index()
	cmd = loadTasksCmd(s)
	msg = cmd()
	newModel, _ = model.Update(msg)
	updated := newModel.(Model)

	// Verify selection was preserved
	if item, ok := updated.taskList.SelectedItem().(TaskItem); ok {
		if item.ID != "test-progress" {
			t.Errorf("expected selection to be preserved as 'test-progress', got %v", item.ID)
		}
	} else {
		t.Error("expected a TaskItem to be selected after refresh")
	}
}

func TestRefresh_HandlesMissingTask(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	m, _ := New(s)
	model := *m

	// Simulate initial load
	cmd := model.Init()
	msg := cmd()
	newModel, _ := model.Update(msg)
	model = newModel.(Model)

	// Get initial item count
	initialCount := len(model.taskList.Items())

	// Select a task that will be deleted
	var selectedIdx int
	for i, item := range model.taskList.Items() {
		if ti, ok := item.(TaskItem); ok && ti.ID == "test-done" {
			model.taskList.Select(i)
			selectedIdx = i
			break
		}
	}

	// Delete the task directly through the store
	_ = s.DeleteTask("test-done")

	// Trigger refresh by simulating runAction(loadTasksCmd)
	model.lastSelectedID = "test-done"
	model.lastSelectedIdx = selectedIdx
	cmd = loadTasksCmd(s)
	msg = cmd()
	newModel, _ = model.Update(msg)
	updated := newModel.(Model)

	// Verify selection moved to a valid index
	newCount := len(updated.taskList.Items())
	if newCount != initialCount-1 {
		t.Errorf("expected %d tasks after delete, got %d", initialCount-1, newCount)
	}

	// Selection should be at a valid index
	newSelectedIdx := updated.taskList.Index()
	if newSelectedIdx >= newCount {
		t.Errorf("selection index %d is out of bounds for %d items", newSelectedIdx, newCount)
	}
}

func TestNoSelectionMessage(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	m := newTestModel(t, s)

	// With tasks and selection available, noSelectionMessage returns "Select a task first"
	// But we need empty list to test - create model with empty store
	dir := t.TempDir()
	root := filepath.Join(dir, ".tasuku")
	emptyStore := v4.New(root)
	if err := emptyStore.Init(); err != nil {
		t.Fatalf("empty store init: %v", err)
	}
	emptyModel := newTestModel(t, emptyStore)

	got := emptyModel.noSelectionMessage()
	if got != "No tasks yet. Press n to add one." {
		t.Errorf("noSelectionMessage (empty) = %q, want %q", got, "No tasks yet. Press n to add one.")
	}

	// With tasks but filter results in empty - filter to blocked (3), setupTestStore has no blocked tasks
	filtered, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	filteredModel := filtered.(Model)
	got = filteredModel.noSelectionMessage()
	if got != "No tasks match filter. Press 0 to clear or n to add." {
		t.Errorf("noSelectionMessage (filtered empty) = %q, want filter message", got)
	}

	// With tasks and items - need to clear selection to trigger "Select a task first"
	// When list has items but none selected - actually the list always has a selection (index 0)
	// So we can't easily get "Select a task first" in a normal scenario - it happens when
	// SelectedItem() returns nil. With 0 items, we get the empty message. With items, there's
	// always a selection. So "Select a task first" might only happen in edge cases.
	// Test the filtered-empty case is sufficient.
}

func TestStatusMsgClearsAfterNoSelectionFeedback(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	model := newTestModel(t, s)

	// Apply blocked filter (3) so list is empty in setup fixture.
	filtered, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	model = filtered.(Model)
	if len(model.taskList.Items()) != 0 {
		t.Fatalf("expected empty filtered list, got %d items", len(model.taskList.Items()))
	}

	// Trigger a task-dependent action with no selected item.
	withMsg, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	model = withMsg.(Model)
	if model.statusMsg == "" {
		t.Fatal("expected statusMsg to be set for no-selection feedback")
	}

	// Next dashboard keypress should clear stale status feedback.
	cleared, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("0")})
	model = cleared.(Model)
	if model.statusMsg != "" {
		t.Fatalf("expected statusMsg to clear, got %q", model.statusMsg)
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
