// Package rules provides CLI commands for syncing learnings/decisions to editor rules.
package rules

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/rules"
	"github.com/iheanyi/tasuku/internal/store"
)

func newRulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Sync learnings and decisions to editor rules",
		Long: `Sync Tasuku learnings and decisions to editor-specific rules directories.

This enables automatic loading of learnings by Claude Code (.claude/rules/),
Cursor (.cursor/rules/), and other AI tools without manual intervention.

Detected editors (based on project files):
  - Claude Code: .claude/ directory or CLAUDE.md file
  - Cursor: .cursor/ directory or .cursorrules file
  - Codex: .codex/ directory or CODEX.md file
  - OpenCode: .opencode/ directory or opencode.json file
  - Copilot CLI: .github/hooks/ or .copilot/ directory
  - Gemini: .gemini/ directory or GEMINI.md file

Subcommands:
  sync   - Sync learnings and decisions to detected editors
  clean  - Remove Tasuku-generated rules files
  status - Show sync status and detected editors`,
	}

	cmd.AddCommand(newSyncCmd())
	cmd.AddCommand(newCleanCmd())
	cmd.AddCommand(newStatusCmd())

	return cmd
}

// Cmd is the parent command for rules operations.
var Cmd = newRulesCmd()

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync learnings and decisions to editor rules",
		Long: `Sync all Tasuku learnings and decisions to detected editor rules directories.

This creates or updates:
  - .claude/rules/tasuku/learnings.md (general learnings)
  - .claude/rules/tasuku/learnings-<scope>.md (scoped learnings with paths frontmatter)
  - .claude/rules/tasuku/decisions.md (all decisions)

Same structure for Cursor (.cursor/rules/tasuku/), Codex (.codex/rules/tasuku/),
OpenCode (.opencode/rules/tasuku/), Copilot CLI (.github/rules/tasuku/),
and Gemini (.gemini/rules/tasuku/).

Path-scoped learnings are written to separate files with YAML frontmatter
containing the 'paths' field, which editors use for conditional application.

Examples:
  tk rules sync                    # Sync to all detected editors
  tk rules sync --tool claude      # Sync only to Claude Code
  tk rules sync --tool cursor      # Sync only to Cursor
  tk rules sync --tool copilot     # Sync only to Copilot CLI
  tk learn "insight" && tk rules sync  # Add and sync`,
		RunE: runSync,
	}

	cmd.Flags().String("tool", "", "Target specific tool: claude, cursor, codex, opencode, copilot, gemini")

	return cmd
}

func newCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove Tasuku-generated rules files",
		Long: `Remove all Tasuku-generated files from editor rules directories.

This removes files from:
  - .claude/rules/tasuku/
  - .cursor/rules/tasuku/
  - .codex/rules/tasuku/
  - .opencode/rules/tasuku/
  - .github/rules/tasuku/ (Copilot CLI)
  - .gemini/rules/tasuku/

The source learnings and decisions in .tasuku/ are preserved.

Examples:
  tk rules clean                # Clean all detected editors
  tk rules clean --tool claude  # Clean only Claude Code rules
  tk rules clean --tool copilot # Clean only Copilot CLI rules`,
		RunE: runClean,
	}

	cmd.Flags().String("tool", "", "Target specific tool: claude, cursor, codex, opencode, copilot, gemini")

	return cmd
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show rules sync status",
		Long:  `Show detected editors and current sync status.`,
		RunE:  runStatus,
	}
}

func runSync(cmd *cobra.Command, args []string) error {
	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		return err
	}
	f, err := s.Read()
	if err != nil {
		return err
	}

	toolFilter, _ := cmd.Flags().GetString("tool")

	var results []rules.SyncResult
	if toolFilter != "" {
		results, err = rules.SyncToTool(f.Context.Learnings, f.Context.Decisions, toolFilter)
	} else {
		results, err = rules.Sync(f.Context.Learnings, f.Context.Decisions)
	}
	if err != nil {
		return err
	}

	for _, result := range results {
		if len(result.Errors) > 0 {
			fmt.Printf("%s: sync completed with errors\n", result.Editor)
			for _, e := range result.Errors {
				fmt.Printf("  Error: %s\n", e)
			}
		} else {
			fmt.Printf("%s: synced %d files\n", result.Editor, len(result.FilesWritten))
			for _, path := range result.FilesWritten {
				fmt.Printf("  - %s\n", path)
			}
		}
	}

	return nil
}

func runClean(cmd *cobra.Command, args []string) error {
	toolFilter, _ := cmd.Flags().GetString("tool")

	var removed []string
	var err error

	if toolFilter != "" {
		removed, err = rules.CleanTool(toolFilter)
	} else {
		removed, err = rules.Clean()
	}
	if err != nil {
		return err
	}

	if len(removed) == 0 {
		fmt.Println("No Tasuku rules files to clean.")
		return nil
	}

	fmt.Printf("Removed %d files:\n", len(removed))
	for _, path := range removed {
		fmt.Printf("  - %s\n", path)
	}
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	targets := rules.GetTargets()

	if len(targets) == 0 {
		fmt.Println("No supported editors detected.")
		fmt.Println("\nTo enable sync, create one of:")
		fmt.Println("  - .claude/ directory (for Claude Code)")
		fmt.Println("  - CLAUDE.md file (for Claude Code)")
		fmt.Println("  - .cursor/ directory (for Cursor)")
		fmt.Println("  - .cursorrules file (for Cursor)")
		return nil
	}

	fmt.Printf("Detected editors (%d):\n", len(targets))
	for _, target := range targets {
		fmt.Printf("  - %s → %s\n", target.Name, target.RulesDir)
	}

	// Show stats
	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		return err
	}
	f, err := s.Read()
	if err != nil {
		return nil // Just show targets if no store
	}

	scopedCount := 0
	for _, l := range f.Context.Learnings {
		if l.Scope != "" {
			scopedCount++
		}
	}

	fmt.Printf("\nContent to sync:\n")
	fmt.Printf("  - %d learnings (%d scoped)\n", len(f.Context.Learnings), scopedCount)
	fmt.Printf("  - %d decisions\n", len(f.Context.Decisions))

	fmt.Println("\nRun 'tk rules sync' to sync content to editors.")
	return nil
}
