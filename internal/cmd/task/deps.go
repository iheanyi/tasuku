package task

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/iheanyi/tasuku/internal/cmd/config"
	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

var depsCmd = &cobra.Command{
	Use:   "deps [task-id]",
	Short: "Show task dependency tree",
	Long: `Show task blocking relationships as a tree.

Without arguments, shows all blocking relationships.
With a task ID, shows the dependency chain for that task.

The output shows:
  - Which tasks are blocked by other tasks
  - The status of each task in the chain
  - Hints about which blockers can be started

Examples:
  tk task deps              # Show all blocking relationships
  tk task deps auth-system  # Show what blocks auth-system
  tk task deps -f json      # Output as JSON`,
	Aliases: []string{"tree", "blocks"},
	Args:    cobra.MaximumNArgs(1),
	RunE:    runTaskDeps,
}

func runTaskDeps(cmd *cobra.Command, args []string) error {
	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		return err
	}
	f, err := s.Read()
	if err != nil {
		return err
	}

	if len(args) == 1 {
		taskID := args[0]
		return outputTaskDepsForTask(taskID, f.Tasks)
	}

	return outputAllTaskDeps(f.Tasks)
}

func findBlockedTasks(taskID string, tasks map[string]task.Task) []string {
	var blocks []string
	for id, t := range tasks {
		for _, blocker := range t.BlockedBy {
			if blocker == taskID {
				blocks = append(blocks, id)
				break
			}
		}
	}
	sort.Strings(blocks)
	return blocks
}

type taskDepInfo struct {
	ID        string   `json:"id" yaml:"id"`
	Status    string   `json:"status" yaml:"status"`
	BlockedBy []string `json:"blocked_by,omitempty" yaml:"blocked_by,omitempty"`
	Blocks    []string `json:"blocks,omitempty" yaml:"blocks,omitempty"`
}

func outputTaskDepsForTask(taskID string, tasks map[string]task.Task) error {
	t, exists := tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	blocks := findBlockedTasks(taskID, tasks)

	switch config.OutputFormat {
	case "json":
		info := taskDepInfo{
			ID:        taskID,
			Status:    string(t.Status),
			BlockedBy: t.BlockedBy,
			Blocks:    blocks,
		}
		data, _ := json.MarshalIndent(info, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		info := taskDepInfo{
			ID:        taskID,
			Status:    string(t.Status),
			BlockedBy: t.BlockedBy,
			Blocks:    blocks,
		}
		data, _ := yaml.Marshal(info)
		fmt.Print(string(data))
	default:
		fmt.Printf("%s [%s]\n", taskID, t.Status)

		if len(t.BlockedBy) > 0 {
			fmt.Println("+-- blocked by:")
			for i, blockerID := range t.BlockedBy {
				connector := "|   +--"
				if i == len(t.BlockedBy)-1 {
					connector = "|   \\--"
				}
				blockerStatus := "not found"
				hint := ""
				if blocker, exists := tasks[blockerID]; exists {
					blockerStatus = string(blocker.Status)
					switch blocker.Status {
					case task.StatusReady:
						hint = " <- can start"
					case task.StatusInProgress:
						hint = " <- in progress"
					case task.StatusDone:
						hint = " <- done"
					}
				}
				fmt.Printf("%s %s [%s]%s\n", connector, blockerID, blockerStatus, hint)
			}
		}

		if len(blocks) > 0 {
			fmt.Println("+-- blocks:")
			for i, blockedID := range blocks {
				connector := "    +--"
				if i == len(blocks)-1 {
					connector := "    \\--"
					_ = connector // use the value
				}
				blockedStatus := "not found"
				if blocked, exists := tasks[blockedID]; exists {
					blockedStatus = string(blocked.Status)
				}
				fmt.Printf("%s %s [%s]\n", connector, blockedID, blockedStatus)
			}
		}

		if len(t.BlockedBy) == 0 && len(blocks) == 0 {
			fmt.Println("  (no dependencies)")
		}
	}
	return nil
}

func outputAllTaskDeps(tasks map[string]task.Task) error {
	var deps []taskDepInfo
	for id, t := range tasks {
		blocks := findBlockedTasks(id, tasks)
		if len(t.BlockedBy) > 0 || len(blocks) > 0 {
			deps = append(deps, taskDepInfo{
				ID:        id,
				Status:    string(t.Status),
				BlockedBy: t.BlockedBy,
				Blocks:    blocks,
			})
		}
	}

	sort.Slice(deps, func(i, j int) bool {
		return deps[i].ID < deps[j].ID
	})

	switch config.OutputFormat {
	case "json":
		data, _ := json.MarshalIndent(deps, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(deps)
		fmt.Print(string(data))
	default:
		if len(deps) == 0 {
			fmt.Println("No task dependencies found")
			return nil
		}

		fmt.Println("Task Dependencies:")
		fmt.Println()

		for _, dep := range deps {
			t := tasks[dep.ID]
			fmt.Printf("+-- %s [%s]\n", dep.ID, dep.Status)

			if len(t.BlockedBy) > 0 {
				blockerList := make([]string, 0, len(t.BlockedBy))
				for _, blockerID := range t.BlockedBy {
					blockerStatus := "?"
					if blocker, exists := tasks[blockerID]; exists {
						blockerStatus = string(blocker.Status)
					}
					blockerList = append(blockerList, fmt.Sprintf("%s[%s]", blockerID, blockerStatus))
				}
				fmt.Printf("|   blocked by: %s\n", strings.Join(blockerList, ", "))
			}

			if len(dep.Blocks) > 0 {
				blocksList := make([]string, 0, len(dep.Blocks))
				for _, blockedID := range dep.Blocks {
					blockedStatus := "?"
					if blocked, exists := tasks[blockedID]; exists {
						blockedStatus = string(blocked.Status)
					}
					blocksList = append(blocksList, fmt.Sprintf("%s[%s]", blockedID, blockedStatus))
				}
				fmt.Printf("|   blocks: %s\n", strings.Join(blocksList, ", "))
			}

			fmt.Println("|")
		}
	}
	return nil
}
