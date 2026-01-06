package skills

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Cmd is the root skills command
var Cmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage Claude Code skills for Tasuku",
	Long: `Install and manage Claude Code slash command skills for Tasuku.

Skills provide quick access to common Tasuku operations via slash commands
like /tasuku:list, /tasuku:ready, /tasuku:start, etc.`,
}

func init() {
	Cmd.AddCommand(installCmd)
	Cmd.AddCommand(uninstallCmd)
	Cmd.AddCommand(listCmd)
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Tasuku skills to current project or globally",
	Long: `Install Tasuku slash command skills to make them available in Claude Code.

By default, installs to the current project's .claude/skills directory.
Use --global to install to ~/.claude/skills for all projects.

This enables commands like:
  /tasuku-add     - Create a new task
  /tasuku-list    - List all tasks
  /tasuku-ready   - Show ready tasks
  /tasuku-start   - Start working on a task
  /tasuku-done    - Mark a task complete
  /tasuku-learn   - Record learnings
  /tasuku-context - Get full project context
  /tasuku-stats   - Show task statistics`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		global, _ := cmd.Flags().GetBool("global")
		return installSkills(force, global)
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove Tasuku skills from current project or globally",
	Long: `Remove Tasuku skills from Claude Code.

By default, removes from the current project's .claude/skills directory.
Use --global to remove from ~/.claude/skills.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		global, _ := cmd.Flags().GetBool("global")
		return uninstallSkills(global)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available Tasuku skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		listSkills()
		return nil
	},
}

func init() {
	installCmd.Flags().Bool("force", false, "Overwrite existing skills")
	installCmd.Flags().Bool("global", false, "Install to ~/.claude/skills for all projects")
	uninstallCmd.Flags().Bool("global", false, "Remove from ~/.claude/skills")
}

// Skill definitions embedded in the binary
var skillDefinitions = map[string]string{
	"list": `---
name: list
description: List all tasks with optional status filtering. Use when user wants to see tasks, check project status, or get an overview.
---

# List Tasks

Display all tasks in the project, optionally filtered by status.

## Usage

Run the ` + "`tk task list`" + ` command to show all tasks:

` + "```bash" + `
tk task list                    # All tasks
tk task list --status ready     # Only ready tasks
tk task list --status in_progress  # Only in-progress tasks
tk task list --status blocked   # Only blocked tasks
tk task list --status done      # Only completed tasks
tk task list --tree             # Hierarchical view with subtasks
tk task list --format json      # Output as JSON
` + "```" + `

## Output Format

Tasks are displayed with:
- Status symbol: ` + "`[-]`" + ` ready, ` + "`[*]`" + ` in_progress, ` + "`[!]`" + ` blocked, ` + "`[x]`" + ` done
- Task ID
- Description
- Blockers (if any)

## When to Use

- Starting a session to see what needs to be done
- Checking overall project progress
- Finding tasks by status
- Reviewing blocked tasks
`,

	"ready": `---
name: ready
description: Show tasks ready to work on, sorted by priority. Use when user wants to pick up new work or see what's available.
---

# Ready Tasks

Show all tasks that are ready to be worked on, sorted by priority.

## Usage

` + "```bash" + `
tk task ready                   # Show ready tasks sorted by priority
tk task ready --format json     # Output as JSON
` + "```" + `

## Priority Order

Tasks are sorted by priority level:
- **Critical (0)**: Urgent, blocking issues
- **High (1)**: Important, do soon
- **Normal (2)**: Default priority
- **Low (3)**: Can wait
- **Backlog (4)**: Future work

## When to Use

- Looking for the next task to work on
- Starting a new work session
- Checking what's available after completing a task

## Best Practice

After viewing ready tasks, use ` + "`tk task start <id>`" + ` to begin work on one.
`,

	"context": `---
name: context
description: Get full project context including tasks, learnings, and decisions. Use at session start or when needing complete project state.
---

# Project Context

Load the complete project context including all tasks, learnings, and decisions.

## Usage

` + "```bash" + `
tk context show                 # Full context as JSON
tk context show --format yaml   # Full context as YAML
` + "```" + `

## What's Included

The context contains:

### Tasks
- All active tasks with status, priority, and blockers
- Subtask relationships
- Owner and claim information

### Learnings
- Insights discovered while working
- "Never do X" and "Always use Y" patterns
- Codebase-specific knowledge

### Decisions
- Architectural choices made
- Alternatives that were considered
- Reasoning behind each decision

## When to Use

- Starting a new session to understand project state
- Onboarding to an unfamiliar project
- Before making decisions that might duplicate past work
- Understanding why things were built a certain way
`,

	"stats": `---
name: stats
description: Show project statistics and progress. Use when user wants metrics, completion status, or progress overview.
---

# Task Statistics

Display project statistics including task counts, completion rates, and progress.

## Usage

` + "```bash" + `
tk task stats                   # Show statistics
tk task stats --format json     # Output as JSON
` + "```" + `

## Metrics Displayed

- Total task count
- Tasks by status (ready, in_progress, blocked, done)
- Completion percentage
- Blocked task count

## When to Use

- Getting a quick project health check
- Reporting progress to stakeholders
- Understanding project velocity
- Identifying bottlenecks (high blocked count)

## Interpreting Results

- High blocked count: Dependencies need resolution
- Many in_progress: Consider focusing on completion
- Low ready count: May need to plan next tasks
`,

	"start": `---
name: start
description: Start working on a task. Use when beginning work, picking up a new task, or resuming work.
---

# Start Task

Begin working on a task by marking it as in_progress.

## Usage

` + "```bash" + `
tk task start <task-id>                  # Start a task
tk task start <task-id> --timer          # Start task and begin timing
tk task start <task-id> --unblock        # Clear blockers and start
tk task start <task-id> --timer --unblock  # All options combined
` + "```" + `

## Flags

- ` + "`--timer`" + `: Also start a time tracking timer on the task
- ` + "`--unblock`" + `: Clear any blockers before starting (for blocked tasks)

## When to Use

- Picking up a new task from the ready list
- Resuming work on a task
- Claiming a task to indicate active work

## Best Practices

1. Only have one task in_progress at a time for focus
2. Use ` + "`--timer`" + ` if tracking time spent
3. Use ` + "`tk task pause <id>`" + ` if you need to switch tasks

## After Starting

- Record learnings with ` + "`tk learn \"insight\"`" + `
- Add notes with ` + "`tk note add <id> \"note\"`" + `
- When done, use ` + "`tk task done <id>`" + `
`,

	"done": `---
name: done
description: Mark a task as completed. Use when finishing work, completing a feature, or closing out a task.
---

# Complete Task

Mark a task as done when work is finished.

## Usage

` + "```bash" + `
tk task done <task-id>          # Mark task as complete
` + "```" + `

## Automatic Behavior

- If a timer is running on the task, it will be automatically stopped
- The elapsed time is added to the task's total duration

## When to Use

- Finishing implementation of a feature
- Completing a bug fix
- Closing out any piece of work

## Best Practices

1. Verify the work is actually complete before marking done
2. Record any learnings discovered during the work
3. Check if completing this task unblocks others

## After Completing

Consider:
- Recording learnings: ` + "`tk learn \"what you discovered\"`" + `
- Archiving if no longer needed: ` + "`tk task archive add <id>`" + `
- Starting the next ready task: ` + "`tk task ready`" + `
`,

	"add": `---
name: add
description: Create a new task. Use when user wants to add work items, create todos, or break down features into tasks.
---

# Add Task

Create a new task in Tasuku.

## Usage

` + "```bash" + `
tk task add "Task description"              # Create with auto-generated ID
tk task add "Task description" --id my-id   # Create with custom ID
tk task add "Subtask" --parent parent-id    # Create as subtask
tk task add "Urgent fix" --priority high    # Create with priority
` + "```" + `

## Options

- ` + "`--id`" + `: Custom task ID (otherwise auto-generated from description)
- ` + "`--parent`" + `: Parent task ID to create as subtask
- ` + "`--priority`" + `: Priority level (critical, high, normal, low, backlog)

## Priority Levels

| Level | Name | When to Use |
|-------|------|-------------|
| 0 | critical | Blocking issues, urgent bugs |
| 1 | high | Important, do soon |
| 2 | normal | Default priority |
| 3 | low | Can wait |
| 4 | backlog | Future work, ideas |

## When to Use

- Breaking down a feature into subtasks
- Capturing new work items
- Creating follow-up tasks during implementation
- Adding bugs or issues discovered during work

## Best Practices

1. Use descriptive task names that explain the goal
2. Use ` + "`--parent`" + ` to organize related work as subtasks
3. Set priority based on urgency and importance
4. Consider using ` + "`tk task start <id>`" + ` immediately if starting work
`,

	"learn": `---
name: learn
description: Record a learning or insight. Use when discovering important patterns, gotchas, or knowledge that should be remembered.
---

# Record Learning

Capture insights discovered while working on the project.

## Usage

` + "```bash" + `
tk learn "The API rate limits to 100 req/min"     # Add a learning
tk learn "Never use sync calls in handlers"       # Record a "never do" pattern
tk learn "Always validate input before DB write"  # Record an "always do" pattern
tk learnings                                      # List all learnings
` + "```" + `

## What to Record

Good learnings include:

- **API behaviors**: "The auth endpoint returns 401 for expired tokens, not 403"
- **Code patterns**: "Use the ` + "`withRetry`" + ` wrapper for all external API calls"
- **Gotchas**: "The config file must be loaded before initializing the logger"
- **Performance**: "Batch inserts are 10x faster than individual inserts"
- **Never/Always rules**: "Never store passwords in plain text"

## When to Use

- After debugging a tricky issue
- When discovering undocumented behavior
- After making a decision that future work should know about
- When finding a pattern that works well (or poorly)

## Promoting Learnings

For learnings that should be permanent documentation:

` + "```bash" + `
tk promote 1                    # Promote learning #1 to context file
tk promote 1 --to CLAUDE.md     # Promote to specific file
tk promote 1 --keep             # Keep in learnings after promoting
` + "```" + `

## Best Practices

1. Be specific - include context that makes the learning actionable
2. Use "Never" or "Always" prefixes for rules
3. Promote important learnings to permanent docs
4. Review learnings periodically to refresh memory
`,
}

func getSkillsBaseDir(global bool) (string, error) {
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(home, ".claude", "skills"), nil
	}
	return filepath.Join(".claude", "skills"), nil
}

func installSkills(force, global bool) error {
	baseDir, err := getSkillsBaseDir(global)
	if err != nil {
		return err
	}

	location := "project"
	if global {
		location = "global"
	}

	// Create each skill in its own directory with SKILL.md
	installed := 0
	for name, content := range skillDefinitions {
		skillDir := filepath.Join(baseDir, "tasuku-"+name)

		// Check if already exists
		if _, err := os.Stat(skillDir); err == nil && !force {
			fmt.Printf("Skill /tasuku-%s already installed (use --force to overwrite)\n", name)
			continue
		}

		// Create directory
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return fmt.Errorf("failed to create skill directory %s: %w", name, err)
		}

		// Write SKILL.md
		skillPath := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write skill %s: %w", name, err)
		}
		installed++
	}

	// Also create main tasuku skill as an overview
	mainSkillDir := filepath.Join(baseDir, "tasuku")
	if _, err := os.Stat(mainSkillDir); err == nil && !force {
		fmt.Println("Main /tasuku skill already installed (use --force to overwrite)")
	} else {
		if err := os.MkdirAll(mainSkillDir, 0755); err != nil {
			return fmt.Errorf("failed to create main skill directory: %w", err)
		}

		mainSkill := `---
name: tasuku
description: Task management for AI agents. Use /tasuku for an overview or specific skills like /tasuku-list, /tasuku-add, /tasuku-start, /tasuku-done.
---

# Tasuku Task Management

Tasuku is an agent-first task management system. Manage tasks, track learnings, and coordinate agents.

## Quick Reference

` + "```bash" + `
tk task add "description" # Create a task
tk task list              # See all tasks
tk task ready             # What can I work on?
tk task start <id>        # Begin work (add --timer to track time)
tk task done <id>         # Complete task (auto-stops timer)
tk task pause <id>        # Pause work (auto-stops timer)
tk learn "insight"        # Record learning
tk context show           # Full context
tk task stats             # Project statistics
` + "```" + `

## Available Skills

Use specific skills for detailed guidance:

- **/tasuku-add** - Create a new task
- **/tasuku-list** - List all tasks with optional filtering
- **/tasuku-ready** - Show tasks ready to work on
- **/tasuku-start** - Start working on a task
- **/tasuku-done** - Mark a task complete
- **/tasuku-learn** - Record learnings and insights
- **/tasuku-context** - Get full project context
- **/tasuku-stats** - Show task statistics

## Task Lifecycle

1. ` + "`tk task add \"description\"`" + ` - Create task
2. ` + "`tk task start <id>`" + ` - Begin work
3. ` + "`tk learn \"insight\"`" + ` - Record learnings
4. ` + "`tk task done <id>`" + ` - Complete task

## Time Tracking

- ` + "`tk task start <id> --timer`" + ` - Start with timer
- ` + "`tk task timer status`" + ` - See running timers
- Timers auto-stop on ` + "`done`" + ` or ` + "`pause`" + `
`

		if err := os.WriteFile(filepath.Join(mainSkillDir, "SKILL.md"), []byte(mainSkill), 0644); err != nil {
			return fmt.Errorf("failed to write main skill: %w", err)
		}
		installed++
	}

	if installed == 0 {
		fmt.Println("All skills already installed.")
		return nil
	}

	fmt.Printf("Installed %d Tasuku skill(s) to %s (%s)\n", installed, baseDir, location)
	fmt.Println("\nAvailable slash commands:")
	fmt.Println("  /tasuku         - Overview and quick reference")
	fmt.Println("  /tasuku-add     - Create a new task")
	fmt.Println("  /tasuku-list    - List all tasks")
	fmt.Println("  /tasuku-ready   - Show ready tasks")
	fmt.Println("  /tasuku-start   - Start a task")
	fmt.Println("  /tasuku-done    - Complete a task")
	fmt.Println("  /tasuku-learn   - Record learnings")
	fmt.Println("  /tasuku-context - Get full context")
	fmt.Println("  /tasuku-stats   - Show statistics")
	fmt.Println("\nRestart Claude Code for skills to take effect.")

	return nil
}

func uninstallSkills(global bool) error {
	baseDir, err := getSkillsBaseDir(global)
	if err != nil {
		return err
	}

	location := "project"
	if global {
		location = "global"
	}

	removed := 0

	// Remove individual skill directories
	for name := range skillDefinitions {
		skillDir := filepath.Join(baseDir, "tasuku-"+name)
		if _, err := os.Stat(skillDir); err == nil {
			if err := os.RemoveAll(skillDir); err != nil {
				return fmt.Errorf("failed to remove skill %s: %w", name, err)
			}
			removed++
		}
	}

	// Remove main tasuku skill
	mainSkillDir := filepath.Join(baseDir, "tasuku")
	if _, err := os.Stat(mainSkillDir); err == nil {
		if err := os.RemoveAll(mainSkillDir); err != nil {
			return fmt.Errorf("failed to remove main skill: %w", err)
		}
		removed++
	}

	if removed == 0 {
		fmt.Println("No Tasuku skills installed.")
		return nil
	}

	fmt.Printf("Removed %d Tasuku skill(s) from %s (%s)\n", removed, baseDir, location)
	return nil
}

func listSkills() {
	fmt.Println("Tasuku Skills:")
	fmt.Println()
	for name, content := range skillDefinitions {
		// Extract description from content
		desc := ""
		lines := []byte(content)
		inFrontmatter := false
		for _, line := range splitLines(string(lines)) {
			if line == "---" {
				if inFrontmatter {
					break
				}
				inFrontmatter = true
				continue
			}
			if inFrontmatter && len(line) > 12 && line[:12] == "description:" {
				desc = line[13:]
				break
			}
		}
		fmt.Printf("  /tasuku:%s\n", name)
		fmt.Printf("    %s\n\n", desc)
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
