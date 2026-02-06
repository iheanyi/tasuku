package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/cmdutil"
	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

func newTodoCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "todo-check",
		Short: "Check if TodoWrite items should persist to Tasuku",
		Long: `Called by Claude Code PostToolUse hook after tool use.

Analyzes tool output and suggests actions:
  - TodoWrite: Persist project-level tasks to Tasuku
  - Bash (tests): Prompt to track test failures, detect fixes
  - Bash (git): Link commits to tasks

Features (all enabled by default):
  - bugfix_learning: Prompt for learnings after bug fixes
  - project_task: Suggest persisting project-level tasks
  - test_failure: Detect test failures and suggest tracking
  - test_fix_learning: Prompt for learning when tests pass after failure
  - git_commit: Link commits to related tasks

The test_fix_learning feature is critical: it detects when you fix a failing
test and prompts you to document the learning IMMEDIATELY, preventing
knowledge loss.

Configuration:
  --quiet           Minimal output mode
  --disable=X,Y     Disable specific features
  --list-features   Show available features

The hook receives tool info via TOOL_NAME and TOOL_INPUT environment variables.

Examples:
  tk hooks todo-check                           # Default (all features)
  tk hooks todo-check --quiet                   # Minimal output
  tk hooks todo-check --disable=test_failure    # Skip test detection
  tk hooks todo-check --list-features           # Show features`,
		RunE: runTodoCheck,
	}

	cmd.Flags().Bool("quiet", false, "Minimal output mode")
	cmd.Flags().String("disable", "", "Comma-separated features to disable")
	cmd.Flags().Bool("list-features", false, "List available features")

	return cmd
}

var todoCheckFeatures = []string{
	"bugfix_learning",
	"project_task",
	"test_failure",
	"test_fix_learning",     // Prompt for learning when tests pass after failure
	"git_commit",
	"investigation_pattern", // Prompt for learning after deep file investigation + edit
}

var todoCheckQuietFeatures = map[string]bool{
	"bugfix_learning":       true,  // Keep in quiet mode
	"project_task":          false,
	"test_failure":          true,  // Keep in quiet mode
	"test_fix_learning":     true,  // Keep in quiet mode - this is critical
	"git_commit":            false,
	"investigation_pattern": true,  // Keep in quiet mode - important for learning capture
}

func runTodoCheck(cmd *cobra.Command, args []string) error {
	listFeatures, _ := cmd.Flags().GetBool("list-features")
	if listFeatures {
		fmt.Println("Available todo-check features:")
		for _, f := range todoCheckFeatures {
			fmt.Printf("  - %s\n", f)
		}
		return nil
	}

	quiet, _ := cmd.Flags().GetBool("quiet")
	disableStr, _ := cmd.Flags().GetString("disable")

	disabled := parseDisabledFeatures(disableStr)
	config := buildFeatureConfig(todoCheckFeatures, todoCheckQuietFeatures, quiet, disabled)

	return hookTodoCheck(config)
}

func newPromptCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prompt-check",
		Short: "Detect task-related intent in user prompts",
		Long: `Called by Claude Code UserPromptSubmit hook when user sends a message.

Analyzes the user's prompt to detect task-related intent and surface context.

Features (all enabled by default):
  Context surfacing:
  - session_continuity: Show in-progress tasks on "continue"/"resume"
  - decision_lookup:    Surface related decisions for questions
  - learning_lookup:    Surface related learnings for questions
  - task_reference:     Show context when task ID mentioned
  - task_surfacing:     Find related tasks by keyword matching

  Nudges:
  - rule_detection:     Detect never/always patterns to record
  - bug_detection:      Prompt to track bug reports
  - work_detection:     Suggest creating task for significant work
  - stuck_detection:    Offer help when user seems stuck
  - shipping_check:     Pre-ship checklist on deploy/release
  - learning_capture:   Capture "TIL"/"I learned" as learnings
  - decision_capture:   Prompt to record "X or Y" decisions
  - scope_warning:      Warn about scope expansion mid-task
  - architecture_explanation: Detect "because we"/"why" explanations for decisions
  - preference_stated:  Capture user preferences for consistency

Configuration:
  --quiet           Minimal output (context only, no nudges)
  --disable=X,Y     Disable specific features
  --list-features   Show available features

The hook receives the user prompt via USER_PROMPT environment variable.

Examples:
  tk hooks prompt-check                              # Default (all features)
  tk hooks prompt-check --quiet                      # Context only, no nudges
  tk hooks prompt-check --disable=shipping_check     # Skip shipping prompts
  tk hooks prompt-check --list-features              # Show features`,
		RunE: runPromptCheck,
	}

	cmd.Flags().Bool("quiet", false, "Minimal output mode (context surfacing only)")
	cmd.Flags().String("disable", "", "Comma-separated features to disable")
	cmd.Flags().Bool("list-features", false, "List available features")

	return cmd
}

var promptCheckFeatures = []string{
	// Context surfacing
	"session_continuity",
	"decision_lookup",
	"learning_lookup",
	"task_reference",
	"task_surfacing",
	// Nudges
	"rule_detection",
	"bug_detection",
	"work_detection",
	"stuck_detection",
	"shipping_check",
	"learning_capture",
	"decision_capture",
	"scope_warning",
	"architecture_explanation",
	"preference_stated",
}

// Features kept in quiet mode (context surfacing only)
var promptCheckQuietFeatures = map[string]bool{
	"session_continuity":       true,
	"decision_lookup":          true,
	"learning_lookup":          true,
	"task_reference":           true,
	"task_surfacing":           true,
	"rule_detection":           true,  // Rules are important enough to keep
	"bug_detection":            false,
	"work_detection":           false,
	"stuck_detection":          false,
	"shipping_check":           false,
	"learning_capture":         true,  // Direct capture is low noise
	"decision_capture":         false,
	"scope_warning":            false,
	"architecture_explanation": true,  // Important for decision capture
	"preference_stated":        true,  // Low noise preference capture
}

func runPromptCheck(cmd *cobra.Command, args []string) error {
	listFeatures, _ := cmd.Flags().GetBool("list-features")
	if listFeatures {
		fmt.Println("Available prompt-check features:")
		fmt.Println("\nContext surfacing:")
		for _, f := range []string{"session_continuity", "decision_lookup", "learning_lookup", "task_reference", "task_surfacing"} {
			fmt.Printf("  - %s\n", f)
		}
		fmt.Println("\nNudges:")
		for _, f := range []string{"rule_detection", "bug_detection", "work_detection", "stuck_detection", "shipping_check", "learning_capture", "decision_capture", "scope_warning", "architecture_explanation", "preference_stated"} {
			fmt.Printf("  - %s\n", f)
		}
		return nil
	}

	quiet, _ := cmd.Flags().GetBool("quiet")
	disableStr, _ := cmd.Flags().GetString("disable")

	disabled := parseDisabledFeatures(disableStr)
	config := buildFeatureConfig(promptCheckFeatures, promptCheckQuietFeatures, quiet, disabled)

	return hookPromptCheck(config)
}

// hookTodoCheck analyzes tool output and suggests actions
// Handles TodoWrite (task persistence) and Bash (test failures, git commits)
func hookTodoCheck(config featureConfig) error {
	toolName := os.Getenv("TOOL_NAME")
	toolInput := os.Getenv("TOOL_INPUT")
	toolOutput := os.Getenv("TOOL_OUTPUT")

	if toolInput == "" && toolOutput == "" {
		return nil
	}

	switch toolName {
	case "TodoWrite":
		return handleTodoWriteCheck(config, toolInput)
	case "Bash":
		return handleBashCheck(config, toolInput, toolOutput)
	case "Read":
		return handleReadCheck(config, toolInput)
	case "Edit":
		return handleEditCheck(config, toolInput)
	default:
		// For backwards compatibility, assume TodoWrite if no TOOL_NAME
		if toolInput != "" {
			return handleTodoWriteCheck(config, toolInput)
		}
	}

	return nil
}

// handleTodoWriteCheck analyzes TodoWrite output for project-level items and bug fixes
func handleTodoWriteCheck(config featureConfig, toolInput string) error {
	// Parse the TodoWrite JSON
	var input struct {
		Todos []struct {
			Content    string `json:"content"`
			Status     string `json:"status"`
			ActiveForm string `json:"activeForm"`
		} `json:"todos"`
	}

	if err := json.Unmarshal([]byte(toolInput), &input); err != nil {
		return nil
	}

	if len(input.Todos) == 0 {
		return nil
	}

	// Check if Tasuku is initialized
	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		return err
	}
	var existingTasks map[string]bool
	if s.Exists() {
		f, err := s.Read()
		if err == nil {
			existingTasks = make(map[string]bool)
			for id := range f.Tasks {
				existingTasks[id] = true
			}
		}
	}

	// Track completed bug fixes for learning prompt
	var completedBugFixes []string
	var suggestions []string

	for _, todo := range input.Todos {
		// Check for completed bug fixes - prompt for learnings
		if config["bugfix_learning"] && todo.Status == "completed" && isBugFixTask(todo.Content) {
			completedBugFixes = append(completedBugFixes, todo.Content)
		}

		if config["project_task"] && shouldPersistTask(todo.Content) {
			id := generateID(todo.Content)
			if existingTasks != nil && existingTasks[id] {
				continue
			}
			suggestions = append(suggestions, todo.Content)
		}
	}

	// Prompt for learnings after bug fixes
	if len(completedBugFixes) > 0 {
		fmt.Println("🎯 BUG FIX COMPLETED - RECORD YOUR LEARNINGS NOW!")
		fmt.Println()
		for _, fix := range completedBugFixes {
			fmt.Printf("   ✓ %s\n", cmdutil.Truncate(fix, 60))
		}
		fmt.Println()
		fmt.Println("📝 Document what you learned:")
		fmt.Println("   → /tasuku:learn \"root cause: ...\"")
		fmt.Println("   → /tasuku:learn \"Never X\" or \"Always Y\" (for rules)")
		fmt.Println()
	}

	// Suggest persisting project-level tasks
	if len(suggestions) > 0 {
		fmt.Println("💡 Some TodoWrite items look project-level:")
		for _, s := range suggestions {
			fmt.Printf("   → %s\n", cmdutil.Truncate(s, 60))
		}
		fmt.Println()
		fmt.Println("Consider: /tasuku:add \"description\" --priority high")
	}

	return nil
}

// handleBashCheck analyzes Bash command output for test failures and git commits
func handleBashCheck(config featureConfig, toolInput, toolOutput string) error {
	// Parse command from input
	var bashInput struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(toolInput), &bashInput); err != nil {
		return nil
	}

	command := strings.ToLower(bashInput.Command)
	outputLower := strings.ToLower(toolOutput)

	// === TEST DETECTION (failure tracking and fix learning) ===
	if isTestCommand(command) {
		isFailure := detectTestFailure(outputLower)
		isSuccess := detectTestSuccess(outputLower)

		// === TEST FAILURE: Save state and show warning ===
		if config["test_failure"] && isFailure {
			// Save failure state for later "test fix" detection
			saveTestFailureState(bashInput.Command)

			fmt.Println("🔴 TEST FAILURE DETECTED")
			fmt.Println()
			fmt.Println("   Track the fix:")
			fmt.Println("   → /tasuku:add \"Fix failing tests\" --tag bug --priority high")
			fmt.Println()
		}

		// === TEST FIX: Detect success after recent failure ===
		// This is the key feature: prompt for learning when tests pass after failure
		if config["test_fix_learning"] && isSuccess && !isFailure {
			// Check if there was a recent test failure (within 30 minutes)
			const maxAge = 30 * time.Minute
			if isRecentTestFailure(maxAge) {
				fmt.Println("✅ TESTS PASSING AFTER FAILURE - DOCUMENT YOUR FIX!")
				fmt.Println()
				fmt.Println("   You just fixed a bug. Record the learning NOW:")
				fmt.Println("   → What was the root cause?")
				fmt.Println("   → What rule prevents this in the future?")
				fmt.Println()
				fmt.Println("   /tasuku:learn \"Never X\" or \"Always Y\"")
				fmt.Println()
				fmt.Println("   Or use /tasuku:reflect for guided extraction.")
				fmt.Println()

				// Clear the state so we don't prompt again
				clearTestFailureState()
			}
		}
	}

	// === GIT COMMIT TASK LINKING ===
	if config["git_commit"] && isGitCommitCommand(command) {
		handleGitCommitLink(toolOutput)
	}

	return nil
}

// isTestCommand checks if a command is running tests
func isTestCommand(command string) bool {
	testPatterns := []string{
		"go test", "npm test", "yarn test", "pnpm test",
		"pytest", "python -m pytest", "python3 -m pytest",
		"jest", "vitest", "mocha", "cargo test", "mix test",
		"rspec", "bundle exec rspec", "rails test",
		"make test", "make check",
	}
	for _, pattern := range testPatterns {
		if strings.Contains(command, pattern) {
			return true
		}
	}
	return false
}

// detectTestFailure checks test output for failure indicators
func detectTestFailure(output string) bool {
	failurePatterns := []string{
		"fail", "failed", "failure", "error:",
		"panic:", "assertion failed", "expected",
		"not equal", "mismatch", "exception",
		"exit status 1", "exit code 1",
		"tests failed", "test failed",
	}
	for _, pattern := range failurePatterns {
		if strings.Contains(output, pattern) {
			return true
		}
	}
	return false
}

// isGitCommitCommand checks if a command is a git commit
func isGitCommitCommand(command string) bool {
	return strings.Contains(command, "git commit")
}

// handleGitCommitLink suggests linking commits to tasks
func handleGitCommitLink(output string) {
	// Check if Tasuku is initialized
	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		return
	}
	if !s.Exists() {
		return
	}

	f, err := s.Read()
	if err != nil {
		return
	}

	// Find in-progress tasks
	var inProgress []taskWithID
	for id, t := range f.Tasks {
		if t.Status == task.StatusInProgress {
			inProgress = append(inProgress, taskWithID{id: id, task: t})
		}
	}

	if len(inProgress) == 0 {
		return
	}

	// Check if commit message already references a task
	outputLower := strings.ToLower(output)
	for _, item := range inProgress {
		if strings.Contains(outputLower, strings.ToLower(item.id)) {
			// Already references task
			fmt.Printf("📝 Commit references task: %s\n", item.id)
			fmt.Println("   Mark as done?")
			fmt.Printf("   → tk task done %s\n", item.id)
			fmt.Println()
			return
		}
	}

	// Suggest linking to in-progress task
	if len(inProgress) == 1 {
		item := inProgress[0]
		fmt.Printf("📝 Commit may relate to: %s\n", item.id)
		fmt.Printf("   %s\n", cmdutil.Truncate(item.task.Description, 50))
		fmt.Println("   Mark as done?")
		fmt.Printf("   → tk task done %s\n", item.id)
		fmt.Println()
	} else if len(inProgress) > 1 {
		fmt.Printf("📝 %d tasks in progress - consider marking one done:\n", len(inProgress))
		for _, item := range inProgress {
			fmt.Printf("   → tk task done %s\n", item.id)
		}
		fmt.Println()
	}
}

// hookPromptCheck analyzes user prompts for task-related intent and rule patterns
func hookPromptCheck(config featureConfig) error {
	// Get user prompt from environment (set by Claude Code hook)
	userPrompt := os.Getenv("USER_PROMPT")
	if userPrompt == "" {
		return nil
	}

	// Skip very short prompts (likely follow-ups or confirmations)
	if len(userPrompt) < 15 {
		return nil
	}

	promptLower := strings.ToLower(userPrompt)

	// Check if Tasuku is initialized
	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		return err
	}
	if !s.Exists() {
		return nil
	}

	f, err := s.Read()
	if err != nil {
		return nil
	}

	// Track what we've output to avoid too much noise
	outputCount := 0
	const maxOutputs = 2 // Limit to avoid overwhelming the user

	// === 1. SESSION CONTINUITY DETECTION ===
	if config["session_continuity"] && outputCount < maxOutputs && detectSessionContinuity(promptLower) {
		inProgressTasks := getInProgressTasks(f)
		if len(inProgressTasks) > 0 {
			fmt.Println("🔄 Continuing session - in-progress tasks:")
			for _, item := range inProgressTasks {
				fmt.Printf("   - %s: %s\n", item.id, cmdutil.Truncate(item.task.Description, 50))
				if notes := f.Context.Notes[item.id]; len(notes) > 0 {
					lastNote := notes[len(notes)-1]
					fmt.Printf("     Last note: %s\n", cmdutil.Truncate(lastNote.Text, 60))
				}
			}
			fmt.Println()
			outputCount++
		}
	}

	// === 2. DECISION LOOKUP ===
	if config["decision_lookup"] && outputCount < maxOutputs && looksLikeQuestion(promptLower) {
		relevantDecisions := findRelevantDecisions(f.Context.Decisions, promptLower)
		if len(relevantDecisions) > 0 {
			fmt.Println("📚 Related decisions found:")
			for _, d := range relevantDecisions {
				fmt.Printf("   - %s: Chose %s\n", d.ID, cmdutil.Truncate(d.Chose, 40))
				if d.Because != "" {
					fmt.Printf("     Because: %s\n", cmdutil.Truncate(d.Because, 50))
				}
			}
			fmt.Println()
			outputCount++
		}
	}

	// === 3. LEARNING LOOKUP ===
	if config["learning_lookup"] && outputCount < maxOutputs && looksLikeQuestion(promptLower) {
		relevantLearnings := findRelevantLearnings(f.Context.Learnings, promptLower)
		if len(relevantLearnings) > 0 {
			fmt.Println("💡 Related learnings found:")
			for _, l := range relevantLearnings {
				fmt.Printf("   - %s\n", cmdutil.Truncate(l.Text, 70))
			}
			fmt.Println()
			outputCount++
		}
	}

	// === 4. RULE PATTERN DETECTION ===
	if config["rule_detection"] && outputCount < maxOutputs && task.IsRuleLearning(userPrompt) && looksLikeInstruction(userPrompt) {
		rulePortion := extractRulePortion(userPrompt)
		if rulePortion != "" && !hasSimilarLearning(f.Context.Learnings, rulePortion) {
			fmt.Println("📝 RULE DETECTED in your message:")
			fmt.Printf("   \"%s\"\n", cmdutil.Truncate(rulePortion, 80))
			fmt.Println()
			fmt.Println("   Record this as a project rule:")
			fmt.Printf("   → /tasuku:learn \"%s\"\n", escapeForShell(rulePortion))
			fmt.Println()
			outputCount++
		}
	}

	// === 5. DIRECT LEARNING CAPTURE ===
	// Detect "TIL", "I learned", "note to self" patterns
	if config["learning_capture"] && outputCount < maxOutputs && detectLearningIntent(promptLower) {
		learningContent := extractLearningContent(userPrompt)
		if learningContent != "" && !hasSimilarLearning(f.Context.Learnings, learningContent) {
			fmt.Println("💡 Capture this learning?")
			fmt.Printf("   → /tasuku:learn \"%s\"\n", escapeForShell(learningContent))
			fmt.Println()
			outputCount++
		}
	}

	// === 6. DECISION POINT DETECTION ===
	// Detect "should we use X or Y" comparison patterns
	if config["decision_capture"] && outputCount < maxOutputs && detectDecisionPoint(promptLower) {
		fmt.Println("🤔 This looks like a decision point.")
		fmt.Println("   After deciding, record it:")
		fmt.Println("   → tk decide --id <name> --chose \"X\" --over \"Y\" --because \"reason\"")
		fmt.Println()
		outputCount++
	}

	// === 7. EXPLICIT TASK ID REFERENCE ===
	if config["task_reference"] {
		for id, t := range f.Tasks {
			if strings.Contains(promptLower, strings.ToLower(id)) {
				fmt.Printf("📋 Task referenced: %s\n", id)
				fmt.Printf("   Status: %s\n", t.Status)
				fmt.Printf("   %s\n", cmdutil.Truncate(t.Description, 50))
				if t.Status == task.StatusReady {
					fmt.Printf("   → Consider: tk task start %s\n", id)
				}
				fmt.Println()
				return nil // Task ID reference is definitive
			}
		}
	}

	// === 8. RELATED TASK SURFACING ===
	if config["task_surfacing"] && outputCount < maxOutputs {
		relatedTasks := findRelatedTasks(f.Tasks, promptLower)
		if len(relatedTasks) > 0 {
			fmt.Println("📋 Related tasks found:")
			for _, item := range relatedTasks {
				statusIcon := getStatusIcon(item.task.Status)
				fmt.Printf("   %s %s: %s\n", statusIcon, item.id, cmdutil.Truncate(item.task.Description, 45))
			}
			fmt.Println()
			outputCount++
		}
	}

	// === 9. SCOPE WARNING ===
	// Warn about scope expansion when there's an in-progress task
	if config["scope_warning"] && outputCount < maxOutputs && detectScopeExpansion(promptLower) {
		inProgress := getInProgressTasks(f)
		if len(inProgress) > 0 {
			currentTask := inProgress[0]
			// Check if new work seems unrelated to current task
			if !isRelatedWork(promptLower, currentTask.task.Description) {
				fmt.Println("⚠️  Scope expansion detected.")
				fmt.Printf("   Currently working on: %s\n", currentTask.id)
				fmt.Println("   Create separate task for new work?")
				fmt.Println("   → /tasuku:add \"description\" --priority normal")
				fmt.Println()
				outputCount++
			}
		}
	}

	// === 10. STUCK/FRUSTRATION DETECTION ===
	if config["stuck_detection"] && outputCount < maxOutputs && detectStuckPattern(promptLower) {
		fmt.Println("🤔 Sounds like you're stuck. Options:")
		fmt.Println("   → Search learnings: /tasuku:list or tk find \"keyword\"")
		fmt.Println("   → Track blocker: /tasuku:add \"...\" --tag blocker")
		fmt.Println("   → Break it down: /tasuku:add \"...\" --parent <current-task>")
		fmt.Println()
		outputCount++
	}

	// === 11. SHIPPING/DEPLOYMENT CHECKPOINT ===
	if config["shipping_check"] && outputCount < maxOutputs && detectShippingIntent(promptLower) {
		warnings := checkPreShipState(f)
		if len(warnings) > 0 {
			fmt.Println("🚀 Pre-ship checklist:")
			for _, w := range warnings {
				fmt.Printf("   ⚠️  %s\n", w)
			}
			fmt.Println()
			outputCount++
		}
	}

	// === 12. BUG REPORT DETECTION ===
	if config["bug_detection"] && outputCount < maxOutputs && detectBugReport(promptLower) {
		if !hasRelatedBugTask(f.Tasks, promptLower) {
			fmt.Println("🐛 This sounds like a bug report.")
			fmt.Println("   Track it:")
			fmt.Println("   → /tasuku:add \"description\" --tag bug --priority high")
			fmt.Println()
			outputCount++
		}
	}

	// === 13. SIGNIFICANT WORK DETECTION ===
	if config["work_detection"] && outputCount < maxOutputs && detectSignificantWork(promptLower) {
		hasInProgress := false
		for _, t := range f.Tasks {
			if t.Status == task.StatusInProgress {
				hasInProgress = true
				break
			}
		}
		if !hasInProgress {
			fmt.Println("💡 This looks like significant work.")
			fmt.Println("   Consider creating a task to track it:")
			fmt.Println("   → /tasuku:add \"description\" --priority high")
			fmt.Println()
		}
	}

	// === 14. ARCHITECTURE EXPLANATION DETECTION ===
	// Detect when user explains "why" or "because we" about design decisions
	if config["architecture_explanation"] && outputCount < maxOutputs && detectArchitectureExplanation(userPrompt) {
		decisionContent := extractDecisionContent(userPrompt)
		if decisionContent != "" {
			fmt.Println("🏛️  This sounds like an architectural decision.")
			fmt.Println("   Record it so future agents understand WHY:")
			fmt.Printf("   → tk decide --id <name> --chose \"...\" --over \"...\" --because \"%s\"\n", escapeForShell(cmdutil.Truncate(decisionContent, 60)))
			fmt.Println()
			fmt.Println("   Agent: If you just learned this, call tk_decide to capture it.")
			fmt.Println()
			outputCount++
		}
	}

	// === 15. USER PREFERENCE DETECTION ===
	// Detect when user states preferences about approach, style, or tooling
	if config["preference_stated"] && outputCount < maxOutputs && detectUserPreference(userPrompt) {
		preferenceContent := extractPreferenceContent(userPrompt)
		if preferenceContent != "" && !hasSimilarLearning(f.Context.Learnings, preferenceContent) {
			fmt.Println("✨ User preference detected:")
			fmt.Printf("   \"%s\"\n", cmdutil.Truncate(preferenceContent, 70))
			fmt.Println()
			fmt.Println("   Capture for consistency across sessions:")
			fmt.Printf("   → /tasuku:learn \"%s\"\n", escapeForShell(preferenceContent))
			fmt.Println()
			fmt.Println("   Agent: If you acknowledge this, call tk_learn to record it.")
			fmt.Println()
			outputCount++
		}
	}

	return nil
}

// detectLearningIntent checks if user is sharing a learning
func detectLearningIntent(prompt string) bool {
	learningPatterns := []string{
		"til ", "til:", "i learned", "i just learned",
		"note to self", "good to know", "remember that",
		"found out that", "discovered that", "realized that",
		"turns out", "apparently ", "interesting:",
	}
	for _, pattern := range learningPatterns {
		if strings.Contains(prompt, pattern) {
			return true
		}
	}
	return false
}

// extractLearningContent extracts the learning from a prompt
func extractLearningContent(prompt string) string {
	// Try to extract content after learning indicators
	indicators := []string{
		"til ", "til:", "i learned that ", "i just learned that ",
		"note to self: ", "good to know: ", "remember that ",
		"found out that ", "discovered that ", "realized that ",
		"turns out ", "apparently ",
	}
	lower := strings.ToLower(prompt)
	for _, ind := range indicators {
		if idx := strings.Index(lower, ind); idx != -1 {
			content := prompt[idx+len(ind):]
			content = strings.TrimSpace(content)
			// Take first sentence
			if endIdx := strings.IndexAny(content, ".!?"); endIdx != -1 {
				content = content[:endIdx+1]
			}
			if len(content) > 10 && len(content) < 200 {
				return content
			}
		}
	}
	return ""
}

// detectDecisionPoint checks if user is comparing options
func detectDecisionPoint(prompt string) bool {
	decisionPatterns := []string{
		" or ", "should we use", "should i use",
		"which is better", "comparing", "trade-off",
		"pros and cons", "versus", " vs ",
		"what's the best way", "best approach",
		"deciding between", "choice between",
	}
	for _, pattern := range decisionPatterns {
		if strings.Contains(prompt, pattern) {
			return true
		}
	}
	return false
}

// detectScopeExpansion checks if user is adding new work
func detectScopeExpansion(prompt string) bool {
	expansionPatterns := []string{
		"also add", "also implement", "also fix",
		"can you also", "while you're at it",
		"another thing", "one more thing",
		"additionally", "in addition",
		"let's also", "we should also",
	}
	for _, pattern := range expansionPatterns {
		if strings.Contains(prompt, pattern) {
			return true
		}
	}
	return false
}

// isRelatedWork checks if new work relates to current task
func isRelatedWork(newWork, currentTaskDesc string) bool {
	newWords := extractKeywords(newWork)
	currentWords := extractKeywords(strings.ToLower(currentTaskDesc))
	overlap := countOverlap(newWords, currentWords)
	return overlap >= 2
}

// detectStuckPattern checks if user expresses frustration or being stuck
func detectStuckPattern(prompt string) bool {
	stuckPatterns := []string{
		"stuck", "frustrated", "frustrating",
		"can't figure", "cannot figure", "don't understand",
		"keeps failing", "keep getting", "still getting",
		"no idea", "not sure why", "don't know why",
		"help me understand", "what am i missing",
		"i give up", "this is hard",
	}
	for _, pattern := range stuckPatterns {
		if strings.Contains(prompt, pattern) {
			return true
		}
	}
	return false
}

// detectShippingIntent checks if user wants to ship/deploy
func detectShippingIntent(prompt string) bool {
	shippingPatterns := []string{
		"ship it", "ship this", "let's ship",
		"deploy", "release", "push to prod",
		"merge to main", "merge to master",
		"we're done", "looks good to ship",
		"ready to release", "time to deploy",
	}
	for _, pattern := range shippingPatterns {
		if strings.Contains(prompt, pattern) {
			return true
		}
	}
	return false
}

// checkPreShipState checks for issues before shipping
func checkPreShipState(f *task.File) []string {
	var warnings []string

	// Check for in-progress tasks
	var inProgressCount int
	for _, t := range f.Tasks {
		if t.Status == task.StatusInProgress {
			inProgressCount++
		}
	}
	if inProgressCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d task(s) still in_progress", inProgressCount))
	}

	// Check for running timers
	var timerCount int
	for _, t := range f.Tasks {
		if t.IsTimerRunning() {
			timerCount++
		}
	}
	if timerCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d timer(s) still running", timerCount))
	}

	// Check for uncommitted changes (if in git repo)
	if out, err := execCommand("git", "status", "--porcelain"); err == nil && len(strings.TrimSpace(out)) > 0 {
		lineCount := len(strings.Split(strings.TrimSpace(out), "\n"))
		warnings = append(warnings, fmt.Sprintf("%d uncommitted file(s)", lineCount))
	}

	return warnings
}

// detectSessionContinuity checks if user wants to continue previous work
func detectSessionContinuity(prompt string) bool {
	continuityPatterns := []string{
		"continue", "resume", "pick up", "where we left",
		"keep working", "back to", "let's continue",
		"continuing", "picking up", "get back to",
	}
	for _, pattern := range continuityPatterns {
		if strings.Contains(prompt, pattern) {
			return true
		}
	}
	return false
}

// getInProgressTasks returns all in-progress tasks
func getInProgressTasks(f *task.File) []taskWithID {
	var result []taskWithID
	for id, t := range f.Tasks {
		if t.Status == task.StatusInProgress {
			result = append(result, taskWithID{id: id, task: t})
		}
	}
	return result
}

// looksLikeQuestion checks if the prompt is asking a question
func looksLikeQuestion(prompt string) bool {
	questionIndicators := []string{
		"how do", "how should", "how can", "how does",
		"what is", "what are", "what should", "what's the",
		"why do", "why does", "why is", "why are",
		"which", "should i", "should we", "can i", "can we",
		"is there", "are there", "do we", "does the",
		"?",
	}
	for _, indicator := range questionIndicators {
		if strings.Contains(prompt, indicator) {
			return true
		}
	}
	return false
}

// findRelevantDecisions finds decisions matching the prompt keywords
func findRelevantDecisions(decisions []task.Decision, prompt string) []task.Decision {
	promptWords := extractKeywords(prompt)
	if len(promptWords) < 2 {
		return nil
	}

	var relevant []task.Decision
	for _, d := range decisions {
		// Build searchable text from decision
		searchText := strings.ToLower(d.ID + " " + d.Chose + " " + d.Because + " " + strings.Join(d.Over, " "))
		decisionWords := extractKeywords(searchText)

		// Check for significant overlap (at least 2 matching keywords)
		overlap := countOverlap(promptWords, decisionWords)
		if overlap >= 2 {
			relevant = append(relevant, d)
		}
	}

	// Limit results
	if len(relevant) > 3 {
		relevant = relevant[:3]
	}
	return relevant
}

// findRelevantLearnings finds learnings matching the prompt keywords
func findRelevantLearnings(learnings []task.Learning, prompt string) []task.Learning {
	promptWords := extractKeywords(prompt)
	if len(promptWords) < 2 {
		return nil
	}

	var relevant []task.Learning
	for _, l := range learnings {
		learningWords := extractKeywords(strings.ToLower(l.Text))

		// Check for significant overlap (at least 2 matching keywords)
		overlap := countOverlap(promptWords, learningWords)
		if overlap >= 2 {
			relevant = append(relevant, l)
		}
	}

	// Limit results
	if len(relevant) > 3 {
		relevant = relevant[:3]
	}
	return relevant
}

// findRelatedTasks finds tasks matching the prompt keywords (not exact ID match)
func findRelatedTasks(tasks map[string]task.Task, prompt string) []taskWithID {
	promptWords := extractKeywords(prompt)
	if len(promptWords) < 2 {
		return nil
	}

	var related []taskWithID
	for id, t := range tasks {
		// Skip done tasks
		if t.Status == task.StatusDone {
			continue
		}

		// Build searchable text from task
		searchText := strings.ToLower(id + " " + t.Description)
		taskWords := extractKeywords(searchText)

		// Check for significant overlap (at least 2 matching keywords)
		overlap := countOverlap(promptWords, taskWords)
		if overlap >= 2 {
			related = append(related, taskWithID{id: id, task: t})
		}
	}

	// Limit results
	if len(related) > 3 {
		related = related[:3]
	}
	return related
}

// detectBugReport checks if the prompt describes a bug
func detectBugReport(prompt string) bool {
	bugIndicators := []string{
		"bug", "broken", "crash", "crashes", "crashing",
		"error", "errors", "failing", "fails", "failed",
		"not working", "doesn't work", "doesn't work",
		"issue", "problem", "wrong", "incorrect",
		"unexpected", "weird behavior", "strange behavior",
	}
	for _, indicator := range bugIndicators {
		if strings.Contains(prompt, indicator) {
			return true
		}
	}
	return false
}

// hasRelatedBugTask checks if there's already a similar bug task
func hasRelatedBugTask(tasks map[string]task.Task, prompt string) bool {
	promptWords := extractKeywords(prompt)
	if len(promptWords) < 2 {
		return false
	}

	for _, t := range tasks {
		// Check if task has bug tag or bug-like keywords in description
		hasBugTag := false
		for _, tag := range t.Tags {
			if tag == "bug" {
				hasBugTag = true
				break
			}
		}
		if !hasBugTag && !isBugFixTask(t.Description) {
			continue
		}

		// Check keyword overlap
		taskWords := extractKeywords(strings.ToLower(t.Description))
		overlap := countOverlap(promptWords, taskWords)
		if overlap >= 2 {
			return true
		}
	}
	return false
}

// detectSignificantWork checks if prompt suggests significant work
func detectSignificantWork(prompt string) bool {
	workKeywords := []string{
		"implement", "add feature", "build", "create new",
		"fix bug", "fix the", "debug", "resolve issue",
		"refactor", "rewrite", "migrate",
		"set up", "configure", "integrate",
		"add support for", "enable",
	}
	for _, kw := range workKeywords {
		if strings.Contains(prompt, kw) {
			return true
		}
	}
	return false
}

// detectArchitectureExplanation checks if user is explaining an architectural decision
func detectArchitectureExplanation(prompt string) bool {
	lower := strings.ToLower(prompt)
	explanationPatterns := []string{
		"because we", "because it", "because they",
		"we chose", "we decided", "we use",
		"the reason is", "the reason we", "reason being",
		"that's why", "this is why", "which is why",
		"we went with", "we opted for", "we picked",
		"the decision was", "decided to use", "chose to use",
		"designed it this way", "built it this way",
		"architecture is", "pattern is",
		"trade-off", "tradeoff",
	}
	for _, pattern := range explanationPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// extractDecisionContent extracts the decision reasoning from a prompt
func extractDecisionContent(prompt string) string {
	lower := strings.ToLower(prompt)
	// Try to find the explanation portion
	indicators := []string{
		"because ", "the reason is ", "reason being ",
		"that's why ", "this is why ", "which is why ",
	}
	for _, ind := range indicators {
		if idx := strings.Index(lower, ind); idx != -1 {
			content := prompt[idx+len(ind):]
			content = strings.TrimSpace(content)
			// Take first sentence
			if endIdx := strings.IndexAny(content, ".!?"); endIdx != -1 && endIdx < 150 {
				content = content[:endIdx+1]
			} else if len(content) > 150 {
				content = content[:150]
			}
			if len(content) > 15 {
				return content
			}
		}
	}
	return ""
}

// detectUserPreference checks if user is stating a preference
func detectUserPreference(prompt string) bool {
	lower := strings.ToLower(prompt)
	preferencePatterns := []string{
		"i prefer", "i like to", "i always",
		"i don't like", "i hate", "i avoid",
		"please always", "please never", "please don't",
		"always use", "never use", "don't use",
		"my preference", "my style", "my approach",
		"i want you to", "i'd like you to",
		"from now on", "going forward",
		"use this style", "follow this pattern",
	}
	for _, pattern := range preferencePatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// extractPreferenceContent extracts the preference from a prompt
func extractPreferenceContent(prompt string) string {
	lower := strings.ToLower(prompt)
	// Try to find preference indicators and extract content
	indicators := []string{
		"i prefer ", "i like to ", "i always ",
		"please always ", "please never ", "always use ",
		"never use ", "from now on ", "going forward ",
	}
	for _, ind := range indicators {
		if idx := strings.Index(lower, ind); idx != -1 {
			// Get the rest starting from this indicator
			content := prompt[idx:]
			content = strings.TrimSpace(content)
			// Take first sentence or clause
			if endIdx := strings.IndexAny(content, ".!?,;"); endIdx != -1 && endIdx < 120 {
				content = content[:endIdx]
			} else if len(content) > 120 {
				content = content[:120]
			}
			if len(content) > 15 {
				// Clean up and capitalize first letter
				content = strings.TrimSpace(content)
				if len(content) > 0 {
					return strings.ToUpper(string(content[0])) + content[1:]
				}
			}
		}
	}
	return ""
}

// looksLikeInstruction checks if a message looks like an instruction to an agent
// rather than casual conversation
func looksLikeInstruction(text string) bool {
	lower := strings.ToLower(text)

	// Instruction indicators - imperative tone or directive language
	instructionIndicators := []string{
		// Imperative verbs
		"make sure", "ensure", "always", "never", "don't", "do not",
		"use ", "avoid", "prefer", "remember to", "be sure to",
		// Directive language
		"you should", "you must", "you need to", "please ",
		"when you", "if you", "try to",
		// Code/development context
		"the code", "the api", "the function", "the class",
		"sdk", "library", "framework", "codebase",
		"bug", "fix", "implement", "refactor",
	}

	for _, indicator := range instructionIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}

	return false
}

// extractRulePortion extracts the rule-like sentence from a longer message
func extractRulePortion(text string) string {
	// Split by sentence boundaries
	sentences := splitSentences(text)

	// Find the sentence containing the rule pattern
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if task.IsRuleLearning(sentence) && len(sentence) > 10 {
			return sentence
		}
	}

	// If no single sentence matches, return empty (full text might be too noisy)
	return ""
}

// splitSentences splits text into sentences
func splitSentences(text string) []string {
	// Simple sentence splitting on common terminators
	// Replace multiple terminators with single marker
	text = strings.ReplaceAll(text, "...", ".")
	text = strings.ReplaceAll(text, ". ", ".|")
	text = strings.ReplaceAll(text, "! ", "!|")
	text = strings.ReplaceAll(text, "? ", "?|")
	text = strings.ReplaceAll(text, ".\n", ".|\n")
	text = strings.ReplaceAll(text, "!\n", "!|\n")
	text = strings.ReplaceAll(text, "?\n", "?|\n")

	parts := strings.Split(text, "|")
	var sentences []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			sentences = append(sentences, p)
		}
	}
	return sentences
}

// hasSimilarLearning checks if we already have a similar learning recorded
func hasSimilarLearning(learnings []task.Learning, newText string) bool {
	newLower := strings.ToLower(newText)
	newWords := extractKeywords(newLower)

	for _, l := range learnings {
		existingLower := strings.ToLower(l.Text)

		// Exact substring match
		if strings.Contains(existingLower, newLower) || strings.Contains(newLower, existingLower) {
			return true
		}

		// Keyword overlap check (if >60% keywords match, consider similar)
		existingWords := extractKeywords(existingLower)
		overlap := countOverlap(newWords, existingWords)
		if len(newWords) > 0 && float64(overlap)/float64(len(newWords)) > 0.6 {
			return true
		}
	}

	return false
}

// === TEST FAILURE STATE TRACKING ===
// These functions track test failures to detect when an agent fixes a failing test,
// prompting them to document the learning.

// testFailureState holds the state of the last test failure
type testFailureState struct {
	FailedAt time.Time `json:"failed_at"`
	Command  string    `json:"command"`
}

// getTestFailureStatePath returns the path to the test failure state file.
// Uses .tasuku/ if it exists, otherwise ~/.cache/tasuku/
func getTestFailureStatePath() string {
	// Try project-local first
	if _, err := os.Stat(".tasuku"); err == nil {
		return filepath.Join(".tasuku", ".test-failure-state.json")
	}

	// Fall back to cache directory
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	tasukuCache := filepath.Join(cacheDir, "tasuku")
	os.MkdirAll(tasukuCache, 0755)
	return filepath.Join(tasukuCache, "test-failure-state.json")
}

// saveTestFailureState saves the current test failure state
func saveTestFailureState(command string) error {
	state := testFailureState{
		FailedAt: time.Now(),
		Command:  command,
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(getTestFailureStatePath(), data, 0644)
}

// getLastTestFailure reads the last test failure state if it exists
func getLastTestFailure() (*testFailureState, error) {
	data, err := os.ReadFile(getTestFailureStatePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var state testFailureState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// clearTestFailureState removes the test failure state file
func clearTestFailureState() {
	os.Remove(getTestFailureStatePath())
}

// isRecentTestFailure checks if a test failure occurred within the given duration
func isRecentTestFailure(maxAge time.Duration) bool {
	state, err := getLastTestFailure()
	if err != nil || state == nil {
		return false
	}
	return time.Since(state.FailedAt) < maxAge
}

// detectTestSuccess checks test output for success indicators
func detectTestSuccess(output string) bool {
	successPatterns := []string{
		"pass", "passed", "ok ",
		"all tests passed", "tests passed",
		"success", "succeeded",
		"exit status 0", "exit code 0",
		"0 failures", "0 failed", "0 errors",
	}
	for _, pattern := range successPatterns {
		if strings.Contains(output, pattern) {
			return true
		}
	}
	return false
}

// === INVESTIGATION PATTERN DETECTION ===
// These functions track file reads to detect when an agent is deeply investigating
// a file before editing it, suggesting they may have discovered something worth documenting.

// investigationState tracks file read patterns during a session
type investigationState struct {
	FileReads   map[string]int `json:"file_reads"`   // file path -> read count
	LastUpdated time.Time      `json:"last_updated"`
}

const investigationThreshold = 3 // Number of reads before prompting
const investigationMaxAge = 30 * time.Minute

// getInvestigationStatePath returns the path to the investigation state file
func getInvestigationStatePath() string {
	// Try project-local first
	if _, err := os.Stat(".tasuku"); err == nil {
		return filepath.Join(".tasuku", ".investigation-state.json")
	}

	// Fall back to cache directory
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	tasukuCache := filepath.Join(cacheDir, "tasuku")
	os.MkdirAll(tasukuCache, 0755)
	return filepath.Join(tasukuCache, "investigation-state.json")
}

// loadInvestigationState loads the current investigation state
func loadInvestigationState() (*investigationState, error) {
	data, err := os.ReadFile(getInvestigationStatePath())
	if err != nil {
		if os.IsNotExist(err) {
			return &investigationState{
				FileReads:   make(map[string]int),
				LastUpdated: time.Now(),
			}, nil
		}
		return nil, err
	}
	var state investigationState
	if err := json.Unmarshal(data, &state); err != nil {
		return &investigationState{
			FileReads:   make(map[string]int),
			LastUpdated: time.Now(),
		}, nil
	}

	// Reset if state is too old (stale from previous session)
	if time.Since(state.LastUpdated) > investigationMaxAge {
		return &investigationState{
			FileReads:   make(map[string]int),
			LastUpdated: time.Now(),
		}, nil
	}

	if state.FileReads == nil {
		state.FileReads = make(map[string]int)
	}
	return &state, nil
}

// saveInvestigationState saves the current investigation state
func saveInvestigationState(state *investigationState) error {
	state.LastUpdated = time.Now()
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(getInvestigationStatePath(), data, 0644)
}

// recordFileRead increments the read count for a file
func recordFileRead(filePath string) error {
	state, err := loadInvestigationState()
	if err != nil {
		return err
	}
	state.FileReads[filePath]++
	return saveInvestigationState(state)
}

// checkInvestigationPattern checks if editing a file after multiple reads
// Returns (wasInvestigating, readCount)
func checkInvestigationPattern(filePath string) (bool, int) {
	state, err := loadInvestigationState()
	if err != nil {
		return false, 0
	}

	count := state.FileReads[filePath]
	if count >= investigationThreshold {
		// Clear this file's count so we don't prompt again
		delete(state.FileReads, filePath)
		saveInvestigationState(state)
		return true, count
	}
	return false, count
}

// handleReadCheck tracks file reads for investigation pattern detection
func handleReadCheck(config featureConfig, toolInput string) error {
	if !config["investigation_pattern"] {
		return nil
	}

	// Parse the Read tool input
	var input struct {
		FilePath string `json:"file_path"`
	}
	if err := json.Unmarshal([]byte(toolInput), &input); err != nil {
		return nil
	}

	if input.FilePath == "" {
		return nil
	}

	// Record the read
	recordFileRead(input.FilePath)
	return nil
}

// handleEditCheck checks for investigation pattern when editing
func handleEditCheck(config featureConfig, toolInput string) error {
	if !config["investigation_pattern"] {
		return nil
	}

	// Parse the Edit tool input
	var input struct {
		FilePath string `json:"file_path"`
	}
	if err := json.Unmarshal([]byte(toolInput), &input); err != nil {
		return nil
	}

	if input.FilePath == "" {
		return nil
	}

	// Check if this file was being investigated
	wasInvestigating, readCount := checkInvestigationPattern(input.FilePath)
	if wasInvestigating {
		// Get just the filename for display
		filename := filepath.Base(input.FilePath)

		fmt.Println("🔍 INVESTIGATION PATTERN DETECTED")
		fmt.Println()
		fmt.Printf("   You read %s %d times before editing.\n", filename, readCount)
		fmt.Println("   This suggests you discovered something non-obvious!")
		fmt.Println()
		fmt.Println("   📝 Document what you learned:")
		fmt.Println("   → What gotcha or edge case did you find?")
		fmt.Println("   → What assumption was wrong?")
		fmt.Println()
		fmt.Println("   /tasuku:learn \"insight here\"")
		fmt.Println()
	}

	return nil
}
