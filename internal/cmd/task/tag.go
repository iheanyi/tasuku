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
)

func newTagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Manage task tags",
		Long: `Manage tags on tasks for filtering and organization.

Subcommands:
  add       Add a tag to a task
  remove    Remove a tag from a task
  list      List tags on a task (or all tags in project)

Examples:
  tk task tag add my-task backend       # Add 'backend' tag
  tk task tag remove my-task backend    # Remove 'backend' tag
  tk task tag list                      # List all tags in project`,
	}

	cmd.AddCommand(tagAddCmd)
	cmd.AddCommand(tagRemoveCmd)
	cmd.AddCommand(tagListCmd)

	return cmd
}

var tagCmd = newTagCmd()

var tagAddCmd = &cobra.Command{
	Use:   "add <task-id> <tag> [tag...]",
	Short: "Add tag(s) to a task",
	Long: `Add one or more tags to a task.

Tags can be specified as separate arguments or comma-separated.

Examples:
  tk task tag add my-task backend          # Add single tag
  tk task tag add my-task backend,api      # Add comma-separated tags
  tk task tag add my-task backend api db   # Add multiple tags`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s, err := store.DefaultStorageWithWarning()
		if err != nil {
			return err
		}

		// Collect all tags, splitting comma-separated values
		var tags []string
		for _, arg := range args[1:] {
			for _, tag := range strings.Split(arg, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					tags = append(tags, tag)
				}
			}
		}

		for _, tag := range tags {
			if err := s.AddTag(taskID, tag); err != nil {
				return err
			}
			fmt.Printf("Added tag '%s' to task %s\n", tag, taskID)
		}
		return nil
	},
}

var tagRemoveCmd = &cobra.Command{
	Use:     "remove <task-id> <tag>",
	Aliases: []string{"rm"},
	Short:   "Remove a tag from a task",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		tag := args[1]
		s, err := store.DefaultStorageWithWarning()
		if err != nil {
			return err
		}

		if err := s.RemoveTag(taskID, tag); err != nil {
			return err
		}

		fmt.Printf("Removed tag '%s' from task %s\n", tag, taskID)
		return nil
	},
}

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
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
			t, exists := f.Tasks[taskID]
			if !exists {
				return fmt.Errorf("task not found: %s", taskID)
			}

			info := tagListInfo{TaskID: taskID, Tags: t.Tags}
			if info.Tags == nil {
				info.Tags = []string{}
			}

			switch config.OutputFormat {
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

		// List all tags
		tagToTasks := make(map[string][]string)
		for id, t := range f.Tasks {
			for _, tag := range t.Tags {
				tagToTasks[tag] = append(tagToTasks[tag], id)
			}
		}

		var tags []string
		for tag := range tagToTasks {
			tags = append(tags, tag)
		}
		sort.Strings(tags)

		var allTags []allTagsInfo
		for _, tag := range tags {
			tasks := tagToTasks[tag]
			sort.Strings(tasks)
			allTags = append(allTags, allTagsInfo{Tag: tag, Tasks: tasks})
		}

		switch config.OutputFormat {
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
