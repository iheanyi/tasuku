package migrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
)

func TestMigrateCmdStructure(t *testing.T) {
	if Cmd.Use != "migrate" {
		t.Errorf("expected Use to be 'migrate', got %s", Cmd.Use)
	}

	// Check subcommands exist
	subcommands := make(map[string]bool)
	for _, sub := range Cmd.Commands() {
		subcommands[sub.Use] = true
	}

	if !subcommands["beads"] {
		t.Error("expected 'beads' subcommand")
	}
	if !subcommands["v3"] {
		t.Error("expected 'v3' subcommand")
	}
}

func TestBeadsCmdFlags(t *testing.T) {
	for _, sub := range Cmd.Commands() {
		if sub.Use == "beads" {
			dryRunFlag := sub.Flags().Lookup("dry-run")
			if dryRunFlag == nil {
				t.Error("expected --dry-run flag on beads command")
			}
			break
		}
	}
}

func TestV3CmdFlags(t *testing.T) {
	for _, sub := range Cmd.Commands() {
		if sub.Use == "v3" {
			dryRunFlag := sub.Flags().Lookup("dry-run")
			if dryRunFlag == nil {
				t.Error("expected --dry-run flag on v3 command")
			}
			break
		}
	}
}

func TestMigrateBeadsNoDirectory(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "beads")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, ".beads directory not found")
}

func TestMigrateBeadsDryRun(t *testing.T) {
	h := testutil.New(t)

	// Create .beads directory with a test issue
	beadsDir := filepath.Join(h.TempDir(), ".beads")
	os.MkdirAll(beadsDir, 0755)

	issuesFile := filepath.Join(beadsDir, "issues.jsonl")
	issueJSON := `{"id":"TEST-1","title":"Test issue","status":"open","priority":2,"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`
	os.WriteFile(issuesFile, []byte(issueJSON), 0644)

	err := h.Execute(Cmd, "beads", "--dry-run")
	h.AssertNoError(err)
	h.AssertOutputContains("TEST-1")
	h.AssertOutputContains("dry-run")
}

func TestMigrateV3NoFile(t *testing.T) {
	h := testutil.New(t)

	// The test harness uses V3 dir format, so there's no .tasuku.json to migrate
	err := h.Execute(Cmd, "v3")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "no .tasuku.json found")
}

func TestMigrateV3AlreadyMigrated(t *testing.T) {
	h := testutil.New(t)

	// Create both .tasuku.json and .tasuku/ directory
	oldFile := filepath.Join(h.TempDir(), ".tasuku.json")
	os.WriteFile(oldFile, []byte(`{"version":2,"tasks":{},"context":{"learnings":[],"decisions":[]}}`), 0644)

	// .tasuku/ already exists from harness setup
	err := h.Execute(Cmd, "v3")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "already exists")
}

func TestV4CmdFlags(t *testing.T) {
	for _, sub := range Cmd.Commands() {
		if sub.Use == "v4" {
			dryRunFlag := sub.Flags().Lookup("dry-run")
			if dryRunFlag == nil {
				t.Error("expected --dry-run flag on v4 command")
			}
			break
		}
	}
}

func TestV4CmdExists(t *testing.T) {
	// Check v4 subcommand exists
	subcommands := make(map[string]bool)
	for _, sub := range Cmd.Commands() {
		subcommands[sub.Use] = true
	}

	if !subcommands["v4"] {
		t.Error("expected 'v4' subcommand")
	}
}

func TestMigrateV4AlreadyV4(t *testing.T) {
	h := testutil.New(t)

	// Create a V4 config.json
	configPath := filepath.Join(h.TempDir(), ".tasuku", "config.json")
	os.WriteFile(configPath, []byte(`{"version":4}`), 0644)

	err := h.Execute(Cmd, "v4")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "already using V4")
}

func TestMigrateBeadsWithMultipleIssues(t *testing.T) {
	h := testutil.New(t)

	// Create .beads directory with multiple issues
	beadsDir := filepath.Join(h.TempDir(), ".beads")
	os.MkdirAll(beadsDir, 0755)

	issuesFile := filepath.Join(beadsDir, "issues.jsonl")
	issues := `{"id":"TEST-1","title":"First issue","status":"open","priority":1,"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}
{"id":"TEST-2","title":"Second issue","status":"in_progress","priority":2,"created_at":"2024-01-02T00:00:00Z","updated_at":"2024-01-02T00:00:00Z"}
{"id":"TEST-3","title":"Third issue","status":"closed","priority":0,"created_at":"2024-01-03T00:00:00Z","updated_at":"2024-01-03T00:00:00Z"}`
	os.WriteFile(issuesFile, []byte(issues), 0644)

	err := h.Execute(Cmd, "beads", "--dry-run")
	h.AssertNoError(err)
	h.AssertOutputContains("TEST-1")
	h.AssertOutputContains("TEST-2")
	h.AssertOutputContains("TEST-3")
	h.AssertOutputContains("3 issues")
}

func TestMigrateBeadsWithDependencies(t *testing.T) {
	h := testutil.New(t)

	// Create .beads directory with issues that have dependencies
	beadsDir := filepath.Join(h.TempDir(), ".beads")
	os.MkdirAll(beadsDir, 0755)

	issuesFile := filepath.Join(beadsDir, "issues.jsonl")
	issues := `{"id":"TEST-1","title":"Parent issue","status":"open","priority":2,"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}
{"id":"TEST-2","title":"Child issue","status":"blocked","priority":2,"dependencies":[{"type":"blocked_by","target_id":"TEST-1"}],"created_at":"2024-01-02T00:00:00Z","updated_at":"2024-01-02T00:00:00Z"}`
	os.WriteFile(issuesFile, []byte(issues), 0644)

	err := h.Execute(Cmd, "beads", "--dry-run")
	h.AssertNoError(err)
	h.AssertOutputContains("TEST-1")
	h.AssertOutputContains("TEST-2")
}

func TestMigrateBeadsEmptyFile(t *testing.T) {
	h := testutil.New(t)

	// Create .beads directory with empty issues file
	beadsDir := filepath.Join(h.TempDir(), ".beads")
	os.MkdirAll(beadsDir, 0755)

	issuesFile := filepath.Join(beadsDir, "issues.jsonl")
	os.WriteFile(issuesFile, []byte(""), 0644)

	err := h.Execute(Cmd, "beads")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "no issues found")
}

func TestMigrateBeadsWithDescription(t *testing.T) {
	h := testutil.New(t)

	// Create .beads directory with issue that has description
	beadsDir := filepath.Join(h.TempDir(), ".beads")
	os.MkdirAll(beadsDir, 0755)

	issuesFile := filepath.Join(beadsDir, "issues.jsonl")
	issues := `{"id":"TEST-1","title":"Issue with description","description":"This is the description","status":"open","priority":2,"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`
	os.WriteFile(issuesFile, []byte(issues), 0644)

	err := h.Execute(Cmd, "beads", "--dry-run")
	h.AssertNoError(err)
	h.AssertOutputContains("TEST-1")
}

func TestMigrateBeadsVariousStatuses(t *testing.T) {
	h := testutil.New(t)

	// Create .beads directory with various status mappings
	beadsDir := filepath.Join(h.TempDir(), ".beads")
	os.MkdirAll(beadsDir, 0755)

	issuesFile := filepath.Join(beadsDir, "issues.jsonl")
	issues := `{"id":"TEST-1","title":"Open issue","status":"open","priority":2,"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}
{"id":"TEST-2","title":"Active issue","status":"active","priority":2,"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}
{"id":"TEST-3","title":"Deferred issue","status":"deferred","priority":2,"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}
{"id":"TEST-4","title":"Done issue","status":"done","priority":2,"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`
	os.WriteFile(issuesFile, []byte(issues), 0644)

	err := h.Execute(Cmd, "beads", "--dry-run")
	h.AssertNoError(err)
	h.AssertOutputContains("open")
}
