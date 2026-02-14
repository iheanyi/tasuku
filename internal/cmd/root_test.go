package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
)

func TestDoctorFindsProjectCursorConfigFromSubdirectory(t *testing.T) {
	h := testutil.New(t)

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to get test executable path: %v", err)
	}

	cursorConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"tasuku": map[string]interface{}{
				"command": executable,
				"args":    []string{"serve", "mcp", "--dir", h.TempDir()},
				"type":    "stdio",
				"cwd":     h.TempDir(),
			},
		},
	}

	cursorDir := filepath.Join(h.TempDir(), ".cursor")
	if err := os.MkdirAll(cursorDir, 0755); err != nil {
		t.Fatalf("failed to create cursor config dir: %v", err)
	}
	data, err := json.MarshalIndent(cursorConfig, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal cursor config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cursorDir, "mcp.json"), data, 0644); err != nil {
		t.Fatalf("failed to write cursor config: %v", err)
	}

	subDir := filepath.Join(h.TempDir(), "nested", "dir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create nested test dir: %v", err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("failed to chdir to nested dir: %v", err)
	}
	defer os.Chdir(oldWD)

	err = h.Execute(RootCmd, "doctor")
	h.AssertNoError(err)
	h.AssertOutputContains("Cursor (project): configured")
	h.AssertOutputNotContains("Cursor (project): MCP not configured")
}

func TestDoctorCLIToMCPMapArchiveIncludesManageAndTask(t *testing.T) {
	m := doctorCLIToMCPMap()
	archiveTools, ok := m["task archive"]
	if !ok {
		t.Fatal("expected task archive parity mapping to exist")
	}
	if !slices.Contains(archiveTools, "tk_task") {
		t.Fatalf("expected task archive parity mapping to include tk_task, got %v", archiveTools)
	}
	if !slices.Contains(archiveTools, "tk_manage") {
		t.Fatalf("expected task archive parity mapping to include tk_manage, got %v", archiveTools)
	}
}
