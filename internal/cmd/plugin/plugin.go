// Package plugin provides CLI commands for managing Tasuku plugins/skills.
package plugin

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/cmdutil"
	"github.com/iheanyi/tasuku/internal/plugin"
)

func newPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage Tasuku plugin/skills installation",
		Long: `Install and manage Tasuku guided workflows (plugins/skills) for AI tools.

Different AI tools use different formats:
  - Claude Code: Plugins with /tasuku:* slash commands
  - Cursor: Commands in .cursor/commands/tasuku/
  - Copilot CLI: Skills in .github/skills/tasuku/
  - Codex: Skills in .codex/skills/tasuku/

This command provides a unified way to install Tasuku's guided workflows
to any supported tool.

Subcommands:
  install   - Install Tasuku plugin/skills to detected tools
  uninstall - Remove Tasuku plugin/skills
  list      - List available commands/skills
  status    - Show installation status`,
	}

	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newUninstallCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newStatusCmd())

	return cmd
}

// Cmd is the parent command for plugin operations.
var Cmd = newPluginCmd()

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Tasuku plugin/skills to AI tools",
		Long: `Install Tasuku guided workflows to detected AI tools.

By default, installs to all detected tools. Use --tool to target specific tools.

For Claude Code, this guides you to use the native plugin system.
For Copilot CLI and Codex, this installs SKILL.md files.

Examples:
  tk plugin install                    # Install to all detected tools
  tk plugin install --tool copilot     # Install to Copilot CLI only
  tk plugin install --tool codex       # Install to Codex only
  tk plugin install --local            # Install locally instead of globally`,
		RunE: runInstall,
	}

	cmd.Flags().String("tool", "", "Target specific tool: claude, cursor, copilot, codex")
	cmd.Flags().Bool("local", false, "Install to project-local directory")

	return cmd
}

func runInstall(cmd *cobra.Command, args []string) error {
	tool, _ := cmd.Flags().GetString("tool")
	local, _ := cmd.Flags().GetBool("local")

	var targets []plugin.ToolTarget

	if tool != "" {
		t := plugin.GetToolByName(tool)
		if t == nil {
			return fmt.Errorf("unknown tool: %s (valid: claude, cursor, copilot, codex)", tool)
		}
		targets = append(targets, *t)
	} else {
		targets = plugin.GetDetectedTools()
		if len(targets) == 0 {
			fmt.Println("No supported AI tools detected.")
			fmt.Println("\nSupported tools and their detection signals:")
			for _, t := range plugin.GetSupportedTools() {
				fmt.Printf("  - %s: %s\n", t.Name, strings.Join(t.DetectFiles, " or "))
			}
			return nil
		}
	}

	var hasErrors bool
	for _, target := range targets {
		fmt.Printf("\n%s:\n", target.Name)
		result := plugin.InstallToTool(target, local)

		if len(result.Errors) > 0 {
			for _, err := range result.Errors {
				fmt.Printf("  %s\n", err)
			}
			if len(result.FilesAdded) == 0 {
				hasErrors = true
				continue
			}
		}

		if len(result.FilesAdded) > 0 {
			fmt.Printf("  Installed %d skill(s):\n", len(result.FilesAdded))
			for _, f := range result.FilesAdded[:min(5, len(result.FilesAdded))] {
				fmt.Printf("    - %s\n", f)
			}
			if len(result.FilesAdded) > 5 {
				fmt.Printf("    ... and %d more\n", len(result.FilesAdded)-5)
			}
		}
	}

	if hasErrors {
		return fmt.Errorf("some installations failed")
	}

	return nil
}

func newUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove Tasuku plugin/skills from AI tools",
		Long: `Remove Tasuku guided workflows from AI tools.

Examples:
  tk plugin uninstall                  # Uninstall from all detected tools
  tk plugin uninstall --tool copilot   # Uninstall from Copilot CLI only
  tk plugin uninstall --local          # Remove from local directory`,
		RunE: runUninstall,
	}

	cmd.Flags().String("tool", "", "Target specific tool: claude, cursor, copilot, codex")
	cmd.Flags().Bool("local", false, "Uninstall from project-local directory")

	return cmd
}

func runUninstall(cmd *cobra.Command, args []string) error {
	tool, _ := cmd.Flags().GetString("tool")
	local, _ := cmd.Flags().GetBool("local")

	var targets []plugin.ToolTarget

	if tool != "" {
		t := plugin.GetToolByName(tool)
		if t == nil {
			return fmt.Errorf("unknown tool: %s (valid: claude, cursor, copilot, codex)", tool)
		}
		targets = append(targets, *t)
	} else {
		targets = plugin.GetDetectedTools()
		if len(targets) == 0 {
			fmt.Println("No supported AI tools detected.")
			return nil
		}
	}

	for _, target := range targets {
		fmt.Printf("\n%s:\n", target.Name)
		result := plugin.UninstallFromTool(target, local)

		if len(result.Errors) > 0 {
			for _, err := range result.Errors {
				fmt.Printf("  %s\n", err)
			}
		}

		if result.AlreadyDone {
			fmt.Println("  Already uninstalled (no files found)")
		} else if len(result.FilesAdded) > 0 {
			fmt.Printf("  Removed %d file(s)\n", len(result.FilesAdded))
		}
	}

	return nil
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available Tasuku commands/skills",
		Long: `List all available Tasuku commands that can be installed as plugins/skills.

These commands provide guided workflows for task management:
  - Workflow commands (pickup, complete, reflect) guide multi-step processes
  - Basic commands (add, list, done) provide direct task operations`,
		RunE: runList,
	}
}

func runList(cmd *cobra.Command, args []string) error {
	commands, err := plugin.LoadEmbeddedCommands()
	if err != nil {
		return fmt.Errorf("failed to load commands: %w", err)
	}

	// Group by type
	workflow := []plugin.Command{}
	basic := []plugin.Command{}

	for _, c := range commands {
		switch c.Name {
		case "pickup", "complete", "reflect", "help", "tasuku":
			workflow = append(workflow, c)
		default:
			basic = append(basic, c)
		}
	}

	fmt.Println("Workflow Commands (Recommended):")
	for _, c := range workflow {
		fmt.Printf("  %-12s - %s\n", c.Name, cmdutil.Truncate(c.Description, 60))
	}

	fmt.Println("\nBasic Commands:")
	for _, c := range basic {
		fmt.Printf("  %-12s - %s\n", c.Name, cmdutil.Truncate(c.Description, 60))
	}

	fmt.Printf("\nTotal: %d commands\n", len(commands))
	return nil
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show plugin installation status",
		Long: `Show which AI tools are detected and whether Tasuku is installed.

Checks both global and local installation directories.`,
		RunE: runStatus,
	}
}

func runStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("Detected AI Tools:")

	detected := plugin.GetDetectedTools()
	if len(detected) == 0 {
		fmt.Println("  None detected")
		fmt.Println("\nSupported tools and their detection signals:")
		for _, t := range plugin.GetSupportedTools() {
			fmt.Printf("  - %s: %s\n", t.Name, strings.Join(t.DetectFiles, " or "))
		}
		return nil
	}

	for _, tool := range detected {
		fmt.Printf("\n%s:\n", tool.Name)

		// Check local installation
		localInstalled := checkInstalled(tool.LocalDir)
		globalInstalled := checkInstalled(tool.GlobalDir)

		if localInstalled {
			fmt.Printf("  Local:  Installed (%s)\n", tool.LocalDir)
		} else {
			fmt.Printf("  Local:  Not installed\n")
		}

		if globalInstalled {
			fmt.Printf("  Global: Installed (%s)\n", tool.GlobalDir)
		} else {
			fmt.Printf("  Global: Not installed\n")
		}
	}

	return nil
}

// Helper functions

func checkInstalled(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return false
	}

	// Check if there are any .md files
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
