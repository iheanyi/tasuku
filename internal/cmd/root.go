// Package cmd provides the CLI commands for tasuku.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/cmd/agentsmd"
	"github.com/iheanyi/tasuku/internal/cmd/claudemd"
	"github.com/iheanyi/tasuku/internal/cmd/config"
	contextcmd "github.com/iheanyi/tasuku/internal/cmd/context"
	"github.com/iheanyi/tasuku/internal/cmd/decision"
	"github.com/iheanyi/tasuku/internal/cmd/hooks"
	"github.com/iheanyi/tasuku/internal/cmd/learning"
	"github.com/iheanyi/tasuku/internal/cmd/mcpcmd"
	"github.com/iheanyi/tasuku/internal/cmd/migrate"
	"github.com/iheanyi/tasuku/internal/cmd/note"
	plugincmd "github.com/iheanyi/tasuku/internal/cmd/plugin"
	"github.com/iheanyi/tasuku/internal/cmd/pr"
	rulescmd "github.com/iheanyi/tasuku/internal/cmd/rules"
	"github.com/iheanyi/tasuku/internal/cmd/serve"
	taskcmd "github.com/iheanyi/tasuku/internal/cmd/task"
	"github.com/iheanyi/tasuku/internal/cmd/ui"
	"github.com/iheanyi/tasuku/internal/mcp"
	"github.com/iheanyi/tasuku/internal/store"
	v4 "github.com/iheanyi/tasuku/internal/store/v4"
	"github.com/iheanyi/tasuku/internal/task"
	"github.com/iheanyi/tasuku/internal/version"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tk",
		Short: "Tasuku - agent-first task management",
		Long: `tk is an agent-first task management system designed for AI agents
working on codebases.

Design Principles:
  - Pull over push: Agents query when needed, no constant injections
  - Parallel-safe: File locking for multiple simultaneous agents
  - Minimal context: Only load what's needed for the current task
  - Human-readable: JSON file that can be edited by hand

Getting Started:
  tk init                  # Create .tasuku/ directory
  tk task add "My task"    # Add a task
  tk task list             # View all tasks
  tk task start <id>       # Begin working on a task
  tk task done <id>        # Mark task complete

AI Tool Integration:
  tk mcp install           # Auto-configure MCP for Claude Code/Cursor
  tk serve mcp             # Start MCP server (for AI tools)

For full documentation: https://github.com/iheanyi/tasuku`,
		Version: version.Version(),
	}

	// Global flags
	cmd.PersistentFlags().StringVarP(&config.OutputFormat, "format", "f", "table", "Output format: table, json, yaml")

	// Register all subcommands
	cmd.AddCommand(taskcmd.Cmd)
	cmd.AddCommand(learning.Cmd)
	cmd.AddCommand(decision.Cmd)
	cmd.AddCommand(note.Cmd)
	cmd.AddCommand(contextcmd.Cmd)
	cmd.AddCommand(serve.Cmd)
	cmd.AddCommand(mcpcmd.Cmd)
	cmd.AddCommand(hooks.Cmd)
	cmd.AddCommand(migrate.Cmd)
	cmd.AddCommand(pr.Cmd)
	cmd.AddCommand(ui.Cmd)
	cmd.AddCommand(plugincmd.Cmd)
	cmd.AddCommand(rulescmd.Cmd)
	cmd.AddCommand(claudemd.Cmd)
	cmd.AddCommand(agentsmd.Cmd)

	// Root-level commands
	cmd.AddCommand(initCmd)
	cmd.AddCommand(doctorCmd)
	cmd.AddCommand(newValidateCmd())

	// Shortcut commands (documented in CLAUDE.md)
	cmd.AddCommand(newLearnShortcutCmd())
	cmd.AddCommand(newDecideShortcutCmd())
	cmd.AddCommand(newHealthCmd())
	cmd.AddCommand(newSuggestCmd())

	return cmd
}

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate Tasuku storage for correctness",
		Long: `Validate the Tasuku storage for correctness.

Checks performed:
  - Version is supported
  - All tasks have non-empty descriptions
  - All tasks have valid statuses
  - No circular dependencies in blocked_by relationships
  - Referenced blockers exist

This is the same as 'tk context validate'.

Examples:
  tk validate              # Validate storage
  tk validate --format json  # Output as JSON`,
		RunE: contextcmd.RunValidate,
	}
}

// RootCmd is the base command for tk
var RootCmd = newRootCmd()

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Tasuku in current directory",
	Long: `Initialize a new Tasuku project in the current directory.

Creates a .tasuku/ directory with V4 Markdown format:
  - tasks/              Individual task Markdown files with YAML frontmatter
  - archive/            Completed tasks that have been archived
  - context/learnings.md  Discovered insights and knowledge
  - context/decisions.md  Documented architectural choices
  - index.json          Auto-generated index for fast queries

Benefits:
  - Human-readable Markdown with rich formatting support
  - Clean git diffs with one file per task
  - Fast queries via auto-generated index.json
  - Safe for multiple agents working in parallel

If you have a legacy .tasuku.json file, use 'tk migrate v3' then 'tk migrate v4'.

Examples:
  tk init                    # Create .tasuku/ directory (V4 format)
  tk init && tk task add "Setup"  # Initialize and add first task`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if already initialized
		storageType := store.DetectStorageType(".")
		if storageType != store.StorageTypeNone {
			switch storageType {
			case store.StorageTypeDirV4:
				return fmt.Errorf(".tasuku/ directory already exists (V4 Markdown format)")
			case store.StorageTypeDir:
				return fmt.Errorf(".tasuku/ directory already exists (V3 JSON format) - run 'tk migrate v4' to upgrade")
			default:
				return fmt.Errorf(".tasuku.json already exists - run 'tk migrate v3' then 'tk migrate v4' to upgrade")
			}
		}

		s := v4.New(store.DirName)
		if err := s.Init(); err != nil {
			return err
		}
		fmt.Println("Created .tasuku/ directory (V4 Markdown format)")
		fmt.Println("  tasks/              - Task Markdown files with YAML frontmatter")
		fmt.Println("  archive/            - Archived completed tasks")
		fmt.Println("  context/learnings.md - Insights and knowledge")
		fmt.Println("  context/decisions.md - Architectural decisions")
		fmt.Println("  index.json          - Auto-generated index for fast queries")
		fmt.Println()
		fmt.Println("Tip: Commit .tasuku/ to git so tasks travel with your code.")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  tk task add \"Your first task\"")
		fmt.Println("  tk hooks install              # Optional: git hooks")
		fmt.Println("  tk mcp install                # Optional: AI tool integration (Claude Code, Cursor)")
		return nil
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose Tasuku setup and MCP configuration",
	Long: `Check your Tasuku installation and MCP configuration for common issues.

This command verifies:
  - tk binary is accessible and shows its location
  - Tasuku storage exists (.tasuku/ directory or .tasuku.json file)
  - MCP is configured in Claude Code, Cursor, and other AI tools
  - The configured binary path matches the current tk installation
  - The MCP server can start and respond to requests

Run this when Tasuku tools aren't appearing in your AI assistant.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
	},
}

func runDoctor() error {
	fmt.Println("Tasuku Doctor")
	fmt.Println("=============")
	fmt.Println()

	hasErrors := false

	// 1. Check tk binary
	executable, err := os.Executable()
	if err != nil {
		fmt.Println("✗ Could not determine tk binary location")
		hasErrors = true
	} else {
		fmt.Printf("✓ tk binary: %s\n", executable)
	}

	// 2. Check Tasuku storage
	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		return err
	}
	tasukuPath := s.Path()
	if !s.Exists() {
		fmt.Printf("✗ No Tasuku storage found (searched from %s)\n", mustGetwd())
		fmt.Println("  Run 'tk init' to create .tasuku/ directory")
		hasErrors = true
	} else {
		fmt.Printf("✓ Tasuku storage: %s\n", tasukuPath)
	}

	fmt.Println()
	fmt.Println("MCP Configuration")
	fmt.Println("-----------------")

	// 3. Check AI tool configurations
	tools := getSupportedAITools()
	configuredTools := 0
	mismatchedPaths := []string{}
	projectRoot := ""
	if s.Exists() {
		projectRoot = filepath.Dir(tasukuPath)
	}

	for _, tool := range tools {
		settingsPath := tool.SettingsPath
		if projectRoot != "" && !filepath.IsAbs(settingsPath) {
			settingsPath = filepath.Join(projectRoot, settingsPath)
		}

		// Check if settings file exists
		if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
			continue
		}

		data, err := os.ReadFile(settingsPath)
		if err != nil {
			continue
		}

		var settings map[string]interface{}
		if err := json.Unmarshal(data, &settings); err != nil {
			continue
		}

		mcpServers, ok := settings[tool.MCPKey].(map[string]interface{})
		if !ok {
			continue
		}

		tasukuConfig, exists := mcpServers["tasuku"].(map[string]interface{})
		if !exists {
			fmt.Printf("✗ %s: MCP not configured\n", tool.Name)
			fmt.Printf("  Run 'tk mcp install' to configure\n")
			hasErrors = true
			continue
		}

		var configuredPath string
		var configuredArgs []interface{}

		// Handle command as string (Claude/Cursor) or array (OpenCode)
		if cmdStr, ok := tasukuConfig["command"].(string); ok {
			configuredPath = cmdStr
			configuredArgs, _ = tasukuConfig["args"].([]interface{})
		} else if cmdArr, ok := tasukuConfig["command"].([]interface{}); ok && len(cmdArr) > 0 {
			if pathStr, ok := cmdArr[0].(string); ok {
				configuredPath = pathStr
			}
			if len(cmdArr) > 1 {
				configuredArgs = cmdArr[1:]
			}
		}

		// Check if path matches current executable
		if configuredPath != executable {
			fmt.Printf("⚠ %s: configured but path mismatch\n", tool.Name)
			fmt.Printf("  Configured: %s\n", configuredPath)
			fmt.Printf("  Current:    %s\n", executable)
			mismatchedPaths = append(mismatchedPaths, tool.Name)
		} else {
			argsStr := ""
			for _, arg := range configuredArgs {
				if s, ok := arg.(string); ok {
					argsStr += s + " "
				}
			}
			fmt.Printf("✓ %s: configured (%s %s)\n", tool.Name, filepath.Base(configuredPath), strings.TrimSpace(argsStr))
		}
		configuredTools++
	}

	if configuredTools == 0 {
		fmt.Println("✗ No AI tools have Tasuku MCP configured")
		fmt.Println("  Run 'tk mcp install' to auto-configure")
		hasErrors = true
	}

	// 4. Test MCP server
	fmt.Println()
	fmt.Println("MCP Server Test")
	fmt.Println("---------------")

	// Quick test: can we create a server and get tools?
	if _, err := os.Stat(tasukuPath); err == nil {
		mcpServer := mcp.New(s)
		mcpTools := mcpServer.Tools()
		fmt.Printf("✓ MCP server responds with %d tools\n", len(mcpTools))

		// List a few tools
		if len(mcpTools) > 0 {
			toolNames := []string{}
			for _, t := range mcpTools {
				toolNames = append(toolNames, t.Name)
			}
			if len(toolNames) > 5 {
				toolNames = toolNames[:5]
				fmt.Printf("  Tools: %s, ... (+%d more)\n", strings.Join(toolNames, ", "), len(mcpTools)-5)
			} else {
				fmt.Printf("  Tools: %s\n", strings.Join(toolNames, ", "))
			}
		}
	} else {
		fmt.Println("⚠ Cannot test MCP server (no Tasuku storage)")
	}

	// 5. Check CLI/MCP parity
	fmt.Println()
	fmt.Println("CLI/MCP Parity")
	fmt.Println("--------------")

	if _, err := os.Stat(tasukuPath); err == nil {
		mcpServer := mcp.New(s)
		mcpTools := mcpServer.Tools()

		// Build set of MCP tool names
		mcpToolSet := make(map[string]bool)
		for _, t := range mcpTools {
			mcpToolSet[t.Name] = true
		}

		// Define expected MCP tools for CLI commands.
		cliToMCP := doctorCLIToMCPMap()

		missingTools := []string{}
		for cli, expectedTools := range cliToMCP {
			for _, tool := range expectedTools {
				if !mcpToolSet[tool] {
					missingTools = append(missingTools, fmt.Sprintf("%s (missing %s)", cli, tool))
				}
			}
		}

		if len(missingTools) == 0 {
			fmt.Printf("✓ All %d CLI commands have corresponding MCP tools\n", len(cliToMCP))
		} else {
			fmt.Printf("✗ %d CLI commands missing MCP tools:\n", len(missingTools))
			for _, m := range missingTools {
				fmt.Printf("  - %s\n", m)
			}
			hasErrors = true
		}
	} else {
		fmt.Println("⚠ Cannot check parity (no Tasuku storage)")
	}

	// Summary
	fmt.Println()
	if hasErrors {
		fmt.Println("Issues found. See recommendations above.")
		if configuredTools > 0 {
			fmt.Println()
			fmt.Println("If MCP is configured but tools aren't visible:")
			fmt.Println("  1. Restart your AI tool (Claude Code, Cursor, etc.)")
			fmt.Println("  2. Run '/mcp' in Claude Code to check MCP status")
		}
		return nil
	}

	if len(mismatchedPaths) > 0 {
		fmt.Println("Configuration path mismatch detected.")
		fmt.Println("Run 'tk mcp install' to update configuration.")
		return nil
	}

	fmt.Println("Everything looks good!")
	fmt.Println()
	fmt.Println("If tools still aren't visible in your AI assistant:")
	fmt.Println("  1. Restart your AI tool (Claude Code, Cursor, etc.)")
	fmt.Println("  2. Run '/mcp' in Claude Code to check MCP status")

	return nil
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func doctorCLIToMCPMap() map[string][]string {
	return map[string][]string{
		// Task commands
		"task list":     {"tk_list"},
		"task add":      {"tk_add"},
		"task show":     {"tk_show"},
		"task start":    {"tk_start"},
		"task done":     {"tk_done"},
		"task block":    {"tk_block"},
		"task unblock":  {"tk_task"},
		"task pause":    {"tk_task"},
		"task find":     {"tk_find"},
		"task priority": {"tk_task"},
		"task delete":   {"tk_task"},
		"task edit":     {"tk_task"},
		"task owner":    {"tk_task"},
		"task claim":    {"tk_task"},
		"task release":  {"tk_task"},
		"task ready":    {"tk_ready"},
		"task who":      {"tk_task"},
		"task deps":     {"tk_deps"},
		"task stats":    {"tk_stats"},
		"task tag":      {"tk_metadata"},
		"task field":    {"tk_metadata"},
		"task archive":  {"tk_task", "tk_manage"},
		// Context commands
		"learn":        {"tk_learn"},
		"decide":       {"tk_decide"},
		"note":         {"tk_note"},
		"context show": {"tk_context"},
		// Learning management
		"learning list":    {"tk_manage"},
		"learning promote": {"tk_manage"},
		"learning remove":  {"tk_manage"},
		"learning rules":   {"tk_manage"},
		// Decision management
		"decision list":   {"tk_manage"},
		"decision remove": {"tk_manage"},
		// Note management
		"note list":   {"tk_metadata"},
		"note remove": {"tk_metadata"},
		// Root commands
		"suggest": {"tk_suggest"},
		"health":  {"tk_health"},
	}
}

// newLearnShortcutCmd creates a top-level shortcut for adding learnings.
// This allows "tk learn 'insight'" instead of "tk learning add 'insight'".
func newLearnShortcutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "learn \"insight\"",
		Short: "Record an insight (shortcut for 'tk learning add')",
		Long: `Record an insight or knowledge discovered during work.
This is a shortcut for 'tk learning add'.

Examples:
  tk learn "Redis connection pooling significantly improves API latency"
  tk learn "The auth middleware must run before rate limiting" --permanent
  tk learn "Never use raw SQL queries" --rule`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			learningText := args[0]
			permanent, _ := cmd.Flags().GetBool("permanent")
			forceRule, _ := cmd.Flags().GetBool("rule")
			s, err := store.DefaultStorageWithWarning()
			if err != nil {
				return err
			}

			var id string
			var isRule bool

			if forceRule {
				ruleVal := true
				id, isRule, err = s.AddLearningWithRule(learningText, &ruleVal)
			} else {
				id, isRule, err = s.AddLearningWithRule(learningText, nil)
			}
			if err != nil {
				return err
			}

			if permanent {
				if err := appendLearningToCLAUDEmd(learningText); err != nil {
					fmt.Printf("Warning: could not append to CLAUDE.md: %v\n", err)
				} else {
					fmt.Printf("Learning added [%s] (also appended to CLAUDE.md)\n", id)
					return nil
				}
			}

			if isRule {
				fmt.Printf("Learning added [%s] [RULE]\n", id)
				fmt.Println("Hint: Promote rules to permanent docs with: tk learning promote", id)
			} else {
				fmt.Printf("Learning added [%s]\n", id)
			}
			return nil
		},
	}

	cmd.Flags().Bool("permanent", false, "Also append learning to CLAUDE.md")
	cmd.Flags().Bool("rule", false, "Explicitly mark this learning as a rule")

	return cmd
}

// newDecideShortcutCmd creates a top-level shortcut for adding decisions.
// This allows "tk decide --id X --chose Y --because Z" instead of "tk decision add ...".
func newDecideShortcutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decide --id <id> --chose <option> --because <reason>",
		Short: "Record a decision (shortcut for 'tk decision add')",
		Long: `Document an architectural or design decision.
This is a shortcut for 'tk decision add'.

Examples:
  tk decide --id db-choice --chose PostgreSQL --over MySQL,SQLite --because "Better JSON support"
  tk decide --id auth-method --chose JWT --over sessions --because "Stateless and scalable"
  tk decide --id framework --chose Cobra --because "Standard Go CLI library"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := cmd.Flags().GetString("id")
			chose, _ := cmd.Flags().GetString("chose")
			alternatives, _ := cmd.Flags().GetStringSlice("over")
			because, _ := cmd.Flags().GetString("because")

			if id == "" || chose == "" || because == "" {
				return fmt.Errorf("usage: tk decide --id <id> --chose <choice> --over <options> --because <reason>")
			}

			for i := range alternatives {
				alternatives[i] = strings.TrimSpace(alternatives[i])
			}

			d := task.Decision{
				ID:      id,
				Chose:   chose,
				Over:    alternatives,
				Because: because,
			}

			s, err := store.DefaultStorageWithWarning()
			if err != nil {
				return err
			}
			if err := s.AddDecision(d); err != nil {
				return err
			}

			fmt.Printf("Decision recorded: %s\n", id)
			return nil
		},
	}

	cmd.Flags().String("id", "", "Decision ID")
	cmd.Flags().String("chose", "", "The option chosen")
	cmd.Flags().StringSlice("over", nil, "Alternatives considered (repeatable or comma-separated)")
	cmd.Flags().String("because", "", "Reasoning")
	cmd.MarkFlagRequired("id")
	cmd.MarkFlagRequired("chose")
	cmd.MarkFlagRequired("because")

	return cmd
}

// appendLearningToCLAUDEmd appends a learning to CLAUDE.md
func appendLearningToCLAUDEmd(content string) error {
	claudePath := "CLAUDE.md"
	existing, err := os.ReadFile(claudePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	text := string(existing)
	entry := fmt.Sprintf("- %s\n", content)

	if strings.Contains(text, "## Learnings") {
		idx := strings.Index(text, "## Learnings")
		endOfLine := strings.Index(text[idx:], "\n") + idx + 1
		nextSection := strings.Index(text[endOfLine:], "\n## ")
		if nextSection == -1 {
			text = text + entry
		} else {
			insertAt := endOfLine + nextSection
			text = text[:insertAt] + entry + text[insertAt:]
		}
	} else {
		text = text + "\n\n## Learnings\n\n" + entry
	}

	return os.WriteFile(claudePath, []byte(text), 0644)
}
