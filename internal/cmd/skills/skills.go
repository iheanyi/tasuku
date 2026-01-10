package skills

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Cmd is the root skills command
var Cmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage Claude Code skills for Tasuku (DEPRECATED - use plugin instead)",
	Long: `DEPRECATED: Skills have been replaced by plugins.

Tasuku now uses the Claude Code plugin system for slash commands.
Plugins provide better organization and namespaced commands like /tasuku:add.

To install the Tasuku plugin:

1. Add the marketplace (one-time setup):
   In Claude Code, run:
   /plugin add-marketplace /path/to/tasuku

   Or if installed via go install:
   /plugin add-marketplace https://github.com/iheanyi/tasuku

2. Install the plugin:
   /plugin install tasuku

This provides all commands: /tasuku:add, /tasuku:list, /tasuku:start, etc.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(`DEPRECATED: Skills have been replaced by plugins.

To install the Tasuku plugin in Claude Code:

1. Add the marketplace:
   /plugin add-marketplace https://github.com/iheanyi/tasuku

2. Install the plugin:
   /plugin install tasuku

This provides all /tasuku:* commands.`)
		return nil
	},
}

func init() {
	Cmd.AddCommand(installCmd)
	Cmd.AddCommand(uninstallCmd)
	Cmd.AddCommand(listCmd)
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "DEPRECATED - use /plugin install tasuku in Claude Code",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(`DEPRECATED: tk skills install is no longer supported.

The Tasuku plugin provides slash commands through Claude Code's plugin system.

To install:

1. In Claude Code, add the marketplace:
   /plugin add-marketplace https://github.com/iheanyi/tasuku

2. Install the plugin:
   /plugin install tasuku

This provides all commands:
  /tasuku:add     - Create a new task
  /tasuku:list    - List all tasks
  /tasuku:ready   - Show ready tasks
  /tasuku:start   - Start a task
  /tasuku:done    - Complete a task
  /tasuku:learn   - Record learnings
  /tasuku:context - Get full context
  /tasuku:stats   - Show statistics
  ... and more!`)
		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "DEPRECATED - use /plugin uninstall tasuku in Claude Code",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(`DEPRECATED: tk skills uninstall is no longer supported.

To uninstall the Tasuku plugin in Claude Code:
  /plugin uninstall tasuku`)
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available Tasuku plugin commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(`Tasuku Plugin Commands:

Workflow Commands (Recommended):
  /tasuku:pickup    - Guided task selection and start
  /tasuku:complete  - Guided completion with learning capture
  /tasuku:reflect   - Guided learning extraction
  /tasuku:help      - Complete command reference

Basic Commands:
  /tasuku:add       - Create a new task
  /tasuku:list      - List all tasks
  /tasuku:ready     - Show tasks ready to work on
  /tasuku:start     - Begin working on a task
  /tasuku:done      - Mark task complete
  /tasuku:learn     - Record learnings
  /tasuku:decide    - Record decisions
  /tasuku:note      - Add notes to tasks
  /tasuku:show      - View task details
  /tasuku:block     - Mark task as blocked
  /tasuku:context   - Get full project context
  /tasuku:stats     - Show task statistics
  /tasuku:promote   - Promote learnings to docs

To install these commands, run in Claude Code:
  /plugin add-marketplace https://github.com/iheanyi/tasuku
  /plugin install tasuku`)
		return nil
	},
}
