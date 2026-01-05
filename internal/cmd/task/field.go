package task

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/iheanyi/tasuku/internal/cmd/config"
	"github.com/iheanyi/tasuku/internal/store"
)

func newFieldCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "field",
		Short: "Manage custom fields on tasks",
		Long: `Manage custom key-value metadata fields on tasks.

Custom fields allow you to attach arbitrary metadata to tasks
for tracking additional information like estimates, URLs,
component names, etc.

Subcommands:
  set       Set a custom field value
  get       Get a custom field value
  list      List all custom fields on a task
  remove    Remove a custom field

Examples:
  tk task field set my-task estimate "2 hours"
  tk task field get my-task estimate
  tk task field list my-task
  tk task field remove my-task estimate`,
	}

	cmd.AddCommand(fieldSetCmd)
	cmd.AddCommand(fieldGetCmd)
	cmd.AddCommand(fieldListCmd)
	cmd.AddCommand(fieldRemoveCmd)

	return cmd
}

var fieldCmd = newFieldCmd()

var fieldSetCmd = &cobra.Command{
	Use:   "set <task-id> <key> <value>",
	Short: "Set a custom field on a task",
	Long: `Set a custom key-value field on a task.

If the field already exists, its value will be updated.
Field keys are case-sensitive.

Examples:
  tk task field set auth-feature estimate "4 hours"
  tk task field set auth-feature component "backend"
  tk task field set auth-feature jira-link "https://jira.example.com/PROJ-123"`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		key := args[1]
		value := args[2]
		s := store.DefaultStorageWithWarning()

		if err := s.SetField(taskID, key, value); err != nil {
			return err
		}

		fmt.Printf("Set field '%s' = '%s' on task %s\n", key, value, taskID)
		return nil
	},
}

var fieldGetCmd = &cobra.Command{
	Use:   "get <task-id> <key>",
	Short: "Get a custom field value from a task",
	Long: `Get the value of a custom field on a task.

Returns an error if the field does not exist.

Examples:
  tk task field get auth-feature estimate
  tk task field get auth-feature component`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		key := args[1]
		s := store.DefaultStorageWithWarning()

		f, err := s.Read()
		if err != nil {
			return err
		}

		t, exists := f.Tasks[taskID]
		if !exists {
			return fmt.Errorf("task not found: %s", taskID)
		}

		if len(t.Fields) == 0 {
			return fmt.Errorf("task %s has no custom fields", taskID)
		}

		value, hasKey := t.Fields[key]
		if !hasKey {
			return fmt.Errorf("field '%s' not found on task %s", key, taskID)
		}

		switch config.OutputFormat {
		case "json":
			data, _ := json.MarshalIndent(map[string]string{key: value}, "", "  ")
			fmt.Println(string(data))
		case "yaml":
			data, _ := yaml.Marshal(map[string]string{key: value})
			fmt.Print(string(data))
		default:
			fmt.Println(value)
		}
		return nil
	},
}

type fieldListInfo struct {
	TaskID string            `json:"task_id" yaml:"task_id"`
	Fields map[string]string `json:"fields" yaml:"fields"`
}

var fieldListCmd = &cobra.Command{
	Use:   "list <task-id>",
	Short: "List all custom fields on a task",
	Long: `List all custom key-value fields on a task.

Examples:
  tk task field list auth-feature
  tk task field list auth-feature -f json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.DefaultStorageWithWarning()

		f, err := s.Read()
		if err != nil {
			return err
		}

		t, exists := f.Tasks[taskID]
		if !exists {
			return fmt.Errorf("task not found: %s", taskID)
		}

		fields := t.Fields
		if fields == nil {
			fields = make(map[string]string)
		}

		info := fieldListInfo{
			TaskID: taskID,
			Fields: fields,
		}

		switch config.OutputFormat {
		case "json":
			data, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(data))
		case "yaml":
			data, _ := yaml.Marshal(info)
			fmt.Print(string(data))
		default:
			if len(fields) == 0 {
				fmt.Printf("Task %s has no custom fields\n", taskID)
			} else {
				fmt.Printf("Fields on %s:\n", taskID)
				keys := make([]string, 0, len(fields))
				for k := range fields {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Printf("  %s: %s\n", k, fields[k])
				}
			}
		}
		return nil
	},
}

var fieldRemoveCmd = &cobra.Command{
	Use:     "remove <task-id> <key>",
	Aliases: []string{"rm"},
	Short:   "Remove a custom field from a task",
	Long: `Remove a custom field from a task.

Returns an error if the field does not exist.

Examples:
  tk task field remove auth-feature estimate
  tk task field rm auth-feature component`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		key := args[1]
		s := store.DefaultStorageWithWarning()

		if err := s.RemoveField(taskID, key); err != nil {
			return err
		}

		fmt.Printf("Removed field '%s' from task %s\n", key, taskID)
		return nil
	},
}
