package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Tasuku hooks",
		Long: `Install Tasuku hooks for git and AI tools (Claude Code, Codex, OpenCode, Copilot CLI, Cursor).

By default, installs all hooks. Use flags to install specific hooks only.

Git hooks (always local to .git/hooks/):
  - pre-commit: Validates Tasuku storage before commits
  - post-commit: Suggests task status updates based on commit messages

Claude Code hooks (global ~/.claude/ by default, or local ./.claude/ with --local):
  - SessionStart: Shows project context summary when session begins
  - Stop: Reminds about running timers and in-progress tasks
  - ExitPlanMode: Prompts to sync plan to Tasuku tasks

Codex hooks (global ~/.codex/config.toml):
  - notify: Called on agent turn completion

OpenCode hooks (via plugin in ~/.config/opencode/plugin/ or .opencode/plugin/):
  - session.created: Shows context summary at session start
  - session.idle: Reminds about running timers
  - todo.updated: Checks for project-level tasks

Copilot CLI hooks (always local in .github/hooks/):
  - sessionStart: Shows project context summary when session begins
  - sessionEnd: Reminds about running timers and in-progress tasks
  - userPromptSubmitted: Detects task intent in prompts
  - postToolUse: Checks if TodoWrite items should persist

Cursor hooks (global ~/.cursor/ by default, or local ./.cursor/ with --local):
  - sessionStart: Shows project context summary when session begins
  - stop: Reminds about running timers and in-progress tasks
  - preCompact: Shows context summary before compaction
  - postToolUse: Checks if TodoWrite items should persist
  - beforeSubmitPrompt: Detects task intent in prompts

Examples:
  tk hooks install              # Git + Claude + Codex + OpenCode + Copilot + Cursor (global where applicable)
  tk hooks install --local      # Git + Claude + OpenCode + Cursor (local to project)
  tk hooks install --git        # Install only git hooks
  tk hooks install --claude     # Install only Claude Code hooks (global)
  tk hooks install --codex      # Install only Codex hooks
  tk hooks install --opencode   # Install only OpenCode hooks (global)
  tk hooks install --copilot    # Install only Copilot CLI hooks (always local)
  tk hooks install --cursor     # Install only Cursor hooks (global)
  tk hooks install --cursor --local    # Cursor hooks local to project
  tk hooks install --opencode --local  # OpenCode hooks local to project`,
		RunE: runInstall,
	}

	cmd.Flags().Bool("git", false, "Install git hooks only")
	cmd.Flags().Bool("claude", false, "Install Claude Code hooks only")
	cmd.Flags().Bool("codex", false, "Install Codex hooks only")
	cmd.Flags().Bool("opencode", false, "Install OpenCode hooks only")
	cmd.Flags().Bool("copilot", false, "Install Copilot CLI hooks only (always local)")
	cmd.Flags().Bool("cursor", false, "Install Cursor hooks only")
	cmd.Flags().Bool("force", false, "Overwrite existing hooks")
	cmd.Flags().Bool("local", false, "Install hooks to project instead of global")

	return cmd
}

func newUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove Tasuku hooks",
		Long: `Remove Tasuku hooks from git and AI tools (Claude Code, Codex, OpenCode, Copilot CLI, Cursor).

By default, removes all hooks. Use flags to remove specific hooks only.

Examples:
  tk hooks uninstall                     # Remove all hooks
  tk hooks uninstall --git               # Remove only git hooks
  tk hooks uninstall --claude            # Remove Claude Code hooks (global)
  tk hooks uninstall --claude --local    # Remove Claude Code hooks (project)
  tk hooks uninstall --codex             # Remove Codex hooks
  tk hooks uninstall --opencode          # Remove OpenCode hooks (global)
  tk hooks uninstall --opencode --local  # Remove OpenCode hooks (project)
  tk hooks uninstall --copilot           # Remove Copilot CLI hooks
  tk hooks uninstall --cursor            # Remove Cursor hooks (global)
  tk hooks uninstall --cursor --local    # Remove Cursor hooks (project)`,
		RunE: runUninstall,
	}

	cmd.Flags().Bool("git", false, "Remove git hooks only")
	cmd.Flags().Bool("claude", false, "Remove Claude Code hooks only")
	cmd.Flags().Bool("codex", false, "Remove Codex hooks only")
	cmd.Flags().Bool("opencode", false, "Remove OpenCode hooks only")
	cmd.Flags().Bool("copilot", false, "Remove Copilot CLI hooks only")
	cmd.Flags().Bool("cursor", false, "Remove Cursor hooks only")
	cmd.Flags().Bool("local", false, "Remove hooks from project instead of global")

	return cmd
}

func runInstall(cmd *cobra.Command, args []string) error {
	gitOnly, _ := cmd.Flags().GetBool("git")
	claudeOnly, _ := cmd.Flags().GetBool("claude")
	codexOnly, _ := cmd.Flags().GetBool("codex")
	opencodeOnly, _ := cmd.Flags().GetBool("opencode")
	copilotOnly, _ := cmd.Flags().GetBool("copilot")
	cursorOnly, _ := cmd.Flags().GetBool("cursor")
	force, _ := cmd.Flags().GetBool("force")
	local, _ := cmd.Flags().GetBool("local")

	// Determine what to install based on flags
	anySpecific := gitOnly || claudeOnly || codexOnly || opencodeOnly || copilotOnly || cursorOnly
	installGit := !anySpecific || gitOnly
	installClaude := !anySpecific || claudeOnly
	installCodex := !anySpecific || codexOnly
	installOpenCode := !anySpecific || opencodeOnly
	installCopilot := !anySpecific || copilotOnly
	installCursor := !anySpecific || cursorOnly

	var errors []string

	if installGit {
		if err := installGitHooks(); err != nil {
			errors = append(errors, fmt.Sprintf("git: %v", err))
		}
	}

	if installClaude {
		if err := installClaudeHooks(force, local); err != nil {
			errors = append(errors, fmt.Sprintf("claude: %v", err))
		}
	}

	if installCodex {
		// Codex hooks are always global (no local option)
		if err := installCodexHooks(force); err != nil {
			errors = append(errors, fmt.Sprintf("codex: %v", err))
		}
	}

	if installOpenCode {
		if err := installOpenCodeHooks(force, local); err != nil {
			errors = append(errors, fmt.Sprintf("opencode: %v", err))
		}
	}

	if installCopilot {
		// Copilot CLI hooks are always local (in .github/hooks/)
		if err := installCopilotHooks(force); err != nil {
			errors = append(errors, fmt.Sprintf("copilot: %v", err))
		}
	}

	if installCursor {
		if err := installCursorHooks(force, local); err != nil {
			errors = append(errors, fmt.Sprintf("cursor: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("some hooks failed to install:\n  %s", strings.Join(errors, "\n  "))
	}

	return nil
}

func runUninstall(cmd *cobra.Command, args []string) error {
	gitOnly, _ := cmd.Flags().GetBool("git")
	claudeOnly, _ := cmd.Flags().GetBool("claude")
	codexOnly, _ := cmd.Flags().GetBool("codex")
	opencodeOnly, _ := cmd.Flags().GetBool("opencode")
	copilotOnly, _ := cmd.Flags().GetBool("copilot")
	cursorOnly, _ := cmd.Flags().GetBool("cursor")
	local, _ := cmd.Flags().GetBool("local")

	// Determine what to uninstall based on flags
	anySpecific := gitOnly || claudeOnly || codexOnly || opencodeOnly || copilotOnly || cursorOnly
	uninstallGit := !anySpecific || gitOnly
	uninstallClaude := !anySpecific || claudeOnly
	uninstallCodex := !anySpecific || codexOnly
	uninstallOpenCode := !anySpecific || opencodeOnly
	uninstallCopilot := !anySpecific || copilotOnly
	uninstallCursor := !anySpecific || cursorOnly

	var errors []string

	if uninstallGit {
		if err := uninstallGitHooks(); err != nil {
			errors = append(errors, fmt.Sprintf("git: %v", err))
		}
	}

	if uninstallClaude {
		if err := uninstallClaudeHooks(local); err != nil {
			errors = append(errors, fmt.Sprintf("claude: %v", err))
		}
	}

	if uninstallCodex {
		if err := uninstallCodexHooks(); err != nil {
			errors = append(errors, fmt.Sprintf("codex: %v", err))
		}
	}

	if uninstallOpenCode {
		if err := uninstallOpenCodeHooks(local); err != nil {
			errors = append(errors, fmt.Sprintf("opencode: %v", err))
		}
	}

	if uninstallCopilot {
		if err := uninstallCopilotHooks(); err != nil {
			errors = append(errors, fmt.Sprintf("copilot: %v", err))
		}
	}

	if uninstallCursor {
		if err := uninstallCursorHooks(local); err != nil {
			errors = append(errors, fmt.Sprintf("cursor: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("some hooks failed to uninstall:\n  %s", strings.Join(errors, "\n  "))
	}

	return nil
}

// Git hook constants
const (
	tasukuHookStart = "# --- TASUKU HOOK START ---"
	tasukuHookEnd   = "# --- TASUKU HOOK END ---"
)

// getTasukuHookContent returns the Tasuku-specific hook content for a given hook name
func getTasukuHookContent(hookName string) string {
	switch hookName {
	case "pre-commit":
		return `# Tasuku pre-commit hook: validate task storage
if [ -d .tasuku ] || [ -f .tasuku.json ]; then
    tk validate
    if [ $? -ne 0 ]; then
        echo "Tasuku validation failed. Please fix issues before committing."
        exit 1
    fi
fi`
	case "post-commit":
		return `# Tasuku post-commit hook: suggest task status updates and reflection
COMMIT_MSG=$(git log -1 --pretty=%B)

if [[ $COMMIT_MSG =~ \(#([a-zA-Z0-9-]+)\) ]]; then
    TASK_ID="${BASH_REMATCH[1]}"
    echo ""
    echo "Detected task reference: #$TASK_ID"
    echo "Consider: tk done $TASK_ID"
fi

# Prompt for reflection after significant commits
echo ""
echo "💡 Post-commit reflection:"
echo "   - Made an architectural decision? → /tasuku:decide"
echo "   - Discovered a gotcha or insight? → /tasuku:learn"`
	default:
		return ""
	}
}

// wrapTasukuSection wraps content with Tasuku marker comments
func wrapTasukuSection(content string) string {
	return fmt.Sprintf("%s\n%s\n%s", tasukuHookStart, content, tasukuHookEnd)
}

// installHookWithMarkers installs a hook while preserving existing content
func installHookWithMarkers(hookPath, hookName string) error {
	tasukuContent := getTasukuHookContent(hookName)
	if tasukuContent == "" {
		return fmt.Errorf("unknown hook: %s", hookName)
	}

	wrappedContent := wrapTasukuSection(tasukuContent)

	// Check if hook file already exists
	existingContent, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No existing hook, create new one with shebang
			newContent := "#!/bin/bash\n\n" + wrappedContent + "\n"
			return os.WriteFile(hookPath, []byte(newContent), 0755)
		}
		return fmt.Errorf("failed to read existing hook: %w", err)
	}

	existingStr := string(existingContent)

	// Check if Tasuku section already exists
	startIdx := strings.Index(existingStr, tasukuHookStart)
	endIdx := strings.Index(existingStr, tasukuHookEnd)

	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		// Replace existing Tasuku section
		before := existingStr[:startIdx]
		after := existingStr[endIdx+len(tasukuHookEnd):]
		newContent := before + wrappedContent + after
		return os.WriteFile(hookPath, []byte(newContent), 0755)
	}

	// Append Tasuku section to existing hook
	var newContent string
	if strings.HasSuffix(existingStr, "\n") {
		newContent = existingStr + "\n" + wrappedContent + "\n"
	} else {
		newContent = existingStr + "\n\n" + wrappedContent + "\n"
	}

	return os.WriteFile(hookPath, []byte(newContent), 0755)
}

// removeTasukuSection removes only the Tasuku section from a hook file
// Returns (fileDeleted, sectionFound, error)
func removeTasukuSection(hookPath string) (deleted bool, found bool, err error) {
	content, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, fmt.Errorf("failed to read hook: %w", err)
	}

	contentStr := string(content)

	// Find Tasuku section
	startIdx := strings.Index(contentStr, tasukuHookStart)
	endIdx := strings.Index(contentStr, tasukuHookEnd)

	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		// No Tasuku section found
		return false, false, nil
	}

	// Remove the Tasuku section including surrounding whitespace
	before := contentStr[:startIdx]
	after := contentStr[endIdx+len(tasukuHookEnd):]

	// Clean up extra whitespace
	before = strings.TrimRight(before, " \t")
	after = strings.TrimLeft(after, " \t")

	// If before ends with newlines and after starts with newlines, normalize
	before = strings.TrimRight(before, "\n")
	after = strings.TrimLeft(after, "\n")

	var newContent string
	if before != "" && after != "" {
		newContent = before + "\n\n" + after
	} else if before != "" {
		newContent = before + "\n"
	} else if after != "" {
		newContent = after + "\n"
	} else {
		newContent = ""
	}

	// Check if the remaining content is essentially empty (just shebang or whitespace)
	trimmed := strings.TrimSpace(newContent)
	isEmptyOrShebangOnly := trimmed == "" ||
		trimmed == "#!/bin/bash" ||
		trimmed == "#!/bin/sh" ||
		trimmed == "#!/usr/bin/env bash" ||
		trimmed == "#!/usr/bin/env sh"

	if isEmptyOrShebangOnly {
		// Delete the file entirely
		if err := os.Remove(hookPath); err != nil {
			return false, true, fmt.Errorf("failed to delete empty hook: %w", err)
		}
		return true, true, nil
	}

	// Write the cleaned content back
	return false, true, os.WriteFile(hookPath, []byte(newContent), 0755)
}

func installGitHooks() error {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository")
	}

	hooksDir := ".git/hooks"

	hooks := []struct {
		name        string
		description string
	}{
		{"pre-commit", "validates Tasuku storage"},
		{"post-commit", "suggests task status updates"},
	}

	for _, hook := range hooks {
		hookPath := filepath.Join(hooksDir, hook.name)
		if err := installHookWithMarkers(hookPath, hook.name); err != nil {
			return fmt.Errorf("failed to install %s hook: %w", hook.name, err)
		}
	}

	fmt.Println("Git hooks installed:")
	for _, hook := range hooks {
		fmt.Printf("  - %s: %s\n", hook.name, hook.description)
	}
	fmt.Println("\nNote: Existing hook content has been preserved.")

	return nil
}

func uninstallGitHooks() error {
	hooksDir := ".git/hooks"

	hooks := []string{"pre-commit", "post-commit"}
	removedCount := 0

	for _, hook := range hooks {
		hookPath := filepath.Join(hooksDir, hook)
		deleted, found, err := removeTasukuSection(hookPath)
		if err != nil {
			return fmt.Errorf("failed to uninstall %s: %w", hook, err)
		}
		if !found {
			continue
		}
		if deleted {
			fmt.Printf("Removed: %s (file deleted - was empty)\n", hook)
		} else {
			fmt.Printf("Removed Tasuku section from: %s\n", hook)
		}
		removedCount++
	}

	if removedCount == 0 {
		fmt.Println("No Tasuku hooks found to uninstall")
	} else {
		fmt.Println("\nTasuku hooks uninstalled (other hook content preserved)")
	}

	return nil
}
