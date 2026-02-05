package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildMCPInstallCommand(t *testing.T) {
	tests := []struct {
		tool     string
		local    bool
		force    bool
		expected string
	}{
		{"", false, false, "tk mcp install"},
		{"claude", false, false, "tk mcp install --tool claude"},
		{"", true, false, "tk mcp install --local"},
		{"", false, true, "tk mcp install --force"},
		{"cursor", true, true, "tk mcp install --tool cursor --local --force"},
	}

	for _, tt := range tests {
		got := buildMCPInstallCommand(tt.tool, tt.local, tt.force)
		if got != tt.expected {
			t.Errorf("buildMCPInstallCommand(%q, %v, %v) = %q, want %q",
				tt.tool, tt.local, tt.force, got, tt.expected)
		}
	}
}

func TestCheckPluginInstalled(t *testing.T) {
	// Test with non-existent directory
	if checkPluginInstalled("/nonexistent/path") {
		t.Error("expected false for non-existent directory")
	}

	// Test with empty directory
	emptyDir := t.TempDir()
	if checkPluginInstalled(emptyDir) {
		t.Error("expected false for empty directory")
	}

	// Test with directory containing non-md files
	nonMdDir := t.TempDir()
	os.WriteFile(filepath.Join(nonMdDir, "test.txt"), []byte("test"), 0644)
	if checkPluginInstalled(nonMdDir) {
		t.Error("expected false for directory with no .md files")
	}

	// Test with directory containing .md files
	mdDir := t.TempDir()
	os.WriteFile(filepath.Join(mdDir, "skill.md"), []byte("test"), 0644)
	if !checkPluginInstalled(mdDir) {
		t.Error("expected true for directory with .md files")
	}

	// Test with file (not directory)
	file := filepath.Join(t.TempDir(), "file.txt")
	os.WriteFile(file, []byte("test"), 0644)
	if checkPluginInstalled(file) {
		t.Error("expected false for file (not directory)")
	}
}
