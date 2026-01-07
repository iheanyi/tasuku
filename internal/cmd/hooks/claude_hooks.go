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

	"github.com/iheanyi/tasuku/internal/version"
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

func getClaudeSettingsPath(local bool) string {
	if local {
		return filepath.Join(".claude", "settings.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

func readClaudeSettings(local bool) (map[string]interface{}, error) {
	path := getClaudeSettingsPath(local)
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

func writeClaudeSettings(settings map[string]interface{}, local bool) error {
	path := getClaudeSettingsPath(local)

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

// Hook version tracking - stores the tk version when hooks were installed
// so we can warn users when their hooks are outdated.

const hookVersionFile = ".tasuku-hooks-version"

func getHookVersionPath(local bool) string {
	if local {
		return filepath.Join(".claude", hookVersionFile)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", hookVersionFile)
}

// writeHookVersion writes the current tk version to the hook version file.
func writeHookVersion(local bool) error {
	path := getHookVersionPath(local)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(version.Version()), 0644)
}

// readHookVersion reads the installed hook version.
func readHookVersion(local bool) string {
	path := getHookVersionPath(local)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// CheckHookVersion compares installed hook version with current tk version.
// Returns (installedVersion, currentVersion, needsUpdate).
func CheckHookVersion(local bool) (string, string, bool) {
	installed := readHookVersion(local)
	current := version.Version()

	if installed == "" {
		// No version file - hooks may have been installed before version tracking
		return "", current, false
	}

	return installed, current, installed != current
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
			{
				"matcher": "TodoWrite",
				"hooks": []map[string]string{
					{
						"type":    "command",
						"command": fmt.Sprintf("%s hooks todo-check %s", executable, tasukuHookMarker),
					},
				},
			},
		},
		"SessionStart": {
			{
				"hooks": []map[string]string{
					{
						"type":    "command",
						"command": fmt.Sprintf("%s hooks session %s", executable, tasukuHookMarker),
					},
				},
			},
		},
		"Stop": {
			{
				"hooks": []map[string]string{
					{
						"type":    "command",
						"command": fmt.Sprintf("%s hooks stop-reminder %s", executable, tasukuHookMarker),
					},
				},
			},
		},
		"PreCompact": {
			{
				"hooks": []map[string]string{
					{
						"type":    "command",
						"command": fmt.Sprintf("%s hooks pre-compact %s", executable, tasukuHookMarker),
					},
				},
			},
		},
		"SubagentStop": {
			{
				"hooks": []map[string]string{
					{
						"type":    "command",
						"command": fmt.Sprintf("%s hooks subagent-done %s", executable, tasukuHookMarker),
					},
				},
			},
		},
		"UserPromptSubmit": {
			{
				"hooks": []map[string]string{
					{
						"type":    "command",
						"command": fmt.Sprintf("%s hooks prompt-check %s", executable, tasukuHookMarker),
					},
				},
			},
		},
	}
}

// installClaudeHooks installs Tasuku hooks in Claude Code settings.
// Uses smart incremental updates:
//   - Adds new hooks without touching existing ones
//   - Updates hooks whose commands have changed
//   - Preserves user's non-Tasuku hooks
//
// If force is true, all Tasuku hooks are replaced (useful for downgrading).
// If local is true, installs to ./.claude/settings.json instead of ~/.claude/settings.json.
func installClaudeHooks(force, local bool) error {
	settings, err := readClaudeSettings(local)
	if err != nil {
		return err
	}

	// Get or create hooks section
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		hooks = make(map[string]interface{})
	}

	tasukuHooks := getTasukuClaudeHooks()
	addedCount := 0
	updatedCount := 0
	unchangedCount := 0

	for hookType, desiredHooks := range tasukuHooks {
		if len(desiredHooks) == 0 {
			continue
		}

		existing, _ := hooks[hookType].([]interface{})

		// If force mode, remove all Tasuku hooks first, then add fresh
		if force {
			var filtered []interface{}
			for _, h := range existing {
				if !isTasukuHook(h) {
					filtered = append(filtered, h)
				}
			}
			existing = filtered

			// Add all desired hooks
			for _, newHook := range desiredHooks {
				hookMap := copyHookMap(newHook)
				existing = append(existing, hookMap)
				addedCount++
			}
			hooks[hookType] = existing
			continue
		}

		// Smart incremental update: check each desired hook individually
		for _, desiredHook := range desiredHooks {
			matcher := getHookMatcher(desiredHook)
			desiredCommand := getHookCommand(desiredHook)

			// Find existing hook with same matcher (or no matcher for global hooks)
			foundIdx := -1
			existingCommand := ""
			for i, h := range existing {
				if !isTasukuHook(h) {
					continue
				}
				existingMatcher := getHookMatcherFromExisting(h)
				if existingMatcher == matcher {
					foundIdx = i
					existingCommand = getHookCommandFromExisting(h)
					break
				}
			}

			if foundIdx == -1 {
				// Hook doesn't exist - add it
				hookMap := copyHookMap(desiredHook)
				existing = append(existing, hookMap)
				addedCount++
				fmt.Printf("  + Added: %s", hookType)
				if matcher != "" {
					fmt.Printf("/%s", matcher)
				}
				fmt.Println()
			} else if existingCommand != desiredCommand {
				// Hook exists but command changed - update it
				existing[foundIdx] = copyHookMap(desiredHook)
				updatedCount++
				fmt.Printf("  ~ Updated: %s", hookType)
				if matcher != "" {
					fmt.Printf("/%s", matcher)
				}
				fmt.Println()
			} else {
				// Hook exists and is unchanged
				unchangedCount++
			}
		}

		hooks[hookType] = existing
	}

	totalChanges := addedCount + updatedCount
	if totalChanges == 0 {
		fmt.Printf("All %d Tasuku hooks are up to date.\n", unchangedCount)
		return nil
	}

	settings["hooks"] = hooks

	if err := writeClaudeSettings(settings, local); err != nil {
		return err
	}

	location := "global"
	if local {
		location = "project"
	}
	fmt.Println()
	fmt.Printf("Tasuku hooks updated (%s):\n", location)
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
	fmt.Println("Restart Claude Code for hooks to take effect.")

	// Write hook version for update detection
	if err := writeHookVersion(local); err != nil {
		// Non-fatal: warn but don't fail the install
		fmt.Printf("Warning: could not write hook version file: %v\n", err)
	}

	return nil
}

// isTasukuHook checks if a hook entry is a Tasuku hook
func isTasukuHook(h interface{}) bool {
	hook, ok := h.(map[string]interface{})
	if !ok {
		return false
	}

	// Check nested hooks array format
	if innerHooks, ok := hook["hooks"].([]interface{}); ok {
		for _, ih := range innerHooks {
			if innerHook, ok := ih.(map[string]interface{}); ok {
				if command, ok := innerHook["command"].(string); ok {
					if containsTasukuMarker(command) {
						return true
					}
				}
			}
		}
	}

	// Also check simple command format (backwards compat)
	if command, ok := hook["command"].(string); ok {
		if containsTasukuMarker(command) {
			return true
		}
	}

	return false
}

// getHookMatcher extracts the matcher from a desired hook definition
func getHookMatcher(hook map[string]interface{}) string {
	if matcher, ok := hook["matcher"].(string); ok {
		return matcher
	}
	return ""
}

// getHookCommand extracts the command from a desired hook definition
func getHookCommand(hook map[string]interface{}) string {
	if innerHooks, ok := hook["hooks"].([]map[string]string); ok {
		for _, ih := range innerHooks {
			if command, ok := ih["command"]; ok {
				return command
			}
		}
	}
	return ""
}

// getHookMatcherFromExisting extracts the matcher from an existing hook entry
func getHookMatcherFromExisting(h interface{}) string {
	hook, ok := h.(map[string]interface{})
	if !ok {
		return ""
	}
	if matcher, ok := hook["matcher"].(string); ok {
		return matcher
	}
	return ""
}

// getHookCommandFromExisting extracts the command from an existing hook entry
func getHookCommandFromExisting(h interface{}) string {
	hook, ok := h.(map[string]interface{})
	if !ok {
		return ""
	}

	// Check nested hooks array format
	if innerHooks, ok := hook["hooks"].([]interface{}); ok {
		for _, ih := range innerHooks {
			if innerHook, ok := ih.(map[string]interface{}); ok {
				if command, ok := innerHook["command"].(string); ok {
					return command
				}
			}
		}
	}

	// Also check simple command format
	if command, ok := hook["command"].(string); ok {
		return command
	}

	return ""
}

// copyHookMap creates a deep copy of a hook definition
func copyHookMap(hook map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range hook {
		result[k] = v
	}
	return result
}

// uninstallClaudeHooks removes Tasuku hooks from Claude Code settings.
// Other hooks configured by the user are preserved.
// If local is true, removes from ./.claude/settings.json instead of ~/.claude/settings.json.
func uninstallClaudeHooks(local bool) error {
	settings, err := readClaudeSettings(local)
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

	if err := writeClaudeSettings(settings, local); err != nil {
		return err
	}

	location := "global"
	if local {
		location = "project"
	}
	fmt.Printf("Removed %d Tasuku hook(s) from Claude Code (%s).\n", removedCount, location)
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
