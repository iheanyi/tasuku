package serve

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

func TestHTTPCmdFlagPrecedence(t *testing.T) {
	cmd := newHTTPCmd()

	// Set both flags
	cmd.SetArgs([]string{"--port", "8080", "--addr", ":9000"})
	if err := cmd.ParseFlags([]string{"--port", "8080", "--addr", ":9000"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	// Verify addr flag is captured
	addr, _ := cmd.Flags().GetString("addr")
	if addr != ":9000" {
		t.Errorf("expected addr ':9000', got '%s'", addr)
	}

	port, _ := cmd.Flags().GetInt("port")
	if port != 8080 {
		t.Errorf("expected port 8080, got %d", port)
	}
}

func TestHTTPCmdDefaultPort(t *testing.T) {
	cmd := newHTTPCmd()

	// No flags set - should use defaults
	port, _ := cmd.Flags().GetInt("port")
	if port != 3000 {
		t.Errorf("expected default port 3000, got %d", port)
	}

	addr, _ := cmd.Flags().GetString("addr")
	if addr != "" {
		t.Errorf("expected default addr empty, got '%s'", addr)
	}
}

func TestMCPCmdNoFlags(t *testing.T) {
	cmd := newMCPCmd()

	// MCP command should have no flags
	if cmd.Flags().HasFlags() {
		// Check if there are any non-persistent flags
		nonPersistent := false
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if !f.Hidden {
				nonPersistent = true
			}
		})
		if nonPersistent {
			t.Error("expected MCP command to have no flags")
		}
	}
}

func TestServeCmdShortDescriptions(t *testing.T) {
	serveCmd := newServeCmd()

	// Verify short descriptions are meaningful
	if serveCmd.Short == "" {
		t.Error("expected serve command to have short description")
	}

	for _, sub := range serveCmd.Commands() {
		if sub.Short == "" {
			t.Errorf("expected %s subcommand to have short description", sub.Name())
		}
	}
}

func TestHTTPCmdCustomPort(t *testing.T) {
	cmd := newHTTPCmd()

	if err := cmd.ParseFlags([]string{"--port", "4000"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	port, _ := cmd.Flags().GetInt("port")
	if port != 4000 {
		t.Errorf("expected port 4000, got %d", port)
	}
}

func TestHTTPCmdAddrOnly(t *testing.T) {
	cmd := newHTTPCmd()

	if err := cmd.ParseFlags([]string{"--addr", "localhost:5000"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	addr, _ := cmd.Flags().GetString("addr")
	if addr != "localhost:5000" {
		t.Errorf("expected addr 'localhost:5000', got '%s'", addr)
	}

	// Port should still have default
	port, _ := cmd.Flags().GetInt("port")
	if port != 3000 {
		t.Errorf("expected default port 3000, got %d", port)
	}
}
