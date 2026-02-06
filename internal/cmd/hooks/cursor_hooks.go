package hooks

// Cursor editor hooks configuration.
// Cursor uses .cursor/hooks.json for hook configuration.
//
// JSON format:
//
//	{
//	  "version": 1,
//	  "hooks": {
//	    "sessionStart": [
//	      {
//	        "command": "tk hooks session --tasuku-hook",
//	        "timeout": 30
//	      }
//	    ]
//	  }
//	}
//
// Available hooks:
//   - sessionStart: Called when a new session begins
//   - stop: Called when a session ends
//   - preToolUse: Called before a tool is used
//   - postToolUse: Called after a tool is used
//   - beforeShellExecution: Called before shell execution
//   - afterShellExecution: Called after shell execution
//   - preCompact: Called before context compaction
//   - beforeMCPExecution: Called before MCP execution
//   - afterMCPExecution: Called after MCP execution
//   - beforeReadFile: Called before reading a file
//   - afterFileEdit: Called after editing a file
//   - beforeSubmitPrompt: Called before submitting a prompt
//
// Config file locations:
//   - Global: ~/.cursor/hooks.json
//   - Local (project): .cursor/hooks.json

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CursorHooksConfig represents the .cursor/hooks.json structure
type CursorHooksConfig struct {
	Version int                       `json:"version"`
	Hooks   map[string][]CursorHook   `json:"hooks"`
}

// CursorHook represents a single hook entry
type CursorHook struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
	Matcher string `json:"matcher,omitempty"`
}

const cursorHooksFile = "hooks.json"

func getCursorHooksPath(local bool) string {
	if local {
		return filepath.Join(".cursor", cursorHooksFile)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cursor", cursorHooksFile)
}

func readCursorHooks(local bool) (*CursorHooksConfig, error) {
	path := getCursorHooksPath(local)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &CursorHooksConfig{
				Version: 1,
				Hooks:   make(map[string][]CursorHook),
			}, nil
		}
		return nil, err
	}

	var config CursorHooksConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return &config, nil
}

func writeCursorHooks(config *CursorHooksConfig, local bool) error {
	path := getCursorHooksPath(local)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// getTasukuCursorHooks returns the hook configuration for Tasuku in Cursor format
func getTasukuCursorHooks() map[string][]CursorHook {
	return map[string][]CursorHook{
		"sessionStart": {
			{
				Command: fmt.Sprintf("tk hooks session %s", tasukuHookMarker),
				Timeout: 30,
			},
		},
		"stop": {
			{
				Command: fmt.Sprintf("tk hooks stop-reminder %s", tasukuHookMarker),
				Timeout: 30,
			},
		},
		"preCompact": {
			{
				Command: fmt.Sprintf("tk hooks pre-compact %s", tasukuHookMarker),
				Timeout: 30,
			},
		},
		"postToolUse": {
			{
				Command: fmt.Sprintf("tk hooks todo-check %s", tasukuHookMarker),
				Timeout: 30,
			},
		},
		"beforeSubmitPrompt": {
			{
				Command: fmt.Sprintf("tk hooks prompt-check %s", tasukuHookMarker),
				Timeout: 30,
			},
		},
	}
}

// isTasukuCursorHook checks if a hook is a Tasuku hook
func isTasukuCursorHook(h CursorHook) bool {
	return containsTasukuMarker(h.Command)
}

// installCursorHooks installs Tasuku hooks for Cursor editor.
// Uses smart incremental updates:
//   - Adds new hooks without touching existing ones
//   - Updates hooks whose commands have changed
//   - Preserves user's non-Tasuku hooks
//
// If force is true, all Tasuku hooks are replaced (useful for downgrading).
// If local is true, installs to ./.cursor/hooks.json instead of ~/.cursor/hooks.json.
func installCursorHooks(force, local bool) error {
	config, err := readCursorHooks(local)
	if err != nil {
		return err
	}

	tasukuHooks := getTasukuCursorHooks()
	addedCount := 0
	updatedCount := 0
	unchangedCount := 0

	for hookType, desiredHooks := range tasukuHooks {
		existing := config.Hooks[hookType]

		if force {
			// Remove existing Tasuku hooks
			var filtered []CursorHook
			for _, h := range existing {
				if !isTasukuCursorHook(h) {
					filtered = append(filtered, h)
				}
			}
			existing = filtered

			// Add desired hooks
			for _, hook := range desiredHooks {
				existing = append(existing, hook)
				addedCount++
			}
			config.Hooks[hookType] = existing
			continue
		}

		// Smart incremental update
		for _, desiredHook := range desiredHooks {
			foundIdx := -1
			for i, h := range existing {
				if isTasukuCursorHook(h) {
					foundIdx = i
					break
				}
			}

			if foundIdx == -1 {
				// Hook doesn't exist - add it
				existing = append(existing, desiredHook)
				addedCount++
				fmt.Printf("  + Added: %s\n", hookType)
			} else if existing[foundIdx].Command != desiredHook.Command {
				// Hook exists but command changed - update it
				existing[foundIdx] = desiredHook
				updatedCount++
				fmt.Printf("  ~ Updated: %s\n", hookType)
			} else {
				unchangedCount++
			}
		}

		config.Hooks[hookType] = existing
	}

	totalChanges := addedCount + updatedCount
	if totalChanges == 0 {
		fmt.Printf("All %d Tasuku hooks are up to date.\n", unchangedCount)
		return nil
	}

	if err := writeCursorHooks(config, local); err != nil {
		return err
	}

	location := "global"
	if local {
		location = "project"
	}
	fmt.Println()
	fmt.Printf("Tasuku hooks installed for Cursor (%s):\n", location)
	if addedCount > 0 {
		fmt.Printf("  %d added\n", addedCount)
	}
	if updatedCount > 0 {
		fmt.Printf("  %d updated\n", updatedCount)
	}
	if unchangedCount > 0 {
		fmt.Printf("  %d unchanged\n", unchangedCount)
	}
	fmt.Println()
	fmt.Printf("Hooks written to: %s\n", getCursorHooksPath(local))
	fmt.Println("Restart Cursor for hooks to take effect.")

	return nil
}

// uninstallCursorHooks removes Tasuku hooks from Cursor configuration.
// Other hooks configured by the user are preserved.
// If local is true, removes from ./.cursor/hooks.json instead of ~/.cursor/hooks.json.
func uninstallCursorHooks(local bool) error {
	path := getCursorHooksPath(local)

	// Check if hooks file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		location := "global"
		if local {
			location = "project"
		}
		fmt.Printf("No Cursor hooks file found (%s).\n", location)
		return nil
	}

	config, err := readCursorHooks(local)
	if err != nil {
		return err
	}

	removedCount := 0

	for hookType, hooks := range config.Hooks {
		var filtered []CursorHook
		for _, h := range hooks {
			if isTasukuCursorHook(h) {
				removedCount++
				continue
			}
			filtered = append(filtered, h)
		}

		if len(filtered) == 0 {
			delete(config.Hooks, hookType)
		} else {
			config.Hooks[hookType] = filtered
		}
	}

	if removedCount == 0 {
		fmt.Println("No Tasuku hooks found in Cursor configuration.")
		return nil
	}

	// If all hooks are removed, delete the file
	if len(config.Hooks) == 0 {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove empty hooks file: %w", err)
		}
		location := "global"
		if local {
			location = "project"
		}
		fmt.Printf("Removed %d Tasuku hook(s) from Cursor (%s).\n", removedCount, location)
		fmt.Printf("Deleted empty hooks file: %s\n", path)
		return nil
	}

	if err := writeCursorHooks(config, local); err != nil {
		return err
	}

	location := "global"
	if local {
		location = "project"
	}
	fmt.Printf("Removed %d Tasuku hook(s) from Cursor (%s).\n", removedCount, location)
	return nil
}
