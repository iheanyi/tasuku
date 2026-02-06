package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
)

var showCmd = &cobra.Command{
	Use:   "show <task-id>",
	Short: "Show task details",
	Long: `Display detailed information about a specific task.

Information Shown:
  - Task ID and description
  - Current status (ready, in_progress, blocked, done)
  - Priority level
  - Owner (if assigned)
  - Blocked by (list of blocking task IDs)
  - Created and updated timestamps
  - Notes attached to the task

Examples:
  tk task show my-task            # Show details for "my-task"
  tk task show fix-auth -f json   # Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]

		s, err := store.DefaultStorageWithWarning()
		if err != nil {
			return err
		}
		f, err := s.Read()
		if err != nil {
			return err
		}

		t, exists := f.Tasks[taskID]
		if !exists {
			return fmt.Errorf("task not found: %s", taskID)
		}

		notes := f.Context.Notes[taskID]
		return outputTaskDetail(taskID, t, notes, f.Tasks)
	},
}
