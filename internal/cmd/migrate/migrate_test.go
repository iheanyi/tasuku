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
