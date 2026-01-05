package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

func newReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release <task-id>",
		Short: "Release a claimed task",
		Long: `Release a task that was previously claimed.

This clears the owner and claim timestamp, sets status back to ready,
making the task available for other agents to claim.

Use --sync to commit and push the release for multi-worktree coordination.

Examples:
  tk task release auth-feature          # Release the task
  tk task release auth-feature --sync   # Release and push to share with others`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			syncFlag, _ := cmd.Flags().GetBool("sync")
			s := store.DefaultStorageWithWarning()

			if err := s.ReleaseTask(taskID); err != nil {
				return err
			}

			// Set status back to ready (if not done/blocked)
			f, err := s.Read()
			if err == nil {
				if t, exists := f.Tasks[taskID]; exists {
					if t.Status == task.StatusInProgress {
						s.SetStatus(taskID, task.StatusReady)
					}
				}
			}

			fmt.Printf("Task %s released\n", taskID)

			if syncFlag {
				commitMsg := fmt.Sprintf("chore(tasuku): release %s", taskID)
				if err := gitCommitAndPush(commitMsg); err != nil {
					fmt.Printf("Warning: %v\n", err)
				} else {
					fmt.Println("Changes committed and pushed")
				}
			}

			return nil
		},
	}

	cmd.Flags().Bool("sync", false, "Commit and push release for multi-worktree coordination")

	return cmd
}

var releaseCmd = newReleaseCmd()
