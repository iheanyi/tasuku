package serve

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestServeCmd(t *testing.T) {
	cmd := newServeCmd()

	if cmd.Use != "serve" {
		t.Errorf("expected Use to be 'serve', got %s", cmd.Use)
	}

	// Check subcommands exist
	subcommands := cmd.Commands()
	expectedSubs := map[string]bool{
		"mcp":  false,
		"http": false,
	}

	for _, sub := range subcommands {
		if _, ok := expectedSubs[sub.Use]; ok {
			expectedSubs[sub.Use] = true
		}
	}

	for name, found := range expectedSubs {
		if !found {
			t.Errorf("expected subcommand '%s' not found", name)
		}
	}
}

func TestMCPCmd(t *testing.T) {
	cmd := newMCPCmd()

	if cmd.Use != "mcp" {
		t.Errorf("expected Use to be 'mcp', got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestHTTPCmd(t *testing.T) {
	cmd := newHTTPCmd()

	if cmd.Use != "http" {
		t.Errorf("expected Use to be 'http', got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}

	// Check flags
	portFlag := cmd.Flags().Lookup("port")
	if portFlag == nil {
		t.Error("expected --port flag")
	}
	if portFlag.DefValue != "3000" {
		t.Errorf("expected port default to be 3000, got %s", portFlag.DefValue)
	}

	addrFlag := cmd.Flags().Lookup("addr")
	if addrFlag == nil {
		t.Error("expected --addr flag")
	}
}

func TestCmdExported(t *testing.T) {
	// Ensure Cmd is properly exported
	if Cmd == nil {
		t.Error("expected Cmd to be non-nil")
	}
	if Cmd.Use != "serve" {
		t.Errorf("expected Cmd.Use to be 'serve', got %s", Cmd.Use)
	}
}

// Note: We don't test actual server start because it blocks
// Integration tests for server functionality are in the http and mcp packages

func TestNewServeCmd_ReturnsNewInstance(t *testing.T) {
	cmd1 := newServeCmd()
	cmd2 := newServeCmd()

	// Verify we get distinct instances
	if cmd1 == cmd2 {
		t.Error("expected newServeCmd to return distinct instances")
	}
}

// Helper to suppress actual command execution in tests
func suppressRunE(cmd *cobra.Command) {
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}
}
