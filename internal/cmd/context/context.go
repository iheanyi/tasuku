// Package context provides CLI commands for managing project context.
package context

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/iheanyi/tasuku/internal/cmd/config"
	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

func newContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "context",
		Aliases: []string{"ctx"},
		Short:   "Manage project context (learnings, decisions, notes)",
		Long: `Manage and inspect the project context in Tasuku.

Subcommands:
  show      Dump the complete project context for agent consumption
  validate  Validate Tasuku storage for correctness
  schema    Output JSON Schema for Tasuku files

Examples:
  tk context show              # Output full context as JSON
  tk context validate          # Validate Tasuku storage
  tk context schema            # Show JSON schema`,
	}

	cmd.AddCommand(showCmd)
	cmd.AddCommand(validateCmd)
	cmd.AddCommand(schemaCmd)

	return cmd
}

// Cmd is the parent command for all context operations
var Cmd = newContextCmd()

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Dump the complete project context for agent consumption",
	Long: `Output the entire Tasuku storage contents as structured data.

This command is designed for AI agents that need the full project context,
including all tasks, learnings, decisions, and notes.

Output includes:
  - version: Schema version number
  - tasks: All tasks with their statuses, priorities, dependencies
  - context.learnings: Insights discovered during work
  - context.decisions: Documented architectural choices
  - context.notes: Notes attached to tasks

The output format defaults to JSON but can be changed to YAML.

Examples:
  tk context show              # Output as JSON
  tk context show -f yaml      # Output as YAML
  tk context show | jq '.tasks'  # Pipe to jq for processing`,
	RunE: runShow,
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate Tasuku storage",
	Long: `Validate the Tasuku storage for correctness.

Checks performed:
- Version is supported
- All tasks have non-empty descriptions
- All tasks have valid statuses
- No circular dependencies in blocked_by relationships

Examples:
  tk context validate`,
	RunE: RunValidate,
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Output JSON Schema for Tasuku task files",
	Long: `Output the JSON Schema definition for Tasuku task files.

The schema defines:
  - version: Schema version (integer)
  - tasks: Object mapping task IDs to task objects
    - status: ready, in_progress, blocked, or done
    - description: Task description (string)
    - priority: 0-4 (0=critical, 4=backlog)
    - blocked_by: Array of task IDs this task depends on
    - owner: Optional owner identifier
    - created_at/updated_at: ISO 8601 timestamps
  - context: Shared knowledge object
    - learnings: Array of insight strings
    - decisions: Array of decision objects (chose/over/because)
    - notes: Object mapping task IDs to note arrays

Use Cases:
  - IDE validation: Configure your editor to validate Tasuku files
  - Documentation: Reference for file format
  - Tooling: Build tools that work with Tasuku files

Examples:
  tk context schema                   # Print schema to stdout
  tk context schema > tasuku.schema.json  # Save schema to file`,
	RunE: runSchema,
}

func runShow(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
	f, err := s.Read()
	if err != nil {
		return err
	}

	if config.OutputFormat == "yaml" {
		data, _ := yaml.Marshal(f)
		fmt.Print(string(data))
	} else {
		data, _ := json.MarshalIndent(f, "", "  ")
		fmt.Println(string(data))
	}
	return nil
}

// RunValidate validates the Tasuku storage for correctness.
func RunValidate(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
	f, err := s.Read()
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if f.Version < 1 || f.Version > 4 {
		return fmt.Errorf("unsupported version: %d", f.Version)
	}

	for id, t := range f.Tasks {
		if t.Description == "" {
			return fmt.Errorf("task %s has empty description", id)
		}
		switch t.Status {
		case task.StatusReady, task.StatusInProgress, task.StatusBlocked, task.StatusDone:
			// Valid
		default:
			return fmt.Errorf("task %s has invalid status: %s", id, t.Status)
		}
	}

	cycles := detectCircularDependencies(f.Tasks)
	if len(cycles) > 0 {
		fmt.Println("Circular dependencies detected:")
		for _, cycle := range cycles {
			fmt.Printf("  %s\n", strings.Join(cycle, " -> "))
		}
		return fmt.Errorf("found %d circular dependency chain(s)", len(cycles))
	}

	fmt.Println("Validation passed")
	return nil
}

func runSchema(cmd *cobra.Command, args []string) error {
	schema := `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://github.com/iheanyi/tasuku/schema.json",
  "title": "Tasuku File",
  "description": "Schema for Tasuku task management storage (V3 directory format or V4 Markdown format)",
  "type": "object",
  "required": ["version", "tasks", "context"],
  "properties": {
    "version": { "type": "integer", "enum": [1, 2, 3, 4] },
    "tasks": {
      "type": "object",
      "additionalProperties": {
        "type": "object",
        "required": ["status", "description", "blocked_by", "created_at", "updated_at"],
        "properties": {
          "status": { "type": "string", "enum": ["ready", "in_progress", "blocked", "done"] },
          "description": { "type": "string" },
          "priority": { "type": "integer", "minimum": 0, "maximum": 4 },
          "blocked_by": { "type": "array", "items": { "type": "string" } },
          "owner": { "type": ["string", "null"] },
          "claimed_at": { "type": ["string", "null"], "format": "date-time" },
          "parent_id": { "type": ["string", "null"] },
          "tags": { "type": "array", "items": { "type": "string" } },
          "fields": { "type": "object", "additionalProperties": { "type": "string" } },
          "timer_start": { "type": ["string", "null"], "format": "date-time" },
          "duration": { "type": "integer", "minimum": 0, "description": "Duration in nanoseconds" },
          "notes": {
            "type": "array",
            "items": {
              "type": "object",
              "required": ["id", "text", "created_at"],
              "properties": {
                "id": { "type": "string" },
                "text": { "type": "string" },
                "created_at": { "type": "string", "format": "date-time" }
              }
            }
          },
          "created_at": { "type": "string", "format": "date-time" },
          "updated_at": { "type": "string", "format": "date-time" }
        }
      }
    },
    "context": {
      "type": "object",
      "required": ["learnings", "decisions"],
      "properties": {
        "learnings": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["id", "text", "created_at"],
            "properties": {
              "id": { "type": "string" },
              "text": { "type": "string" },
              "is_rule": { "type": "boolean" },
              "created_at": { "type": "string", "format": "date-time" }
            }
          }
        },
        "decisions": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["id", "chose", "over", "because", "created_at"],
            "properties": {
              "id": { "type": "string" },
              "chose": { "type": "string" },
              "over": { "type": "array", "items": { "type": "string" } },
              "because": { "type": "string" },
              "created_at": { "type": "string", "format": "date-time" }
            }
          }
        }
      }
    },
    "archive": {
      "type": "object",
      "additionalProperties": {
        "type": "object",
        "required": ["original_task", "archived_at"],
        "properties": {
          "original_task": { "$ref": "#/properties/tasks/additionalProperties" },
          "summary": { "type": "string" },
          "archived_at": { "type": "string", "format": "date-time" }
        }
      }
    }
  }
}`
	fmt.Println(schema)
	return nil
}

// detectCircularDependencies finds all circular dependency chains in blocked_by relationships.
func detectCircularDependencies(tasks map[string]task.Task) [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	reportedCycles := make(map[string]bool)

	var dfs func(taskID string, path []string) bool
	dfs = func(taskID string, path []string) bool {
		if inStack[taskID] {
			cycleStart := -1
			for i, id := range path {
				if id == taskID {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := append(path[cycleStart:], taskID)
				cycleKey := normalizeCycle(cycle)
				if !reportedCycles[cycleKey] {
					reportedCycles[cycleKey] = true
					cycles = append(cycles, cycle)
				}
			}
			return true
		}

		if visited[taskID] {
			return false
		}

		visited[taskID] = true
		inStack[taskID] = true

		t, exists := tasks[taskID]
		if exists {
			for _, blockerID := range t.BlockedBy {
				dfs(blockerID, append(path, taskID))
			}
		}

		inStack[taskID] = false
		return false
	}

	for taskID := range tasks {
		if !visited[taskID] {
			dfs(taskID, []string{})
		}
	}

	return cycles
}

func normalizeCycle(cycle []string) string {
	if len(cycle) <= 1 {
		return strings.Join(cycle, ",")
	}
	nodes := cycle[:len(cycle)-1]
	minIdx := 0
	for i := 1; i < len(nodes); i++ {
		if nodes[i] < nodes[minIdx] {
			minIdx = i
		}
	}
	rotated := append(nodes[minIdx:], nodes[:minIdx]...)
	return strings.Join(rotated, ",")
}
