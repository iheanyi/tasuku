package task

import (
	"sort"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all tasks",
		Long: `Display all tasks in the project, sorted by status and priority.

Status Icons:
  *  in_progress - Currently being worked on
  -  ready       - Available to start
  !  blocked     - Waiting on other tasks
  x  done        - Completed
  ⌂  archived    - Archived (completed and stored)

Sort Order:
  1. Status: in_progress > ready > blocked > done
  2. Priority: critical (0) > high (1) > normal (2) > low (3) > backlog (4)
  3. Task ID: alphabetically

Filtering:
  Use --status to show only tasks with a specific status.
  Use --status archived to show archived tasks.
  Use --tag to show only tasks with a specific tag.
  Use --owner to show only tasks assigned to a specific owner.

Tree View:
  Use --tree to show tasks in a hierarchical tree format,
  with subtasks indented under their parent tasks.

Examples:
  tk task list                    # List all tasks
  tk task list -s ready           # Show only ready tasks
  tk task list --status done      # Show completed tasks
  tk task list --status archived  # Show archived tasks
  tk task list --tag backend      # Show tasks with 'backend' tag
  tk task list --owner alice      # Show tasks owned by alice
  tk task list -t bug -s ready    # Combine filters
  tk task list -f json            # Output as JSON
  tk task list --tree             # Show tasks in tree view`,
		RunE: func(cmd *cobra.Command, args []string) error {
			status, _ := cmd.Flags().GetString("status")
			tagFilter, _ := cmd.Flags().GetString("tag")
			ownerFilter, _ := cmd.Flags().GetString("owner")
			treeView, _ := cmd.Flags().GetBool("tree")

			s, err := store.DefaultStorageWithWarning()
			if err != nil {
				return err
			}

			// Handle archived status specially - query archive directory
			if status == "archived" {
				return listArchivedTasks(s, tagFilter, ownerFilter)
			}

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
				if ownerFilter != "" {
					if t.Owner == nil || *t.Owner != ownerFilter {
						continue
					}
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

	cmd.Flags().StringP("status", "s", "", "Filter by status: ready, in_progress, blocked, done, archived")
	cmd.Flags().StringP("tag", "t", "", "Filter by tag")
	cmd.Flags().StringP("owner", "o", "", "Filter by owner")
	cmd.Flags().Bool("tree", false, "Show tasks in tree view with subtasks indented")

	return cmd
}

// listArchivedTasks handles listing archived tasks with optional filters
func listArchivedTasks(s store.Storage, tagFilter, ownerFilter string) error {
	archived, err := s.GetArchivedTasks()
	if err != nil {
		return err
	}

	var tasks []taskEntry
	for id, at := range archived {
		// Apply filters to archived tasks
		if tagFilter != "" && !at.Task.HasTag(tagFilter) {
			continue
		}
		if ownerFilter != "" {
			if at.Task.Owner == nil || *at.Task.Owner != ownerFilter {
				continue
			}
		}
		// Create a copy of the embedded Task and set a pseudo-status for display
		t := at.Task
		tasks = append(tasks, taskEntry{ID: id, Task: t, Archived: true, ArchivedAt: at.ArchivedAt})
	}

	// Sort by archived date (most recent first), then by ID
	sort.Slice(tasks, func(i, j int) bool {
		if !tasks[i].ArchivedAt.Equal(tasks[j].ArchivedAt) {
			return tasks[i].ArchivedAt.After(tasks[j].ArchivedAt)
		}
		return tasks[i].ID < tasks[j].ID
	})

	return outputTasks(tasks)
}

var listCmd = newListCmd()
