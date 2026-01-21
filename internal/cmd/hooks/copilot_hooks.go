package hooks

// Copilot CLI hooks configuration.
// Copilot CLI uses .github/hooks/*.json for hook configuration.
//
// JSON format:
//
//	{
//	  "version": 1,
//	  "hooks": {
//	    "sessionStart": [
//	      {
//	        "type": "command",
//	        "bash": "tk hooks session",
//	        "cwd": ".",
//	        "timeoutSec": 30
//	      }
//	    ]
//	  }
//	}
//
// Available hooks:
//   - sessionStart: Called when a new session begins
//   - sessionEnd: Called when a session ends
//   - userPromptSubmitted: Called when user submits a prompt
//   - preToolUse: Called before a tool is used
//   - postToolUse: Called after a tool is used
//   - errorOccurred: Called when an error occurs
//
// Note: Copilot CLI hooks are always local to the repository (.github/hooks/).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CopilotHooksConfig represents the .github/hooks/*.json structure
type CopilotHooksConfig struct {
	Version int                        `json:"version"`
	Hooks   map[string][]CopilotHook   `json:"hooks"`
}

// CopilotHook represents a single hook entry
type CopilotHook struct {
	Type       string            `json:"type"`
	Bash       string            `json:"bash"`
	Cwd        string            `json:"cwd,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	TimeoutSec int               `json:"timeoutSec,omitempty"`
}

const copilotHooksDir = ".github/hooks"
const copilotHooksFile = "tasuku.json"

func getCopilotHooksPath() string {
	return filepath.Join(copilotHooksDir, copilotHooksFile)
}

func readCopilotHooks() (*CopilotHooksConfig, error) {
	path := getCopilotHooksPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &CopilotHooksConfig{
				Version: 1,
				Hooks:   make(map[string][]CopilotHook),
			}, nil
		}
		return nil, err
	}

	var config CopilotHooksConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return &config, nil
}

func writeCopilotHooks(config *CopilotHooksConfig) error {
	path := getCopilotHooksPath()

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

// getTasukuCopilotHooks returns the hook configuration for Tasuku
func getTasukuCopilotHooks() map[string][]CopilotHook {
	// Use "tk" as the command - assumes tk is in PATH
	return map[string][]CopilotHook{
		"sessionStart": {
			{
				Type:       "command",
				Bash:       fmt.Sprintf("tk hooks session %s", tasukuHookMarker),
				Cwd:        ".",
				TimeoutSec: 30,
			},
		},
		"sessionEnd": {
			{
				Type:       "command",
				Bash:       fmt.Sprintf("tk hooks stop-reminder %s", tasukuHookMarker),
				Cwd:        ".",
				TimeoutSec: 30,
			},
		},
		"userPromptSubmitted": {
			{
				Type:       "command",
				Bash:       fmt.Sprintf("tk hooks prompt-check %s", tasukuHookMarker),
				Cwd:        ".",
				TimeoutSec: 30,
			},
		},
		"postToolUse": {
			{
				Type:       "command",
				Bash:       fmt.Sprintf("tk hooks todo-check %s", tasukuHookMarker),
				Cwd:        ".",
				TimeoutSec: 30,
			},
		},
	}
}

// installCopilotHooks installs Tasuku hooks for Copilot CLI.
// Copilot CLI hooks are always local to the repository in .github/hooks/.
func installCopilotHooks(force bool) error {
	// Check if .github directory exists (indicates a GitHub repo)
	if _, err := os.Stat(".github"); os.IsNotExist(err) {
		// Create .github directory if it doesn't exist
		if err := os.MkdirAll(".github", 0755); err != nil {
			return fmt.Errorf("failed to create .github directory: %w", err)
		}
	}

	config, err := readCopilotHooks()
	if err != nil {
		return err
	}

	tasukuHooks := getTasukuCopilotHooks()
	addedCount := 0
	updatedCount := 0
	unchangedCount := 0

	for hookType, desiredHooks := range tasukuHooks {
		existing := config.Hooks[hookType]

		if force {
			// Remove existing Tasuku hooks
			var filtered []CopilotHook
			for _, h := range existing {
				if !isTasukuCopilotHook(h) {
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
				if isTasukuCopilotHook(h) {
					foundIdx = i
					break
				}
			}

			if foundIdx == -1 {
				// Hook doesn't exist - add it
				existing = append(existing, desiredHook)
				addedCount++
				fmt.Printf("  + Added: %s\n", hookType)
			} else if existing[foundIdx].Bash != desiredHook.Bash {
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

	if err := writeCopilotHooks(config); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("Tasuku hooks installed for Copilot CLI:\n")
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
	fmt.Printf("Hooks written to: %s\n", getCopilotHooksPath())
	fmt.Println("Restart Copilot CLI for hooks to take effect.")

	return nil
}

// uninstallCopilotHooks removes Tasuku hooks from Copilot CLI configuration.
func uninstallCopilotHooks() error {
	path := getCopilotHooksPath()

	// Check if hooks file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("No Copilot CLI hooks file found.")
		return nil
	}

	config, err := readCopilotHooks()
	if err != nil {
		return err
	}

	removedCount := 0

	for hookType, hooks := range config.Hooks {
		var filtered []CopilotHook
		for _, h := range hooks {
			if isTasukuCopilotHook(h) {
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
		fmt.Println("No Tasuku hooks found in Copilot CLI configuration.")
		return nil
	}

	// If all hooks are removed, delete the file
	if len(config.Hooks) == 0 {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove empty hooks file: %w", err)
		}
		fmt.Printf("Removed %d Tasuku hook(s) from Copilot CLI.\n", removedCount)
		fmt.Printf("Deleted empty hooks file: %s\n", path)
		return nil
	}

	if err := writeCopilotHooks(config); err != nil {
		return err
	}

	fmt.Printf("Removed %d Tasuku hook(s) from Copilot CLI.\n", removedCount)
	return nil
}

// isTasukuCopilotHook checks if a hook is a Tasuku hook
func isTasukuCopilotHook(h CopilotHook) bool {
	return containsTasukuMarker(h.Bash)
}
