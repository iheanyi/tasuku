package server

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
)

func TestServerCmdStructure(t *testing.T) {
	// Test that the command structure is correct
	if Cmd.Use != "server" {
		t.Errorf("expected Use to be 'server', got %s", Cmd.Use)
	}

	// Check aliases
	found := false
	for _, alias := range Cmd.Aliases {
		if alias == "srv" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'srv' alias")
	}

	// Check subcommands
	subcommands := Cmd.Commands()
	if len(subcommands) != 1 {
		t.Errorf("expected 1 subcommand (start), got %d", len(subcommands))
	}
}

func TestStartCmdFlags(t *testing.T) {
	h := testutil.New(t)

	// Test that flags exist
	startCmd := Cmd.Commands()[0]
	if startCmd.Use != "start" {
		t.Errorf("expected start subcommand, got %s", startCmd.Use)
	}

	// Check flags
	portFlag := startCmd.Flags().Lookup("port")
	if portFlag == nil {
		t.Error("expected --port flag")
	}

	httpFlag := startCmd.Flags().Lookup("http")
	if httpFlag == nil {
		t.Error("expected --http flag")
	}

	_ = h // harness ensures proper cleanup
}

// Note: We don't test actual server start because it blocks
// and requires MCP protocol handling. The MCP server itself
// is tested in internal/mcp/server_test.go
