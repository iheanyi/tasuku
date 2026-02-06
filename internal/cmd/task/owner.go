package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
)

func newOwnerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "owner <task-id> [owner-name]",
		Short: "Manage task ownership",
		Long: `Set, view, or clear the owner of a task.

Examples:
  tk task owner my-task agent-1      # Set owner to agent-1
  tk task owner my-task --clear      # Clear owner
  tk task owner my-task              # Show current owner`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			clearFlag, _ := cmd.Flags().GetBool("clear")
			s, err := store.DefaultStorageWithWarning()
			if err != nil {
				return err
			}

			if len(args) == 2 {
				ownerName := args[1]
				if err := s.SetOwner(taskID, ownerName); err != nil {
					return err
				}
				fmt.Printf("Set owner of %s to: %s\n", taskID, ownerName)
				return nil
			}

			if clearFlag {
				if err := s.ClearOwner(taskID); err != nil {
					return err
				}
				fmt.Printf("Cleared owner of: %s\n", taskID)
				return nil
			}

			// Show current owner
			f, err := s.Read()
			if err != nil {
				return err
			}

			t, exists := f.Tasks[taskID]
			if !exists {
				return fmt.Errorf("task not found: %s", taskID)
			}

			if t.Owner == nil {
				fmt.Printf("Task %s has no owner\n", taskID)
			} else {
				fmt.Printf("Owner of %s: %s\n", taskID, *t.Owner)
			}
			return nil
		},
	}

	cmd.Flags().Bool("clear", false, "Clear the task owner")

	return cmd
}

var ownerCmd = newOwnerCmd()
