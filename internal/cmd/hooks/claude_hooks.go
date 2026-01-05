package hooks

// Claude Code hooks configuration format.
// See official docs for schema reference:
//   - https://code.claude.com/docs/en/hooks.md (reference)
//   - https://code.claude.com/docs/en/hooks-guide.md (guide)
//
// Hook structure in ~/.claude/settings.json:
//
//	{
//	  "hooks": {
//	    "EventName": [
//	      {
//	        "matcher": "ToolPattern",
//	        "hooks": [
//	          { "type": "command", "command": "your-command" }
//	        ]
//	      }
//	    ]
//	  }
//	}
//
// Events: PreToolUse, PostToolUse, PermissionRequest, UserPromptSubmit,
//         Notification, Stop, SubagentStop, PreCompact, SessionStart, SessionEnd
//
// Exit codes: 0 = success, 2 = blocking error (stderr fed to Claude)

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ClaudeSettings represents the Claude Code settings.json structure
type ClaudeSettings struct {
	MCPServers map[string]interface{} `json:"mcpServers,omitempty"`
	Hooks      *ClaudeHooks           `json:"hooks,omitempty"`
	// Preserve other fields
	Other map[string]interface{} `json:"-"`
}

// ClaudeHooks represents the hooks section
type ClaudeHooks struct {
	PreToolUse  []ClaudeHook `json:"PreToolUse,omitempty"`
	PostToolUse []ClaudeHook `json:"PostToolUse,omitempty"`
}

// ClaudeHook represents a single hook
type ClaudeHook struct {
	Matcher string `json:"matcher"`
	Command string `json:"command"`
}

const tasukuHookMarker = "# tasuku-hook"

func getClaudeSettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

func readClaudeSettings() (map[string]interface{}, error) {
	path := getClaudeSettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, err
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings: %w", err)
	}
	return settings, nil
}

func writeClaudeSettings(settings map[string]interface{}) error {
	path := getClaudeSettingsPath()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

func getTasukuClaudeHooks() map[string][]map[string]interface{} {
	// Use "tk" as the command - assumes tk is in PATH after installation
	// This is more portable than using os.Executable() which returns temp build paths
	executable := "tk"

	return map[string][]map[string]interface{}{
		"PreToolUse": {},
		"PostToolUse": {
			{
				"matcher": "ExitPlanMode",
				"hooks": []map[string]string{
					{
						"type":    "command",
						"command": fmt.Sprintf("%s hooks plan-sync \"$PLAN_FILE\" --dry-run %s", executable, tasukuHookMarker),
					},
				},
			},
		},
	}
}

// installClaudeHooks installs Tasuku hooks in Claude Code settings.
// If force is true, existing Tasuku hooks will be replaced.
// If force is false and hooks already exist, they will be skipped with a message.
func installClaudeHooks(force bool) error {
	settings, err := readClaudeSettings()
	if err != nil {
		return err
	}

	// Get or create hooks section
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		hooks = make(map[string]interface{})
	}

	tasukuHooks := getTasukuClaudeHooks()
	installedCount := 0

	for hookType, newHooks := range tasukuHooks {
		if len(newHooks) == 0 {
			continue
		}

		existing, _ := hooks[hookType].([]interface{})

		// Check for existing Tasuku hooks
		hasExisting := false
		for _, h := range existing {
			if hook, ok := h.(map[string]interface{}); ok {
				// Check nested hooks array format
				if innerHooks, ok := hook["hooks"].([]interface{}); ok {
					for _, ih := range innerHooks {
						if innerHook, ok := ih.(map[string]interface{}); ok {
							if command, ok := innerHook["command"].(string); ok {
								if containsTasukuMarker(command) {
									hasExisting = true
									break
								}
							}
						}
					}
				}
				// Also check simple command format (backwards compat)
				if command, ok := hook["command"].(string); ok {
					if containsTasukuMarker(command) {
						hasExisting = true
						break
					}
				}
			}
		}

		if hasExisting && !force {
			fmt.Printf("Tasuku %s hooks already installed (use --force to overwrite)\n", hookType)
			continue
		}

		// Remove existing Tasuku hooks if force
		if hasExisting && force {
			var filtered []interface{}
			for _, h := range existing {
				if hook, ok := h.(map[string]interface{}); ok {
					isTasuku := false
					// Check nested hooks array format
					if innerHooks, ok := hook["hooks"].([]interface{}); ok {
						for _, ih := range innerHooks {
							if innerHook, ok := ih.(map[string]interface{}); ok {
								if command, ok := innerHook["command"].(string); ok {
									if containsTasukuMarker(command) {
										isTasuku = true
										break
									}
								}
							}
						}
					}
					// Also check simple command format (backwards compat)
					if command, ok := hook["command"].(string); ok {
						if containsTasukuMarker(command) {
							isTasuku = true
						}
					}
					if !isTasuku {
						filtered = append(filtered, h)
					}
				} else {
					filtered = append(filtered, h)
				}
			}
			existing = filtered
		}

		// Add new Tasuku hooks
		for _, newHook := range newHooks {
			// Copy all fields from the hook definition
			hookMap := make(map[string]interface{})
			for k, v := range newHook {
				hookMap[k] = v
			}
			existing = append(existing, hookMap)
			installedCount++
		}

		hooks[hookType] = existing
	}

	if installedCount == 0 {
		fmt.Println("No hooks to install.")
		return nil
	}

	settings["hooks"] = hooks

	if err := writeClaudeSettings(settings); err != nil {
		return err
	}

	fmt.Printf("Installed %d Tasuku hook(s) in Claude Code:\n", installedCount)
	fmt.Println("  - PostToolUse/ExitPlanMode: prompts to sync plan to tasks")
	fmt.Println()
	fmt.Println("Restart Claude Code for hooks to take effect.")

	return nil
}

// uninstallClaudeHooks removes Tasuku hooks from Claude Code settings.
// Other hooks configured by the user are preserved.
func uninstallClaudeHooks() error {
	settings, err := readClaudeSettings()
	if err != nil {
		return err
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		fmt.Println("No Claude Code hooks configured.")
		return nil
	}

	removedCount := 0

	for hookType := range hooks {
		existing, ok := hooks[hookType].([]interface{})
		if !ok {
			continue
		}

		var filtered []interface{}
		for _, h := range existing {
			if hook, ok := h.(map[string]interface{}); ok {
				isTasuku := false
				// Check nested hooks array format
				if innerHooks, ok := hook["hooks"].([]interface{}); ok {
					for _, ih := range innerHooks {
						if innerHook, ok := ih.(map[string]interface{}); ok {
							if command, ok := innerHook["command"].(string); ok {
								if containsTasukuMarker(command) {
									isTasuku = true
									break
								}
							}
						}
					}
				}
				// Also check simple command format (backwards compat)
				if command, ok := hook["command"].(string); ok {
					if containsTasukuMarker(command) {
						isTasuku = true
					}
				}
				if isTasuku {
					removedCount++
					continue
				}
			}
			filtered = append(filtered, h)
		}

		if len(filtered) == 0 {
			delete(hooks, hookType)
		} else {
			hooks[hookType] = filtered
		}
	}

	if removedCount == 0 {
		fmt.Println("No Tasuku hooks found in Claude Code settings.")
		return nil
	}

	// Clean up empty hooks section
	if len(hooks) == 0 {
		delete(settings, "hooks")
	} else {
		settings["hooks"] = hooks
	}

	if err := writeClaudeSettings(settings); err != nil {
		return err
	}

	fmt.Printf("Removed %d Tasuku hook(s) from Claude Code.\n", removedCount)
	fmt.Println("Restart Claude Code for changes to take effect.")

	return nil
}

func containsTasukuMarker(command string) bool {
	return len(command) > 0 && (contains(command, tasukuHookMarker) || contains(command, "tk hooks"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
