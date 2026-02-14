package hooks

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

func newTaskCompletedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "task-completed",
		Short: "Show handoff guidance when a teammate completes a task",
		Long: `Called by Claude Code TaskCompleted hook when a teammate marks a task as done.

Shows:
  - Tasks that list the completed task as a blocker (may still have other blockers)
  - Reflection prompts to capture learnings and decisions

Reads JSON from stdin with task_id, task_subject, teammate_name, etc.
Always exits 0 (soft reminder, never blocks).

Examples:
  echo '{"task_id":"auth","task_subject":"Auth system"}' | tk hooks task-completed`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return hookTaskCompleted()
		},
	}
}

type taskCompletedInput struct {
	TaskID          string `json:"task_id"`
	TaskSubject     string `json:"task_subject"`
	TaskDescription string `json:"task_description"`
	TeammateName    string `json:"teammate_name"`
	TeamName        string `json:"team_name"`
}

func hookTaskCompleted() error {
	var input taskCompletedInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		// If we can't parse stdin, still show reflection prompts
		printReflectionPrompts()
		return nil
	}

	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		printTaskCompletedHeader(input.TaskID, input.TaskSubject)
		printReflectionPrompts()
		return nil
	}
	if !s.Exists() {
		printTaskCompletedHeader(input.TaskID, input.TaskSubject)
		printReflectionPrompts()
		return nil
	}

	f, err := s.Read()
	if err != nil {
		printTaskCompletedHeader(input.TaskID, input.TaskSubject)
		printReflectionPrompts()
		return nil
	}

	// Try to resolve a matching Tasuku task
	matchedID := resolveTaskID(input.TaskID, input.TaskSubject, f.Tasks)

	if matchedID != "" {
		printTaskCompletedHeader(matchedID, "")

		// Find tasks blocked by this one
		blocked := task.FindBlockedTasks(matchedID, f.Tasks)
		if len(blocked) > 0 {
			fmt.Println()
			fmt.Printf("Tasks blocked by this task (check remaining blockers):\n")
			for _, depID := range blocked {
				dep := f.Tasks[depID]
				ownerStr := "unassigned"
				if dep.Owner != nil && *dep.Owner != "" {
					ownerStr = *dep.Owner
				}
				fmt.Printf("   - %s [%s] (owner: %s): %s\n", depID, dep.Status, ownerStr, dep.Description)
			}
		}
	} else {
		printTaskCompletedHeader(input.TaskID, input.TaskSubject)
	}

	fmt.Println()
	printReflectionPrompts()

	return nil
}

// resolveTaskID tries to find a matching Tasuku task by exact ID or generated ID from subject.
func resolveTaskID(taskID, taskSubject string, tasks map[string]task.Task) string {
	if taskID != "" {
		if _, exists := tasks[taskID]; exists {
			return taskID
		}
	}

	if taskSubject != "" {
		generatedID := generateID(taskSubject)
		if generatedID != "" {
			if _, exists := tasks[generatedID]; exists {
				return generatedID
			}
		}
	}

	return ""
}

func printTaskCompletedHeader(taskID, taskSubject string) {
	label := taskID
	if label == "" {
		label = taskSubject
	}
	fmt.Printf("=== Task Completed: %s ===\n", label)
}

func printReflectionPrompts() {
	fmt.Println("Reflect on this work:")
	fmt.Println("   - Did you learn any gotchas or patterns? -> /tasuku:learn")
	fmt.Println("   - Did you make architectural decisions? -> /tasuku:decide")
	fmt.Println("   - Any \"never X\" or \"always Y\" rules? -> /tasuku:learn")
	fmt.Println("================================")
}
