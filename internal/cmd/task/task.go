// Package task provides CLI commands for task management.
package task

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/iheanyi/tasuku/internal/cmd/config"
	"github.com/iheanyi/tasuku/internal/task"
)

// Cmd is the parent command for all task operations
var Cmd = &cobra.Command{
	Use:     "task",
	Aliases: []string{"tasks", "t"},
	Short:   "Manage tasks",
	Long: `Manage tasks in your Tasuku project.

Subcommands:
  list      List all tasks
  add       Create a new task
  show      Show task details
  start     Mark task as in_progress
  done      Mark task as complete
  delete    Delete a task
  edit      Update task description
  pause     Pause work on a task
  block     Mark task as blocked
  unblock   Remove blockers from task
  ready     List tasks ready to work on
  find      Search across all content
  priority  Set task priority
  owner     Manage task ownership
  claim     Claim a task for an agent
  release   Release a claimed task
  who       Show claimed tasks by owner
  tag       Manage task tags
  field     Manage custom fields
  timer     Track time spent on tasks

Examples:
  tk task list                 # List all tasks
  tk task add "New feature"    # Add a new task
  tk task start my-task        # Start working on a task
  tk task claim my-task agent1 # Claim task for agent1
  tk task list --tag backend   # Filter by tag
  tk t ls                      # Short alias for list
  tk tasks ready               # Show ready tasks`,
}

func init() {
	// Register all task subcommands
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(addCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(doneCmd)
	Cmd.AddCommand(blockCmd)
	Cmd.AddCommand(unblockCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(editCmd)
	Cmd.AddCommand(pauseCmd)
	Cmd.AddCommand(ownerCmd)
	Cmd.AddCommand(readyCmd)
	Cmd.AddCommand(findCmd)
	Cmd.AddCommand(priorityCmd)
	Cmd.AddCommand(claimCmd)
	Cmd.AddCommand(releaseCmd)
	Cmd.AddCommand(whoCmd)
	Cmd.AddCommand(tagCmd)
	Cmd.AddCommand(fieldCmd)
	Cmd.AddCommand(timerCmd)
	Cmd.AddCommand(statsCmd)
	Cmd.AddCommand(depsCmd)
	Cmd.AddCommand(archiveCmd)
}

// taskEntry holds a task with its ID for sorting/display
type taskEntry struct {
	ID   string
	Task task.Task
}

// outputTasks outputs tasks in the configured format
func outputTasks(tasks []taskEntry) error {
	switch config.OutputFormat {
	case "json":
		output := make(map[string]interface{})
		for _, t := range tasks {
			output[t.ID] = t.Task
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		output := make(map[string]interface{})
		for _, t := range tasks {
			output[t.ID] = t.Task
		}
		data, _ := yaml.Marshal(output)
		fmt.Print(string(data))
	default:
		if len(tasks) == 0 {
			fmt.Println("No tasks found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, t := range tasks {
			icon := getStatusIcon(t.Task.Status)
			desc := truncate(t.Task.Description, 50)

			// Build info parts
			var parts []string
			if len(t.Task.BlockedBy) > 0 {
				parts = append(parts, fmt.Sprintf("blocked by: %s", strings.Join(t.Task.BlockedBy, ", ")))
			}
			if len(t.Task.Tags) > 0 {
				parts = append(parts, fmt.Sprintf("tags: %s", strings.Join(t.Task.Tags, ", ")))
			}

			extra := ""
			if len(parts) > 0 {
				extra = "  (" + strings.Join(parts, ", ") + ")"
			}

			fmt.Fprintf(w, "[%s]  %s\t%s%s\n", icon, t.ID, desc, extra)
		}
		w.Flush()
	}
	return nil
}

// outputTasksTree outputs tasks in a tree format showing parent-child relationships
func outputTasksTree(tasks []taskEntry) error {
	switch config.OutputFormat {
	case "json", "yaml":
		return outputTasks(tasks)
	default:
		if len(tasks) == 0 {
			fmt.Println("No tasks found")
			return nil
		}

		// Build parent-child map
		children := make(map[string][]taskEntry)
		var rootTasks []taskEntry

		for _, t := range tasks {
			if t.Task.ParentID == nil || *t.Task.ParentID == "" {
				rootTasks = append(rootTasks, t)
			} else {
				children[*t.Task.ParentID] = append(children[*t.Task.ParentID], t)
			}
		}

		// Print tree recursively
		var printTree func(entries []taskEntry, indent string)
		printTree = func(entries []taskEntry, indent string) {
			for i, t := range entries {
				icon := getStatusIcon(t.Task.Status)
				desc := truncate(t.Task.Description, 50-len(indent))

				// Determine tree character
				prefix := "├── "
				if i == len(entries)-1 {
					prefix = "└── "
				}

				fmt.Printf("%s%s[%s] %s  %s\n", indent, prefix, icon, t.ID, desc)

				// Print children
				childIndent := indent + "│   "
				if i == len(entries)-1 {
					childIndent = indent + "    "
				}
				if childTasks, ok := children[t.ID]; ok {
					printTree(childTasks, childIndent)
				}
			}
		}

		printTree(rootTasks, "")
	}
	return nil
}

// outputTaskDetail outputs detailed task information
func outputTaskDetail(id string, t task.Task, notes []task.Note, allTasks map[string]task.Task) error {
	switch config.OutputFormat {
	case "json":
		output := map[string]interface{}{
			"id":   id,
			"task": t,
		}
		if len(notes) > 0 {
			output["notes"] = notes
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		output := map[string]interface{}{
			"id":   id,
			"task": t,
		}
		if len(notes) > 0 {
			output["notes"] = notes
		}
		data, _ := yaml.Marshal(output)
		fmt.Print(string(data))
	default:
		fmt.Printf("ID:          %s\n", id)
		fmt.Printf("Description: %s\n", t.Description)
		fmt.Printf("Status:      %s\n", t.Status)
		fmt.Printf("Priority:    %s\n", task.PriorityName(t.GetPriority()))
		if t.Owner != nil {
			fmt.Printf("Owner:       %s\n", *t.Owner)
		}
		if t.ParentID != nil && *t.ParentID != "" {
			fmt.Printf("Parent:      %s\n", *t.ParentID)
		}
		if len(t.BlockedBy) > 0 {
			fmt.Printf("Blocked by:  %s\n", strings.Join(t.BlockedBy, ", "))
		}
		if len(t.Tags) > 0 {
			fmt.Printf("Tags:        %s\n", strings.Join(t.Tags, ", "))
		}
		if len(t.Fields) > 0 {
			fmt.Printf("Fields:\n")
			for k, v := range t.Fields {
				fmt.Printf("  %s: %s\n", k, v)
			}
		}
		fmt.Printf("Created:     %s\n", t.CreatedAt.Format(time.RFC3339))
		fmt.Printf("Updated:     %s\n", t.UpdatedAt.Format(time.RFC3339))

		// Show subtasks if any
		var subtasks []string
		for taskID, taskData := range allTasks {
			if taskData.ParentID != nil && *taskData.ParentID == id {
				subtasks = append(subtasks, taskID)
			}
		}
		if len(subtasks) > 0 {
			sort.Strings(subtasks)
			fmt.Printf("Subtasks:    %s\n", strings.Join(subtasks, ", "))
		}

		if len(notes) > 0 {
			fmt.Println("\nNotes:")
			for i, n := range notes {
				fmt.Printf("  %d. %s (%s)\n", i+1, n.Text, formatRelativeTime(n.CreatedAt))
			}
		}
	}
	return nil
}

func getStatusIcon(status task.Status) string {
	switch status {
	case task.StatusInProgress:
		return "*"
	case task.StatusReady:
		return "-"
	case task.StatusBlocked:
		return "!"
	case task.StatusDone:
		return "x"
	default:
		return "?"
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatRelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// getAgentName returns the current agent/user name for claims.
// Priority: TASUKU_AGENT env var > git user.name > system username > hostname
func getAgentName() string {
	if agent := os.Getenv("TASUKU_AGENT"); agent != "" {
		return agent
	}
	if out, err := exec.Command("git", "config", "user.name").Output(); err == nil {
		if name := strings.TrimSpace(string(out)); name != "" {
			return name
		}
	}
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}
	return "unknown"
}

// gitCommitAndPush commits task changes and pushes to remote
func gitCommitAndPush(message string) error {
	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		return fmt.Errorf("not in a git repository")
	}
	if err := exec.Command("git", "add", ".tasuku/").Run(); err != nil {
		exec.Command("git", "add", ".tasuku.json").Run()
	}
	statusOut, _ := exec.Command("git", "status", "--porcelain", ".tasuku/", ".tasuku.json").Output()
	if len(strings.TrimSpace(string(statusOut))) == 0 {
		return nil
	}
	if err := exec.Command("git", "commit", "-m", message).Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}
	exec.Command("git", "push").Run()
	return nil
}
