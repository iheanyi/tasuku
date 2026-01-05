package task

import (
	"sort"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all tasks",
	Long: `Display all tasks in the project, sorted by status and priority.

Status Icons:
  *  in_progress - Currently being worked on
  -  ready       - Available to start
  !  blocked     - Waiting on other tasks
  x  done        - Completed

Sort Order:
  1. Status: in_progress > ready > blocked > done
  2. Priority: critical (0) > high (1) > normal (2) > low (3) > backlog (4)
  3. Task ID: alphabetically

Filtering:
  Use --status to show only tasks with a specific status.
  Use --tag to show only tasks with a specific tag.

Tree View:
  Use --tree to show tasks in a hierarchical tree format,
  with subtasks indented under their parent tasks.

Examples:
  tk task list                 # List all tasks
  tk task list -s ready        # Show only ready tasks
  tk task list --status done   # Show completed tasks
  tk task list --tag backend   # Show tasks with 'backend' tag
  tk task list -t bug -s ready # Combine filters
  tk task list -f json         # Output as JSON
  tk task list --tree          # Show tasks in tree view`,
	RunE: func(cmd *cobra.Command, args []string) error {
		status, _ := cmd.Flags().GetString("status")
		tagFilter, _ := cmd.Flags().GetString("tag")
		treeView, _ := cmd.Flags().GetBool("tree")

		s := store.DefaultStorageWithWarning()
		f, err := s.Read()
		if err != nil {
			return err
		}

		var tasks []taskEntry

		for id, t := range f.Tasks {
			if status != "" && string(t.Status) != status {
				continue
			}
			if tagFilter != "" && !t.HasTag(tagFilter) {
				continue
			}
			tasks = append(tasks, taskEntry{ID: id, Task: t})
		}

		// Sort by status priority, then by task priority, then by ID
		statusOrder := map[task.Status]int{
			task.StatusInProgress: 0,
			task.StatusReady:      1,
			task.StatusBlocked:    2,
			task.StatusDone:       3,
		}
		sort.Slice(tasks, func(i, j int) bool {
			if statusOrder[tasks[i].Task.Status] != statusOrder[tasks[j].Task.Status] {
				return statusOrder[tasks[i].Task.Status] < statusOrder[tasks[j].Task.Status]
			}
			pi, pj := tasks[i].Task.GetPriority(), tasks[j].Task.GetPriority()
			if pi != pj {
				return pi < pj
			}
			return tasks[i].ID < tasks[j].ID
		})

		if treeView {
			return outputTasksTree(tasks)
		}
		return outputTasks(tasks)
	},
}

func init() {
	listCmd.Flags().StringP("status", "s", "", "Filter by status: ready, in_progress, blocked, done")
	listCmd.Flags().StringP("tag", "t", "", "Filter by tag")
	listCmd.Flags().Bool("tree", false, "Show tasks in tree view with subtasks indented")
}
