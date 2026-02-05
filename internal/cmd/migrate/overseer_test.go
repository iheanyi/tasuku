package migrate

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
	"github.com/iheanyi/tasuku/internal/task"
)

// createTestOverseerDB creates a SQLite database with the Overseer schema in the given directory.
func createTestOverseerDB(t *testing.T, dir string, tasks []OverseerTask, learnings []OverseerLearning, blockers []OverseerBlocker) string {
	t.Helper()
	return createTestOverseerDBWithOptions(t, dir, tasks, learnings, blockers, nil, true, -1)
}

// createTestOverseerDBWithOptions creates a SQLite database with configurable schema options.
// If metadata is non-nil, a task_metadata table is created and populated.
// If includeOptionalCols is false, optional columns (commit_sha, bookmark, start_commit,
// cancelled_at, archived_at) are omitted from the tasks table.
// If schemaVersion >= 0, it is set as PRAGMA user_version.
func createTestOverseerDBWithOptions(t *testing.T, dir string, tasks []OverseerTask, learnings []OverseerLearning, blockers []OverseerBlocker, metadata map[string]string, includeOptionalCols bool, schemaVersion int) string {
	t.Helper()

	overseerDir := filepath.Join(dir, ".overseer")
	if err := os.MkdirAll(overseerDir, 0755); err != nil {
		t.Fatalf("failed to create .overseer dir: %v", err)
	}

	dbPath := filepath.Join(overseerDir, "tasks.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Set schema version if requested
	if schemaVersion >= 0 {
		if _, err := db.Exec("PRAGMA user_version = " + strconv.Itoa(schemaVersion)); err != nil {
			t.Fatalf("failed to set user_version: %v", err)
		}
	}

	// Create tasks table with optional columns
	tasksCols := `id TEXT PRIMARY KEY,
		parent_id TEXT,
		description TEXT NOT NULL,
		context TEXT,
		result TEXT,
		priority INTEGER DEFAULT 1,
		completed BOOLEAN DEFAULT 0,
		cancelled BOOLEAN DEFAULT 0,
		archived BOOLEAN DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		started_at TEXT,
		completed_at TEXT`

	if includeOptionalCols {
		tasksCols += `,
		commit_sha TEXT,
		bookmark TEXT,
		start_commit TEXT,
		cancelled_at TEXT,
		archived_at TEXT`
	}

	_, err = db.Exec("CREATE TABLE tasks (" + tasksCols + ")")
	if err != nil {
		t.Fatalf("failed to create tasks table: %v", err)
	}

	// Create learnings table
	_, err = db.Exec(`CREATE TABLE learnings (
		id TEXT PRIMARY KEY,
		task_id TEXT,
		content TEXT NOT NULL,
		source_task_id TEXT,
		created_at TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("failed to create learnings table: %v", err)
	}

	// Create task_blockers table (Overseer's actual table name)
	_, err = db.Exec(`CREATE TABLE task_blockers (
		task_id TEXT NOT NULL,
		blocker_id TEXT NOT NULL,
		PRIMARY KEY (task_id, blocker_id)
	)`)
	if err != nil {
		t.Fatalf("failed to create task_blockers table: %v", err)
	}

	// Create task_metadata table if metadata provided
	if metadata != nil {
		_, err = db.Exec(`CREATE TABLE task_metadata (
			task_id TEXT NOT NULL,
			data TEXT NOT NULL,
			PRIMARY KEY (task_id)
		)`)
		if err != nil {
			t.Fatalf("failed to create task_metadata table: %v", err)
		}
		for taskID, data := range metadata {
			if _, err := db.Exec(`INSERT INTO task_metadata (task_id, data) VALUES (?, ?)`, taskID, data); err != nil {
				t.Fatalf("failed to insert metadata: %v", err)
			}
		}
	}

	// Insert tasks
	for _, tk := range tasks {
		var parentID, context, result, startedAt, completedAt any
		if tk.ParentID != nil {
			parentID = *tk.ParentID
		}
		if tk.Context != nil {
			context = *tk.Context
		}
		if tk.Result != nil {
			result = *tk.Result
		}
		if tk.StartedAt != nil {
			startedAt = tk.StartedAt.Format(time.RFC3339)
		}
		if tk.CompletedAt != nil {
			completedAt = tk.CompletedAt.Format(time.RFC3339)
		}

		if includeOptionalCols {
			var commitSHA, bookmark, startCommit any
			if tk.CommitSHA != nil {
				commitSHA = *tk.CommitSHA
			}
			if tk.Bookmark != nil {
				bookmark = *tk.Bookmark
			}
			if tk.StartCommit != nil {
				startCommit = *tk.StartCommit
			}

			_, err = db.Exec(`INSERT INTO tasks (id, parent_id, description, context, result, priority,
				completed, cancelled, archived, created_at, updated_at, started_at, completed_at,
				commit_sha, bookmark, start_commit, cancelled_at, archived_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL)`,
				tk.ID, parentID, tk.Description, context, result, tk.Priority,
				tk.Completed, tk.Cancelled, tk.Archived,
				tk.CreatedAt.Format(time.RFC3339), tk.UpdatedAt.Format(time.RFC3339),
				startedAt, completedAt, commitSHA, bookmark, startCommit)
		} else {
			_, err = db.Exec(`INSERT INTO tasks (id, parent_id, description, context, result, priority,
				completed, cancelled, archived, created_at, updated_at, started_at, completed_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				tk.ID, parentID, tk.Description, context, result, tk.Priority,
				tk.Completed, tk.Cancelled, tk.Archived,
				tk.CreatedAt.Format(time.RFC3339), tk.UpdatedAt.Format(time.RFC3339),
				startedAt, completedAt)
		}
		if err != nil {
			t.Fatalf("failed to insert task: %v", err)
		}
	}

	// Insert learnings
	for _, l := range learnings {
		_, err = db.Exec(`INSERT INTO learnings (id, content, created_at) VALUES (?, ?, ?)`,
			l.ID, l.Content, l.CreatedAt.Format(time.RFC3339))
		if err != nil {
			t.Fatalf("failed to insert learning: %v", err)
		}
	}

	// Insert blockers
	for _, b := range blockers {
		_, err = db.Exec(`INSERT INTO task_blockers (task_id, blocker_id) VALUES (?, ?)`,
			b.TaskID, b.BlockerID)
		if err != nil {
			t.Fatalf("failed to insert blocker: %v", err)
		}
	}

	return dbPath
}

// helper to create a string pointer
func strPtr(s string) *string { return &s }

func TestOverseerCmdExists(t *testing.T) {
	subcommands := make(map[string]bool)
	for _, sub := range Cmd.Commands() {
		subcommands[sub.Use] = true
	}

	if !subcommands["overseer"] {
		t.Error("expected 'overseer' subcommand")
	}
}

func TestOverseerCmdFlags(t *testing.T) {
	for _, sub := range Cmd.Commands() {
		if sub.Use == "overseer" {
			if sub.Flags().Lookup("dry-run") == nil {
				t.Error("expected --dry-run flag on overseer command")
			}
			if sub.Flags().Lookup("path") == nil {
				t.Error("expected --path flag on overseer command")
			}
			if sub.Flags().Lookup("force") == nil {
				t.Error("expected --force flag on overseer command")
			}
			break
		}
	}
}

func TestOverseerMissingDB(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "overseer")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, ".overseer/tasks.db not found")
}

func TestOverseerDryRun(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{ID: "01J1234", Description: "Fix login bug", Priority: 0, CreatedAt: now, UpdatedAt: now},
		{ID: "01J5678", Description: "Add dark mode", Priority: 1, StartedAt: &now, CreatedAt: now, UpdatedAt: now},
	}

	createTestOverseerDB(t, h.TempDir(), tasks, nil, nil)

	err := h.Execute(Cmd, "overseer", "--dry-run")
	h.AssertNoError(err)
	h.AssertOutputContains("Overseer Migration Preview")
	h.AssertOutputContains("2 total")
	h.AssertOutputContains("fix-login-bug")
	h.AssertOutputContains("add-dark-mode")
	h.AssertOutputContains("Dry run")

	// Ensure no tasks were actually created
	f, _ := h.Store().Read()
	if len(f.Tasks) > 0 {
		t.Errorf("dry run should not create tasks, got %d", len(f.Tasks))
	}
}

func TestOverseerStatusMapping(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		// ready: not started, not completed, not cancelled, not archived
		{ID: "01A", Description: "Ready task", Priority: 1, CreatedAt: now, UpdatedAt: now},
		// in_progress: has started_at, not completed
		{ID: "01B", Description: "In progress task", Priority: 1, StartedAt: &now, CreatedAt: now, UpdatedAt: now},
		// done: completed
		{ID: "01C", Description: "Completed task", Priority: 1, Completed: true, CompletedAt: &now, CreatedAt: now, UpdatedAt: now},
		// done + cancelled tag: cancelled but not archived
		{ID: "01D", Description: "Cancelled task", Priority: 1, Cancelled: true, CreatedAt: now, UpdatedAt: now},
		// done + archived: archived (must be completed or cancelled)
		{ID: "01E", Description: "Archived task", Priority: 1, Completed: true, Archived: true, CompletedAt: &now, CreatedAt: now, UpdatedAt: now},
	}

	createTestOverseerDB(t, h.TempDir(), tasks, nil, nil)

	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)

	f, err := h.Store().Read()
	if err != nil {
		t.Fatalf("failed to read store: %v", err)
	}

	// Check active tasks
	checkStatus := func(desc string, expectedStatus task.Status) {
		t.Helper()
		for _, tk := range f.Tasks {
			if tk.Description == desc {
				if tk.Status != expectedStatus {
					t.Errorf("task %q: expected status %s, got %s", desc, expectedStatus, tk.Status)
				}
				return
			}
		}
		// Could be archived
		if expectedStatus == task.StatusDone {
			return // archived tasks aren't in f.Tasks
		}
		t.Errorf("task %q not found in active tasks", desc)
	}

	checkStatus("Ready task", task.StatusReady)
	checkStatus("In progress task", task.StatusInProgress)
	checkStatus("Completed task", task.StatusDone)
	checkStatus("Cancelled task", task.StatusDone)

	// Check archived task is in archive
	archived, err := h.Store().GetArchivedTasks()
	if err != nil {
		t.Fatalf("failed to get archived tasks: %v", err)
	}

	foundArchived := false
	for _, a := range archived {
		if a.Description == "Archived task" {
			foundArchived = true
			break
		}
	}
	if !foundArchived {
		t.Error("expected 'Archived task' to be in archive")
	}
}

func TestOverseerPriorityMapping(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{ID: "01A", Description: "High priority", Priority: 0, CreatedAt: now, UpdatedAt: now},
		{ID: "01B", Description: "Medium priority", Priority: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "01C", Description: "Low priority", Priority: 2, CreatedAt: now, UpdatedAt: now},
	}

	createTestOverseerDB(t, h.TempDir(), tasks, nil, nil)

	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)

	f, _ := h.Store().Read()

	checkPriority := func(desc string, expected int) {
		t.Helper()
		for _, tk := range f.Tasks {
			if tk.Description == desc {
				if tk.Priority == nil {
					t.Errorf("task %q: expected priority %d, got nil", desc, expected)
					return
				}
				if *tk.Priority != expected {
					t.Errorf("task %q: expected priority %d, got %d", desc, expected, *tk.Priority)
				}
				return
			}
		}
		t.Errorf("task %q not found", desc)
	}

	checkPriority("High priority", task.PriorityCritical)  // 0 → 0
	checkPriority("Medium priority", task.PriorityNormal)  // 1 → 2
	checkPriority("Low priority", task.PriorityBacklog)    // 2 → 4
}

func TestOverseerParentRemapping(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{ID: "01PARENT", Description: "Parent task", Priority: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "01CHILD", Description: "Child task", Priority: 1, ParentID: strPtr("01PARENT"), CreatedAt: now, UpdatedAt: now},
	}

	createTestOverseerDB(t, h.TempDir(), tasks, nil, nil)

	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)

	f, _ := h.Store().Read()

	// Find the child task
	for _, tk := range f.Tasks {
		if tk.Description == "Child task" {
			if tk.ParentID == nil {
				t.Error("child task should have a parent ID")
				return
			}
			// The parent ID should be the new Tasuku ID, not the old Overseer ULID
			if *tk.ParentID == "01PARENT" {
				t.Error("parent ID should be remapped to Tasuku ID, not Overseer ID")
			}
			// Find parent by new ID
			if _, ok := f.Tasks[*tk.ParentID]; !ok {
				t.Errorf("parent task %q not found", *tk.ParentID)
			}
			return
		}
	}
	t.Error("child task not found")
}

func TestOverseerBlockerRemapping(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{ID: "01BLOCKER", Description: "Blocker task", Priority: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "01BLOCKED", Description: "Blocked task", Priority: 1, CreatedAt: now, UpdatedAt: now},
	}
	blockers := []OverseerBlocker{
		{TaskID: "01BLOCKED", BlockerID: "01BLOCKER"},
	}

	createTestOverseerDB(t, h.TempDir(), tasks, nil, blockers)

	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)

	f, _ := h.Store().Read()

	for _, tk := range f.Tasks {
		if tk.Description == "Blocked task" {
			if len(tk.BlockedBy) == 0 {
				t.Error("blocked task should have blockers")
				return
			}
			// Blockers should use new IDs
			for _, b := range tk.BlockedBy {
				if b == "01BLOCKER" {
					t.Error("blocker ID should be remapped to Tasuku ID")
				}
				if _, ok := f.Tasks[b]; !ok {
					t.Errorf("blocker task %q not found", b)
				}
			}
			return
		}
	}
	t.Error("blocked task not found")
}

func TestOverseerContextAndResultNotes(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{
			ID:          "01A",
			Description: "Task with context",
			Priority:    1,
			Context:     strPtr("Some important context"),
			Result:      strPtr("It worked great"),
			Completed:   true,
			CompletedAt: &now,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	createTestOverseerDB(t, h.TempDir(), tasks, nil, nil)

	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)

	f, _ := h.Store().Read()

	// Find the task's notes
	foundContext := false
	foundResult := false
	for _, notes := range f.Context.Notes {
		for _, note := range notes {
			if note.Text == "Context: Some important context" {
				foundContext = true
			}
			if note.Text == "Result: It worked great" {
				foundResult = true
			}
		}
	}

	if !foundContext {
		t.Error("expected context note to be created")
	}
	if !foundResult {
		t.Error("expected result note to be created")
	}
}

func TestOverseerLearnings(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()

	// Create empty task to satisfy FK expectations (not required but more realistic)
	tasks := []OverseerTask{
		{ID: "01A", Description: "Some task", Priority: 1, CreatedAt: now, UpdatedAt: now},
	}

	learnings := []OverseerLearning{
		{ID: "L1", Content: "Never use global state for config", CreatedAt: now},
		{ID: "L2", Content: "Database connections should be pooled", CreatedAt: now},
		{ID: "L3", Content: "Never use global state for config", CreatedAt: now}, // duplicate
	}

	createTestOverseerDB(t, h.TempDir(), tasks, learnings, nil)

	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)

	f, _ := h.Store().Read()

	// Should have 2 unique learnings (deduped)
	if len(f.Context.Learnings) != 2 {
		t.Errorf("expected 2 learnings (deduplicated), got %d", len(f.Context.Learnings))
	}

	// Check rule detection
	for _, l := range f.Context.Learnings {
		if l.Text == "Never use global state for config" && !l.IsRule {
			t.Error("learning starting with 'Never' should be detected as rule")
		}
	}
}

func TestOverseerEmptyDB(t *testing.T) {
	h := testutil.New(t)

	createTestOverseerDB(t, h.TempDir(), nil, nil, nil)

	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)
	h.AssertOutputContains("0 total")
}

func TestOverseerCancelledTag(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{ID: "01A", Description: "Cancelled task", Priority: 1, Cancelled: true, CreatedAt: now, UpdatedAt: now},
	}

	createTestOverseerDB(t, h.TempDir(), tasks, nil, nil)

	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)

	f, _ := h.Store().Read()

	for _, tk := range f.Tasks {
		if tk.Description == "Cancelled task" {
			if tk.Status != task.StatusDone {
				t.Errorf("cancelled task should have done status, got %s", tk.Status)
			}
			if !tk.HasTag("cancelled") {
				t.Error("cancelled task should have 'cancelled' tag")
			}
			return
		}
	}
	t.Error("cancelled task not found")
}

func TestOverseerArchivedTasks(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{ID: "01A", Description: "Archived completed task", Priority: 1, Completed: true, Archived: true, CompletedAt: &now, CreatedAt: now, UpdatedAt: now},
		{ID: "01B", Description: "Active task", Priority: 1, CreatedAt: now, UpdatedAt: now},
	}

	createTestOverseerDB(t, h.TempDir(), tasks, nil, nil)

	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)

	// Active task should be in tasks
	f, _ := h.Store().Read()
	foundActive := false
	for _, tk := range f.Tasks {
		if tk.Description == "Active task" {
			foundActive = true
		}
		if tk.Description == "Archived completed task" {
			t.Error("archived task should not be in active tasks")
		}
	}
	if !foundActive {
		t.Error("active task not found in tasks")
	}

	// Archived task should be in archive
	archived, _ := h.Store().GetArchivedTasks()
	foundArchived := false
	for _, a := range archived {
		if a.Description == "Archived completed task" {
			foundArchived = true
		}
	}
	if !foundArchived {
		t.Error("archived task not found in archive")
	}
}

func TestOverseerCustomPath(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{ID: "01A", Description: "Custom path task", Priority: 1, CreatedAt: now, UpdatedAt: now},
	}

	// Create DB in a non-standard location
	customDir := filepath.Join(h.TempDir(), "custom")
	os.MkdirAll(customDir, 0755)
	dbPath := filepath.Join(customDir, "my.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE tasks (id TEXT PRIMARY KEY, parent_id TEXT, description TEXT NOT NULL, context TEXT, result TEXT, priority INTEGER DEFAULT 1, completed BOOLEAN DEFAULT 0, cancelled BOOLEAN DEFAULT 0, archived BOOLEAN DEFAULT 0, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, started_at TEXT, completed_at TEXT, commit_sha TEXT, bookmark TEXT, start_commit TEXT, cancelled_at TEXT, archived_at TEXT)`); err != nil {
		t.Fatalf("failed to create tasks table: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE learnings (id TEXT PRIMARY KEY, task_id TEXT, content TEXT NOT NULL, source_task_id TEXT, created_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("failed to create learnings table: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE task_blockers (task_id TEXT NOT NULL, blocker_id TEXT NOT NULL, PRIMARY KEY (task_id, blocker_id))`); err != nil {
		t.Fatalf("failed to create task_blockers table: %v", err)
	}
	for _, tk := range tasks {
		if _, err := db.Exec(`INSERT INTO tasks (id, description, priority, completed, cancelled, archived, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			tk.ID, tk.Description, tk.Priority, tk.Completed, tk.Cancelled, tk.Archived, tk.CreatedAt.Format(time.RFC3339), tk.UpdatedAt.Format(time.RFC3339)); err != nil {
			t.Fatalf("failed to insert task: %v", err)
		}
	}
	db.Close()

	err = h.Execute(Cmd, "overseer", "--path", dbPath)
	h.AssertNoError(err)
	h.AssertOutputContains("custom-path-task")
}

func TestOverseerVCSFields(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{
			ID:          "01A",
			Description: "Task with VCS",
			Priority:    1,
			CommitSHA:   strPtr("abc123"),
			Bookmark:    strPtr("feature-branch"),
			StartCommit: strPtr("def456"),
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	createTestOverseerDB(t, h.TempDir(), tasks, nil, nil)

	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)

	f, _ := h.Store().Read()

	for _, tk := range f.Tasks {
		if tk.Description == "Task with VCS" {
			if tk.Fields["overseer_commit_sha"] != "abc123" {
				t.Errorf("expected commit_sha field, got %v", tk.Fields)
			}
			if tk.Fields["overseer_bookmark"] != "feature-branch" {
				t.Errorf("expected bookmark field, got %v", tk.Fields)
			}
			if tk.Fields["overseer_start_commit"] != "def456" {
				t.Errorf("expected start_commit field, got %v", tk.Fields)
			}
			return
		}
	}
	t.Error("task with VCS fields not found")
}

func TestOverseerCircularParentReference(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{ID: "01A", Description: "Task A", Priority: 1, ParentID: strPtr("01B"), CreatedAt: now, UpdatedAt: now},
		{ID: "01B", Description: "Task B", Priority: 1, ParentID: strPtr("01A"), CreatedAt: now, UpdatedAt: now},
	}

	createTestOverseerDB(t, h.TempDir(), tasks, nil, nil)

	err := h.Execute(Cmd, "overseer")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "circular parent reference")
}

func TestOverseerSelfParentReference(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{ID: "01A", Description: "Self parent", Priority: 1, ParentID: strPtr("01A"), CreatedAt: now, UpdatedAt: now},
	}

	createTestOverseerDB(t, h.TempDir(), tasks, nil, nil)

	err := h.Execute(Cmd, "overseer")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "circular parent reference")
}

func TestOverseerOrphanedSubtask(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	// Child references a parent ID that doesn't exist in the tasks list
	tasks := []OverseerTask{
		{ID: "01CHILD", Description: "Orphaned child", Priority: 1, ParentID: strPtr("NONEXISTENT"), CreatedAt: now, UpdatedAt: now},
	}

	createTestOverseerDB(t, h.TempDir(), tasks, nil, nil)

	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)
	h.AssertOutputContains("Warning: orphaned subtask")

	f, _ := h.Store().Read()
	for _, tk := range f.Tasks {
		if tk.Description == "Orphaned child" {
			if tk.ParentID != nil {
				t.Error("orphaned subtask should have nil parent ID")
			}
			return
		}
	}
	t.Error("orphaned child task not found")
}

func TestOverseerIdempotency(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{ID: "01A", Description: "Idempotent task", Priority: 1, CreatedAt: now, UpdatedAt: now},
	}

	createTestOverseerDB(t, h.TempDir(), tasks, nil, nil)

	// First run
	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)

	// Second run — should fail without --force
	err = h.Execute(Cmd, "overseer")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "conflict")

	// Third run with --force — should succeed
	err = h.Execute(Cmd, "overseer", "--force")
	h.AssertNoError(err)

	// Should have exactly one task, not duplicated
	f, _ := h.Store().Read()
	count := 0
	for _, tk := range f.Tasks {
		if tk.Description == "Idempotent task" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 task after force migration, got %d", count)
	}
}

func TestOverseerAlreadyMigrated(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{ID: "01A", Description: "First task", Priority: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "01B", Description: "Second task", Priority: 1, CreatedAt: now, UpdatedAt: now},
	}

	createTestOverseerDB(t, h.TempDir(), tasks, nil, nil)

	// First run succeeds
	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)

	// Second run without --force should fail with conflict details
	err = h.Execute(Cmd, "overseer")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "existing task(s) that would conflict")
	h.AssertErrorContainsMsg(err, "--force")
}

func TestOverseerMetadata(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{ID: "01A", Description: "Task with metadata", Priority: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "01B", Description: "Task without metadata", Priority: 1, CreatedAt: now, UpdatedAt: now},
	}

	metadata := map[string]string{
		"01A": `{"key": "value", "count": 42}`,
	}

	createTestOverseerDBWithOptions(t, h.TempDir(), tasks, nil, nil, metadata, true, -1)

	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)
	h.AssertOutputContains("Metadata:  1")

	f, _ := h.Store().Read()

	for _, tk := range f.Tasks {
		if tk.Description == "Task with metadata" {
			if tk.Fields["overseer_metadata"] != `{"key": "value", "count": 42}` {
				t.Errorf("expected overseer_metadata field, got %v", tk.Fields)
			}
			return
		}
	}
	t.Error("task with metadata not found")
}

func TestOverseerDynamicColumns(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{ID: "01A", Description: "Minimal schema task", Priority: 1, CreatedAt: now, UpdatedAt: now},
	}

	// Create DB without optional columns (commit_sha, bookmark, start_commit, cancelled_at, archived_at)
	createTestOverseerDBWithOptions(t, h.TempDir(), tasks, nil, nil, nil, false, -1)

	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)

	f, _ := h.Store().Read()
	found := false
	for _, tk := range f.Tasks {
		if tk.Description == "Minimal schema task" {
			found = true
			// Should still work — optional fields just won't be populated
			if len(tk.Fields) > 0 {
				t.Errorf("expected no fields for minimal schema task, got %v", tk.Fields)
			}
		}
	}
	if !found {
		t.Error("minimal schema task not found")
	}

	// Stderr should mention missing columns
	h.AssertErrorContains("missing columns")
}

func TestOverseerBareMinimumSchema(t *testing.T) {
	h := testutil.New(t)

	// Create a DB with only id and description — the absolute minimum
	overseerDir := filepath.Join(h.TempDir(), ".overseer")
	os.MkdirAll(overseerDir, 0755)
	dbPath := filepath.Join(overseerDir, "tasks.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE tasks (id TEXT PRIMARY KEY, description TEXT NOT NULL)`); err != nil {
		t.Fatalf("failed to create tasks table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO tasks (id, description) VALUES ('01A', 'Bare bones task')`); err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	db.Close()

	err = h.Execute(Cmd, "overseer")
	h.AssertNoError(err)

	f, _ := h.Store().Read()
	found := false
	for _, tk := range f.Tasks {
		if tk.Description == "Bare bones task" {
			found = true
			if tk.Status != task.StatusReady {
				t.Errorf("expected ready status, got %s", tk.Status)
			}
			// Timestamps should be defaulted to roughly now
			if tk.CreatedAt.IsZero() {
				t.Error("expected non-zero created_at default")
			}
			if tk.Priority == nil {
				t.Error("expected non-nil priority default")
			} else if *tk.Priority != task.PriorityNormal {
				t.Errorf("expected normal priority default, got %d", *tk.Priority)
			}
		}
	}
	if !found {
		t.Error("bare bones task not found")
	}
}

func TestOverseerSchemaVersionWarning(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{ID: "01A", Description: "Future schema task", Priority: 1, CreatedAt: now, UpdatedAt: now},
	}

	// Create DB with schema version higher than our tested max
	createTestOverseerDBWithOptions(t, h.TempDir(), tasks, nil, nil, nil, true, 99)

	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)

	// Warning should be on stderr
	h.AssertErrorContains("Warning: Overseer schema version 99")
}

func TestOverseerArchivedConflict(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{ID: "01A", Description: "Will be archived", Priority: 1, Completed: true, Archived: true, CompletedAt: &now, CreatedAt: now, UpdatedAt: now},
	}

	createTestOverseerDB(t, h.TempDir(), tasks, nil, nil)

	// First run archives the task
	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)

	// Second run should detect the archived conflict
	err = h.Execute(Cmd, "overseer")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "conflict")
	h.AssertErrorContainsMsg(err, "archived")
}

func TestOverseerForceDoesNotDuplicateNotes(t *testing.T) {
	h := testutil.New(t)

	now := time.Now().UTC()
	tasks := []OverseerTask{
		{
			ID:          "01A",
			Description: "Task with context",
			Priority:    1,
			Context:     strPtr("Important context"),
			Result:      strPtr("Great result"),
			Completed:   true,
			CompletedAt: &now,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
	learnings := []OverseerLearning{
		{ID: "L1", Content: "Never forget this", CreatedAt: now},
	}

	createTestOverseerDB(t, h.TempDir(), tasks, learnings, nil)

	// First run
	err := h.Execute(Cmd, "overseer")
	h.AssertNoError(err)

	// Force re-run
	err = h.Execute(Cmd, "overseer", "--force")
	h.AssertNoError(err)

	f, _ := h.Store().Read()

	// Notes should not be duplicated — exactly 2 (context + result)
	totalNotes := 0
	for _, notes := range f.Context.Notes {
		totalNotes += len(notes)
	}
	if totalNotes != 2 {
		t.Errorf("expected 2 notes after force re-run (no duplication), got %d", totalNotes)
	}

	// Learnings should not be duplicated — exactly 1
	if len(f.Context.Learnings) != 1 {
		t.Errorf("expected 1 learning after force re-run (no duplication), got %d", len(f.Context.Learnings))
	}
}

func TestOverseerDBPathValidation(t *testing.T) {
	h := testutil.New(t)

	// A path with query parameters should be rejected
	err := h.Execute(Cmd, "overseer", "--path", "/tmp/db.sqlite?mode=rw")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "query parameters")
}
