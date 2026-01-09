package hooks

import (
	"fmt"
	"strings"
	"testing"
)

func TestHookParity(t *testing.T) {
	// Get Claude hooks (reference implementation)
	claudeHooks := getTasukuClaudeHooks()

	// Get OpenCode plugin source (target implementation)
	openCodeSource := tasukuOpenCodePlugin

	// Define the mapping of critical features that must exist in both
	// Feature Name -> {Claude Command, OpenCode Event}
	features := map[string]struct {
		claudeCmd     string
		openCodeEvent string
	}{
		"Session Context": {
			claudeCmd:     "hooks session",
			openCodeEvent: "session.created",
		},
		"Stop Reminder": {
			claudeCmd:     "hooks stop-reminder",
			openCodeEvent: "session.idle",
		},
		"Todo Check": {
			claudeCmd:     "hooks todo-check",
			openCodeEvent: "todo.updated",
		},
		"Prompt Check": {
			claudeCmd:     "hooks prompt-check",
			openCodeEvent: "message.created",
		},
		"Subagent Done": {
			claudeCmd:     "hooks subagent-done",
			openCodeEvent: "tool.execute.after",
		},
	}

	// 1. Verify Claude hooks contain the expected commands
	t.Run("Claude_Has_Features", func(t *testing.T) {
		for name, feature := range features {
			found := false
			// Search recursively in claude hooks structure
			for _, hooks := range claudeHooks {
				for _, hook := range hooks {
					if cmd := getHookCommand(hook); strings.Contains(cmd, feature.claudeCmd) {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				t.Errorf("Claude hooks missing feature: %s (expected command: %s)", name, feature.claudeCmd)
			}
		}
	})

	// 2. Verify OpenCode plugin contains the expected logic
	t.Run("OpenCode_Has_Parity", func(t *testing.T) {
		for name, feature := range features {
			// Check for event handler
			if !strings.Contains(openCodeSource, feature.openCodeEvent) {
				t.Errorf("OpenCode plugin missing event handler for %s: %s", name, feature.openCodeEvent)
			}

			// Check for command execution
			cmdParts := strings.Split(feature.claudeCmd, " ")
			if len(cmdParts) >= 2 {
				subCmd := cmdParts[1]
				// We expect the JS code to call tk('hooks', 'subCmd')
				// Check for the presence of the subcommand string
				if !strings.Contains(openCodeSource, fmt.Sprintf("'%s'", subCmd)) &&
					!strings.Contains(openCodeSource, fmt.Sprintf("\"%s\"", subCmd)) {
					t.Errorf("OpenCode plugin missing command logic for %s: expected usage of '%s'", name, subCmd)
				}
			}
		}
	})

	// 3. Verify specific handling for Bash checks in OpenCode
	t.Run("OpenCode_Tool_Handling", func(t *testing.T) {
		if !strings.Contains(openCodeSource, "tool === 'bash'") {
			t.Error("OpenCode plugin missing specific handling for 'bash' tool")
		}
		if !strings.Contains(openCodeSource, "tool === 'task'") {
			t.Error("OpenCode plugin missing specific handling for 'task' tool")
		}
	})
}
