// Package store provides file-based storage with locking.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/iheanyi/tasuku/internal/task"
)

// ErrNotInitialized is returned when no Tasuku storage exists.
var ErrNotInitialized = errors.New("no Tasuku storage found - run 'tk init' to create one")

const (
	DefaultFileName = ".tasuku.json"
	LockTimeout     = 5 * time.Second
)

// Store manages the task file with locking.
type Store struct {
	path string
}

// New creates a new store for the given path.
func New(path string) *Store {
	return &Store{path: path}
}

// Default creates a store using the default filename.
func Default() *Store {
	return New(DefaultFileName)
}

// FindUp searches for .tasuku.json in the current directory and parent directories.
// Stops at the git root (where .git exists) or filesystem root.
// Returns the path to the file if found, or an empty string if not found.
func FindUp() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		path := filepath.Join(dir, DefaultFileName)
		if _, err := os.Stat(path); err == nil {
			return path
		}

		// Stop at git root - .tasuku.json should be at or below repo root
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			// Found .git but no .tasuku.json here - don't go higher
			return ""
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return ""
		}
		dir = parent
	}
}

// DefaultWithFindUp creates a store by searching up directories for .tasuku.json.
// Falls back to current directory if not found (for init command).
func DefaultWithFindUp() *Store {
	if path := FindUp(); path != "" {
		return New(path)
	}
	return New(DefaultFileName)
}

// Path returns the path to the task file.
func (s *Store) Path() string {
	return s.path
}

// Exists checks if the task file exists.
func (s *Store) Exists() bool {
	_, err := os.Stat(s.path)
	return err == nil
}

// Init creates a new task file.
func (s *Store) Init() error {
	if s.Exists() {
		return fmt.Errorf("store: %s already exists", s.path)
	}

	f := task.NewFile()
	return s.write(f)
}

// Read loads the task file.
func (s *Store) Read() (*task.File, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotInitialized
		}
		return nil, fmt.Errorf("store: failed to read %s: %w", s.path, err)
	}

	var f task.File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("store: failed to parse %s: %w", s.path, err)
	}

	return &f, nil
}

// write saves the task file without locking (internal use).
func (s *Store) write(f *task.File) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("store: failed to marshal: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("store: failed to write %s: %w", s.path, err)
	}

	return nil
}

// Update reads, modifies, and writes the task file atomically with a lock.
func (s *Store) Update(fn func(*task.File) error) error {
	f, err := os.OpenFile(s.path, os.O_RDWR, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotInitialized
		}
		return fmt.Errorf("store: failed to open %s: %w", s.path, err)
	}
	defer f.Close()

	// Acquire exclusive lock
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("store: failed to acquire lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	// Read current state
	data, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("store: failed to read: %w", err)
	}

	var tf task.File
	if err := json.Unmarshal(data, &tf); err != nil {
		return fmt.Errorf("store: failed to parse: %w", err)
	}

	// Apply modification
	if err := fn(&tf); err != nil {
		return err
	}

	// Write back
	return s.write(&tf)
}

// AddTask adds a new task with the given ID.
func (s *Store) AddTask(id, description string) error {
	return s.AddTaskWithPriority(id, description, nil)
}

// AddTaskWithPriority adds a new task with the given ID and optional priority.
func (s *Store) AddTaskWithPriority(id, description string, priority *int) error {
	return s.Update(func(f *task.File) error {
		if _, exists := f.Tasks[id]; exists {
			return fmt.Errorf("store: task %q already exists", id)
		}
		t := task.NewTask(description)
		t.Priority = priority
		f.Tasks[id] = t
		return nil
	})
}

// SetPriority changes a task's priority.
func (s *Store) SetPriority(id string, priority int) error {
	return s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}
		t.Priority = &priority
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t
		return nil
	})
}

// SetStatus changes a task's status.
func (s *Store) SetStatus(id string, status task.Status) error {
	return s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		if !task.ValidTransition(t.Status, status) {
			return fmt.Errorf("store: invalid transition from %s to %s", t.Status, status)
		}

		t.Status = status
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t
		return nil
	})
}

// SetStatusAndRead sets a task's status and returns the updated file.
// This avoids a redundant Read() call when you need the file state after setting status.
func (s *Store) SetStatusAndRead(id string, status task.Status) (*task.File, error) {
	if err := s.SetStatus(id, status); err != nil {
		return nil, err
	}
	return s.Read()
}

// MarkDoneAndUnblock marks a task as done and automatically unblocks any tasks
// that were blocked by it. Returns the list of task IDs that were unblocked.
func (s *Store) MarkDoneAndUnblock(id string) ([]string, error) {
	var unblocked []string

	err := s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		if !task.ValidTransition(t.Status, task.StatusDone) {
			return fmt.Errorf("store: invalid transition from %s to %s", t.Status, task.StatusDone)
		}

		// Mark the task as done
		t.Status = task.StatusDone
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t

		// Find and unblock tasks that were blocked by this task
		for taskID, blockedTask := range f.Tasks {
			if blockedTask.Status != task.StatusBlocked {
				continue
			}

			// Check if blocked by the completed task
			wasBlockedByUs := false
			for _, blockerID := range blockedTask.BlockedBy {
				if blockerID == id {
					wasBlockedByUs = true
					break
				}
			}

			if !wasBlockedByUs {
				continue
			}

			// Check if all blockers are now done
			allBlockersDone := true
			for _, blockerID := range blockedTask.BlockedBy {
				if blocker, exists := f.Tasks[blockerID]; exists && blocker.Status != task.StatusDone {
					allBlockersDone = false
					break
				}
			}

			if allBlockersDone {
				blockedTask.Status = task.StatusReady
				blockedTask.BlockedBy = []string{}
				blockedTask.UpdatedAt = time.Now().UTC()
				f.Tasks[taskID] = blockedTask
				unblocked = append(unblocked, taskID)
			}
		}

		return nil
	})

	return unblocked, err
}

// SetDescription updates a task's description.
func (s *Store) SetDescription(id string, description string) error {
	return s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		t.Description = description
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t
		return nil
	})
}

// AddLearning adds a learning to context and returns its ID.
// It auto-detects if the learning is a rule (never/always pattern).
func (s *Store) AddLearning(text string) (string, error) {
	id, _, err := s.AddLearningWithRule(text, nil)
	return id, err
}

// AddLearningWithRule adds a learning with an explicit rule flag.
// If forceRule is nil, auto-detection is used. Otherwise, the provided value is used.
func (s *Store) AddLearningWithRule(text string, forceRule *bool) (string, bool, error) {
	id := task.GenerateShortID()
	var isRule bool

	err := s.Update(func(f *task.File) error {
		// Determine if this is a rule
		if forceRule != nil {
			isRule = *forceRule
		} else {
			isRule = task.IsRuleLearning(text)
		}

		learning := task.Learning{
			ID:        id,
			Text:      text,
			IsRule:    isRule,
			CreatedAt: time.Now().UTC(),
		}
		f.Context.Learnings = append(f.Context.Learnings, learning)
		return nil
	})
	return id, isRule, err
}

// RemoveLearning removes a learning by ID and returns the removed learning text.
func (s *Store) RemoveLearning(id string) (string, error) {
	var removedText string
	err := s.Update(func(f *task.File) error {
		for i, l := range f.Context.Learnings {
			if l.ID == id {
				removedText = l.Text
				f.Context.Learnings = append(f.Context.Learnings[:i], f.Context.Learnings[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("learning %q not found", id)
	})
	return removedText, err
}

// FindLearningByText finds a learning by partial text match (case-insensitive).
func (s *Store) FindLearningByText(query string) (*task.Learning, error) {
	f, err := s.Read()
	if err != nil {
		return nil, err
	}

	lowerQuery := strings.ToLower(query)
	for i := range f.Context.Learnings {
		if strings.Contains(strings.ToLower(f.Context.Learnings[i].Text), lowerQuery) {
			return &f.Context.Learnings[i], nil
		}
	}
	return nil, fmt.Errorf("no learning found matching %q", query)
}

// AddDecision adds a decision to context.
func (s *Store) AddDecision(d task.Decision) error {
	return s.Update(func(f *task.File) error {
		f.Context.Decisions = append(f.Context.Decisions, d)
		return nil
	})
}

// AddNote adds a note to a specific task and returns the generated note ID.
func (s *Store) AddNote(taskID, noteText string) (string, error) {
	var noteID string
	err := s.Update(func(f *task.File) error {
		if _, exists := f.Tasks[taskID]; !exists {
			return fmt.Errorf("store: task %q not found", taskID)
		}
		if f.Context.Notes == nil {
			f.Context.Notes = make(map[string][]task.Note)
		}
		noteID = task.GenerateShortID()
		note := task.Note{
			ID:        noteID,
			Text:      noteText,
			CreatedAt: time.Now().UTC(),
		}
		f.Context.Notes[taskID] = append(f.Context.Notes[taskID], note)
		return nil
	})
	return noteID, err
}

// RemoveNote removes a note from a task by its ID.
func (s *Store) RemoveNote(taskID, noteID string) (string, error) {
	var removedText string
	err := s.Update(func(f *task.File) error {
		taskNotes, exists := f.Context.Notes[taskID]
		if !exists || len(taskNotes) == 0 {
			return fmt.Errorf("store: no notes found for task %q", taskID)
		}

		for i, note := range taskNotes {
			if note.ID == noteID {
				removedText = note.Text
				f.Context.Notes[taskID] = append(taskNotes[:i], taskNotes[i+1:]...)

				// If task has no more notes, remove the key from the map
				if len(f.Context.Notes[taskID]) == 0 {
					delete(f.Context.Notes, taskID)
				}
				return nil
			}
		}
		return fmt.Errorf("store: note %q not found for task %q", noteID, taskID)
	})
	return removedText, err
}

// BlockTask marks a task as blocked by other tasks.
func (s *Store) BlockTask(id string, blockers []string) error {
	return s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		// Verify blockers exist and prevent self-blocking
		for _, blocker := range blockers {
			if blocker == id {
				return fmt.Errorf("store: task %q cannot block itself", id)
			}
			if _, exists := f.Tasks[blocker]; !exists {
				return fmt.Errorf("store: blocker task %q not found", blocker)
			}
		}

		t.Status = task.StatusBlocked
		t.BlockedBy = blockers
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t
		return nil
	})
}

// UnblockTask removes all blockers and sets status to ready.
func (s *Store) UnblockTask(id string) error {
	return s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		t.Status = task.StatusReady
		t.BlockedBy = []string{}
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t
		return nil
	})
}

// RemoveBlocker removes a specific blocker from a task.
// If no blockers remain, sets status to ready.
func (s *Store) RemoveBlocker(id string, blocker string) error {
	return s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		newBlockers := make([]string, 0, len(t.BlockedBy))
		found := false
		for _, b := range t.BlockedBy {
			if b == blocker {
				found = true
			} else {
				newBlockers = append(newBlockers, b)
			}
		}

		if !found {
			return fmt.Errorf("store: task %q is not blocked by %q", id, blocker)
		}

		t.BlockedBy = newBlockers
		if len(newBlockers) == 0 {
			t.Status = task.StatusReady
		}
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t
		return nil
	})
}

// DeleteTask removes a task permanently.
func (s *Store) DeleteTask(id string) error {
	return s.Update(func(f *task.File) error {
		if _, exists := f.Tasks[id]; !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		delete(f.Tasks, id)

		// Also remove notes for this task
		if f.Context.Notes != nil {
			delete(f.Context.Notes, id)
		}

		return nil
	})
}

// EditTask updates a task's description. Alias for SetDescription.
func (s *Store) EditTask(id string, description string) error {
	return s.SetDescription(id, description)
}

// SetOwner sets the owner of a task.
func (s *Store) SetOwner(id string, owner string) error {
	return s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		t.Owner = &owner
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t
		return nil
	})
}

// ClearOwner removes the owner from a task.
func (s *Store) ClearOwner(id string) error {
	return s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		t.Owner = nil
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t
		return nil
	})
}

// ClaimTask claims a task for an agent, setting owner and ClaimedAt.
func (s *Store) ClaimTask(id string, owner string) error {
	return s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		// Check if already claimed by another agent
		if t.Owner != nil && *t.Owner != owner {
			// Check if claim is stale
			if !t.IsClaimStale(task.DefaultClaimTimeout) {
				return fmt.Errorf("store: task %q is already claimed by %q", id, *t.Owner)
			}
			// Claim is stale, allow takeover
		}

		now := time.Now().UTC()
		t.Owner = &owner
		t.ClaimedAt = &now
		t.UpdatedAt = now
		f.Tasks[id] = t
		return nil
	})
}

// ReleaseTask releases a task claim, clearing owner and ClaimedAt.
func (s *Store) ReleaseTask(id string) error {
	return s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		t.Owner = nil
		t.ClaimedAt = nil
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t
		return nil
	})
}

// AddTag adds a tag to a task.
func (s *Store) AddTag(id string, tag string) error {
	return s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		// Check if tag already exists
		for _, existing := range t.Tags {
			if existing == tag {
				return nil // Already has tag, no-op
			}
		}

		t.Tags = append(t.Tags, tag)
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t
		return nil
	})
}

// RemoveTag removes a tag from a task.
func (s *Store) RemoveTag(id string, tag string) error {
	return s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		newTags := make([]string, 0, len(t.Tags))
		found := false
		for _, existing := range t.Tags {
			if existing == tag {
				found = true
			} else {
				newTags = append(newTags, existing)
			}
		}

		if !found {
			return fmt.Errorf("store: task %q does not have tag %q", id, tag)
		}

		t.Tags = newTags
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t
		return nil
	})
}

// AddTaskWithTags adds a new task with tags and optional priority.
func (s *Store) AddTaskWithTags(id, description string, priority *int, tags []string) error {
	return s.Update(func(f *task.File) error {
		if _, exists := f.Tasks[id]; exists {
			return fmt.Errorf("store: task %q already exists", id)
		}
		t := task.NewTask(description)
		t.Priority = priority
		t.Tags = tags
		f.Tasks[id] = t
		return nil
	})
}

// AddSubtask adds a new task as a subtask of an existing task.
func (s *Store) AddSubtask(id, description, parentID string) error {
	return s.Update(func(f *task.File) error {
		if _, exists := f.Tasks[id]; exists {
			return fmt.Errorf("store: task %q already exists", id)
		}
		if _, exists := f.Tasks[parentID]; !exists {
			return fmt.Errorf("store: parent task %q not found (create it first with: tk task add \"description\" --id %s)", parentID, parentID)
		}
		t := task.NewTask(description)
		t.ParentID = &parentID
		f.Tasks[id] = t
		return nil
	})
}

// SetParent sets or clears the parent of a task.
func (s *Store) SetParent(id string, parentID *string) error {
	return s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}
		if parentID != nil && *parentID != "" {
			if _, exists := f.Tasks[*parentID]; !exists {
				return fmt.Errorf("store: parent task %q not found (create it first with: tk task add \"description\" --id %s)", *parentID, *parentID)
			}
			// Prevent self-parenting
			if *parentID == id {
				return fmt.Errorf("store: task cannot be its own parent")
			}
		}
		t.ParentID = parentID
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t
		return nil
	})
}

// GetSubtasks returns all subtasks of a given task.
func (s *Store) GetSubtasks(parentID string) (map[string]task.Task, error) {
	f, err := s.Read()
	if err != nil {
		return nil, err
	}

	result := make(map[string]task.Task)
	for id, t := range f.Tasks {
		if t.ParentID != nil && *t.ParentID == parentID {
			result[id] = t
		}
	}
	return result, nil
}

// SetField sets a custom field on a task.
func (s *Store) SetField(id string, key, value string) error {
	return s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		if t.Fields == nil {
			t.Fields = make(map[string]string)
		}
		t.Fields[key] = value
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t
		return nil
	})
}

// RemoveField removes a custom field from a task.
func (s *Store) RemoveField(id string, key string) error {
	return s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		if len(t.Fields) == 0 {
			return fmt.Errorf("store: task %q has no custom fields", id)
		}

		if _, hasKey := t.Fields[key]; !hasKey {
			return fmt.Errorf("store: task %q does not have field %q", id, key)
		}

		delete(t.Fields, key)
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t
		return nil
	})
}

// StartTimer starts a timer on a task. Returns an error if a timer is already running.
func (s *Store) StartTimer(id string) error {
	return s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		if t.TimerStart != nil {
			return fmt.Errorf("store: task %q already has a timer running (started %s)", id, t.TimerStart.Format(time.RFC3339))
		}

		now := time.Now().UTC()
		t.TimerStart = &now
		t.UpdatedAt = now
		f.Tasks[id] = t
		return nil
	})
}

// StopTimer stops a running timer on a task and adds the elapsed time to Duration.
// Returns the elapsed time for this session.
func (s *Store) StopTimer(id string) (time.Duration, error) {
	var elapsed time.Duration
	err := s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		if t.TimerStart == nil {
			return fmt.Errorf("store: task %q has no timer running", id)
		}

		now := time.Now().UTC()
		elapsed = now.Sub(*t.TimerStart)

		// Add elapsed time to accumulated duration
		t.Duration = task.Duration(time.Duration(t.Duration) + elapsed)
		t.TimerStart = nil
		t.UpdatedAt = now
		f.Tasks[id] = t
		return nil
	})
	return elapsed, err
}

// StopTimerIfRunning stops a timer only if one is running.
// Returns elapsed time, whether a timer was actually running, and any error.
func (s *Store) StopTimerIfRunning(id string) (time.Duration, bool, error) {
	var elapsed time.Duration
	var wasRunning bool
	err := s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		if t.TimerStart == nil {
			return nil // No timer running, nothing to do
		}

		wasRunning = true
		now := time.Now().UTC()
		elapsed = now.Sub(*t.TimerStart)

		t.Duration = task.Duration(time.Duration(t.Duration) + elapsed)
		t.TimerStart = nil
		t.UpdatedAt = now
		f.Tasks[id] = t
		return nil
	})
	return elapsed, wasRunning, err
}

// GetActiveTimers returns all tasks with running timers.
func (s *Store) GetActiveTimers() (map[string]task.Task, error) {
	f, err := s.Read()
	if err != nil {
		return nil, err
	}

	result := make(map[string]task.Task)
	for id, t := range f.Tasks {
		if t.TimerStart != nil {
			result[id] = t
		}
	}
	return result, nil
}

// =============================================================================
// Archive Operations
// =============================================================================

// ArchiveTask moves a done task to the archive.
// The task must have status "done" to be archived.
func (s *Store) ArchiveTask(id string, summary string) error {
	return s.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("store: task %q not found", id)
		}

		if t.Status != task.StatusDone {
			return fmt.Errorf("store: task %q must be done to archive (current status: %s)", id, t.Status)
		}

		// Initialize archive map if nil
		if f.Archive == nil {
			f.Archive = make(map[string]task.ArchivedTask)
		}

		// Create archived task
		archived := task.ArchivedTask{
			Task:       t,
			ArchivedAt: time.Now().UTC(),
			Summary:    summary,
			TotalTime:  t.Duration,
		}

		// Move to archive
		f.Archive[id] = archived
		delete(f.Tasks, id)

		// Clean up notes for this task (keep them in archive context)
		// Notes are left in f.Context.Notes[id] for historical reference

		return nil
	})
}

// ArchiveDoneTasks archives all done tasks older than the given duration.
// Returns the IDs of archived tasks.
func (s *Store) ArchiveDoneTasks(olderThan time.Duration) ([]string, error) {
	var archived []string
	cutoff := time.Now().Add(-olderThan)

	err := s.Update(func(f *task.File) error {
		// Initialize archive map if nil
		if f.Archive == nil {
			f.Archive = make(map[string]task.ArchivedTask)
		}

		for id, t := range f.Tasks {
			if t.Status == task.StatusDone && t.UpdatedAt.Before(cutoff) {
				// Create archived task
				archivedTask := task.ArchivedTask{
					Task:       t,
					ArchivedAt: time.Now().UTC(),
					TotalTime:  t.Duration,
				}

				f.Archive[id] = archivedTask
				delete(f.Tasks, id)
				archived = append(archived, id)
			}
		}
		return nil
	})

	return archived, err
}

// RestoreTask moves an archived task back to active tasks.
// The task is restored with status "ready" by default.
func (s *Store) RestoreTask(id string) error {
	return s.Update(func(f *task.File) error {
		archived, exists := f.Archive[id]
		if !exists {
			return fmt.Errorf("store: archived task %q not found", id)
		}

		// Check if ID conflicts with existing task
		if _, exists := f.Tasks[id]; exists {
			return fmt.Errorf("store: cannot restore - task %q already exists in active tasks", id)
		}

		// Restore the task
		restoredTask := archived.Task
		restoredTask.Status = task.StatusReady
		restoredTask.UpdatedAt = time.Now().UTC()

		f.Tasks[id] = restoredTask
		delete(f.Archive, id)

		return nil
	})
}

// GetArchivedTasks returns all archived tasks.
func (s *Store) GetArchivedTasks() (map[string]task.ArchivedTask, error) {
	f, err := s.Read()
	if err != nil {
		return nil, err
	}

	if f.Archive == nil {
		return make(map[string]task.ArchivedTask), nil
	}
	return f.Archive, nil
}

// GetArchivedTask returns a single archived task by ID.
func (s *Store) GetArchivedTask(id string) (*task.ArchivedTask, error) {
	f, err := s.Read()
	if err != nil {
		return nil, err
	}

	if f.Archive == nil {
		return nil, fmt.Errorf("store: archived task %q not found", id)
	}

	archived, exists := f.Archive[id]
	if !exists {
		return nil, fmt.Errorf("store: archived task %q not found", id)
	}

	return &archived, nil
}

// ClearArchive removes all archived tasks permanently.
func (s *Store) ClearArchive() (int, error) {
	var count int
	err := s.Update(func(f *task.File) error {
		if f.Archive != nil {
			count = len(f.Archive)
			f.Archive = make(map[string]task.ArchivedTask)
		}
		return nil
	})
	return count, err
}
