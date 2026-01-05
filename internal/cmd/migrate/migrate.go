// Package migrate provides CLI commands for migrating from other systems.
package migrate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Import tasks from another task management system",
		Long: `Migrate tasks from another task management system into Tasuku.

Available subcommands:
  beads    Import from Beads (.beads/issues.jsonl)
  v3       Migrate to V3 directory-based storage format

Run 'tk migrate <subcommand> --help' for details on each source.`,
	}

	cmd.AddCommand(newBeadsCmd())
	cmd.AddCommand(newV3Cmd())

	return cmd
}

// Cmd is the parent command for all migrate operations
var Cmd = newMigrateCmd()

func newBeadsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "beads",
		Short: "Import tasks from Beads issue tracker",
		Long: `Migrate tasks from a Beads (.beads/issues.jsonl) directory to Tasuku.

This will:
  - Import all issues as tasks
  - Map Beads statuses to Tasuku statuses
  - Preserve priority and dependencies
  - Import descriptions and notes
  - Keep the original .beads/ directory intact

Use --dry-run to preview what would be imported without making changes.

Examples:
  tk migrate beads             # Import from Beads
  tk migrate beads --dry-run   # Preview migration without changes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			return migrateFromBeads(dryRun)
		},
	}

	cmd.Flags().Bool("dry-run", false, "Preview migration without making changes")

	return cmd
}

func newV3Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "v3",
		Short: "Migrate to V3 directory-based storage format",
		Long: `Migrate from the single .tasuku.json file to the V3 directory-based format.

The V3 format uses a .tasuku/ directory with one file per task:
  .tasuku/
  ├── config.json       # Version and settings
  ├── tasks/            # One JSON file per task
  │   ├── task-1.json
  │   └── task-2.json
  ├── archive/          # Archived tasks
  └── context/          # Learnings, decisions, notes

Benefits of V3 format:
  - No merge conflicts when multiple agents work in parallel
  - Each task can be edited independently
  - Cleaner git history (changes show which task was modified)
  - Archive/restore is just moving files

The original .tasuku.json will be renamed to .tasuku.json.bak.

Use --dry-run to preview what would be migrated without making changes.

Examples:
  tk migrate v3            # Migrate to V3 format
  tk migrate v3 --dry-run  # Preview migration`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			return migrateToV3(dryRun)
		},
	}

	cmd.Flags().Bool("dry-run", false, "Preview migration without making changes")

	return cmd
}

func migrateToV3(dryRun bool) error {
	// Check for existing .tasuku.json (use migration-specific function)
	oldStore := store.GetV2StoreForMigration()
	if oldStore == nil {
		return fmt.Errorf("no .tasuku.json found to migrate")
	}

	oldPath := oldStore.Path()
	newPath := filepath.Join(filepath.Dir(oldPath), ".tasuku")

	// Check if already migrated
	if info, err := os.Stat(newPath); err == nil && info.IsDir() {
		return fmt.Errorf(".tasuku/ directory already exists - already migrated?")
	}

	// Read old format
	f, err := oldStore.Read()
	if err != nil {
		return fmt.Errorf("failed to read .tasuku.json: %w", err)
	}

	fmt.Println("V3 Migration Preview")
	fmt.Println("====================")
	fmt.Printf("Source: %s\n", oldPath)
	fmt.Printf("Target: %s/\n", newPath)
	fmt.Println()

	fmt.Printf("Tasks to migrate: %d\n", len(f.Tasks))
	fmt.Printf("Archived tasks: %d\n", len(f.Archive))
	fmt.Printf("Learnings: %d\n", len(f.Context.Learnings))
	fmt.Printf("Decisions: %d\n", len(f.Context.Decisions))

	noteCount := 0
	for _, notes := range f.Context.Notes {
		noteCount += len(notes)
	}
	fmt.Printf("Notes: %d\n", noteCount)
	fmt.Println()

	if dryRun {
		fmt.Println("Dry run - no changes made.")
		fmt.Println("Run without --dry-run to perform migration.")
		return nil
	}

	// Create new directory store
	newStore := store.NewDirStore(newPath)
	if err := newStore.Init(); err != nil {
		return fmt.Errorf("failed to create .tasuku/ directory: %w", err)
	}

	// Migrate tasks
	for id, t := range f.Tasks {
		if err := newStore.AddTaskWithTags(id, t.Description, t.Priority, t.Tags); err != nil {
			return fmt.Errorf("failed to migrate task %s: %w", id, err)
		}
		// Update additional fields
		newStore.Update(func(nf *task.File) error {
			nt := nf.Tasks[id]
			nt.Status = t.Status
			nt.BlockedBy = t.BlockedBy
			nt.Owner = t.Owner
			nt.ClaimedAt = t.ClaimedAt
			nt.Fields = t.Fields
			nt.TimerStart = t.TimerStart
			nt.Duration = t.Duration
			nt.ParentID = t.ParentID
			nt.CreatedAt = t.CreatedAt
			nt.UpdatedAt = t.UpdatedAt
			nf.Tasks[id] = nt
			return nil
		})
		fmt.Printf("  ✓ Task: %s\n", id)
	}

	// Migrate learnings
	for _, l := range f.Context.Learnings {
		newStore.AddLearningWithRule(l.Text, &l.IsRule)
	}
	if len(f.Context.Learnings) > 0 {
		fmt.Printf("  ✓ Learnings: %d\n", len(f.Context.Learnings))
	}

	// Migrate decisions
	for _, d := range f.Context.Decisions {
		newStore.AddDecision(d)
	}
	if len(f.Context.Decisions) > 0 {
		fmt.Printf("  ✓ Decisions: %d\n", len(f.Context.Decisions))
	}

	// Migrate notes
	for taskID, notes := range f.Context.Notes {
		for _, note := range notes {
			newStore.AddNote(taskID, note.Text)
		}
	}
	if noteCount > 0 {
		fmt.Printf("  ✓ Notes: %d\n", noteCount)
	}

	// Migrate archive
	for id, archived := range f.Archive {
		// Create task, set status to done, then archive
		newStore.AddTask(id, archived.Description)
		newStore.Update(func(nf *task.File) error {
			t := nf.Tasks[id]
			t.Status = task.StatusDone
			t.Priority = archived.Priority
			t.Tags = archived.Tags
			t.Fields = archived.Fields
			t.Duration = archived.Duration
			t.CreatedAt = archived.CreatedAt
			t.UpdatedAt = archived.UpdatedAt
			nf.Tasks[id] = t
			return nil
		})
		newStore.ArchiveTask(id, archived.Summary)
		fmt.Printf("  ✓ Archived: %s\n", id)
	}

	// Backup old file
	backupPath := oldPath + ".bak"
	if err := os.Rename(oldPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup old file: %w", err)
	}

	fmt.Println()
	fmt.Println("Migration complete!")
	fmt.Printf("  Old file backed up to: %s\n", backupPath)
	fmt.Printf("  New format at: %s/\n", newPath)
	fmt.Println()
	fmt.Println("You can safely delete the backup file after verifying the migration.")

	return nil
}

// BeadsIssue represents a Beads issue from issues.jsonl
type BeadsIssue struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	Notes        string   `json:"notes,omitempty"`
	Status       string   `json:"status,omitempty"`
	Priority     int      `json:"priority"`
	IssueType    string   `json:"issue_type,omitempty"`
	Assignee     string   `json:"assignee,omitempty"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
	ClosedAt     string   `json:"closed_at,omitempty"`
	CloseReason  string   `json:"close_reason,omitempty"`
	Labels       []string `json:"labels,omitempty"`
	Dependencies []struct {
		Type     string `json:"type"`
		TargetID string `json:"target_id"`
	} `json:"dependencies,omitempty"`
}

func migrateFromBeads(dryRun bool) error {
	if _, err := os.Stat(".beads"); os.IsNotExist(err) {
		return fmt.Errorf(".beads directory not found")
	}

	fmt.Println("Migrating from Beads...")

	issuesFile := ".beads/issues.jsonl"
	content, err := os.ReadFile(issuesFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", issuesFile, err)
	}

	lines := strings.Split(string(content), "\n")
	var issues []BeadsIssue

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var issue BeadsIssue
		if err := json.Unmarshal([]byte(line), &issue); err != nil {
			fmt.Printf("  Warning: failed to parse line: %v\n", err)
			continue
		}
		issues = append(issues, issue)
	}

	if len(issues) == 0 {
		return fmt.Errorf("no issues found in %s", issuesFile)
	}

	if dryRun {
		fmt.Printf("Found %d issues to migrate:\n", len(issues))
		for _, issue := range issues {
			fmt.Printf("  - %s: %s (%s)\n", issue.ID, issue.Title, issue.Status)
		}
		fmt.Println("\nRun without --dry-run to perform migration")
		return nil
	}

	s := store.DefaultStorageWithWarning()
	if !s.Exists() {
		if err := s.Init(); err != nil {
			return err
		}
	}

	migrated := 0
	for _, issue := range issues {
		id := strings.ToLower(issue.ID)
		id = strings.ReplaceAll(id, " ", "-")

		status := task.StatusReady
		switch strings.ToLower(issue.Status) {
		case "open":
			status = task.StatusReady
		case "in_progress", "in-progress", "active":
			status = task.StatusInProgress
		case "blocked", "deferred":
			status = task.StatusBlocked
		case "closed", "done":
			status = task.StatusDone
		}

		createdAt := time.Now().UTC()
		if issue.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, issue.CreatedAt); err == nil {
				createdAt = t
			}
		}
		updatedAt := createdAt
		if issue.UpdatedAt != "" {
			if t, err := time.Parse(time.RFC3339, issue.UpdatedAt); err == nil {
				updatedAt = t
			}
		}

		var priority *int
		if issue.Priority >= 0 && issue.Priority <= 4 {
			priority = &issue.Priority
		}

		var blockedBy []string
		for _, dep := range issue.Dependencies {
			if dep.Type == "blocks" || dep.Type == "blocked_by" {
				blockedBy = append(blockedBy, strings.ToLower(dep.TargetID))
			}
		}

		if err := s.Update(func(f *task.File) error {
			f.Tasks[id] = task.Task{
				Status:      status,
				Description: issue.Title,
				Priority:    priority,
				BlockedBy:   blockedBy,
				Owner:       nil,
				CreatedAt:   createdAt,
				UpdatedAt:   updatedAt,
			}

			if f.Context.Notes == nil {
				f.Context.Notes = make(map[string][]task.Note)
			}
			if issue.Description != "" {
				f.Context.Notes[id] = append(f.Context.Notes[id], task.Note{Text: "Description: " + issue.Description, CreatedAt: createdAt})
			}
			if issue.Notes != "" {
				f.Context.Notes[id] = append(f.Context.Notes[id], task.Note{Text: issue.Notes, CreatedAt: createdAt})
			}
			if issue.CloseReason != "" {
				f.Context.Notes[id] = append(f.Context.Notes[id], task.Note{Text: "Close reason: " + issue.CloseReason, CreatedAt: updatedAt})
			}

			return nil
		}); err != nil {
			fmt.Printf("  Warning: failed to add %s: %v\n", id, err)
			continue
		}

		priorityStr := ""
		if priority != nil {
			priorityStr = fmt.Sprintf(" [P%d]", *priority)
		}
		fmt.Printf("  Migrated: %s -> %s (%s%s)\n", issue.ID, id, status, priorityStr)
		migrated++
	}

	fmt.Printf("\nMigration complete: %d tasks imported\n", migrated)
	fmt.Println("Original .beads/ preserved (delete manually if desired)")

	return nil
}
