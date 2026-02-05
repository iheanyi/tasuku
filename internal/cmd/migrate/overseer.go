package migrate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

func newOverseerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "overseer",
		Short: "Import tasks and learnings from Overseer",
		Long: `Migrate tasks and learnings from an Overseer SQLite database (.overseer/tasks.db) to Tasuku.

This will:
  - Import all tasks with mapped statuses and priorities
  - Preserve parent-child relationships
  - Convert blocker relationships
  - Import learnings with rule auto-detection
  - Convert context and result fields to task notes
  - Store VCS fields (commit_sha, bookmark, start_commit) as custom fields
  - Import task_metadata as custom fields
  - Keep the original .overseer/ directory intact

Use --dry-run to preview what would be imported without making changes.
Use --force to overwrite tasks that already exist in the store.

Examples:
  tk migrate overseer                              # Import from .overseer/tasks.db
  tk migrate overseer --dry-run                    # Preview migration
  tk migrate overseer --path /path/to/tasks.db     # Import from custom path
  tk migrate overseer --force                      # Overwrite existing tasks`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			path, _ := cmd.Flags().GetString("path")
			force, _ := cmd.Flags().GetBool("force")
			return migrateFromOverseer(path, dryRun, force)
		},
	}

	cmd.Flags().Bool("dry-run", false, "Preview migration without making changes")
	cmd.Flags().String("path", "", "Path to Overseer database file")
	cmd.Flags().Bool("force", false, "Overwrite existing tasks with the same ID")

	return cmd
}

// mapOverseerStatus computes Tasuku status from Overseer boolean fields.
func mapOverseerStatus(t OverseerTask) task.Status {
	if t.Archived {
		return task.StatusDone // will be archived after creation
	}
	if t.Cancelled {
		return task.StatusDone // with "cancelled" tag
	}
	if t.Completed {
		return task.StatusDone
	}
	if t.StartedAt != nil {
		return task.StatusInProgress
	}
	return task.StatusReady
}

// mapOverseerPriority converts Overseer priority to Tasuku priority.
// Overseer: 0=high, 1=medium, 2=low
// Tasuku: 0=critical, 1=high, 2=normal, 3=low, 4=backlog
func mapOverseerPriority(p int) int {
	switch p {
	case 0:
		return task.PriorityCritical // 0 → 0
	case 1:
		return task.PriorityNormal // 1 → 2
	case 2:
		return task.PriorityBacklog // 2 → 4
	default:
		return task.PriorityNormal
	}
}

// sortTasksByDepth sorts tasks so parents come before children.
// Tasks with nil parent_id come first, then children, then grandchildren, etc.
// Returns an error if a circular parent reference is detected.
func sortTasksByDepth(tasks []OverseerTask) ([]OverseerTask, error) {
	// Build parent lookup
	taskByID := make(map[string]*OverseerTask)
	for i := range tasks {
		taskByID[tasks[i].ID] = &tasks[i]
	}

	// Calculate depth for each task using DFS with cycle detection
	depth := make(map[string]int)
	visiting := make(map[string]bool) // tracks nodes in current DFS path
	var cycleErr error

	var getDepth func(id string) int
	getDepth = func(id string) int {
		if d, ok := depth[id]; ok {
			return d
		}
		if visiting[id] {
			cycleErr = fmt.Errorf("circular parent reference detected involving task %q", id)
			depth[id] = 0
			return 0
		}
		t, ok := taskByID[id]
		if !ok || t.ParentID == nil {
			depth[id] = 0
			return 0
		}
		visiting[id] = true
		d := getDepth(*t.ParentID) + 1
		delete(visiting, id)
		depth[id] = d
		return d
	}

	for _, t := range tasks {
		getDepth(t.ID)
		if cycleErr != nil {
			return nil, cycleErr
		}
	}

	sorted := make([]OverseerTask, len(tasks))
	copy(sorted, tasks)
	sort.SliceStable(sorted, func(i, j int) bool {
		return depth[sorted[i].ID] < depth[sorted[j].ID]
	})

	return sorted, nil
}

func migrateFromOverseer(customPath string, dryRun bool, force bool) error {
	// 1. Locate database
	dbPath, err := findOverseerDB(customPath)
	if err != nil {
		return err
	}

	// 2. Read all data
	result, err := readOverseerDB(dbPath)
	if err != nil {
		return err
	}

	// 3. Build blocker lookup
	blockersByTask := make(map[string][]string)
	for _, b := range result.Blockers {
		blockersByTask[b.TaskID] = append(blockersByTask[b.TaskID], b.BlockerID)
	}

	// 4. Generate ID mappings (Overseer ULID → Tasuku kebab-case)
	existingIDs := make(map[string]struct{})
	idMap := make(map[string]string) // overseer ID → tasuku ID
	for _, t := range result.Tasks {
		newID := task.GenerateTaskID(t.Description, existingIDs)
		idMap[t.ID] = newID
		existingIDs[newID] = struct{}{}
	}

	// 5. Categorize tasks
	var activeTasks, cancelledTasks, archivedTasks []OverseerTask
	for _, t := range result.Tasks {
		if t.Archived {
			archivedTasks = append(archivedTasks, t)
		} else if t.Cancelled {
			cancelledTasks = append(cancelledTasks, t)
		} else {
			activeTasks = append(activeTasks, t)
		}
	}

	// 6. Deduplicate learnings by content
	seen := make(map[string]bool)
	var uniqueLearnings []OverseerLearning
	for _, l := range result.Learnings {
		if !seen[l.Content] {
			seen[l.Content] = true
			uniqueLearnings = append(uniqueLearnings, l)
		}
	}

	// 7. Print preview summary
	fmt.Println("Overseer Migration Preview")
	fmt.Println("==========================")
	fmt.Printf("Source: %s\n", dbPath)
	fmt.Println()
	fmt.Printf("Tasks:     %d total\n", len(result.Tasks))
	fmt.Printf("  Active:    %d\n", len(activeTasks))
	fmt.Printf("  Cancelled: %d\n", len(cancelledTasks))
	fmt.Printf("  Archived:  %d\n", len(archivedTasks))
	fmt.Printf("Learnings: %d (%d unique)\n", len(result.Learnings), len(uniqueLearnings))
	fmt.Printf("Blockers:  %d\n", len(result.Blockers))
	if len(result.Metadata) > 0 {
		fmt.Printf("Metadata:  %d\n", len(result.Metadata))
	}
	fmt.Println()

	if dryRun {
		fmt.Println("Tasks to migrate:")
		sorted, err := sortTasksByDepth(result.Tasks)
		if err != nil {
			return err
		}
		for _, t := range sorted {
			status := mapOverseerStatus(t)
			newID := idMap[t.ID]
			extra := ""
			if t.Cancelled && !t.Archived {
				extra = " [cancelled]"
			}
			if t.Archived {
				extra = " [archived]"
			}
			if t.ParentID != nil {
				if parentNewID, ok := idMap[*t.ParentID]; ok {
					extra += fmt.Sprintf(" (parent: %s)", parentNewID)
				}
			}
			fmt.Printf("  %s → %s (%s%s)\n", t.ID, newID, status, extra)
		}
		fmt.Println()
		fmt.Println("Dry run - no changes made.")
		fmt.Println("Run without --dry-run to perform migration.")
		return nil
	}

	// 8. Get or create store
	s := store.DefaultStorageWithWarning()
	if !s.Exists() {
		if err := s.Init(); err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}
	}

	// 9. Check for already-migrated conflicts
	existing, err := s.Read()
	if err != nil {
		return fmt.Errorf("failed to read existing store: %w", err)
	}
	existingArchive, err := s.GetArchivedTasks()
	if err != nil {
		return fmt.Errorf("failed to read existing archive: %w", err)
	}

	var conflicts []string
	for _, ot := range result.Tasks {
		newID := idMap[ot.ID]
		if _, ok := existing.Tasks[newID]; ok {
			conflicts = append(conflicts, newID)
		} else if _, ok := existingArchive[newID]; ok {
			conflicts = append(conflicts, newID+" (archived)")
		}
	}
	if len(conflicts) > 0 && !force {
		return fmt.Errorf("found %d existing task(s) that would conflict:\n  %s\n\nUse --force to overwrite",
			len(conflicts), strings.Join(conflicts, "\n  "))
	}

	// 10. Sort tasks by depth for parent-first creation
	allSorted, err := sortTasksByDepth(result.Tasks)
	if err != nil {
		return err
	}

	// 11. Batch-create all tasks, notes, and learnings in a single Update call
	migratedTasks := 0
	if err := s.Update(func(f *task.File) error {
		for _, ot := range allSorted {
			newID := idMap[ot.ID]
			status := mapOverseerStatus(ot)
			priority := mapOverseerPriority(ot.Priority)

			// Remap parent ID
			var parentID *string
			if ot.ParentID != nil {
				if newParentID, ok := idMap[*ot.ParentID]; ok {
					parentID = &newParentID
				} else {
					fmt.Printf("  Warning: orphaned subtask %s (parent %s not found), creating without parent\n", newID, *ot.ParentID)
				}
			}

			// Remap blocked_by
			var blockedBy []string
			if taskBlockers, ok := blockersByTask[ot.ID]; ok {
				for _, blockerOldID := range taskBlockers {
					if newBlockerID, ok := idMap[blockerOldID]; ok {
						blockedBy = append(blockedBy, newBlockerID)
					}
				}
			}

			// Build tags
			var tags []string
			if ot.Cancelled && !ot.Archived {
				tags = append(tags, "cancelled")
			}

			// Build fields from VCS data and metadata
			fields := make(map[string]string)
			if ot.CommitSHA != nil && *ot.CommitSHA != "" {
				fields["overseer_commit_sha"] = *ot.CommitSHA
			}
			if ot.Bookmark != nil && *ot.Bookmark != "" {
				fields["overseer_bookmark"] = *ot.Bookmark
			}
			if ot.StartCommit != nil && *ot.StartCommit != "" {
				fields["overseer_start_commit"] = *ot.StartCommit
			}
			if md, ok := result.Metadata[ot.ID]; ok {
				fields["overseer_metadata"] = md
			}

			t := task.Task{
				Status:      status,
				Description: ot.Description,
				Priority:    &priority,
				ParentID:    parentID,
				BlockedBy:   blockedBy,
				Tags:        tags,
				CreatedAt:   ot.CreatedAt,
				UpdatedAt:   ot.UpdatedAt,
			}
			if len(fields) > 0 {
				t.Fields = fields
			}
			if t.BlockedBy == nil {
				t.BlockedBy = []string{}
			}

			f.Tasks[newID] = t

			// Add context and result as notes (clear existing notes on --force to prevent duplication)
			if f.Context.Notes == nil {
				f.Context.Notes = make(map[string][]task.Note)
			}
			if force {
				delete(f.Context.Notes, newID)
			}
			if ot.Context != nil && *ot.Context != "" {
				f.Context.Notes[newID] = append(f.Context.Notes[newID], task.Note{
					ID:        task.GenerateShortID(),
					Text:      "Context: " + *ot.Context,
					CreatedAt: ot.CreatedAt,
				})
			}
			if ot.Result != nil && *ot.Result != "" {
				noteTime := ot.UpdatedAt
				if ot.CompletedAt != nil {
					noteTime = *ot.CompletedAt
				}
				f.Context.Notes[newID] = append(f.Context.Notes[newID], task.Note{
					ID:        task.GenerateShortID(),
					Text:      "Result: " + *ot.Result,
					CreatedAt: noteTime,
				})
			}

			migratedTasks++
			label := string(status)
			if ot.Archived {
				label = "archived"
			}
			fmt.Printf("  ✓ %s (%s)\n", newID, label)
		}

		// Migrate learnings in the same Update call (skip duplicates that already exist)
		existingLearningTexts := make(map[string]bool, len(f.Context.Learnings))
		for _, l := range f.Context.Learnings {
			existingLearningTexts[l.Text] = true
		}
		for _, ol := range uniqueLearnings {
			if existingLearningTexts[ol.Content] {
				continue
			}
			learning := task.Learning{
				ID:        task.GenerateShortID(),
				Text:      ol.Content,
				IsRule:    task.IsRuleLearning(ol.Content),
				CreatedAt: ol.CreatedAt,
			}
			f.Context.Learnings = append(f.Context.Learnings, learning)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to write migration data: %w", err)
	}

	if len(uniqueLearnings) > 0 {
		fmt.Printf("  ✓ Learnings: %d\n", len(uniqueLearnings))
	}

	// 12. Archive tasks that should be archived (requires separate calls since
	// ArchiveTask does a read-modify-write and needs the task to exist first)
	migratedArchived := 0
	for _, ot := range allSorted {
		if !ot.Archived {
			continue
		}
		newID := idMap[ot.ID]
		if err := s.ArchiveTask(newID, ""); err != nil {
			fmt.Printf("  Warning: failed to archive task %s: %v\n", newID, err)
		} else {
			migratedArchived++
		}
	}

	fmt.Println()
	fmt.Printf("Migration complete: %d tasks (%d archived), %d learnings\n", migratedTasks, migratedArchived, len(uniqueLearnings))
	fmt.Println("Original .overseer/ preserved (delete manually if desired)")

	return nil
}
