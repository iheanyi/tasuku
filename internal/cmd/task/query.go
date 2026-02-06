package task

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/cmdutil"
	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "Show tasks ready to work on",
	Long: `List all tasks that are ready to be started, sorted by priority.

A task is considered "ready" when:
  - Status is "ready" (not blocked, in_progress, or done)
  - All blocking tasks (blocked_by) are completed

Examples:
  tk task ready                     # List ready tasks
  tk task ready -f json             # Output as JSON`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.DefaultStorageWithWarning()
		if err != nil {
			return err
		}
		f, err := s.Read()
		if err != nil {
			return err
		}

		var tasks []taskEntry

		for id, t := range f.Tasks {
			if t.Status == task.StatusReady {
				blocked := false
				for _, blockerID := range t.BlockedBy {
					if blocker, exists := f.Tasks[blockerID]; exists && blocker.Status != task.StatusDone {
						blocked = true
						break
					}
				}
				if !blocked {
					tasks = append(tasks, taskEntry{ID: id, Task: t})
				}
			}
		}

		sort.Slice(tasks, func(i, j int) bool {
			pi, pj := tasks[i].Task.GetPriority(), tasks[j].Task.GetPriority()
			if pi != pj {
				return pi < pj
			}
			return tasks[i].ID < tasks[j].ID
		})

		return outputTasks(tasks)
	},
}

var findCmd = &cobra.Command{
	Use:   "find <query>",
	Short: "Search across all content",
	Long: `Search for tasks, learnings, and decisions containing the query string.

The search is case-insensitive and matches partial strings.

Examples:
  tk task find auth               # Find items mentioning "auth"
  tk task find "login bug"        # Search for phrase`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.ToLower(args[0])
		s, err := store.DefaultStorageWithWarning()
		if err != nil {
			return err
		}
		f, err := s.Read()
		if err != nil {
			return err
		}

		// Search tasks
		fmt.Println("Tasks:")
		found := false
		for id, t := range f.Tasks {
			if strings.Contains(strings.ToLower(id), query) ||
				strings.Contains(strings.ToLower(t.Description), query) {
				icon := getStatusIcon(t.Status)
				fmt.Printf("  [%s] %s: %s\n", icon, id, cmdutil.Truncate(t.Description, 50))
				found = true
			}
		}
		if !found {
			fmt.Println("  (none)")
		}

		// Search learnings
		fmt.Println("\nLearnings:")
		found = false
		for _, l := range f.Context.Learnings {
			if strings.Contains(strings.ToLower(l.Text), query) {
				fmt.Printf("  - %s\n", cmdutil.Truncate(l.Text, 60))
				found = true
			}
		}
		if !found {
			fmt.Println("  (none)")
		}

		// Search decisions
		fmt.Println("\nDecisions:")
		found = false
		for _, d := range f.Context.Decisions {
			if strings.Contains(strings.ToLower(d.ID), query) ||
				strings.Contains(strings.ToLower(d.Chose), query) ||
				strings.Contains(strings.ToLower(d.Because), query) {
				fmt.Printf("  - %s: chose %s\n", d.ID, cmdutil.Truncate(d.Chose, 40))
				found = true
			}
		}
		if !found {
			fmt.Println("  (none)")
		}

		return nil
	},
}
