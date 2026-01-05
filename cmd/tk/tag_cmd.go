package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/iheanyi/tasuku/internal/store"
)

// =============================================================================
// Tag Management Commands (V2.0)
// =============================================================================

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage task tags",
	Long: `Manage tags on tasks for filtering and organization.

Tags allow you to categorize and filter tasks by topic, component,
or any other grouping you find useful.

Subcommands:
  add       Add a tag to a task
  remove    Remove a tag from a task
  list      List tags on a task (or all tags in project)

Examples:
  tk task tag my-task add backend       # Add 'backend' tag
  tk task tag my-task remove backend    # Remove 'backend' tag
  tk task tag my-task list              # List tags on task
  tk task tag list                      # List all tags in project`,
}

var tagAddCmd = &cobra.Command{
	Use:   "add <task-id> <tag>",
	Short: "Add a tag to a task",
	Long: `Add a tag to a task.

Tags are case-sensitive strings. You can add multiple tags by
calling this command multiple times or using comma-separated values
with the --tag flag on 'tk task add'.

Examples:
  tk task tag add auth-feature backend
  tk task tag add auth-feature security`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		tag := args[1]
		s := store.Default()

		if err := s.AddTag(taskID, tag); err != nil {
			return err
		}

		fmt.Printf("Added tag '%s' to task %s\n", tag, taskID)
		return nil
	},
}

var tagRemoveCmd = &cobra.Command{
	Use:     "remove <task-id> <tag>",
	Aliases: []string{"rm"},
	Short:   "Remove a tag from a task",
	Long: `Remove a tag from a task.

Examples:
  tk task tag remove auth-feature backend
  tk task tag rm auth-feature security`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		tag := args[1]
		s := store.Default()

		if err := s.RemoveTag(taskID, tag); err != nil {
			return err
		}

		fmt.Printf("Removed tag '%s' from task %s\n", tag, taskID)
		return nil
	},
}

// tagListInfo holds tag listing information for output
type tagListInfo struct {
	TaskID string   `json:"task_id,omitempty" yaml:"task_id,omitempty"`
	Tags   []string `json:"tags" yaml:"tags"`
}

type allTagsInfo struct {
	Tag   string   `json:"tag" yaml:"tag"`
	Tasks []string `json:"tasks" yaml:"tasks"`
}

var tagListCmd = &cobra.Command{
	Use:   "list [task-id]",
	Short: "List tags on a task or all tags in project",
	Long: `List tags.

Without a task ID, lists all unique tags used in the project along
with which tasks have each tag.

With a task ID, lists all tags on that specific task.

Examples:
  tk task tag list                  # List all tags in project
  tk task tag list auth-feature     # List tags on specific task
  tk task tag list -f json          # Output as JSON`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.Default()
		f, err := s.Read()
		if err != nil {
			return err
		}

		if len(args) == 1 {
			// List tags for specific task
			taskID := args[0]
			t, exists := f.Tasks[taskID]
			if !exists {
				return fmt.Errorf("task not found: %s", taskID)
			}

			info := tagListInfo{
				TaskID: taskID,
				Tags:   t.Tags,
			}
			if info.Tags == nil {
				info.Tags = []string{}
			}

			switch outputFormat {
			case "json":
				data, _ := json.MarshalIndent(info, "", "  ")
				fmt.Println(string(data))
			case "yaml":
				data, _ := yaml.Marshal(info)
				fmt.Print(string(data))
			default:
				if len(t.Tags) == 0 {
					fmt.Printf("Task %s has no tags\n", taskID)
				} else {
					fmt.Printf("Tags on %s: %s\n", taskID, strings.Join(t.Tags, ", "))
				}
			}
			return nil
		}

		// List all tags in project
		tagToTasks := make(map[string][]string)
		for id, t := range f.Tasks {
			for _, tag := range t.Tags {
				tagToTasks[tag] = append(tagToTasks[tag], id)
			}
		}

		// Sort tags
		var tags []string
		for tag := range tagToTasks {
			tags = append(tags, tag)
		}
		sort.Strings(tags)

		// Build output
		var allTags []allTagsInfo
		for _, tag := range tags {
			tasks := tagToTasks[tag]
			sort.Strings(tasks)
			allTags = append(allTags, allTagsInfo{
				Tag:   tag,
				Tasks: tasks,
			})
		}

		switch outputFormat {
		case "json":
			data, _ := json.MarshalIndent(allTags, "", "  ")
			fmt.Println(string(data))
		case "yaml":
			data, _ := yaml.Marshal(allTags)
			fmt.Print(string(data))
		default:
			if len(allTags) == 0 {
				fmt.Println("No tags in project")
			} else {
				fmt.Println("Tags in project:")
				for _, t := range allTags {
					fmt.Printf("  %s (%d tasks): %s\n", t.Tag, len(t.Tasks), strings.Join(t.Tasks, ", "))
				}
			}
		}
		return nil
	},
}

func init() {
	tagCmd.AddCommand(tagAddCmd)
	tagCmd.AddCommand(tagRemoveCmd)
	tagCmd.AddCommand(tagListCmd)
}
