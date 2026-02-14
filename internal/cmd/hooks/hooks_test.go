package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
	"github.com/iheanyi/tasuku/internal/cmdutil"
	"github.com/iheanyi/tasuku/internal/task"
)

func TestSessionCmd(t *testing.T) {
	h := testutil.New(t)

	// Add some tasks
	h.AddTaskWithStatus("task-1", "Ready task", task.StatusReady)
	h.AddTaskWithStatus("task-2", "In progress task", task.StatusInProgress)
	h.AddTaskWithStatus("task-3", "Done task", task.StatusDone)
	h.AddLearning("Test learning")
	h.AddDecision("test-dec", "A", []string{"B"}, "reason")

	err := h.Execute(Cmd, "session")
	h.AssertNoError(err)
	h.AssertOutputContains("Tasuku Context")
	h.AssertOutputContains("Tasks:")
	h.AssertOutputContains("ready")
	h.AssertOutputContains("in_progress")
	h.AssertOutputContains("done")
	h.AssertOutputContains("Learnings:")
	h.AssertOutputContains("Decisions:")
}

func TestSessionCmdStaleTimerWarning(t *testing.T) {
	h := testutil.New(t)

	// Add a task with a stale timer (started 5 hours ago)
	h.AddTaskWithStatus("stale-timer-task", "Task with stale timer", task.StatusInProgress)
	staleTime := time.Now().Add(-5 * time.Hour)
	h.StartTimerAt("stale-timer-task", staleTime)

	err := h.Execute(Cmd, "session")
	h.AssertNoError(err)
	h.AssertOutputContains("Stale timers")
	h.AssertOutputContains("stale-timer-task")
	h.AssertOutputContains("tk task timer stop")
}

func TestSessionCmdNoStaleTimerWarning(t *testing.T) {
	h := testutil.New(t)

	// Add a task with a recent timer (started 1 hour ago - not stale)
	h.AddTaskWithStatus("recent-timer-task", "Task with recent timer", task.StatusInProgress)
	recentTime := time.Now().Add(-1 * time.Hour)
	h.StartTimerAt("recent-timer-task", recentTime)

	err := h.Execute(Cmd, "session")
	h.AssertNoError(err)
	h.AssertOutputNotContains("Stale timers")
}

func TestSessionCmdNoStorage(t *testing.T) {
	// Create temp dir without tasuku storage
	tempDir, _ := os.MkdirTemp("", "tasuku-test-nostorage-*")
	defer os.RemoveAll(tempDir)

	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Session should succeed silently when no storage exists
	err := sessionCmd.RunE(sessionCmd, []string{})
	if err != nil {
		t.Errorf("session should not error when no storage, got: %v", err)
	}
}

func TestSyncCmdNoStorage(t *testing.T) {
	// Create temp dir without tasuku storage
	tempDir, _ := os.MkdirTemp("", "tasuku-test-nostorage-*")
	defer os.RemoveAll(tempDir)

	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Sync should error when no storage exists
	err := syncCmd.RunE(syncCmd, []string{})
	if err == nil || !strings.Contains(err.Error(), "no Tasuku storage") {
		t.Errorf("expected 'no Tasuku storage' error, got: %v", err)
	}
}

func TestInstallHooksNoGit(t *testing.T) {
	h := testutil.New(t)

	// The temp dir is not a git repo
	err := h.Execute(Cmd, "install")
	h.AssertError(err)
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("expected 'not a git repository' error, got: %v", err)
	}
}

func TestInstallAndUninstallHooks(t *testing.T) {
	h := testutil.New(t)

	// Initialize a git repo in the temp dir
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	err := h.Execute(Cmd, "install")
	h.AssertNoError(err)
	h.AssertOutputContains("Git hooks installed")

	// Check hooks were created
	preCommit := filepath.Join(h.TempDir(), ".git", "hooks", "pre-commit")
	if _, err := os.Stat(preCommit); os.IsNotExist(err) {
		t.Error("pre-commit hook should exist")
	}
	postCommit := filepath.Join(h.TempDir(), ".git", "hooks", "post-commit")
	if _, err := os.Stat(postCommit); os.IsNotExist(err) {
		t.Error("post-commit hook should exist")
	}

	// Uninstall
	err = h.Execute(Cmd, "uninstall")
	h.AssertNoError(err)
	h.AssertOutputContains("uninstalled")
}

func TestInstallHooksPreservesExisting(t *testing.T) {
	h := testutil.New(t)

	// Initialize git hooks dir
	hooksDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(hooksDir, 0755)

	// Create existing pre-commit hook
	existingContent := "#!/bin/bash\necho 'existing hook'"
	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	os.WriteFile(preCommitPath, []byte(existingContent), 0755)

	// Install tasuku hooks
	err := h.Execute(Cmd, "install")
	h.AssertNoError(err)

	// Check that existing content is preserved
	content, _ := os.ReadFile(preCommitPath)
	if !strings.Contains(string(content), "existing hook") {
		t.Error("existing hook content should be preserved")
	}
	if !strings.Contains(string(content), "TASUKU HOOK") {
		t.Error("tasuku hook should be added")
	}
}

func TestGenerateID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple Task", "simple-task"},
		{"Task with CAPS", "task-with-caps"},
		{"Task with special chars!@#", "task-with-special-chars"},
		{"  Leading and trailing spaces  ", "leading-and-trailing-spaces"},
		{"Multiple   spaces", "multiple-spaces"},
	}

	for _, tt := range tests {
		got := generateID(tt.input)
		if got != tt.expected {
			t.Errorf("generateID(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestShouldPersistTask(t *testing.T) {
	tests := []struct {
		description   string
		shouldPersist bool
	}{
		{"Implement user authentication", true},
		{"Add feature for dark mode", true},
		{"Fix bug in login flow", true},
		{"Refactor database layer", true},
		{"fix type error in UserService", false},
		{"Update file with new config", false},
		{"Run tests for coverage", false},
		{"Add comment explaining logic", false},
		{"Rename variable to be clearer", false},
	}

	for _, tt := range tests {
		got := shouldPersistTask(tt.description)
		if got != tt.shouldPersist {
			t.Errorf("shouldPersistTask(%q) = %v, want %v", tt.description, got, tt.shouldPersist)
		}
	}
}

func TestTestFailureStateTracking(t *testing.T) {
	// Save original directory and change to temp dir
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Create .tasuku directory for local state storage
	os.MkdirAll(".tasuku", 0755)

	// Clean up any existing state
	clearTestFailureState()

	// Initially, no failure should be detected
	if isRecentTestFailure(30 * time.Minute) {
		t.Error("expected no recent failure initially")
	}

	// Save a test failure
	if err := saveTestFailureState("go test ./..."); err != nil {
		t.Fatalf("failed to save test failure state: %v", err)
	}

	// Now should detect recent failure
	if !isRecentTestFailure(30 * time.Minute) {
		t.Error("expected recent failure after saving state")
	}

	// Read the state
	state, err := getLastTestFailure()
	if err != nil {
		t.Fatalf("failed to get last test failure: %v", err)
	}
	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.Command != "go test ./..." {
		t.Errorf("expected command 'go test ./...', got %q", state.Command)
	}

	// Clear the state
	clearTestFailureState()

	// Should no longer detect failure
	if isRecentTestFailure(30 * time.Minute) {
		t.Error("expected no recent failure after clearing state")
	}
}

func TestDetectTestSuccess(t *testing.T) {
	tests := []struct {
		output  string
		success bool
	}{
		{"PASS\nok  github.com/example/pkg 0.123s", true},
		{"Tests passed!", true},
		{"All tests passed", true},
		{"exit status 0", true},
		{"0 failures, 10 successes", true},
		{"FAIL\nFailed: TestSomething", false},
		{"Error: assertion failed", false},
		{"Some random output", false},
	}

	for _, tt := range tests {
		got := detectTestSuccess(strings.ToLower(tt.output))
		if got != tt.success {
			t.Errorf("detectTestSuccess(%q) = %v, want %v", tt.output, got, tt.success)
		}
	}
}

func TestIsTestCommand(t *testing.T) {
	tests := []struct {
		command string
		isTest  bool
	}{
		{"go test ./...", true},
		{"npm test", true},
		{"pytest", true},
		{"yarn test", true},
		{"jest", true},
		{"make test", true},
		{"go build", false},
		{"npm install", false},
		{"echo hello", false},
	}

	for _, tt := range tests {
		got := isTestCommand(tt.command)
		if got != tt.isTest {
			t.Errorf("isTestCommand(%q) = %v, want %v", tt.command, got, tt.isTest)
		}
	}
}

func TestDetectTestFailure(t *testing.T) {
	tests := []struct {
		output  string
		failure bool
	}{
		{"fail\n--- fail: TestSomething", true},
		{"failed", true},
		{"error:", true},
		{"panic: runtime error", true},
		{"exit status 1", true},
		{"tests failed", true},
		{"assertion failed", true},
		{"pass\nok  github.com/example", false},
		{"all tests passed", false},
	}

	for _, tt := range tests {
		got := detectTestFailure(tt.output)
		if got != tt.failure {
			t.Errorf("detectTestFailure(%q) = %v, want %v", tt.output, got, tt.failure)
		}
	}
}

func TestDetectBugReport(t *testing.T) {
	tests := []struct {
		prompt string
		isBug  bool
	}{
		{"there's a bug in the login page", true},
		{"error when clicking submit button", true},
		{"crash occurs on page load", true},
		{"fix this broken feature", true},
		{"failing test in the CI", true},
		{"not working after deploy", true},
		{"weird behavior with auth", true},
		{"add a new feature for dark mode", false},
		{"implement user authentication", false},
		{"refactor the database layer", false},
	}

	for _, tt := range tests {
		got := detectBugReport(tt.prompt)
		if got != tt.isBug {
			t.Errorf("detectBugReport(%q) = %v, want %v", tt.prompt, got, tt.isBug)
		}
	}
}

func TestDetectSignificantWork(t *testing.T) {
	tests := []struct {
		prompt      string
		significant bool
	}{
		{"implement user authentication", true},
		{"add feature for payments", true},
		{"create new dashboard page", true},
		{"build a REST API", true},
		{"set up the database schema", true},
		{"refactor the auth module", true},
		{"integrate stripe payments", true},
		{"fix this typo", false},
		{"rename this variable", false},
		{"what does this code do", false},
		{"how does the auth work", false},
	}

	for _, tt := range tests {
		got := detectSignificantWork(tt.prompt)
		if got != tt.significant {
			t.Errorf("detectSignificantWork(%q) = %v, want %v", tt.prompt, got, tt.significant)
		}
	}
}

func TestLooksLikeQuestion(t *testing.T) {
	tests := []struct {
		prompt     string
		isQuestion bool
	}{
		{"what is the purpose of this function?", true},
		{"how does authentication work?", true},
		{"why does this test fail?", true},
		{"can you explain this code?", true},
		{"implement user login", false},
		{"fix the bug in payment", false},
		{"add dark mode support", false},
	}

	for _, tt := range tests {
		got := looksLikeQuestion(tt.prompt)
		if got != tt.isQuestion {
			t.Errorf("looksLikeQuestion(%q) = %v, want %v", tt.prompt, got, tt.isQuestion)
		}
	}
}

func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		status task.Status
		icon   string
	}{
		{task.StatusReady, "○"},
		{task.StatusInProgress, "●"},
		{task.StatusBlocked, "⊘"},
		{task.StatusDone, "✓"},
	}

	for _, tt := range tests {
		got := getStatusIcon(tt.status)
		if got != tt.icon {
			t.Errorf("getStatusIcon(%v) = %q, want %q", tt.status, got, tt.icon)
		}
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly ten", 10, "exactly..."},
		{"a very long string that exceeds the limit", 20, "a very long strin..."},
		{"", 10, ""},
		{"abc", 3, "abc"},
	}

	for _, tt := range tests {
		got := cmdutil.Truncate(tt.input, tt.maxLen)
		if got != tt.expected {
			t.Errorf("cmdutil.Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
		}
	}
}

func TestIsBugFixTask(t *testing.T) {
	tests := []struct {
		desc  string
		isBug bool
	}{
		{"fix login bug", true},
		{"Fix authentication issue", true},
		{"bugfix for payment", true},
		{"repair broken link", true},
		{"add new feature", false},
		{"implement dark mode", false},
		{"refactor database", false},
	}

	for _, tt := range tests {
		got := isBugFixTask(tt.desc)
		if got != tt.isBug {
			t.Errorf("isBugFixTask(%q) = %v, want %v", tt.desc, got, tt.isBug)
		}
	}
}

func TestIsGitCommitCommand(t *testing.T) {
	tests := []struct {
		command  string
		isCommit bool
	}{
		{"git commit -m 'message'", true},
		{"git commit --amend", true},
		{"git status", false},
		{"git push", false},
		{"npm install", false},
	}

	for _, tt := range tests {
		got := isGitCommitCommand(tt.command)
		if got != tt.isCommit {
			t.Errorf("isGitCommitCommand(%q) = %v, want %v", tt.command, got, tt.isCommit)
		}
	}
}

func TestDetectLearningIntent(t *testing.T) {
	tests := []struct {
		prompt     string
		isLearning bool
	}{
		{"til that Go maps are not concurrent-safe", true},
		{"i learned that React hooks must be called in order", true},
		{"realized that the API requires auth", true},
		{"turns out the cache was stale", true},
		{"note to self: always check nil", true},
		{"implement user login", false},
		{"fix the bug", false},
	}

	for _, tt := range tests {
		got := detectLearningIntent(tt.prompt)
		if got != tt.isLearning {
			t.Errorf("detectLearningIntent(%q) = %v, want %v", tt.prompt, got, tt.isLearning)
		}
	}
}

func TestDetectDecisionPoint(t *testing.T) {
	tests := []struct {
		prompt     string
		isDecision bool
	}{
		{"should we use React or Vue?", true},
		{"either Redux or Context for state", true},
		{"choosing between PostgreSQL or MongoDB", true},
		{"implement the login feature", false},
		{"fix the auth bug", false},
	}

	for _, tt := range tests {
		got := detectDecisionPoint(tt.prompt)
		if got != tt.isDecision {
			t.Errorf("detectDecisionPoint(%q) = %v, want %v", tt.prompt, got, tt.isDecision)
		}
	}
}

func TestDetectShippingIntent(t *testing.T) {
	tests := []struct {
		prompt string
		isShip bool
	}{
		{"let's deploy this to production", true},
		{"ready to release", true},
		{"ship it", true},
		{"merge to main", true},
		{"push to prod", true},
		{"time to deploy the changes", true},
		{"implement new feature", false},
		{"fix the bug", false},
	}

	for _, tt := range tests {
		got := detectShippingIntent(tt.prompt)
		if got != tt.isShip {
			t.Errorf("detectShippingIntent(%q) = %v, want %v", tt.prompt, got, tt.isShip)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{3600 * time.Second, "1h0m"},
		{3661 * time.Second, "1h1m"},
		{0, "0s"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.duration)
		if got != tt.expected {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.expected)
		}
	}
}

func TestInvestigationPatternTracking(t *testing.T) {
	// Save original directory and change to temp dir
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Create .tasuku directory for local state storage
	os.MkdirAll(".tasuku", 0755)

	// Clean up any existing state
	os.Remove(getInvestigationStatePath())

	testFile := "/path/to/some/file.go"

	// Initially, should not detect investigation pattern
	wasInvestigating, count := checkInvestigationPattern(testFile)
	if wasInvestigating {
		t.Error("expected no investigation pattern initially")
	}
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}

	// Record file reads (less than threshold)
	recordFileRead(testFile)
	recordFileRead(testFile)

	// Still should not trigger (only 2 reads, threshold is 3)
	wasInvestigating, count = checkInvestigationPattern(testFile)
	if wasInvestigating {
		t.Error("expected no investigation pattern with only 2 reads")
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}

	// Add one more read to reach threshold
	recordFileRead(testFile)

	// Now should trigger
	wasInvestigating, count = checkInvestigationPattern(testFile)
	if !wasInvestigating {
		t.Error("expected investigation pattern with 3 reads")
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}

	// After triggering, count should be cleared
	wasInvestigating, count = checkInvestigationPattern(testFile)
	if wasInvestigating {
		t.Error("expected no investigation pattern after clearing")
	}
	if count != 0 {
		t.Errorf("expected count 0 after clearing, got %d", count)
	}
}

func TestInvestigationStateExpiry(t *testing.T) {
	// Save original directory and change to temp dir
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Create .tasuku directory for local state storage
	os.MkdirAll(".tasuku", 0755)

	// Create state with old timestamp
	state := &investigationState{
		FileReads: map[string]int{
			"/some/file.go": 5,
		},
		LastUpdated: time.Now().Add(-2 * time.Hour), // 2 hours ago, beyond 30 min max
	}
	data, _ := json.Marshal(state)
	os.WriteFile(getInvestigationStatePath(), data, 0644)

	// Loading should reset due to stale state
	loaded, err := loadInvestigationState()
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}
	if len(loaded.FileReads) != 0 {
		t.Errorf("expected empty file reads after stale state reset, got %d entries", len(loaded.FileReads))
	}
}

func TestHandleReadCheck(t *testing.T) {
	// Save original directory and change to temp dir
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Create .tasuku directory for local state storage
	os.MkdirAll(".tasuku", 0755)

	// Clean up any existing state
	os.Remove(getInvestigationStatePath())

	tests := []struct {
		name           string
		config         featureConfig
		toolInput      string
		expectRecorded bool
	}{
		{
			name:           "records read when feature enabled",
			config:         featureConfig{"investigation_pattern": true},
			toolInput:      `{"file_path": "/path/to/file.go"}`,
			expectRecorded: true,
		},
		{
			name:           "does not record when feature disabled",
			config:         featureConfig{"investigation_pattern": false},
			toolInput:      `{"file_path": "/path/to/another.go"}`,
			expectRecorded: false,
		},
		{
			name:           "handles empty file_path",
			config:         featureConfig{"investigation_pattern": true},
			toolInput:      `{"file_path": ""}`,
			expectRecorded: false,
		},
		{
			name:           "handles invalid JSON",
			config:         featureConfig{"investigation_pattern": true},
			toolInput:      `not json`,
			expectRecorded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear state before each test
			os.Remove(getInvestigationStatePath())

			err := handleReadCheck(tt.config, tt.toolInput)
			if err != nil {
				t.Errorf("handleReadCheck returned error: %v", err)
			}

			// Check if file was recorded
			state, _ := loadInvestigationState()
			var input struct {
				FilePath string `json:"file_path"`
			}
			json.Unmarshal([]byte(tt.toolInput), &input)

			recorded := state.FileReads[input.FilePath] > 0
			if recorded != tt.expectRecorded {
				t.Errorf("expected recorded=%v, got %v", tt.expectRecorded, recorded)
			}
		})
	}
}

func TestHandleEditCheck(t *testing.T) {
	// Save original directory and change to temp dir
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Create .tasuku directory for local state storage
	os.MkdirAll(".tasuku", 0755)

	tests := []struct {
		name          string
		config        featureConfig
		setupReads    int
		toolInput     string
		expectTrigger bool
	}{
		{
			name:          "triggers when investigation pattern detected",
			config:        featureConfig{"investigation_pattern": true},
			setupReads:    3, // At threshold
			toolInput:     `{"file_path": "/path/to/file.go"}`,
			expectTrigger: true,
		},
		{
			name:          "does not trigger below threshold",
			config:        featureConfig{"investigation_pattern": true},
			setupReads:    2, // Below threshold
			toolInput:     `{"file_path": "/path/to/file2.go"}`,
			expectTrigger: false,
		},
		{
			name:          "does not trigger when feature disabled",
			config:        featureConfig{"investigation_pattern": false},
			setupReads:    5,
			toolInput:     `{"file_path": "/path/to/file3.go"}`,
			expectTrigger: false,
		},
		{
			name:          "handles empty file_path",
			config:        featureConfig{"investigation_pattern": true},
			setupReads:    5,
			toolInput:     `{"file_path": ""}`,
			expectTrigger: false,
		},
		{
			name:          "handles invalid JSON",
			config:        featureConfig{"investigation_pattern": true},
			setupReads:    5,
			toolInput:     `not json`,
			expectTrigger: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear state before each test
			os.Remove(getInvestigationStatePath())

			// Setup reads if needed
			var input struct {
				FilePath string `json:"file_path"`
			}
			json.Unmarshal([]byte(tt.toolInput), &input)

			if input.FilePath != "" && tt.setupReads > 0 && tt.config["investigation_pattern"] {
				for i := 0; i < tt.setupReads; i++ {
					recordFileRead(input.FilePath)
				}
			}

			// Check the pattern before calling handleEditCheck
			_, countBefore := checkInvestigationPattern(input.FilePath)
			wouldTrigger := countBefore >= investigationThreshold

			err := handleEditCheck(tt.config, tt.toolInput)
			if err != nil {
				t.Errorf("handleEditCheck returned error: %v", err)
			}

			// For feature disabled or invalid input, wouldTrigger should be false regardless
			if !tt.config["investigation_pattern"] || input.FilePath == "" {
				wouldTrigger = false
			}

			if wouldTrigger != tt.expectTrigger {
				t.Errorf("expected trigger=%v, got %v (countBefore=%d)", tt.expectTrigger, wouldTrigger, countBefore)
			}
		})
	}
}

func TestRunPlanSync(t *testing.T) {
	h := testutil.New(t)

	// Create a test plan file
	planContent := `# Implementation Plan

## Tasks
- [ ] Implement user authentication feature
- [ ] Add dark mode toggle setting
- [x] Fix type error in login (already done)
- [ ] Run tests for coverage

## Notes
Some notes here that aren't tasks.
`
	planPath := filepath.Join(h.TempDir(), "plan.md")
	os.WriteFile(planPath, []byte(planContent), 0644)

	// Run plan-sync with dry-run
	err := h.Execute(Cmd, "plan-sync", "--dry-run", planPath)
	h.AssertNoError(err)
	h.AssertOutputContains("Would create tasks:")
	h.AssertOutputContains("implement-user-authentication")
}

func TestRunPlanSyncCreatesTasks(t *testing.T) {
	h := testutil.New(t)

	// Create a test plan file with project-level tasks
	planContent := `# Plan
- [ ] Implement new feature for export
- [ ] Add API endpoint for users
`
	planPath := filepath.Join(h.TempDir(), "plan.md")
	os.WriteFile(planPath, []byte(planContent), 0644)

	// Run plan-sync without dry-run
	err := h.Execute(Cmd, "plan-sync", planPath)
	h.AssertNoError(err)
	h.AssertOutputContains("Creating tasks:")

	// Verify tasks were actually created
	err = h.Execute(Cmd, "plan-sync", "--dry-run", planPath)
	h.AssertNoError(err)
	h.AssertOutputContains("Already exists:")
}

func TestRunPlanSyncNoItems(t *testing.T) {
	h := testutil.New(t)

	// Create an empty plan file
	planPath := filepath.Join(h.TempDir(), "empty.md")
	os.WriteFile(planPath, []byte("# Empty Plan\n\nNo tasks here."), 0644)

	err := h.Execute(Cmd, "plan-sync", "--dry-run", planPath)
	h.AssertNoError(err)
	h.AssertOutputContains("No actionable items")
}

func TestRunPlanSyncAllFlag(t *testing.T) {
	h := testutil.New(t)

	// Create a plan file with session-level tasks
	planContent := `# Plan
- [ ] Fix type error in service
- [ ] Run tests for coverage
- [ ] Update config file
`
	planPath := filepath.Join(h.TempDir(), "plan.md")
	os.WriteFile(planPath, []byte(planContent), 0644)

	// Without --all, these should be skipped
	err := h.Execute(Cmd, "plan-sync", "--dry-run", planPath)
	h.AssertNoError(err)
	h.AssertOutputContains("Skipped (session-level)")

	// With --all, they should be created
	h.ResetOutput()
	err = h.Execute(Cmd, "plan-sync", "--dry-run", "--all", planPath)
	h.AssertNoError(err)
	h.AssertOutputContains("Would create tasks:")
}

func TestRunPlanSyncNoStorage(t *testing.T) {
	// Create temp dir without tasuku storage
	tempDir, _ := os.MkdirTemp("", "tasuku-test-plansync-*")
	defer os.RemoveAll(tempDir)

	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Create a plan file
	planContent := `- [ ] Implement feature`
	planPath := filepath.Join(tempDir, "plan.md")
	os.WriteFile(planPath, []byte(planContent), 0644)

	// Plan-sync should error when no storage exists
	cmd := newPlanSyncCmd()
	err := cmd.RunE(cmd, []string{planPath})
	if err == nil || !strings.Contains(err.Error(), "no Tasuku storage") {
		t.Errorf("expected 'no Tasuku storage' error, got: %v", err)
	}
}

func TestRunPlanSyncInvalidFile(t *testing.T) {
	h := testutil.New(t)

	// Run plan-sync with non-existent file
	err := h.Execute(Cmd, "plan-sync", "/nonexistent/plan.md")
	h.AssertError(err)
}

func TestParseDisabledFeatures(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: map[string]bool{},
		},
		{
			name:     "single feature",
			input:    "shipping_check",
			expected: map[string]bool{"shipping_check": true},
		},
		{
			name:     "multiple features",
			input:    "shipping_check,scope_warning,rule_detection",
			expected: map[string]bool{"shipping_check": true, "scope_warning": true, "rule_detection": true},
		},
		{
			name:     "features with spaces",
			input:    " shipping_check , scope_warning ",
			expected: map[string]bool{"shipping_check": true, "scope_warning": true},
		},
		{
			name:     "empty values in list",
			input:    "shipping_check,,scope_warning",
			expected: map[string]bool{"shipping_check": true, "scope_warning": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDisabledFeatures(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d features, got %d", len(tt.expected), len(result))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("expected %s=%v, got %v", k, v, result[k])
				}
			}
		})
	}
}

func TestBuildFeatureConfig(t *testing.T) {
	allFeatures := []string{"feature1", "feature2", "feature3", "quiet_feature"}
	quietFeatures := map[string]bool{"quiet_feature": true}

	tests := []struct {
		name     string
		quiet    bool
		disabled map[string]bool
		expected featureConfig
	}{
		{
			name:     "all enabled",
			quiet:    false,
			disabled: map[string]bool{},
			expected: featureConfig{"feature1": true, "feature2": true, "feature3": true, "quiet_feature": true},
		},
		{
			name:     "quiet mode - only quiet features enabled",
			quiet:    true,
			disabled: map[string]bool{},
			expected: featureConfig{"feature1": false, "feature2": false, "feature3": false, "quiet_feature": true},
		},
		{
			name:     "some disabled",
			quiet:    false,
			disabled: map[string]bool{"feature1": true, "feature3": true},
			expected: featureConfig{"feature1": false, "feature2": true, "feature3": false, "quiet_feature": true},
		},
		{
			name:     "quiet mode with disabled",
			quiet:    true,
			disabled: map[string]bool{"quiet_feature": true},
			expected: featureConfig{"feature1": false, "feature2": false, "feature3": false, "quiet_feature": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildFeatureConfig(allFeatures, quietFeatures, tt.quiet, tt.disabled)
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("expected %s=%v, got %v", k, v, result[k])
				}
			}
		})
	}
}

func TestStopReminderCmd(t *testing.T) {
	h := testutil.New(t)

	// Add an in-progress task with a running timer
	h.AddTaskWithStatus("task-1", "In progress task", task.StatusInProgress)
	staleTime := time.Now().Add(-30 * time.Minute)
	h.StartTimerAt("task-1", staleTime)

	err := h.Execute(Cmd, "stop-reminder")
	h.AssertNoError(err)
	h.AssertOutputContains("Running timers")
	h.AssertOutputContains("task-1")
}

func TestStopReminderNoReminders(t *testing.T) {
	h := testutil.New(t)

	// Add only done tasks
	h.AddTaskWithStatus("task-1", "Done task", task.StatusDone)

	err := h.Execute(Cmd, "stop-reminder")
	h.AssertNoError(err)
	// Should have no reminders output
}

func TestCodexNotifyCmd(t *testing.T) {
	h := testutil.New(t)

	// Test with valid JSON
	jsonPayload := `{"type":"agent-turn-complete","thread-id":"123"}`
	err := h.Execute(Cmd, "codex-notify", jsonPayload)
	h.AssertNoError(err)
}

func TestCodexNotifyNoArgs(t *testing.T) {
	h := testutil.New(t)

	// Test with no arguments
	err := h.Execute(Cmd, "codex-notify")
	h.AssertNoError(err)
}

func TestPreCompactCmd(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "pre-compact")
	h.AssertNoError(err)
}

func TestSubagentDoneCmd(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "subagent-done")
	h.AssertNoError(err)
}

func TestDetectArchitectureExplanation(t *testing.T) {
	tests := []struct {
		prompt         string
		isArchitecture bool
	}{
		{"we chose Go because it compiles to a single binary", true},
		{"because we need fast startup for CLI tools", true},
		{"the reason is that JSON parses faster than YAML", true},
		{"that's why we use flock for file locking", true},
		{"we went with MCP over REST for native integration", true},
		{"we opted for local-first design", true},
		{"the decision was to use constructor pattern", true},
		{"implement user login", false},
		{"fix the bug in payment", false},
		{"add dark mode support", false},
		{"what is the purpose of this function", false},
	}

	for _, tt := range tests {
		got := detectArchitectureExplanation(tt.prompt)
		if got != tt.isArchitecture {
			t.Errorf("detectArchitectureExplanation(%q) = %v, want %v", tt.prompt, got, tt.isArchitecture)
		}
	}
}

func TestExtractDecisionContent(t *testing.T) {
	tests := []struct {
		prompt   string
		expected string
	}{
		{"we chose Go because it compiles to a single binary", "it compiles to a single binary"},
		{"the reason is that JSON parses faster", "that JSON parses faster"},
		{"that's why we use flock for file locking", "we use flock for file locking"},
		{"implement user login", ""}, // No decision content
		{"", ""},                     // Empty prompt
	}

	for _, tt := range tests {
		got := extractDecisionContent(tt.prompt)
		if got != tt.expected {
			t.Errorf("extractDecisionContent(%q) = %q, want %q", tt.prompt, got, tt.expected)
		}
	}
}

func TestDetectUserPreference(t *testing.T) {
	tests := []struct {
		prompt       string
		isPreference bool
	}{
		{"i prefer explicit error handling over panic", true},
		{"please always use descriptive variable names", true},
		{"never use magic numbers in the code", true},
		{"always use context for cancellation", true},
		{"from now on use early returns", true},
		{"i'd like you to follow the existing patterns", true},
		{"my preference is for smaller functions", true},
		{"implement user login", false},
		{"fix the bug in payment", false},
		{"what does this code do", false},
	}

	for _, tt := range tests {
		got := detectUserPreference(tt.prompt)
		if got != tt.isPreference {
			t.Errorf("detectUserPreference(%q) = %v, want %v", tt.prompt, got, tt.isPreference)
		}
	}
}

func TestExtractPreferenceContent(t *testing.T) {
	tests := []struct {
		prompt     string
		hasContent bool
	}{
		{"i prefer explicit error handling over panic", true},
		{"please always use descriptive variable names", true},
		{"from now on use early returns for guard clauses", true},
		{"implement user login", false}, // No preference content
		{"", false},                     // Empty prompt
	}

	for _, tt := range tests {
		got := extractPreferenceContent(tt.prompt)
		hasContent := got != ""
		if hasContent != tt.hasContent {
			t.Errorf("extractPreferenceContent(%q) hasContent=%v, want %v (got: %q)", tt.prompt, hasContent, tt.hasContent, got)
		}
	}
}

// Integration tests for prompt-check command with new nudges

func TestPromptCheckArchitectureExplanation(t *testing.T) {
	h := testutil.New(t)

	// Set the USER_PROMPT environment variable
	os.Setenv("USER_PROMPT", "We chose Go because it compiles to a single binary and has fast startup times for CLI tools")
	defer os.Unsetenv("USER_PROMPT")

	err := h.Execute(Cmd, "prompt-check")
	h.AssertNoError(err)
	h.AssertOutputContains("architectural decision")
	h.AssertOutputContains("tk decide")
}

func TestPromptCheckArchitectureExplanationDisabled(t *testing.T) {
	h := testutil.New(t)

	os.Setenv("USER_PROMPT", "We chose Go because it compiles to a single binary")
	defer os.Unsetenv("USER_PROMPT")

	err := h.Execute(Cmd, "prompt-check", "--disable=architecture_explanation")
	h.AssertNoError(err)
	h.AssertOutputNotContains("architectural decision")
}

func TestPromptCheckPreferenceStated(t *testing.T) {
	h := testutil.New(t)

	os.Setenv("USER_PROMPT", "I prefer explicit error handling over panic recovery in this codebase")
	defer os.Unsetenv("USER_PROMPT")

	err := h.Execute(Cmd, "prompt-check")
	h.AssertNoError(err)
	h.AssertOutputContains("preference detected")
	h.AssertOutputContains("tk_learn")
}

func TestPromptCheckPreferenceStatedDisabled(t *testing.T) {
	h := testutil.New(t)

	os.Setenv("USER_PROMPT", "I prefer explicit error handling over panic recovery")
	defer os.Unsetenv("USER_PROMPT")

	err := h.Execute(Cmd, "prompt-check", "--disable=preference_stated")
	h.AssertNoError(err)
	h.AssertOutputNotContains("preference detected")
}

func TestPromptCheckQuietModeIncludesNewNudges(t *testing.T) {
	h := testutil.New(t)

	// Architecture explanation should show in quiet mode (it's enabled)
	os.Setenv("USER_PROMPT", "We chose JSON because it parses faster than YAML")
	defer os.Unsetenv("USER_PROMPT")

	err := h.Execute(Cmd, "prompt-check", "--quiet")
	h.AssertNoError(err)
	h.AssertOutputContains("architectural decision")
}

func TestPromptCheckShortPromptSkipped(t *testing.T) {
	h := testutil.New(t)

	// Very short prompts should be skipped (< 15 chars)
	os.Setenv("USER_PROMPT", "yes")
	defer os.Unsetenv("USER_PROMPT")

	err := h.Execute(Cmd, "prompt-check")
	h.AssertNoError(err)
	// Should produce no output for short prompts
	if h.Stdout() != "" {
		t.Errorf("expected empty output for short prompt, got: %s", h.Stdout())
	}
}

func TestPromptCheckNoStorageNoError(t *testing.T) {
	// Create temp dir WITHOUT tasuku storage
	tempDir, _ := os.MkdirTemp("", "tasuku-test-nostorage-*")
	defer os.RemoveAll(tempDir)

	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	os.Setenv("USER_PROMPT", "We chose Go because it's fast")
	defer os.Unsetenv("USER_PROMPT")

	// Should not error, just silently return
	cmd := newPromptCheckCmd()
	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Errorf("prompt-check should not error without storage, got: %v", err)
	}
}

func TestPromptCheckPreferenceNotDuplicated(t *testing.T) {
	h := testutil.New(t)

	// Add an existing learning that matches the preference
	h.AddLearning("I prefer explicit error handling over panic")

	os.Setenv("USER_PROMPT", "I prefer explicit error handling over panic in this codebase")
	defer os.Unsetenv("USER_PROMPT")

	err := h.Execute(Cmd, "prompt-check")
	h.AssertNoError(err)
	// Should NOT prompt for preference since similar learning exists
	h.AssertOutputNotContains("preference detected")
}

func TestPromptCheckListFeaturesIncludesNewNudges(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "prompt-check", "--list-features")
	h.AssertNoError(err)
	h.AssertOutputContains("architecture_explanation")
	h.AssertOutputContains("preference_stated")
}

func TestPromptCheckEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		prompt       string
		expectOutput bool
		expectedText string
	}{
		{
			name:         "unicode in architecture explanation",
			prompt:       "We chose 日本語 because it's better for i18n",
			expectOutput: true,
			expectedText: "architectural decision",
		},
		{
			name:         "very long preference",
			prompt:       "I prefer " + strings.Repeat("explicit error handling with proper context ", 20),
			expectOutput: true,
			expectedText: "preference detected",
		},
		{
			name:         "mixed case preference",
			prompt:       "I PREFER using interfaces over concrete types",
			expectOutput: true,
			expectedText: "preference detected",
		},
		{
			name:         "because in non-architecture context",
			prompt:       "The test failed because the mock was wrong",
			expectOutput: false, // "because the" isn't a pattern, but "because it/we/they" are
			expectedText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testutil.New(t)

			os.Setenv("USER_PROMPT", tt.prompt)
			defer os.Unsetenv("USER_PROMPT")

			err := h.Execute(Cmd, "prompt-check")
			h.AssertNoError(err)

			if tt.expectOutput && tt.expectedText != "" {
				h.AssertOutputContains(tt.expectedText)
			}
		})
	}
}

// Copilot CLI hooks tests

func TestInstallCopilotHooks(t *testing.T) {
	h := testutil.New(t)

	// Initialize a git repo in the temp dir (needed for general hooks install)
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Install Copilot hooks only
	err := h.Execute(Cmd, "install", "--copilot")
	h.AssertNoError(err)
	h.AssertOutputContains("Copilot CLI")

	// Check hooks file was created
	hooksFile := filepath.Join(h.TempDir(), ".github", "hooks", "tasuku.json")
	if _, err := os.Stat(hooksFile); os.IsNotExist(err) {
		t.Error("Copilot hooks file should exist")
	}

	// Verify the content
	data, err := os.ReadFile(hooksFile)
	if err != nil {
		t.Fatalf("failed to read hooks file: %v", err)
	}

	var config CopilotHooksConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse hooks file: %v", err)
	}

	if config.Version != 1 {
		t.Errorf("expected version 1, got %d", config.Version)
	}

	// Check that hooks were added
	if len(config.Hooks["sessionStart"]) != 1 {
		t.Errorf("expected 1 sessionStart hook, got %d", len(config.Hooks["sessionStart"]))
	}
	if len(config.Hooks["sessionEnd"]) != 1 {
		t.Errorf("expected 1 sessionEnd hook, got %d", len(config.Hooks["sessionEnd"]))
	}
	if len(config.Hooks["userPromptSubmitted"]) != 1 {
		t.Errorf("expected 1 userPromptSubmitted hook, got %d", len(config.Hooks["userPromptSubmitted"]))
	}
	if len(config.Hooks["postToolUse"]) != 1 {
		t.Errorf("expected 1 postToolUse hook, got %d", len(config.Hooks["postToolUse"]))
	}
}

func TestUninstallCopilotHooks(t *testing.T) {
	h := testutil.New(t)

	// Initialize git repo
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Install first
	err := h.Execute(Cmd, "install", "--copilot")
	h.AssertNoError(err)

	// Then uninstall
	err = h.Execute(Cmd, "uninstall", "--copilot")
	h.AssertNoError(err)
	h.AssertOutputContains("Removed")

	// Check that file was deleted (since it only had Tasuku hooks)
	hooksFile := filepath.Join(h.TempDir(), ".github", "hooks", "tasuku.json")
	if _, err := os.Stat(hooksFile); !os.IsNotExist(err) {
		t.Error("Copilot hooks file should be deleted when empty")
	}
}

func TestCopilotHooksIdempotent(t *testing.T) {
	h := testutil.New(t)

	// Initialize git repo
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Install twice
	err := h.Execute(Cmd, "install", "--copilot")
	h.AssertNoError(err)

	err = h.Execute(Cmd, "install", "--copilot")
	h.AssertNoError(err)
	h.AssertOutputContains("up to date")

	// Should still have only one hook per type
	hooksFile := filepath.Join(h.TempDir(), ".github", "hooks", "tasuku.json")
	data, _ := os.ReadFile(hooksFile)

	var config CopilotHooksConfig
	json.Unmarshal(data, &config)

	if len(config.Hooks["sessionStart"]) != 1 {
		t.Errorf("expected 1 sessionStart hook after double install, got %d", len(config.Hooks["sessionStart"]))
	}
}

func TestCopilotHooksForceUpdate(t *testing.T) {
	h := testutil.New(t)

	// Initialize git repo
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Create hooks dir and file with old content
	hooksDir := filepath.Join(h.TempDir(), ".github", "hooks")
	os.MkdirAll(hooksDir, 0755)
	oldConfig := CopilotHooksConfig{
		Version: 1,
		Hooks: map[string][]CopilotHook{
			"sessionStart": {
				{
					Type: "command",
					Bash: "tk hooks session # tasuku-hook-old-version",
				},
			},
		},
	}
	data, _ := json.Marshal(oldConfig)
	os.WriteFile(filepath.Join(hooksDir, "tasuku.json"), data, 0644)

	// Force reinstall
	err := h.Execute(Cmd, "install", "--copilot", "--force")
	h.AssertNoError(err)

	// Should have updated all hooks
	newData, _ := os.ReadFile(filepath.Join(hooksDir, "tasuku.json"))
	var newConfig CopilotHooksConfig
	json.Unmarshal(newData, &newConfig)

	// Force should add all hooks, not just update existing
	if len(newConfig.Hooks["sessionEnd"]) != 1 {
		t.Error("force install should add all hooks including sessionEnd")
	}
}

func TestCopilotHooksPreservesOther(t *testing.T) {
	h := testutil.New(t)

	// Initialize git repo
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Create hooks dir and file with a non-Tasuku hook
	hooksDir := filepath.Join(h.TempDir(), ".github", "hooks")
	os.MkdirAll(hooksDir, 0755)
	existingConfig := CopilotHooksConfig{
		Version: 1,
		Hooks: map[string][]CopilotHook{
			"sessionStart": {
				{
					Type: "command",
					Bash: "echo 'other hook'",
				},
			},
		},
	}
	data, _ := json.Marshal(existingConfig)
	os.WriteFile(filepath.Join(hooksDir, "tasuku.json"), data, 0644)

	// Install Tasuku hooks
	err := h.Execute(Cmd, "install", "--copilot")
	h.AssertNoError(err)

	// Should have both hooks
	newData, _ := os.ReadFile(filepath.Join(hooksDir, "tasuku.json"))
	var newConfig CopilotHooksConfig
	json.Unmarshal(newData, &newConfig)

	if len(newConfig.Hooks["sessionStart"]) != 2 {
		t.Errorf("expected 2 sessionStart hooks (original + Tasuku), got %d", len(newConfig.Hooks["sessionStart"]))
	}
}

func TestUninstallCopilotHooksPreservesOther(t *testing.T) {
	h := testutil.New(t)

	// Initialize git repo
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Create hooks dir and file with both Tasuku and non-Tasuku hooks
	hooksDir := filepath.Join(h.TempDir(), ".github", "hooks")
	os.MkdirAll(hooksDir, 0755)
	existingConfig := CopilotHooksConfig{
		Version: 1,
		Hooks: map[string][]CopilotHook{
			"sessionStart": {
				{
					Type: "command",
					Bash: "echo 'other hook'",
				},
				{
					Type: "command",
					Bash: "tk hooks session # tasuku-hook",
				},
			},
		},
	}
	data, _ := json.Marshal(existingConfig)
	os.WriteFile(filepath.Join(hooksDir, "tasuku.json"), data, 0644)

	// Uninstall Tasuku hooks
	err := h.Execute(Cmd, "uninstall", "--copilot")
	h.AssertNoError(err)

	// Should still have the non-Tasuku hook
	newData, _ := os.ReadFile(filepath.Join(hooksDir, "tasuku.json"))
	var newConfig CopilotHooksConfig
	json.Unmarshal(newData, &newConfig)

	if len(newConfig.Hooks["sessionStart"]) != 1 {
		t.Errorf("expected 1 sessionStart hook after uninstall, got %d", len(newConfig.Hooks["sessionStart"]))
	}
	if newConfig.Hooks["sessionStart"][0].Bash != "echo 'other hook'" {
		t.Error("non-Tasuku hook should be preserved")
	}
}

// --- TaskCompleted hook tests ---

func TestTaskCompletedWithDependents(t *testing.T) {
	h := testutil.New(t)

	h.AddTaskWithStatus("auth-system", "Implement auth system", task.StatusDone)
	h.AddTaskWithStatus("user-dashboard", "Build user dashboard", task.StatusReady)
	h.Store().BlockTask("user-dashboard", []string{"auth-system"})
	h.Store().SetOwner("user-dashboard", "worker-2")

	input := map[string]string{
		"task_id":       "auth-system",
		"task_subject":  "Auth system",
		"teammate_name": "worker-1",
	}

	output := captureStdoutWithStdin(t, input, func() {
		hookTaskCompleted()
	})

	if !strings.Contains(output, "Task Completed: auth-system") {
		t.Errorf("expected header with task ID, got:\n%s", output)
	}
	if !strings.Contains(output, "Tasks blocked by this task") {
		t.Errorf("expected blocked-by-this-task section, got:\n%s", output)
	}
	if !strings.Contains(output, "user-dashboard") {
		t.Errorf("expected dependent task ID, got:\n%s", output)
	}
	if !strings.Contains(output, "worker-2") {
		t.Errorf("expected owner name, got:\n%s", output)
	}
	if !strings.Contains(output, "Reflect on this work") {
		t.Errorf("expected reflection prompts, got:\n%s", output)
	}
}

func TestTaskCompletedNoMatch(t *testing.T) {
	h := testutil.New(t)
	_ = h // storage exists but no matching task

	input := map[string]string{
		"task_id":      "nonexistent",
		"task_subject": "Some unknown task",
	}

	output := captureStdoutWithStdin(t, input, func() {
		hookTaskCompleted()
	})

	if !strings.Contains(output, "Task Completed: nonexistent") {
		t.Errorf("expected header with task_id, got:\n%s", output)
	}
	if !strings.Contains(output, "Reflect on this work") {
		t.Errorf("expected reflection prompts, got:\n%s", output)
	}
	// Should NOT have blocked-by-this-task section
	if strings.Contains(output, "Tasks blocked by this task") {
		t.Errorf("should not have blocked-by-this-task section for non-matching task, got:\n%s", output)
	}
}

func TestTaskCompletedNoDependents(t *testing.T) {
	h := testutil.New(t)

	// Task exists but nothing is blocked by it
	h.AddTaskWithStatus("simple-task", "Simple standalone task", task.StatusDone)

	input := map[string]string{
		"task_id":      "simple-task",
		"task_subject": "Simple standalone task",
	}

	output := captureStdoutWithStdin(t, input, func() {
		hookTaskCompleted()
	})

	if !strings.Contains(output, "Task Completed: simple-task") {
		t.Errorf("expected header, got:\n%s", output)
	}
	if !strings.Contains(output, "Reflect on this work") {
		t.Errorf("expected reflection prompts, got:\n%s", output)
	}
	if strings.Contains(output, "Tasks blocked by this task") {
		t.Errorf("should not have blocked-by-this-task section when no dependents, got:\n%s", output)
	}
}

func TestTaskCompletedSubjectMatch(t *testing.T) {
	h := testutil.New(t)

	// Task ID is generated from subject
	h.AddTaskWithStatus("auth-system", "Auth system", task.StatusDone)
	h.AddTaskWithStatus("user-profile", "User profile page", task.StatusReady)
	h.Store().BlockTask("user-profile", []string{"auth-system"})

	// Use task_subject that generates "auth-system" via generateID
	input := map[string]string{
		"task_id":      "no-match-id",
		"task_subject": "Auth System",
	}

	output := captureStdoutWithStdin(t, input, func() {
		hookTaskCompleted()
	})

	if !strings.Contains(output, "Task Completed: auth-system") {
		t.Errorf("expected subject-matched task ID, got:\n%s", output)
	}
	if !strings.Contains(output, "Tasks blocked by this task") {
		t.Errorf("expected blocked-by-this-task section via subject match, got:\n%s", output)
	}
}

func TestTaskCompletedSubjectMatchUsesCanonicalIDRules(t *testing.T) {
	h := testutil.New(t)

	// Task ID reflects task.GenerateTaskID normalization (digits are dropped).
	h.AddTaskWithStatus("oauth-v-support", "OAuth v2 support", task.StatusDone)
	h.AddTaskWithStatus("post-auth-cleanup", "Post auth cleanup", task.StatusReady)
	h.Store().BlockTask("post-auth-cleanup", []string{"oauth-v-support"})

	input := map[string]string{
		"task_id":      "no-match-id",
		"task_subject": "OAuth v2 support",
	}

	output := captureStdoutWithStdin(t, input, func() {
		hookTaskCompleted()
	})

	if !strings.Contains(output, "Task Completed: oauth-v-support") {
		t.Errorf("expected canonical subject-matched task ID, got:\n%s", output)
	}
	if !strings.Contains(output, "post-auth-cleanup") {
		t.Errorf("expected dependent task in output, got:\n%s", output)
	}
}

// --- TeammateIdle hook tests ---

func TestTeammateIdleWithDependents(t *testing.T) {
	h := testutil.New(t)

	h.AddTaskWithStatus("build-api", "Build the API", task.StatusInProgress)
	h.Store().SetOwner("build-api", "worker-1")
	h.AddTaskWithStatus("write-docs", "Write API docs", task.StatusReady)
	h.Store().BlockTask("write-docs", []string{"build-api"})

	input := map[string]string{
		"teammate_name": "worker-1",
		"team_name":     "my-team",
	}

	output := captureStdoutWithStdin(t, input, func() {
		hookTeammateIdle()
	})

	if !strings.Contains(output, "Teammate Idle: worker-1") {
		t.Errorf("expected header, got:\n%s", output)
	}
	if !strings.Contains(output, "Your tasks blocking others") {
		t.Errorf("expected blocking section, got:\n%s", output)
	}
	if !strings.Contains(output, "build-api") {
		t.Errorf("expected owned task ID, got:\n%s", output)
	}
	if !strings.Contains(output, "write-docs") {
		t.Errorf("expected dependent task ID, got:\n%s", output)
	}
	if strings.Contains(output, "Before going idle, reflect") {
		t.Errorf("did not expect reflection prompts for teammate idle, got:\n%s", output)
	}
}

func TestTeammateIdleNoDependents(t *testing.T) {
	h := testutil.New(t)

	// Teammate owns a task but nothing depends on it
	h.AddTaskWithStatus("solo-task", "Solo task", task.StatusInProgress)
	h.Store().SetOwner("solo-task", "worker-1")

	input := map[string]string{
		"teammate_name": "worker-1",
		"team_name":     "my-team",
	}

	output := captureStdoutWithStdin(t, input, func() {
		hookTeammateIdle()
	})

	if !strings.Contains(output, "Teammate Idle: worker-1") {
		t.Errorf("expected header, got:\n%s", output)
	}
	if strings.Contains(output, "Your tasks blocking others") {
		t.Errorf("should not have blocking section, got:\n%s", output)
	}
	if strings.Contains(output, "Before going idle, reflect") {
		t.Errorf("did not expect reflection prompts for teammate idle, got:\n%s", output)
	}
}

func TestTeammateIdleNoOwnedTasks(t *testing.T) {
	h := testutil.New(t)

	// Tasks exist but not owned by this teammate
	h.AddTaskWithStatus("other-task", "Other task", task.StatusInProgress)
	h.Store().SetOwner("other-task", "someone-else")

	input := map[string]string{
		"teammate_name": "worker-1",
		"team_name":     "my-team",
	}

	output := captureStdoutWithStdin(t, input, func() {
		hookTeammateIdle()
	})

	if !strings.Contains(output, "Teammate Idle: worker-1") {
		t.Errorf("expected header, got:\n%s", output)
	}
	if strings.Contains(output, "Your tasks blocking others") {
		t.Errorf("should not have blocking section, got:\n%s", output)
	}
	if strings.Contains(output, "Before going idle, reflect") {
		t.Errorf("did not expect reflection prompts for teammate idle, got:\n%s", output)
	}
}

func TestTeammateIdleSkipsDoneOwnedTasks(t *testing.T) {
	h := testutil.New(t)

	// Done tasks can still appear in BlockedBy, but should not be reported as blockers.
	h.AddTaskWithStatus("done-owner-task", "Already completed task", task.StatusDone)
	h.Store().SetOwner("done-owner-task", "worker-1")
	h.AddTaskWithStatus("dependent-task", "Still blocked by other work", task.StatusBlocked)
	h.Store().BlockTask("dependent-task", []string{"done-owner-task", "active-blocker"})

	input := map[string]string{
		"teammate_name": "worker-1",
		"team_name":     "my-team",
	}

	output := captureStdoutWithStdin(t, input, func() {
		hookTeammateIdle()
	})

	if strings.Contains(output, "Your tasks blocking others") {
		t.Errorf("expected done owned task to be excluded from blockers, got:\n%s", output)
	}
}

// captureStdoutWithStdin pipes JSON to stdin and captures stdout from fn.
func captureStdoutWithStdin(t *testing.T, stdinData interface{}, fn func()) string {
	t.Helper()

	jsonBytes, err := json.Marshal(stdinData)
	if err != nil {
		t.Fatalf("failed to marshal stdin JSON: %v", err)
	}

	// Set up stdin pipe
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	oldStdin := os.Stdin
	os.Stdin = stdinR

	go func() {
		stdinW.Write(jsonBytes)
		stdinW.Close()
	}()

	// Set up stdout capture
	oldStdout := os.Stdout
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		os.Stdin = oldStdin
		stdinR.Close()
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = stdoutW

	fn()

	stdoutW.Close()
	os.Stdout = oldStdout
	os.Stdin = oldStdin
	stdinR.Close()

	var buf strings.Builder
	outBytes := make([]byte, 4096)
	for {
		n, readErr := stdoutR.Read(outBytes)
		if n > 0 {
			buf.Write(outBytes[:n])
		}
		if readErr != nil {
			break
		}
	}
	stdoutR.Close()

	return buf.String()
}

func TestCopilotHooksNoGithubDir(t *testing.T) {
	h := testutil.New(t)

	// Initialize git repo but don't create .github directory
	gitDir := filepath.Join(h.TempDir(), ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Should create .github/hooks directory automatically
	err := h.Execute(Cmd, "install", "--copilot")
	h.AssertNoError(err)

	// Check .github/hooks was created
	hooksDir := filepath.Join(h.TempDir(), ".github", "hooks")
	if _, err := os.Stat(hooksDir); os.IsNotExist(err) {
		t.Error(".github/hooks directory should be created")
	}
}
