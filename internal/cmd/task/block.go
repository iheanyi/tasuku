package task

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
)

func newBlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block <task-id>",
		Short: "Mark task as blocked",
		Long: `Mark a task as blocked by one or more other tasks.

Use the --by flag to specify which tasks are blocking.

Examples:
  tk task block my-task --by other-task
  tk task block my-task --by task-a,task-b`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			blockers, _ := cmd.Flags().GetStringSlice("by")
			s := store.DefaultStorageWithWarning()

			if err := s.BlockTask(taskID, blockers); err != nil {
				return err
			}

			fmt.Printf("Blocked: %s (by: %s)\n", taskID, strings.Join(blockers, ", "))
			return nil
		},
	}

	cmd.Flags().StringSlice("by", nil, "Blocking task IDs (repeatable or comma-separated)")
	cmd.MarkFlagRequired("by")

	return cmd
}

var blockCmd = newBlockCmd()
