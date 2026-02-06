// Package note provides CLI commands for managing task notes.
package note

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/iheanyi/tasuku/internal/cmd/config"
	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

func newNoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "note",
		Short:   "Manage notes",
		Long:    `Manage notes attached to tasks.`,
		Aliases: []string{"notes"},
	}

	cmd.AddCommand(listCmd)
	cmd.AddCommand(addCmd)
	cmd.AddCommand(removeCmd)

	return cmd
}

// Cmd is the parent command for all note operations
var Cmd = newNoteCmd()

var listCmd = &cobra.Command{
	Use:   "list [task-id]",
	Short: "List notes for a task or all notes",
	Long: `Display notes recorded in the project context.

If task-id is provided, show notes for that specific task.
If no task-id is provided, show all notes grouped by task.

Examples:
  tk note list                    # List all notes grouped by task
  tk note list my-task            # List notes for "my-task"
  tk note list -f json            # Output as JSON`,
	Args: cobra.MaximumNArgs(1),
	RunE: runList,
}

var addCmd = &cobra.Command{
	Use:   "add <task-id> <text>",
	Short: "Add a note to a task",
	Long: `Attach a note to a specific task for additional context.

Notes are useful for:
  - Recording progress updates
  - Documenting blockers or issues encountered
  - Capturing implementation details
  - Leaving messages for other agents

Notes appear when you run 'tk task show' for the task.

Examples:
  tk note add my-task "Started implementation of auth flow"
  tk note add fix-bug "Root cause: null pointer in UserService"
  tk note add feature "Waiting for API spec from backend team"`,
	Args: cobra.ExactArgs(2),
	RunE: runAdd,
}

var removeCmd = &cobra.Command{
	Use:   "remove <task-id> <note-id>",
	Short: "Remove a note from a task",
	Long: `Remove a note from a task by its ID.

Use 'tk note list <task-id>' to see available notes and their IDs.

Examples:
  tk note remove my-task a3x9k2    # Remove note with ID "a3x9k2" from "my-task"
  tk note remove fix-bug b7m4p1    # Remove note with ID "b7m4p1" from "fix-bug"`,
	Args: cobra.ExactArgs(2),
	RunE: runRemove,
}

func runList(cmd *cobra.Command, args []string) error {
	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		return err
	}
	f, err := s.Read()
	if err != nil {
		return err
	}

	notes := f.Context.Notes
	if len(notes) == 0 {
		fmt.Println("No notes recorded yet.")
		fmt.Println("Use: tk note add <task-id> \"your note here\"")
		return nil
	}

	if len(args) == 1 {
		taskID := args[0]
		taskNotes, exists := notes[taskID]
		if !exists || len(taskNotes) == 0 {
			return fmt.Errorf("no notes found for task: %s", taskID)
		}

		switch config.OutputFormat {
		case "json":
			data, _ := json.MarshalIndent(map[string][]task.Note{taskID: taskNotes}, "", "  ")
			fmt.Println(string(data))
		case "yaml":
			data, _ := yaml.Marshal(map[string][]task.Note{taskID: taskNotes})
			fmt.Print(string(data))
		default:
			fmt.Printf("Notes for %s:\n", taskID)
			for _, note := range taskNotes {
				fmt.Printf("  [%s] %s\n", note.ID, note.Text)
			}
		}
		return nil
	}

	switch config.OutputFormat {
	case "json":
		data, _ := json.MarshalIndent(notes, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(notes)
		fmt.Print(string(data))
	default:
		var taskIDs []string
		for taskID := range notes {
			taskIDs = append(taskIDs, taskID)
		}
		sort.Strings(taskIDs)

		totalNotes := 0
		for _, taskNotes := range notes {
			totalNotes += len(taskNotes)
		}

		fmt.Printf("Notes (%d total across %d tasks):\n\n", totalNotes, len(notes))
		for _, taskID := range taskIDs {
			taskNotes := notes[taskID]
			fmt.Printf("  [%s]\n", taskID)
			for _, note := range taskNotes {
				fmt.Printf("    [%s] %s\n", note.ID, note.Text)
			}
			fmt.Println()
		}
	}
	return nil
}

func runAdd(cmd *cobra.Command, args []string) error {
	taskID := args[0]
	note := args[1]

	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		return err
	}
	noteID, err := s.AddNote(taskID, note)
	if err != nil {
		return err
	}

	fmt.Printf("Note [%s] added to: %s\n", noteID, taskID)
	return nil
}

func runRemove(cmd *cobra.Command, args []string) error {
	taskID := args[0]
	noteID := args[1]

	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		return err
	}
	removedText, err := s.RemoveNote(taskID, noteID)
	if err != nil {
		return err
	}

	fmt.Printf("Removed note: %s\n", removedText)
	return nil
}
