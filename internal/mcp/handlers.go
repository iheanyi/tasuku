package mcp

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/iheanyi/tasuku/internal/rules"
	"github.com/iheanyi/tasuku/internal/task"
)

// HandleToolCall processes a tool call and returns the result.
func (s *Server) HandleToolCall(name string, args map[string]interface{}) (interface{}, error) {
	switch name {
	// Tier 1: Core tools (kept individual)
	case "tk_help":
		return s.handleHelp(args)
	case "tk_list":
		return s.handleList(args)
	case "tk_add":
		return s.handleAdd(args)
	case "tk_start":
		return s.handleStart(args)
	case "tk_done":
		return s.handleDone(args)
	case "tk_show":
		return s.handleShow(args)
	case "tk_note":
		return s.handleNote(args)
	case "tk_context":
		return s.handleContext(args)
	case "tk_find":
		return s.handleFind(args)
	case "tk_learn":
		return s.handleLearn(args)
	case "tk_decide":
		return s.handleDecide(args)
	case "tk_block":
		return s.handleBlock(args)
	case "tk_ready":
		return s.handleReady(args)
	case "tk_deps":
		return s.handleDeps(args)
	case "tk_suggest":
		return s.handleSuggest(args)

	// Tier 2: Consolidated tools
	case "tk_task":
		return s.handleTaskAction(args)
	case "tk_metadata":
		return s.handleMetadataAction(args)
	case "tk_manage":
		return s.handleManageAction(args)

	// Tier 3: Useful but less frequent tools
	case "tk_stats":
		return s.handleStats(args)
	case "tk_health":
		return s.handleHealth(args)

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// handleHelp returns contextual help for Tasuku tools
func (s *Server) handleHelp(args map[string]interface{}) (interface{}, error) {
	topic, _ := args["topic"].(string)
	command, _ := args["command"].(string)

	if command != "" {
		return s.getCommandHelp(command), nil
	}

	switch topic {
	case "tasks":
		return map[string]interface{}{
			"topic":       "Task Operations",
			"description": "Task lifecycle management",
			"tools": map[string]string{
				"tk_add":   "Create a new task with description, optional ID, priority, and tags",
				"tk_start": "Mark task as in_progress, optionally start timer",
				"tk_done":  "Mark task as completed, auto-stops timer",
				"tk_show":  "View task details including notes and dependencies",
				"tk_list":  "List tasks with optional filters (status, tag, owner)",
			},
			"consolidated_tool": "tk_task",
			"tk_task_actions": map[string]string{
				"edit":     "Update task description",
				"delete":   "Permanently delete a task",
				"pause":    "Revert in_progress → ready",
				"block":    "Mark task as blocked by others",
				"unblock":  "Remove blockers",
				"priority": "Set priority (0-4 or critical/high/normal/low/backlog)",
				"owner":    "Set or clear task owner",
				"archive":  "Archive a done task",
				"restore":  "Restore archived task",
				"claim":    "Claim for exclusive agent work",
				"release":  "Release claimed task",
				"who":      "Show task assignments by owner",
			},
		}, nil

	case "metadata":
		return map[string]interface{}{
			"topic":       "Metadata Operations",
			"description": "Manage tags, fields, and notes on tasks",
			"tool":        "tk_metadata",
			"actions": map[string]string{
				"tag_add":      "Add a tag (bug, feature, urgent, etc.)",
				"tag_remove":   "Remove a tag",
				"field_set":    "Set custom field (estimate, component, pr, etc.)",
				"field_remove": "Remove custom field",
				"note_list":    "List notes for a task or all notes",
				"note_remove":  "Remove a note",
			},
			"common_tags":   []string{"bug", "feature", "refactor", "docs", "test", "security", "performance", "tech-debt", "urgent"},
			"common_fields": []string{"estimate", "component", "pr", "issue", "approach", "reviewer"},
		}, nil

	case "knowledge":
		return map[string]interface{}{
			"topic":       "Knowledge Capture",
			"description": "Record learnings and decisions for future sessions",
			"tools": map[string]string{
				"tk_learn":   "Record a learning or insight (use 'Never X' or 'Always Y' for rules)",
				"tk_decide":  "Record an architectural decision with reasoning",
				"tk_context": "Get full project context including tasks, learnings, decisions",
				"tk_find":    "Search across all content",
			},
			"consolidated_tool": "tk_manage",
			"tk_manage_actions": map[string]string{
				"learning_list":    "List all learnings",
				"learning_promote": "Promote to permanent docs",
				"learning_remove":  "Remove a learning",
				"learning_rules":   "List rule learnings (never/always)",
				"decision_list":    "List all decisions",
				"decision_remove":  "Remove a decision",
			},
		}, nil

	case "multiagent":
		return map[string]interface{}{
			"topic":       "Multi-Agent Coordination",
			"description": "Coordinate work between multiple agents",
			"tool":        "tk_task",
			"actions": map[string]string{
				"claim":   "Claim task for exclusive work: tk_task action=claim id=<id> agent=<name>",
				"release": "Release when done: tk_task action=release id=<id>",
				"who":     "See assignments: tk_task action=who",
				"owner":   "Set owner: tk_task action=owner id=<id> owner=<name>",
			},
			"workflow": "1. Check who is working on what (action=who)\n2. Claim a task before starting\n3. Release when done or pausing",
		}, nil

	case "archive":
		return map[string]interface{}{
			"topic":       "Archive Operations",
			"description": "Archive completed tasks to keep lists lean",
			"tools": map[string]string{
				"tk_task (archive)": "Archive a done task: tk_task action=archive id=<id>",
				"tk_task (restore)": "Restore archived task: tk_task action=restore id=<id>",
				"tk_manage":         "List/bulk archive operations",
				"tk_list":           "Include archived: tk_list include_archived=true",
			},
			"tk_manage_actions": map[string]string{
				"archive_list": "List all archived tasks",
				"archive_all":  "Archive done tasks older than duration (e.g., '7d')",
			},
		}, nil

	case "install":
		return map[string]interface{}{
			"topic":       "Installation",
			"description": "Install Tasuku via CLI (not available via MCP)",
			"cli_commands": map[string]string{
				"tk mcp install":    "Install MCP server to AI tools",
				"tk plugin install": "Install slash commands",
				"tk hooks install":  "Install session hooks",
			},
			"note": "Installation commands are CLI-only for security. Run them in your terminal.",
		}, nil

	default: // overview
		return map[string]interface{}{
			"topic":       "Tasuku Overview",
			"description": "Agent-first task management system",
			"tool_count":  17,
			"tools": map[string]string{
				"tk_help":     "This help system",
				"tk_list":     "List tasks",
				"tk_add":      "Create task",
				"tk_start":    "Start working",
				"tk_done":     "Complete task",
				"tk_block":    "Block a task",
				"tk_show":     "Task details",
				"tk_note":     "Add note",
				"tk_context":  "Full project state",
				"tk_find":     "Search everything",
				"tk_learn":    "Record learning",
				"tk_decide":   "Record decision",
				"tk_task":     "Consolidated task ops (edit, delete, pause, block, etc.)",
				"tk_metadata": "Consolidated metadata ops (tags, fields, notes)",
				"tk_manage":   "Consolidated management (learnings, decisions, archive)",
				"tk_stats":    "Project statistics",
				"tk_health":   "Health check",
			},
			"topics": []string{"tasks", "metadata", "knowledge", "multiagent", "archive", "install"},
			"tip":    "Use tk_help topic=<topic> or tk_help command=<tool> for details",
		}, nil
	}
}

// getCommandHelp returns detailed help for a specific command
func (s *Server) getCommandHelp(command string) map[string]interface{} {
	switch command {
	case "tk_task":
		return map[string]interface{}{
			"command":     "tk_task",
			"description": "Consolidated task operations",
			"required":    []string{"action"},
			"actions": []map[string]interface{}{
				{"action": "edit", "params": "id, description", "example": `tk_task action=edit id=my-task description="Updated desc"`},
				{"action": "delete", "params": "id", "example": `tk_task action=delete id=my-task`},
				{"action": "pause", "params": "id", "example": `tk_task action=pause id=my-task`},
				{"action": "block", "params": "id, blocked_by", "example": `tk_task action=block id=my-task blocked_by=["blocker-1"]`},
				{"action": "unblock", "params": "id, from (optional)", "example": `tk_task action=unblock id=my-task`},
				{"action": "priority", "params": "id, priority", "example": `tk_task action=priority id=my-task priority=high`},
				{"action": "owner", "params": "id, owner", "example": `tk_task action=owner id=my-task owner=agent-1`},
				{"action": "archive", "params": "id, summary (optional)", "example": `tk_task action=archive id=my-task`},
				{"action": "restore", "params": "id", "example": `tk_task action=restore id=my-task`},
				{"action": "claim", "params": "id, agent", "example": `tk_task action=claim id=my-task agent=worker-1`},
				{"action": "release", "params": "id", "example": `tk_task action=release id=my-task`},
				{"action": "who", "params": "none", "example": `tk_task action=who`},
			},
		}
	case "tk_metadata":
		return map[string]interface{}{
			"command":     "tk_metadata",
			"description": "Manage tags, fields, and notes",
			"required":    []string{"action"},
			"actions": []map[string]interface{}{
				{"action": "tag_add", "params": "id, tag", "example": `tk_metadata action=tag_add id=my-task tag=bug`},
				{"action": "tag_remove", "params": "id, tag", "example": `tk_metadata action=tag_remove id=my-task tag=bug`},
				{"action": "field_set", "params": "id, key, value", "example": `tk_metadata action=field_set id=my-task key=estimate value="2h"`},
				{"action": "field_remove", "params": "id, key", "example": `tk_metadata action=field_remove id=my-task key=estimate`},
				{"action": "note_list", "params": "task_id (optional)", "example": `tk_metadata action=note_list task_id=my-task`},
				{"action": "note_remove", "params": "task_id, note_id", "example": `tk_metadata action=note_remove task_id=my-task note_id=n1`},
			},
		}
	case "tk_manage":
		return map[string]interface{}{
			"command":     "tk_manage",
			"description": "Manage learnings, decisions, and archive",
			"required":    []string{"action"},
			"actions": []map[string]interface{}{
				{"action": "learning_list", "params": "none", "example": `tk_manage action=learning_list`},
				{"action": "learning_promote", "params": "id, to (optional), keep (optional)", "example": `tk_manage action=learning_promote id=l1`},
				{"action": "learning_remove", "params": "id", "example": `tk_manage action=learning_remove id=l1`},
				{"action": "learning_rules", "params": "none", "example": `tk_manage action=learning_rules`},
				{"action": "decision_list", "params": "none", "example": `tk_manage action=decision_list`},
				{"action": "decision_remove", "params": "id", "example": `tk_manage action=decision_remove id=auth-choice`},
				{"action": "archive_list", "params": "none", "example": `tk_manage action=archive_list`},
				{"action": "archive_all", "params": "older_than", "example": `tk_manage action=archive_all older_than=7d`},
			},
		}
	default:
		return map[string]interface{}{
			"error":           fmt.Sprintf("Unknown command: %s", command),
			"available_tools": []string{"tk_help", "tk_list", "tk_add", "tk_start", "tk_done", "tk_show", "tk_note", "tk_context", "tk_find", "tk_learn", "tk_decide", "tk_task", "tk_metadata", "tk_manage", "tk_stats", "tk_health"},
		}
	}
}

// handleTaskAction handles consolidated task operations
func (s *Server) handleTaskAction(args map[string]interface{}) (interface{}, error) {
	action, _ := args["action"].(string)
	if action == "" {
		return nil, fmt.Errorf("action is required")
	}

	switch action {
	case "edit":
		return s.handleEdit(args)
	case "delete":
		return s.handleDelete(args)
	case "pause":
		return s.handlePause(args)
	case "block":
		return s.handleBlock(args)
	case "unblock":
		return s.handleUnblock(args)
	case "priority":
		return s.handlePriority(args)
	case "owner":
		return s.handleOwner(args)
	case "archive":
		// Map 'id' to 'task_id' for archive handler
		if id, ok := args["id"].(string); ok && id != "" {
			args["task_id"] = id
		}
		return s.handleArchive(args)
	case "restore":
		// Map 'id' to 'task_id' for restore handler
		if id, ok := args["id"].(string); ok && id != "" {
			args["task_id"] = id
		}
		return s.handleArchiveRestore(args)
	case "claim":
		return s.handleClaim(args)
	case "release":
		return s.handleRelease(args)
	case "who":
		return s.handleWho(args)
	default:
		return nil, fmt.Errorf("unknown action: %s (valid: edit, delete, pause, block, unblock, priority, owner, archive, restore, claim, release, who)", action)
	}
}

// handleMetadataAction handles consolidated metadata operations
func (s *Server) handleMetadataAction(args map[string]interface{}) (interface{}, error) {
	action, _ := args["action"].(string)
	if action == "" {
		return nil, fmt.Errorf("action is required")
	}

	switch action {
	case "tag_add":
		return s.handleTagAdd(args)
	case "tag_remove":
		return s.handleTagRemove(args)
	case "field_set":
		return s.handleFieldSet(args)
	case "field_remove":
		return s.handleFieldRemove(args)
	case "note_list":
		return s.handleNoteList(args)
	case "note_remove":
		return s.handleNoteRemove(args)
	default:
		return nil, fmt.Errorf("unknown action: %s (valid: tag_add, tag_remove, field_set, field_remove, note_list, note_remove)", action)
	}
}

// handleManageAction handles consolidated management operations
func (s *Server) handleManageAction(args map[string]interface{}) (interface{}, error) {
	action, _ := args["action"].(string)
	if action == "" {
		return nil, fmt.Errorf("action is required")
	}

	switch action {
	case "learning_list":
		return s.handleLearningList(args)
	case "learning_promote":
		return s.handleLearningPromote(args)
	case "learning_remove":
		return s.handleLearningRemove(args)
	case "learning_rules":
		return s.handleLearningRules(args)
	case "decision_list":
		return s.handleDecisionList(args)
	case "decision_remove":
		return s.handleDecisionRemove(args)
	case "archive_list":
		return s.handleArchiveList(args)
	case "archive_all":
		return s.handleArchiveAll(args)
	default:
		return nil, fmt.Errorf("unknown action: %s (valid: learning_list, learning_promote, learning_remove, learning_rules, decision_list, decision_remove, archive_list, archive_all)", action)
	}
}

func (s *Server) handleList(args map[string]interface{}) (interface{}, error) {
	// Parse filter arguments
	status, _ := args["status"].(string)
	tagFilter, _ := args["tag"].(string)
	ownerFilter, _ := args["owner"].(string)
	treeView, _ := args["tree"].(bool)
	includeArchived, _ := args["include_archived"].(bool)

	type taskResult struct {
		ID          string       `json:"id"`
		Status      string       `json:"status"`
		Description string       `json:"description"`
		BlockedBy   []string     `json:"blocked_by,omitempty"`
		Owner       *string      `json:"owner,omitempty"`
		ParentID    string       `json:"parent_id,omitempty"`
		Priority    int          `json:"priority,omitempty"`
		Tags        []string     `json:"tags,omitempty"`
		Children    []taskResult `json:"children,omitempty"`
	}

	// Use index-based read: reads 1 file (index.json) instead of N task files.
	summaries, err := s.store.ListFromIndex()
	if err != nil {
		return nil, err
	}

	// Filter tasks from index summaries
	var results []taskResult
	for _, t := range summaries {
		if status != "" && t.Status != status {
			continue
		}
		if tagFilter != "" && !t.HasTag(tagFilter) {
			continue
		}
		if ownerFilter != "" {
			if t.Owner == nil || *t.Owner != ownerFilter {
				continue
			}
		}
		results = append(results, taskResult{
			ID:          t.ID,
			Status:      t.Status,
			Description: t.Description,
			BlockedBy:   t.BlockedBy,
			Owner:       t.Owner,
			ParentID:    t.ParentID,
			Priority:    t.GetPriority(),
			Tags:        t.Tags,
		})
	}

	// Include archived tasks if requested
	if includeArchived {
		archived, err := s.store.GetArchivedTasks()
		if err == nil {
			for id, t := range archived {
				// Apply the same filters to archived tasks
				if status != "" && status != "archived" {
					continue
				}
				if tagFilter != "" && !t.HasTag(tagFilter) {
					continue
				}
				if ownerFilter != "" {
					if t.Owner == nil || *t.Owner != ownerFilter {
						continue
					}
				}
				results = append(results, taskResult{
					ID:          id,
					Status:      "archived",
					Description: t.Description,
					Tags:        t.Tags,
				})
			}
		}
	}

	// Sort by status priority, then by task priority, then by ID
	statusOrder := map[task.Status]int{
		task.StatusInProgress:   0,
		task.StatusReady:        1,
		task.StatusBlocked:      2,
		task.StatusDone:         3,
		task.StatusArchived: 4,
	}
	slices.SortFunc(results, func(a, b taskResult) int {
		// Status order first
		aStatus := statusOrder[task.Status(a.Status)]
		bStatus := statusOrder[task.Status(b.Status)]
		if aStatus != bStatus {
			return aStatus - bStatus
		}
		// Priority second
		if a.Priority != b.Priority {
			return a.Priority - b.Priority
		}
		// ID alphabetically last
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})

	// If tree view requested, nest children under parents
	if treeView {
		// Build lookup map
		taskMap := make(map[string]*taskResult)
		for i := range results {
			taskMap[results[i].ID] = &results[i]
		}

		// Build tree
		var rootTasks []taskResult
		for i := range results {
			t := &results[i]
			if t.ParentID == "" {
				rootTasks = append(rootTasks, *t)
			} else if parent, ok := taskMap[t.ParentID]; ok {
				parent.Children = append(parent.Children, *t)
			} else {
				// Parent not in filtered results, treat as root
				rootTasks = append(rootTasks, *t)
			}
		}
		return rootTasks, nil
	}

	return results, nil
}

func (s *Server) handleAdd(args map[string]interface{}) (interface{}, error) {
	desc, _ := args["description"].(string)
	id, _ := args["id"].(string)
	priorityStr, _ := args["priority"].(string)
	parentID, _ := args["parent_id"].(string)

	// Parse tags - handle both []interface{} from JSON and []string
	var tags []string
	if tagsRaw, ok := args["tags"].([]interface{}); ok {
		for _, t := range tagsRaw {
			if s, ok := t.(string); ok {
				tags = append(tags, strings.TrimSpace(s))
			}
		}
	} else if tagsStr, ok := args["tags"].([]string); ok {
		for _, t := range tagsStr {
			tags = append(tags, strings.TrimSpace(t))
		}
	}

	// Use index-based read for duplicate detection (1 file instead of N)
	summaries, err := s.store.ListFromIndex()
	if err != nil {
		return nil, err
	}

	// Generate ID if not provided, checking for collisions
	existingIDs := make(map[string]struct{})
	for _, t := range summaries {
		existingIDs[t.ID] = struct{}{}
	}

	if id == "" {
		id = task.GenerateTaskID(desc, existingIDs)
	}

	// Check for potential duplicates (similar descriptions)
	descLower := strings.ToLower(desc)
	var potentialDuplicates []string
	for _, t := range summaries {
		if t.Status != string(task.StatusDone) {
			existingLower := strings.ToLower(t.Description)
			// Check for significant overlap
			if strings.Contains(existingLower, descLower) || strings.Contains(descLower, existingLower) {
				potentialDuplicates = append(potentialDuplicates, t.ID)
			}
		}
	}

	// Parse priority (supports both numeric and named values)
	var priorityPtr *int
	if priorityStr != "" {
		priority := task.ParsePriority(priorityStr)
		if priority == -1 {
			return nil, fmt.Errorf("invalid priority: %s (use 0-4 or critical/high/normal/low/backlog)", priorityStr)
		}
		priorityPtr = &priority
	}

	// Create task - either as subtask or regular task
	if parentID != "" {
		if err := s.store.AddSubtask(id, desc, parentID); err != nil {
			return nil, err
		}
		// Apply priority and tags after creation
		if priorityPtr != nil {
			s.store.SetPriority(id, *priorityPtr)
		}
		for _, tag := range tags {
			s.store.AddTag(id, tag)
		}
	} else {
		if err := s.store.AddTaskWithTags(id, desc, priorityPtr, tags); err != nil {
			return nil, err
		}
	}

	result := map[string]interface{}{"id": id, "status": "created"}

	if priorityPtr != nil {
		result["priority"] = task.PriorityName(*priorityPtr)
	}
	if len(tags) > 0 {
		result["tags"] = tags
	}
	if parentID != "" {
		result["parent_id"] = parentID
	}
	if len(potentialDuplicates) > 0 {
		result["warning"] = fmt.Sprintf("Potential duplicate tasks found: %v. Consider checking these before proceeding.", potentialDuplicates)
	}

	return result, nil
}

func (s *Server) handleStart(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	startTimer, _ := args["start_timer"].(bool)
	unblock, _ := args["unblock"].(bool)

	// Use index for in_progress check (1 file instead of N), then full read only for notes
	summaries, err := s.store.ListFromIndex()
	if err != nil {
		return nil, err
	}

	// Check for other in_progress tasks (warning)
	var otherInProgress []string
	for _, t := range summaries {
		if t.Status == string(task.StatusInProgress) && t.ID != id {
			otherInProgress = append(otherInProgress, t.ID)
		}
	}

	// Read full state only for notes (context from previous sessions)
	var notes []task.Note
	if f, readErr := s.store.Read(); readErr == nil {
		notes = f.Context.Notes[id]
	}

	// Unblock if requested (for blocked tasks)
	if unblock {
		if err := s.store.UnblockTask(id); err != nil {
			return nil, fmt.Errorf("failed to unblock task: %w", err)
		}
	}

	if err := s.store.SetStatus(id, task.StatusInProgress); err != nil {
		return nil, err
	}

	result := map[string]interface{}{"id": id, "status": "in_progress"}

	if unblock {
		result["unblocked"] = true
	}

	if startTimer {
		if err := s.store.StartTimer(id); err != nil {
			result["timer_warning"] = err.Error()
		} else {
			result["timer_started"] = true
		}
	}

	// Add warnings and context
	if len(otherInProgress) > 0 {
		result["warning"] = fmt.Sprintf("You have %d other task(s) in_progress: %v. Consider pausing them with tk_pause.", len(otherInProgress), otherInProgress)
	}

	if len(notes) > 0 {
		// Show recent notes for context
		type noteInfo struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		}
		var recentNotes []noteInfo
		for i := len(notes) - 1; i >= 0 && i >= len(notes)-3; i-- {
			recentNotes = append(recentNotes, noteInfo{ID: notes[i].ID, Text: notes[i].Text})
		}
		result["previous_notes"] = recentNotes
		result["hint"] = "Review previous notes above for context from earlier sessions."
	}

	return result, nil
}

func (s *Server) handleDone(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	// Auto-stop timer if running
	elapsed, wasRunning, err := s.store.StopTimerIfRunning(id)
	if err != nil {
		return nil, err
	}

	// SetStatusAndRead combines setting status + reading file (avoids redundant read)
	f, err := s.store.SetStatusAndRead(id, task.StatusDone)
	if err != nil {
		return nil, err
	}

	// Get the task description for analysis
	completedTask := f.Tasks[id]

	// Find tasks that were blocked by this task and are now unblocked
	var nowUnblocked []string
	for taskID, t := range f.Tasks {
		if t.Status == task.StatusBlocked {
			// Check if this task was the only blocker
			allBlockersDone := true
			wasBlockedByUs := false
			for _, blockerID := range t.BlockedBy {
				if blockerID == id {
					wasBlockedByUs = true
				}
				if blocker, exists := f.Tasks[blockerID]; exists && blocker.Status != task.StatusDone {
					allBlockersDone = false
					break
				}
			}
			if wasBlockedByUs && allBlockersDone {
				nowUnblocked = append(nowUnblocked, taskID)
			}
		}
	}

	result := map[string]interface{}{"id": id, "status": "done"}
	if wasRunning {
		result["timer_stopped"] = true
		result["elapsed"] = elapsed.String()
	}

	// Add smart suggestions
	var hints []string

	if len(nowUnblocked) > 0 {
		result["unblocked_tasks"] = nowUnblocked
		hints = append(hints, fmt.Sprintf("Completing this task unblocked %d task(s): %v", len(nowUnblocked), nowUnblocked))
	}

	// Check if task has notes worth reviewing for learnings
	if notes := f.Context.Notes[id]; len(notes) > 0 {
		hints = append(hints, "This task has notes - consider if any insights should be recorded as learnings with tk_learn.")
	}

	// PRIORITY: Strong learning prompt for bug-fix tasks
	if isBugFixTaskDescription(completedTask.Description) || isBugFixTaskDescription(id) {
		result["is_bug_fix"] = true
		hints = append(hints, "🎯 BUG FIX COMPLETED! MANDATORY: Record what you learned with tk_learn:")
		hints = append(hints, "  → Root cause: tk_learn \"The bug was caused by...\"")
		hints = append(hints, "  → Prevention rule: tk_learn \"Never X\" or \"Always Y\"")
		hints = append(hints, "  ⚠️ Skipping this means the same bug WILL happen again!")
	} else {
		// Reflection prompt - always shown for non-bug-fix tasks
		hints = append(hints, "REFLECT: Did completing this task involve decisions (tk_decide) or reveal learnings (tk_learn) worth preserving?")
	}

	// Suggest archiving if appropriate
	hints = append(hints, "Consider archiving with tk_archive if this task is fully verified and no longer needs visibility.")

	if len(hints) > 0 {
		result["hints"] = hints
	}

	return result, nil
}

// isBugFixTaskDescription checks if a task description indicates bug fix work
func isBugFixTaskDescription(description string) bool {
	desc := strings.ToLower(description)

	bugKeywords := []string{
		"fix", "bug", "debug", "resolve", "repair",
		"patch", "hotfix", "error", "issue", "problem",
		"broken", "crash", "failing", "failed",
	}

	for _, kw := range bugKeywords {
		if strings.Contains(desc, kw) {
			return true
		}
	}
	return false
}

func (s *Server) handleBlock(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	var blockers []string
	if b, ok := args["blocked_by"].([]interface{}); ok {
		for _, v := range b {
			if s, ok := v.(string); ok {
				blockers = append(blockers, s)
			}
		}
	}

	if err := s.store.BlockTask(id, blockers); err != nil {
		return nil, err
	}

	return map[string]interface{}{"id": id, "status": "blocked", "blocked_by": blockers}, nil
}

// autoSyncRules syncs learnings/decisions to editor rules if targets exist.
// Returns the number of editors synced to, or 0 if sync was skipped/failed.
func (s *Server) autoSyncRules() int {
	targets := rules.GetTargets()
	if len(targets) == 0 {
		return 0
	}

	f, err := s.store.Read()
	if err != nil {
		return 0
	}

	if _, err := rules.Sync(f.Context.Learnings, f.Context.Decisions); err != nil {
		return 0
	}

	return len(targets)
}

func (s *Server) handleLearn(args map[string]interface{}) (interface{}, error) {
	insight, _ := args["insight"].(string)
	scope, _ := args["scope"].(string)

	var id string
	var isRule bool
	var err error

	if scope != "" {
		id, isRule, err = s.store.AddLearningWithScope(insight, scope, nil)
	} else {
		id, isRule, err = s.store.AddLearningWithRule(insight, nil)
	}
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{"id": id, "status": "added", "is_rule": isRule}
	if scope != "" {
		result["scope"] = scope
	}

	// Auto-sync to editor rules if targets exist
	if synced := s.autoSyncRules(); synced > 0 {
		result["synced_to"] = synced
	}

	return result, nil
}

func (s *Server) handleDecide(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	chose, _ := args["chose"].(string)
	because, _ := args["because"].(string)

	var over []string
	if o, ok := args["over"].([]interface{}); ok {
		for _, v := range o {
			if str, ok := v.(string); ok {
				over = append(over, str)
			}
		}
	}

	d := task.Decision{
		ID:        id,
		Chose:     chose,
		Over:      over,
		Because:   because,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.store.AddDecision(d); err != nil {
		return nil, err
	}

	result := map[string]interface{}{"id": id, "status": "recorded"}

	// Auto-sync to editor rules if targets exist
	if synced := s.autoSyncRules(); synced > 0 {
		result["synced_to"] = synced
	}

	return result, nil
}

func (s *Server) handleNote(args map[string]interface{}) (interface{}, error) {
	taskID, _ := args["task_id"].(string)
	note, _ := args["note"].(string)

	id, err := s.store.AddNote(taskID, note)
	if err != nil {
		return nil, err
	}

	return map[string]string{"id": id, "task_id": taskID, "status": "added"}, nil
}

func (s *Server) handleContext(args map[string]interface{}) (interface{}, error) {
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	// Generate smart suggestions based on project state
	var suggestions []string

	// Count tasks by status
	statusCounts := map[string]int{}
	var highPriorityReady []string
	var staleInProgress []string
	now := time.Now()

	for id, t := range f.Tasks {
		statusCounts[string(t.Status)]++

		// Find high-priority ready tasks
		if t.Status == task.StatusReady && t.GetPriority() <= task.PriorityHigh {
			highPriorityReady = append(highPriorityReady, id)
		}

		// Find stale in_progress tasks (>24h without update)
		if t.Status == task.StatusInProgress && now.Sub(t.UpdatedAt) > 24*time.Hour {
			staleInProgress = append(staleInProgress, id)
		}
	}

	// Generate suggestions
	if len(highPriorityReady) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("High-priority tasks ready: %v", highPriorityReady))
	}

	if len(staleInProgress) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("Stale in_progress tasks (>24h): %v - consider updating or pausing", staleInProgress))
	}

	if statusCounts["blocked"] > 3 {
		suggestions = append(suggestions, fmt.Sprintf("%d blocked tasks - review blockers to unblock progress", statusCounts["blocked"]))
	}

	// Check for rule learnings that should be promoted
	ruleCount := 0
	for _, l := range f.Context.Learnings {
		if l.IsRule {
			ruleCount++
		}
	}
	if ruleCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("%d rule learnings ready for promotion to docs - use tk_learning_rules", ruleCount))
	}

	// Build enhanced response
	result := map[string]interface{}{
		"tasks":   f.Tasks,
		"version": f.Version,
		"task_counts": map[string]int{
			"ready":       statusCounts["ready"],
			"in_progress": statusCounts["in_progress"],
			"blocked":     statusCounts["blocked"],
			"done":        statusCounts["done"],
			"total":       len(f.Tasks),
		},
	}

	// Check if rules have actually been synced (not just that editors are detected)
	// Only skip learnings if rules files exist - avoids missing data if sync hasn't happened
	if rules.HasSyncedRules() {
		// Rules files exist - only include decisions, learnings are auto-loaded by editor
		result["context"] = map[string]interface{}{
			"decisions": f.Context.Decisions,
		}
		result["rules_sync_active"] = true
		result["rules_sync_note"] = "Learnings omitted - auto-loaded from editor rules directories"
	} else {
		// No rules synced yet - include full context
		result["context"] = f.Context
	}

	if len(suggestions) > 0 {
		result["suggestions"] = suggestions
	}

	return result, nil
}

func (s *Server) handleTimerStart(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	if err := s.store.StartTimer(id); err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "status": "timer_started"}, nil
}

func (s *Server) handleTimerStop(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	elapsed, err := s.store.StopTimer(id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":      id,
		"status":  "timer_stopped",
		"elapsed": elapsed.String(),
	}, nil
}

func (s *Server) handleTimerStatus(args map[string]interface{}) (interface{}, error) {
	timers, err := s.store.GetActiveTimers()
	if err != nil {
		return nil, err
	}

	type timerInfo struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		StartedAt   string `json:"started_at"`
		Elapsed     string `json:"elapsed"`
		Warning     string `json:"warning,omitempty"`
	}

	var results []timerInfo
	var warnings []string

	for id, t := range timers {
		elapsed := time.Since(*t.TimerStart)
		info := timerInfo{
			ID:          id,
			Description: t.Description,
			StartedAt:   t.TimerStart.Format(time.RFC3339),
			Elapsed:     elapsed.Truncate(time.Second).String(),
		}

		// Add warnings for long-running timers
		if elapsed > 4*time.Hour {
			info.Warning = "Running for over 4 hours - consider stopping if not actively working"
			warnings = append(warnings, fmt.Sprintf("%s: running for %s", id, info.Elapsed))
		} else if elapsed > 2*time.Hour {
			info.Warning = "Running for over 2 hours"
		}

		results = append(results, info)
	}

	response := map[string]interface{}{
		"timers": results,
		"count":  len(results),
	}

	if len(warnings) > 0 {
		response["warnings"] = warnings
		response["hint"] = "Long-running timers detected. If you're not actively working, stop them with `tk task timer stop <id>`."
	}

	if len(results) == 0 {
		response["hint"] = "No active timers. Start one with `tk task timer start <id>` when beginning focused work."
	}

	return response, nil
}

func (s *Server) handleFieldSet(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	key, _ := args["key"].(string)
	value, _ := args["value"].(string)

	if err := s.store.SetField(id, key, value); err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "key": key, "value": value, "status": "set"}, nil
}

func (s *Server) handleFieldRemove(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	key, _ := args["key"].(string)

	if err := s.store.RemoveField(id, key); err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "key": key, "status": "removed"}, nil
}

func (s *Server) handleTagAdd(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	tag, _ := args["tag"].(string)

	if err := s.store.AddTag(id, tag); err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "tag": tag, "status": "added"}, nil
}

func (s *Server) handleTagRemove(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	tag, _ := args["tag"].(string)

	if err := s.store.RemoveTag(id, tag); err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "tag": tag, "status": "removed"}, nil
}

func (s *Server) handleArchive(args map[string]interface{}) (interface{}, error) {
	taskID, _ := args["task_id"].(string)
	summary, _ := args["summary"].(string)

	if err := s.store.ArchiveTask(taskID, summary); err != nil {
		return nil, err
	}
	return map[string]string{"id": taskID, "status": "archived"}, nil
}

func (s *Server) handleArchiveRestore(args map[string]interface{}) (interface{}, error) {
	taskID, _ := args["task_id"].(string)

	if err := s.store.RestoreTask(taskID); err != nil {
		return nil, err
	}
	return map[string]string{"id": taskID, "status": "restored"}, nil
}

func (s *Server) handleArchiveList(args map[string]interface{}) (interface{}, error) {
	archived, err := s.store.GetArchivedTasks()
	if err != nil {
		return nil, err
	}

	type archivedResult struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		Summary     string `json:"summary,omitempty"`
		ArchivedAt  string `json:"archived_at"`
	}

	var results []archivedResult
	for id, t := range archived {
		results = append(results, archivedResult{
			ID:          id,
			Description: t.Description,
			Summary:     t.Summary,
			ArchivedAt:  t.ArchivedAt.Format(time.RFC3339),
		})
	}

	return results, nil
}

func (s *Server) handleArchiveAll(args map[string]interface{}) (interface{}, error) {
	olderThan, _ := args["older_than"].(string)
	if olderThan == "" {
		return nil, fmt.Errorf("older_than is required")
	}

	// Parse duration string (e.g., "7d", "24h", "2w")
	duration, err := parseDurationString(olderThan)
	if err != nil {
		return nil, fmt.Errorf("invalid duration format '%s': %w", olderThan, err)
	}

	archived, err := s.store.ArchiveDoneTasks(duration)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"archived_count": len(archived),
		"archived_tasks": archived,
		"message":        fmt.Sprintf("Archived %d task(s) older than %s", len(archived), olderThan),
	}, nil
}

// parseDurationString parses a human-friendly duration string like "7d", "24h", "2w"
func parseDurationString(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("duration too short")
	}

	unit := s[len(s)-1]
	valueStr := s[:len(s)-1]

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", valueStr)
	}

	switch unit {
	case 'h':
		return time.Duration(value) * time.Hour, nil
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown unit: %c (use h, d, or w)", unit)
	}
}

func (s *Server) handleShow(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	t, exists := f.Tasks[id]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	notes := f.Context.Notes[id]

	type noteInfo struct {
		ID        string `json:"id"`
		Text      string `json:"text"`
		CreatedAt string `json:"created_at"`
	}

	var noteResults []noteInfo
	for _, n := range notes {
		noteResults = append(noteResults, noteInfo{
			ID:        n.ID,
			Text:      n.Text,
			CreatedAt: n.CreatedAt.Format(time.RFC3339),
		})
	}

	result := map[string]interface{}{
		"id":          id,
		"description": t.Description,
		"status":      string(t.Status),
		"priority":    t.Priority,
		"created_at":  t.CreatedAt.Format(time.RFC3339),
		"updated_at":  t.UpdatedAt.Format(time.RFC3339),
	}

	if t.Owner != nil {
		result["owner"] = *t.Owner
	}
	if len(t.BlockedBy) > 0 {
		result["blocked_by"] = t.BlockedBy
	}
	if len(t.Tags) > 0 {
		result["tags"] = t.Tags
	}
	if len(t.Fields) > 0 {
		result["fields"] = t.Fields
	}
	if t.Duration > 0 {
		result["duration"] = t.Duration.String()
	}
	if len(noteResults) > 0 {
		result["notes"] = noteResults
	}

	// Include dependency information (what this task blocks and is blocked by)
	type depInfo struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}

	// Find what this task is blocked by (with details)
	if len(t.BlockedBy) > 0 {
		var blockedByDetails []depInfo
		for _, blockerID := range t.BlockedBy {
			if blocker, exists := f.Tasks[blockerID]; exists {
				blockedByDetails = append(blockedByDetails, depInfo{
					ID:          blockerID,
					Description: blocker.Description,
					Status:      string(blocker.Status),
				})
			}
		}
		if len(blockedByDetails) > 0 {
			result["blocked_by_details"] = blockedByDetails
		}
	}

	// Find what tasks this one blocks
	var blocks []depInfo
	for otherID, other := range f.Tasks {
		for _, blockerID := range other.BlockedBy {
			if blockerID == id {
				blocks = append(blocks, depInfo{
					ID:          otherID,
					Description: other.Description,
					Status:      string(other.Status),
				})
				break
			}
		}
	}
	if len(blocks) > 0 {
		result["blocks"] = blocks
	}

	return result, nil
}

func (s *Server) handleDelete(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	err := s.store.Update(func(f *task.File) error {
		if _, exists := f.Tasks[id]; !exists {
			return fmt.Errorf("task not found: %s", id)
		}

		// Delete the task
		delete(f.Tasks, id)

		// Remove notes for this task
		delete(f.Context.Notes, id)

		// Remove this task from any blocked_by arrays in other tasks
		for tid, t := range f.Tasks {
			newBlockedBy := []string{}
			for _, blockerID := range t.BlockedBy {
				if blockerID != id {
					newBlockedBy = append(newBlockedBy, blockerID)
				}
			}
			if len(newBlockedBy) != len(t.BlockedBy) {
				t.BlockedBy = newBlockedBy
				// If no more blockers, set to ready
				if len(newBlockedBy) == 0 && t.Status == task.StatusBlocked {
					t.Status = task.StatusReady
				}
				t.UpdatedAt = time.Now().UTC()
				f.Tasks[tid] = t
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return map[string]string{"id": id, "status": "deleted"}, nil
}

func (s *Server) handleEdit(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	description, _ := args["description"].(string)

	if err := s.store.SetDescription(id, description); err != nil {
		return nil, err
	}

	return map[string]string{"id": id, "status": "updated", "description": description}, nil
}

func (s *Server) handlePause(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	// Auto-stop timer if running
	elapsed, wasRunning, err := s.store.StopTimerIfRunning(id)
	if err != nil {
		return nil, err
	}

	err = s.store.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("task not found: %s", id)
		}

		if t.Status != task.StatusInProgress {
			return fmt.Errorf("task %s is not in_progress (current status: %s)", id, t.Status)
		}

		t.Status = task.StatusReady
		t.Owner = nil
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t

		return nil
	})

	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{"id": id, "status": "ready"}
	if wasRunning {
		result["timer_stopped"] = true
		result["elapsed"] = elapsed.String()
	}

	// Prompt for note about why paused
	result["hint"] = "Consider adding a note about why this was paused and current progress: tk_note task_id='" + id + "' note='...'"

	return result, nil
}

func (s *Server) handleUnblock(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	fromBlocker, _ := args["from"].(string)

	if fromBlocker == "" {
		// Clear all blockers
		if err := s.store.UnblockTask(id); err != nil {
			return nil, err
		}
		return map[string]interface{}{"id": id, "status": "ready", "removed": "all"}, nil
	}

	// Partial unblock: remove only the specified blocker
	err := s.store.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("task %q not found", id)
		}

		found := false
		newBlockers := []string{}
		for _, b := range t.BlockedBy {
			if b == fromBlocker {
				found = true
			} else {
				newBlockers = append(newBlockers, b)
			}
		}

		if !found {
			return fmt.Errorf("task %q is not blocked by %q", id, fromBlocker)
		}

		t.BlockedBy = newBlockers
		if len(newBlockers) == 0 {
			t.Status = task.StatusReady
		}
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t

		return nil
	})

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{"id": id, "removed": fromBlocker}, nil
}

func (s *Server) handleFind(args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	queryLower := strings.ToLower(query)

	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	type searchResult struct {
		Type    string `json:"type"`
		ID      string `json:"id"`
		Content string `json:"content"`
	}

	var results []searchResult

	// Search tasks
	for id, t := range f.Tasks {
		if strings.Contains(strings.ToLower(t.Description), queryLower) ||
			strings.Contains(strings.ToLower(id), queryLower) {
			results = append(results, searchResult{
				Type:    "task",
				ID:      id,
				Content: t.Description,
			})
		}
	}

	// Search notes
	for taskID, notes := range f.Context.Notes {
		for _, n := range notes {
			if strings.Contains(strings.ToLower(n.Text), queryLower) {
				results = append(results, searchResult{
					Type:    "note",
					ID:      taskID + "/" + n.ID,
					Content: n.Text,
				})
			}
		}
	}

	// Search learnings
	for _, l := range f.Context.Learnings {
		if strings.Contains(strings.ToLower(l.Text), queryLower) {
			results = append(results, searchResult{
				Type:    "learning",
				ID:      l.ID,
				Content: l.Text,
			})
		}
	}

	// Search decisions
	for _, d := range f.Context.Decisions {
		if strings.Contains(strings.ToLower(d.ID), queryLower) ||
			strings.Contains(strings.ToLower(d.Chose), queryLower) ||
			strings.Contains(strings.ToLower(d.Because), queryLower) {
			results = append(results, searchResult{
				Type:    "decision",
				ID:      d.ID,
				Content: d.Chose + " because: " + d.Because,
			})
		}
	}

	return results, nil
}

func (s *Server) handlePriority(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	priorityStr, _ := args["priority"].(string)

	var priority int
	switch priorityStr {
	case "0", "critical":
		priority = task.PriorityCritical
	case "1", "high":
		priority = task.PriorityHigh
	case "2", "normal":
		priority = task.PriorityNormal
	case "3", "low":
		priority = task.PriorityLow
	case "4", "backlog":
		priority = task.PriorityBacklog
	default:
		return nil, fmt.Errorf("invalid priority: %s (use 0-4 or critical/high/normal/low/backlog)", priorityStr)
	}

	if err := s.store.SetPriority(id, priority); err != nil {
		return nil, err
	}

	return map[string]interface{}{"id": id, "priority": priority, "status": "updated"}, nil
}

func (s *Server) handleOwner(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	owner, hasOwner := args["owner"].(string)

	if !hasOwner || owner == "" {
		// Clear owner
		if err := s.store.ClearOwner(id); err != nil {
			return nil, err
		}
		return map[string]string{"id": id, "status": "owner_cleared"}, nil
	}

	// Set owner
	if err := s.store.SetOwner(id, owner); err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "owner": owner, "status": "owner_set"}, nil
}

func (s *Server) handleClaim(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	agent, _ := args["agent"].(string)

	// Check current state for warnings
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	t, exists := f.Tasks[id]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	result := map[string]interface{}{"id": id, "agent": agent}

	// Warn if already claimed by someone else
	if t.Owner != nil && *t.Owner != "" && *t.Owner != agent {
		result["warning"] = fmt.Sprintf("Task was claimed by '%s'. Overriding claim.", *t.Owner)
	}

	// Check if agent already has other claimed tasks
	var otherClaimed []string
	for taskID, task := range f.Tasks {
		if task.Owner != nil && *task.Owner == agent && taskID != id {
			otherClaimed = append(otherClaimed, taskID)
		}
	}
	if len(otherClaimed) > 0 {
		result["note"] = fmt.Sprintf("You already have %d other claimed task(s): %v", len(otherClaimed), otherClaimed)
	}

	if err := s.store.ClaimTask(id, agent); err != nil {
		return nil, err
	}

	result["status"] = "claimed"
	return result, nil
}

func (s *Server) handleRelease(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	// Check if task has notes before releasing
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	notes := f.Context.Notes[id]

	if err := s.store.ReleaseTask(id); err != nil {
		return nil, err
	}

	result := map[string]interface{}{"id": id, "status": "released"}

	if len(notes) == 0 {
		result["hint"] = "Consider adding a note about current progress before releasing: tk_note"
	}

	return result, nil
}

func (s *Server) handleSuggest(args map[string]interface{}) (interface{}, error) {
	description, _ := args["description"].(string)
	desc := strings.ToLower(description)

	// Keywords that indicate project-level tasks (should persist to tk)
	projectKeywords := []string{
		"implement", "add feature", "build", "create", "develop",
		"fix bug", "bugfix", "hotfix", "patch",
		"refactor", "rewrite", "redesign", "rearchitect",
		"migrate", "upgrade", "update dependency",
		"integrate", "connect", "setup", "configure",
		"support", "enable", "add support",
		"milestone", "epic", "feature", "story",
		"api endpoint", "database", "schema",
		"authentication", "authorization", "security",
		"performance", "optimize", "cache",
		"deploy", "release", "ship",
	}

	// Keywords that indicate session-level tasks (TodoWrite only)
	sessionKeywords := []string{
		"fix type error", "fix typo", "fix lint",
		"update file", "edit file", "modify file",
		"read file", "check file", "review file",
		"run test", "run build", "run script",
		"verify", "check", "confirm", "ensure",
		"debug", "investigate", "look into",
		"format", "cleanup", "tidy",
		"add comment", "add docstring", "add import",
		"remove unused", "delete unused",
		"rename variable", "rename function",
	}

	shouldPersist := false
	reason := "No strong project-level indicators found"
	matchedKeyword := ""

	// Check for project keywords
	for _, kw := range projectKeywords {
		if strings.Contains(desc, kw) {
			shouldPersist = true
			matchedKeyword = kw
			reason = fmt.Sprintf("Contains project-level keyword '%s' - this looks like a feature, bug, or significant change that should be tracked across sessions", kw)
			break
		}
	}

	// Session keywords can override if they match
	for _, kw := range sessionKeywords {
		if strings.Contains(desc, kw) {
			shouldPersist = false
			matchedKeyword = kw
			reason = fmt.Sprintf("Contains session-level keyword '%s' - this looks like an implementation step that doesn't need to persist", kw)
			break
		}
	}

	result := map[string]interface{}{
		"should_persist":       shouldPersist,
		"reason":               reason,
		"matched_keyword":      matchedKeyword,
		"original_description": description,
	}

	if shouldPersist {
		result["suggested_command"] = fmt.Sprintf("tk task add %q", description)
		result["recommendation"] = "Add this to tk for persistent tracking across sessions"
	} else {
		result["recommendation"] = "Keep this in TodoWrite only - it's a session-level implementation step"
	}

	return result, nil
}

func (s *Server) handleReady(args map[string]interface{}) (interface{}, error) {
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	type readyTask struct {
		ID          string  `json:"id"`
		Description string  `json:"description"`
		Priority    int     `json:"priority"`
		Owner       *string `json:"owner,omitempty"`
	}

	var results []readyTask
	inProgressCount := 0
	blockedCount := 0

	for id, t := range f.Tasks {
		if t.Status == task.StatusInProgress {
			inProgressCount++
		}
		if t.Status == task.StatusBlocked {
			blockedCount++
		}
		if t.Status != task.StatusReady {
			continue
		}
		// Check if actually blocked
		blocked := false
		for _, blockerID := range t.BlockedBy {
			if blocker, exists := f.Tasks[blockerID]; exists && blocker.Status != task.StatusDone {
				blocked = true
				break
			}
		}
		if !blocked {
			results = append(results, readyTask{
				ID:          id,
				Description: t.Description,
				Priority:    t.GetPriority(),
				Owner:       t.Owner,
			})
		}
	}

	// Sort by priority (lower number = higher priority) - O(n log n) vs O(n²) bubble sort
	slices.SortFunc(results, func(a, b readyTask) int {
		return a.Priority - b.Priority
	})

	// Build response with stats
	response := map[string]interface{}{
		"tasks": results,
		"stats": map[string]int{
			"ready":       len(results),
			"in_progress": inProgressCount,
			"blocked":     blockedCount,
		},
	}

	if len(results) > 0 {
		response["recommendation"] = fmt.Sprintf("Highest priority task: %s", results[0].ID)
	} else if inProgressCount > 0 {
		response["recommendation"] = "No ready tasks - focus on completing in_progress work"
	} else if blockedCount > 0 {
		response["recommendation"] = "All tasks blocked - review blockers with tk_list status=blocked"
	}

	return response, nil
}

func (s *Server) handleWho(args map[string]interface{}) (interface{}, error) {
	// Use index-based read: reads 1 file (index.json) instead of N task files.
	summaries, err := s.store.ListFromIndex()
	if err != nil {
		return nil, err
	}

	type ownedTask struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}

	ownerMap := make(map[string][]ownedTask)
	for _, t := range summaries {
		if t.Owner != nil && *t.Owner != "" {
			ownerMap[*t.Owner] = append(ownerMap[*t.Owner], ownedTask{
				ID:          t.ID,
				Description: t.Description,
				Status:      t.Status,
			})
		}
	}

	return ownerMap, nil
}

func (s *Server) handleDeps(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	// Use index-based read (1 file instead of N) - all needed fields are in the index
	summaries, err := s.store.ListFromIndex()
	if err != nil {
		return nil, err
	}

	// Build lookup map
	type depInfo struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}

	var thisTask *task.TaskSummary
	summaryMap := make(map[string]*task.TaskSummary, len(summaries))
	for i := range summaries {
		summaryMap[summaries[i].ID] = &summaries[i]
		if summaries[i].ID == id {
			thisTask = &summaries[i]
		}
	}

	if thisTask == nil {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	// Find what this task is blocked by
	var blockedBy []depInfo
	for _, blockerID := range thisTask.BlockedBy {
		if blocker, exists := summaryMap[blockerID]; exists {
			blockedBy = append(blockedBy, depInfo{
				ID:          blockerID,
				Description: blocker.Description,
				Status:      blocker.Status,
			})
		}
	}

	// Find what this task blocks
	var blocks []depInfo
	for _, other := range summaries {
		for _, blockerID := range other.BlockedBy {
			if blockerID == id {
				blocks = append(blocks, depInfo{
					ID:          other.ID,
					Description: other.Description,
					Status:      other.Status,
				})
				break
			}
		}
	}

	return map[string]interface{}{
		"id":          id,
		"description": thisTask.Description,
		"status":      thisTask.Status,
		"blocked_by":  blockedBy,
		"blocks":      blocks,
	}, nil
}

func (s *Server) handleStats(args map[string]interface{}) (interface{}, error) {
	// Use index-based read: reads 1 file (index.json) instead of N task files.
	summaries, err := s.store.ListFromIndex()
	if err != nil {
		return nil, err
	}

	statusCounts := map[string]int{
		"ready":       0,
		"in_progress": 0,
		"blocked":     0,
		"done":        0,
	}

	priorityCounts := map[string]int{
		"critical": 0,
		"high":     0,
		"normal":   0,
		"low":      0,
		"backlog":  0,
	}

	total := len(summaries)
	for _, t := range summaries {
		statusCounts[t.Status]++

		switch t.GetPriority() {
		case task.PriorityCritical:
			priorityCounts["critical"]++
		case task.PriorityHigh:
			priorityCounts["high"]++
		case task.PriorityNormal:
			priorityCounts["normal"]++
		case task.PriorityLow:
			priorityCounts["low"]++
		case task.PriorityBacklog:
			priorityCounts["backlog"]++
		}
	}

	completionRate := 0.0
	if total > 0 {
		completionRate = float64(statusCounts["done"]) / float64(total) * 100
	}

	// Get learnings/decisions counts from the index (avoids reading context files).
	learningsCount, decisionsCount, _ := s.store.ContextCounts()

	return map[string]interface{}{
		"total":           total,
		"by_status":       statusCounts,
		"by_priority":     priorityCounts,
		"completion_rate": fmt.Sprintf("%.1f%%", completionRate),
		"learnings_count": learningsCount,
		"decisions_count": decisionsCount,
	}, nil
}

func (s *Server) handleLearningList(args map[string]interface{}) (interface{}, error) {
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	type learningResult struct {
		ID        string `json:"id"`
		Text      string `json:"text"`
		IsRule    bool   `json:"is_rule"`
		Scope     string `json:"scope,omitempty"`
		CreatedAt string `json:"created_at"`
	}

	var results []learningResult
	for _, l := range f.Context.Learnings {
		results = append(results, learningResult{
			ID:        l.ID,
			Text:      l.Text,
			IsRule:    l.IsRule,
			Scope:     l.Scope,
			CreatedAt: l.CreatedAt.Format(time.RFC3339),
		})
	}

	return results, nil
}

func (s *Server) handleLearningPromote(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	targetFile, _ := args["to"].(string)
	keep, _ := args["keep"].(bool)

	if targetFile == "" {
		targetFile = detectContextFile()
	}

	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	// Find the learning
	var foundLearning *task.Learning
	var foundIdx int
	for i := range f.Context.Learnings {
		if f.Context.Learnings[i].ID == id {
			foundLearning = &f.Context.Learnings[i]
			foundIdx = i
			break
		}
	}

	if foundLearning == nil {
		return nil, fmt.Errorf("learning not found: %s", id)
	}

	// Append to context file
	if err := appendToContextFile(targetFile, foundLearning.Text); err != nil {
		return nil, fmt.Errorf("failed to write to %s: %w", targetFile, err)
	}

	// Remove from learnings if not keeping
	if !keep {
		if _, err := s.store.RemoveLearning(id); err != nil {
			return nil, err
		}
	}

	result := map[string]interface{}{
		"id":          id,
		"promoted_to": targetFile,
		"text":        foundLearning.Text,
		"kept":        keep,
	}

	// Auto-sync to editor rules if learning was removed
	if !keep {
		if synced := s.autoSyncRules(); synced > 0 {
			result["synced_to"] = synced
		}
	}

	_ = foundIdx // Used for potential future optimization

	return result, nil
}

func (s *Server) handleLearningRemove(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	removedText, err := s.store.RemoveLearning(id)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{"id": id, "removed": removedText}

	// Auto-sync to editor rules to remove stale learning
	if synced := s.autoSyncRules(); synced > 0 {
		result["synced_to"] = synced
	}

	return result, nil
}

func (s *Server) handleLearningRules(args map[string]interface{}) (interface{}, error) {
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	type ruleResult struct {
		ID        string `json:"id"`
		Text      string `json:"text"`
		Scope     string `json:"scope,omitempty"`
		CreatedAt string `json:"created_at"`
		Hint      string `json:"hint"`
	}

	var results []ruleResult
	for _, l := range f.Context.Learnings {
		if l.IsRule {
			results = append(results, ruleResult{
				ID:        l.ID,
				Text:      l.Text,
				Scope:     l.Scope,
				CreatedAt: l.CreatedAt.Format(time.RFC3339),
				Hint:      "Consider promoting with tk_learning_promote",
			})
		}
	}

	return map[string]interface{}{
		"rules":          results,
		"count":          len(results),
		"recommendation": "Rule learnings (never/always patterns) should be promoted to permanent docs when they prove useful",
	}, nil
}

func (s *Server) handleRulesSync(args map[string]interface{}) (interface{}, error) {
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	results, err := rules.Sync(f.Context.Learnings, f.Context.Decisions)
	if err != nil {
		return nil, err
	}

	type syncResultOutput struct {
		Editor       string   `json:"editor"`
		FilesWritten []string `json:"files_written"`
		Errors       []string `json:"errors,omitempty"`
	}

	var output []syncResultOutput
	totalFiles := 0
	for _, r := range results {
		output = append(output, syncResultOutput{
			Editor:       r.Editor,
			FilesWritten: r.FilesWritten,
			Errors:       r.Errors,
		})
		totalFiles += len(r.FilesWritten)
	}

	return map[string]interface{}{
		"results":     output,
		"total_files": totalFiles,
		"message":     fmt.Sprintf("Synced to %d editor(s), %d files written", len(results), totalFiles),
	}, nil
}

func (s *Server) handleDecisionList(args map[string]interface{}) (interface{}, error) {
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	type decisionResult struct {
		ID        string   `json:"id"`
		Chose     string   `json:"chose"`
		Over      []string `json:"over"`
		Because   string   `json:"because"`
		CreatedAt string   `json:"created_at"`
	}

	var results []decisionResult
	for _, d := range f.Context.Decisions {
		results = append(results, decisionResult{
			ID:        d.ID,
			Chose:     d.Chose,
			Over:      d.Over,
			Because:   d.Because,
			CreatedAt: d.CreatedAt.Format(time.RFC3339),
		})
	}

	return results, nil
}

func (s *Server) handleDecisionRemove(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	err := s.store.Update(func(f *task.File) error {
		for i, d := range f.Context.Decisions {
			if d.ID == id {
				f.Context.Decisions = append(f.Context.Decisions[:i], f.Context.Decisions[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("decision not found: %s", id)
	})

	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{"id": id, "status": "removed"}

	// Auto-sync to editor rules to remove stale decision
	if synced := s.autoSyncRules(); synced > 0 {
		result["synced_to"] = synced
	}

	return result, nil
}

func (s *Server) handleNoteList(args map[string]interface{}) (interface{}, error) {
	taskID, hasTaskID := args["task_id"].(string)

	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	type noteResult struct {
		TaskID    string `json:"task_id"`
		NoteID    string `json:"note_id"`
		Text      string `json:"text"`
		CreatedAt string `json:"created_at"`
	}

	var results []noteResult

	if hasTaskID && taskID != "" {
		// List notes for specific task
		notes := f.Context.Notes[taskID]
		for _, n := range notes {
			results = append(results, noteResult{
				TaskID:    taskID,
				NoteID:    n.ID,
				Text:      n.Text,
				CreatedAt: n.CreatedAt.Format(time.RFC3339),
			})
		}
	} else {
		// List all notes
		for tid, notes := range f.Context.Notes {
			for _, n := range notes {
				results = append(results, noteResult{
					TaskID:    tid,
					NoteID:    n.ID,
					Text:      n.Text,
					CreatedAt: n.CreatedAt.Format(time.RFC3339),
				})
			}
		}
	}

	return results, nil
}

func (s *Server) handleNoteRemove(args map[string]interface{}) (interface{}, error) {
	taskID, _ := args["task_id"].(string)
	noteID, _ := args["note_id"].(string)

	removedText, err := s.store.RemoveNote(taskID, noteID)
	if err != nil {
		return nil, err
	}

	return map[string]string{"task_id": taskID, "note_id": noteID, "removed": removedText}, nil
}

func (s *Server) handleHealth(args map[string]interface{}) (interface{}, error) {
	// Use index-based read: reads 1 file (index.json) instead of N task files.
	// The index contains all metadata needed for health checks: status, priority,
	// updated_at (for staleness), and timer_start (for long-running timers).
	summaries, err := s.store.ListFromIndex()
	if err != nil {
		return nil, err
	}

	now := time.Now()

	// Collect metrics
	statusCounts := map[string]int{}
	priorityCounts := map[string]int{}
	var staleInProgress []string
	var staleDone []string
	var longRunningTimers []string
	var highPriorityBlocked []string

	for _, t := range summaries {
		statusCounts[t.Status]++

		switch t.GetPriority() {
		case task.PriorityCritical:
			priorityCounts["critical"]++
		case task.PriorityHigh:
			priorityCounts["high"]++
		case task.PriorityNormal:
			priorityCounts["normal"]++
		case task.PriorityLow:
			priorityCounts["low"]++
		case task.PriorityBacklog:
			priorityCounts["backlog"]++
		}

		// Stale in_progress (>24h)
		if t.Status == string(task.StatusInProgress) && now.Sub(t.UpdatedAt) > 24*time.Hour {
			staleInProgress = append(staleInProgress, t.ID)
		}

		// Stale done tasks (>7 days, not archived)
		if t.Status == string(task.StatusDone) && now.Sub(t.UpdatedAt) > 7*24*time.Hour {
			staleDone = append(staleDone, t.ID)
		}

		// High priority blocked
		if t.Status == string(task.StatusBlocked) && t.GetPriority() <= task.PriorityHigh {
			highPriorityBlocked = append(highPriorityBlocked, t.ID)
		}

		// Long-running timers
		if t.TimerStart != nil && now.Sub(*t.TimerStart) > 4*time.Hour {
			longRunningTimers = append(longRunningTimers, t.ID)
		}
	}

	// Get learnings/decisions counts from the index (avoids reading context files).
	learningsCount, decisionsCount, _ := s.store.ContextCounts()

	// For rule count, we use the learnings count as a signal that rules may exist.
	// The index doesn't track rule vs non-rule breakdown, but if there are learnings
	// with "Never" or "Always" prefixes they are auto-classified as rules.
	// We only need to do a full read if we want the exact rule count.
	ruleCount := 0
	if learningsCount > 0 {
		// Read just to get rule count — this reads context/learnings.md (1 small file)
		// plus all task files via Read(). Future optimization: add a dedicated
		// ReadLearnings() method to avoid reading task files.
		if f, readErr := s.store.Read(); readErr == nil {
			for _, l := range f.Context.Learnings {
				if l.IsRule {
					ruleCount++
				}
			}
		}
	}

	// Build recommendations
	var recommendations []string

	if len(staleInProgress) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("STALE: %d in_progress task(s) not updated in 24h: %v - update or pause them", len(staleInProgress), staleInProgress))
	}

	if len(highPriorityBlocked) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("BLOCKED: %d high-priority task(s) blocked: %v - unblock to make progress", len(highPriorityBlocked), highPriorityBlocked))
	}

	if len(longRunningTimers) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("TIMERS: %d timer(s) running 4+ hours: %v - stop if not active", len(longRunningTimers), longRunningTimers))
	}

	if len(staleDone) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("ARCHIVE: %d done task(s) older than 7 days: consider archiving with tk_archive", len(staleDone)))
	}

	if ruleCount > 0 {
		recommendations = append(recommendations, fmt.Sprintf("PROMOTE: %d rule learning(s) ready for promotion to docs - use tk_learning_rules", ruleCount))
	}

	// Calculate health score (simple heuristic)
	healthScore := 100
	healthScore -= len(staleInProgress) * 10
	healthScore -= len(highPriorityBlocked) * 15
	healthScore -= len(longRunningTimers) * 5
	if healthScore < 0 {
		healthScore = 0
	}

	var healthStatus string
	if healthScore >= 80 {
		healthStatus = "healthy"
	} else if healthScore >= 50 {
		healthStatus = "needs attention"
	} else {
		healthStatus = "unhealthy"
	}

	return map[string]interface{}{
		"health_score":  healthScore,
		"health_status": healthStatus,
		"task_counts": map[string]int{
			"total":       len(summaries),
			"ready":       statusCounts["ready"],
			"in_progress": statusCounts["in_progress"],
			"blocked":     statusCounts["blocked"],
			"done":        statusCounts["done"],
		},
		"priority_counts": priorityCounts,
		"issues": map[string]interface{}{
			"stale_in_progress":     staleInProgress,
			"high_priority_blocked": highPriorityBlocked,
			"long_running_timers":   longRunningTimers,
			"stale_done_count":      len(staleDone),
			"rule_learnings":        ruleCount,
		},
		"recommendations": recommendations,
		"learnings_count": learningsCount,
		"decisions_count": decisionsCount,
	}, nil
}

// Helper functions for learning promote

func detectContextFile() string {
	contextFiles := []string{
		"CLAUDE.md",
		".cursorrules",
		".github/copilot-instructions.md",
		"AGENTS.md",
		"AI.md",
	}

	for _, cf := range contextFiles {
		if _, err := os.Stat(cf); err == nil {
			return cf
		}
	}

	return "CLAUDE.md"
}

func appendToContextFile(filePath, learning string) error {
	existing, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	text := string(existing)
	entry := fmt.Sprintf("- %s\n", learning)

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
		if len(text) > 0 && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text += "\n## Learnings\n\n" + entry
	}

	return os.WriteFile(filePath, []byte(text), 0644)
}

// generateID creates a kebab-case ID from description (without collision check).
// Used for tests. Production code should use task.GenerateTaskID with existingIDs.
func generateID(desc string) string {
	return task.GenerateTaskID(desc, nil)
}
