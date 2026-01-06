package task

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
)

func newBlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block <task-id> [task-id...]",
		Short: "Mark task(s) as blocked",
		Long: `Mark one or more tasks as blocked by other tasks.

Use the --by flag to specify which tasks are blocking.

Examples:
  tk task block my-task --by other-task
  tk task block my-task --by task-a,task-b
  tk task block task-1 task-2 --by blocker     # Block multiple tasks`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			blockers, _ := cmd.Flags().GetStringSlice("by")
			s := store.DefaultStorageWithWarning()

			for _, taskID := range args {
				if err := s.BlockTask(taskID, blockers); err != nil {
					return fmt.Errorf("%s: %w", taskID, err)
				}
				fmt.Printf("Blocked: %s (by: %s)\n", taskID, strings.Join(blockers, ", "))
			}
			return nil
		},
	}

	cmd.Flags().StringSlice("by", nil, "Blocking task IDs (repeatable or comma-separated)")
	cmd.MarkFlagRequired("by")

	return cmd
}

var blockCmd = newBlockCmd()
