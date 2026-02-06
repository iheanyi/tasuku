package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

func newClaimCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim <task-id> [agent-name]",
		Short: "Claim a task for an agent",
		Long: `Claim a task for exclusive work by an agent.

This sets the task's owner, marks it as in_progress, and records a claim
timestamp. If another agent tries to claim the same task, they will be
rejected unless the claim is stale (older than 2 hours by default).

If agent-name is not provided, it's auto-detected from:
  1. TASUKU_AGENT environment variable
  2. Git user.name
  3. System username
  4. Hostname

Use --sync to commit and push the claim for multi-worktree coordination.

Examples:
  tk task claim auth-feature              # Claim with auto-detected name
  tk task claim auth-feature agent-1      # Claim for specific agent
  tk task claim auth-feature --sync       # Claim and push to share with others`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			agentName := ""
			if len(args) == 2 {
				agentName = args[1]
			} else {
				agentName = getAgentName()
			}

			syncFlag, _ := cmd.Flags().GetBool("sync")
			s, err := store.DefaultStorageWithWarning()
			if err != nil {
				return err
			}

			if err := s.ClaimTask(taskID, agentName); err != nil {
				return err
			}

			if err := s.SetStatus(taskID, task.StatusInProgress); err != nil {
				return err
			}

			fmt.Printf("Task %s claimed by %s\n", taskID, agentName)

			if syncFlag {
				commitMsg := fmt.Sprintf("chore(tasuku): claim %s for %s", taskID, agentName)
				if err := gitCommitAndPush(commitMsg); err != nil {
					fmt.Printf("Warning: %v\n", err)
				} else {
					fmt.Println("Changes committed and pushed")
				}
			}

			return nil
		},
	}

	cmd.Flags().Bool("sync", false, "Commit and push claim for multi-worktree coordination")

	return cmd
}

var claimCmd = newClaimCmd()
