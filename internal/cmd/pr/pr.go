// Package pr provides CLI commands for GitHub PR integration.
package pr

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

const ghInstallMessage = `GitHub CLI (gh) is not installed.

To install gh:
  macOS:   brew install gh
  Ubuntu:  sudo apt install gh
  Windows: winget install --id GitHub.cli

After installation, authenticate with:
  gh auth login

For more information: https://cli.github.com/`

// hasGhCLI checks if the GitHub CLI is installed and available.
func hasGhCLI() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

// Cmd is the parent command for PR operations.
var Cmd = &cobra.Command{
	Use:   "pr",
	Short: "GitHub Pull Request integration",
	Long: `Integrate with GitHub Pull Requests using the gh CLI.

Requires the GitHub CLI (gh) to be installed and authenticated.
If gh is not installed, commands will show installation instructions.

Subcommands:
  create    Create a new pull request
  list      List pull requests

Examples:
  tk pr create                          # Create a PR
  tk pr create --task auth-feature      # Create PR linked to a task
  tk pr create --task auth-feature --done  # Create PR and mark task done
  tk pr list                            # List open PRs
  tk pr list --state all                # List all PRs`,
}

// Flags for pr create
var (
	prTaskID   string
	prMarkDone bool
)

func init() {
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(listCmd)

	// Add flags to pr create
	createCmd.Flags().StringVar(&prTaskID, "task", "", "Task ID to link to the PR")
	createCmd.Flags().BoolVar(&prMarkDone, "done", false, "Mark the task as done after creating PR")

	// Allow unknown flags to pass through to gh
	createCmd.Flags().SetInterspersed(false)
	listCmd.Flags().SetInterspersed(false)
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new pull request",
	Long: `Create a new GitHub pull request, optionally linked to a task.

When a task is specified with --task, the task description is included
in the PR body. Use --done to automatically mark the task as complete
after the PR is created successfully.

All other flags are passed through to 'gh pr create'.

Examples:
  tk pr create                          # Create a PR
  tk pr create --task auth-feature      # Include task in PR body
  tk pr create --task auth-feature --done  # Mark task done after PR
  tk pr create --title "My PR" --body "Description"  # Pass flags to gh
  tk pr create --draft                  # Create draft PR`,
	RunE: runCreate,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests",
	Long: `List GitHub pull requests.

All flags are passed through to 'gh pr list'.

Examples:
  tk pr list                    # List open PRs
  tk pr list --state all        # List all PRs
  tk pr list --state merged     # List merged PRs
  tk pr list --author @me       # List your PRs
  tk pr list --json number,title,state  # JSON output`,
	RunE: runList,
}

func runCreate(cmd *cobra.Command, args []string) error {
	if !hasGhCLI() {
		fmt.Println(ghInstallMessage)
		return nil
	}

	// Build gh pr create command
	ghArgs := []string{"pr", "create"}

	// If a task is linked, add task info to PR body
	var hasLinkedTask bool
	var taskDescription string
	if prTaskID != "" {
		s := store.DefaultStorageWithWarning()
		f, err := s.Read()
		if err != nil {
			return fmt.Errorf("failed to read tasks: %w", err)
		}

		t, exists := f.Tasks[prTaskID]
		if !exists {
			return fmt.Errorf("task not found: %s", prTaskID)
		}
		hasLinkedTask = true
		taskDescription = t.Description

		// Build task context for PR body
		taskContext := buildTaskContext(prTaskID, t, f)

		// Check if user already provided --body flag
		hasBody := false
		for _, arg := range args {
			if strings.HasPrefix(arg, "--body") || arg == "-b" {
				hasBody = true
				break
			}
		}

		if !hasBody {
			ghArgs = append(ghArgs, "--body", taskContext)
		}
	}

	// Pass through remaining args to gh
	ghArgs = append(ghArgs, args...)

	// Execute gh pr create
	ghCmd := exec.Command("gh", ghArgs...)
	ghCmd.Stdin = os.Stdin
	ghCmd.Stdout = os.Stdout
	ghCmd.Stderr = os.Stderr

	err := ghCmd.Run()
	if err != nil {
		return fmt.Errorf("gh pr create failed: %w", err)
	}

	// If task was linked and --done flag was set, mark task as done
	if hasLinkedTask && prMarkDone {
		s := store.DefaultStorageWithWarning()
		if err := s.SetStatus(prTaskID, task.StatusDone); err != nil {
			fmt.Printf("Warning: PR created but failed to mark task %s as done: %v\n", prTaskID, err)
		} else {
			fmt.Printf("Marked task %s as done\n", prTaskID)
		}
	} else if hasLinkedTask {
		fmt.Printf("Linked to task: %s\n", taskDescription)
	}

	return nil
}

// buildTaskContext creates a formatted string for the PR body with task information.
func buildTaskContext(taskID string, t task.Task, f *task.File) string {
	var buf bytes.Buffer

	buf.WriteString("## Task\n\n")
	buf.WriteString(fmt.Sprintf("**ID:** %s\n", taskID))
	buf.WriteString(fmt.Sprintf("**Description:** %s\n", t.Description))
	buf.WriteString(fmt.Sprintf("**Status:** %s\n", t.Status))

	if t.Priority != nil {
		buf.WriteString(fmt.Sprintf("**Priority:** %s\n", task.PriorityName(*t.Priority)))
	}

	if len(t.Tags) > 0 {
		buf.WriteString(fmt.Sprintf("**Tags:** %s\n", strings.Join(t.Tags, ", ")))
	}

	// Add notes if any exist
	if notes, exists := f.Context.Notes[taskID]; exists && len(notes) > 0 {
		buf.WriteString("\n### Notes\n\n")
		for _, note := range notes {
			buf.WriteString(fmt.Sprintf("- %s\n", note.Text))
		}
	}

	buf.WriteString("\n---\n")
	buf.WriteString("*Created with [Tasuku](https://github.com/iheanyi/tasuku)*\n")

	return buf.String()
}

func runList(cmd *cobra.Command, args []string) error {
	if !hasGhCLI() {
		fmt.Println(ghInstallMessage)
		return nil
	}

	// Build gh pr list command
	ghArgs := []string{"pr", "list"}
	ghArgs = append(ghArgs, args...)

	// Execute gh pr list
	ghCmd := exec.Command("gh", ghArgs...)
	ghCmd.Stdin = os.Stdin
	ghCmd.Stdout = os.Stdout
	ghCmd.Stderr = os.Stderr

	return ghCmd.Run()
}
