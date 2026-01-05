// Package store provides file-based storage with locking.
// This file implements directory-based storage for V3.

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

const (
	DirName        = ".tasuku"
	ConfigFileName = "config.json"
	TasksDir       = "tasks"
	ArchiveDir     = "archive"
	ContextDir     = "context"
)

// DirConfig holds the configuration stored in .tasuku/config.json
type DirConfig struct {
	Version int `json:"version"`
}

// DirStore manages tasks using a directory structure with one file per task.
// This eliminates merge conflicts when multiple agents work in parallel.
type DirStore struct {
	root string // Path to .tasuku directory
}

// NewDirStore creates a new directory-based store.
func NewDirStore(root string) *DirStore {
	return &DirStore{root: root}
}

// DefaultDirStore creates a store using the default directory name in current dir.
func DefaultDirStore() *DirStore {
	return NewDirStore(DirName)
}

// FindDirUp searches for .tasuku directory in current and parent directories.
func FindDirUp() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		path := filepath.Join(dir, DirName)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}

		// Stop at git root
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return ""
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// DirStoreWithFindUp creates a store by searching up directories for .tasuku.
func DirStoreWithFindUp() *DirStore {
	if path := FindDirUp(); path != "" {
		return NewDirStore(path)
	}
	return NewDirStore(DirName)
}

// Path returns the root directory path.
func (s *DirStore) Path() string {
	return s.root
}

// Exists checks if the .tasuku directory exists.
func (s *DirStore) Exists() bool {
	info, err := os.Stat(s.root)
	return err == nil && info.IsDir()
}

// Init creates a new .tasuku directory structure.
func (s *DirStore) Init() error {
	if s.Exists() {
		return fmt.Errorf("store: %s already exists", s.root)
	}

	// Create directory structure
	dirs := []string{
		s.root,
		filepath.Join(s.root, TasksDir),
		filepath.Join(s.root, ArchiveDir),
		filepath.Join(s.root, ContextDir),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("store: failed to create %s: %w", dir, err)
		}
	}

	// Write config
	config := DirConfig{Version: 3}
	configPath := filepath.Join(s.root, ConfigFileName)
	data, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("store: failed to write config: %w", err)
	}

	// Initialize empty context files
	learningsPath := filepath.Join(s.root, ContextDir, "learnings.json")
	decisionsPath := filepath.Join(s.root, ContextDir, "decisions.json")

	if err := os.WriteFile(learningsPath, []byte("[]"), 0644); err != nil {
		return fmt.Errorf("store: failed to write learnings: %w", err)
	}
	if err := os.WriteFile(decisionsPath, []byte("[]"), 0644); err != nil {
		return fmt.Errorf("store: failed to write decisions: %w", err)
	}

	return nil
}

// taskPath returns the path to a task file.
func (s *DirStore) taskPath(id string) string {
	return filepath.Join(s.root, TasksDir, id+".json")
}

// archivePath returns the path to an archived task file.
func (s *DirStore) archivePath(id string) string {
	return filepath.Join(s.root, ArchiveDir, id+".json")
}

// notesPath returns the path to a task's notes file.
func (s *DirStore) notesPath(id string) string {
	return filepath.Join(s.root, ContextDir, "notes", id+".json")
}

// TaskWithID wraps a Task with its ID for storage.
type TaskWithID struct {
	ID string `json:"id"`
	task.Task
}

// readTask reads a single task file.
func (s *DirStore) readTask(id string) (*task.Task, error) {
	path := s.taskPath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("store: task %q not found", id)
		}
		return nil, fmt.Errorf("store: failed to read task %s: %w", id, err)
	}

	var twid TaskWithID
	if err := json.Unmarshal(data, &twid); err != nil {
		return nil, fmt.Errorf("store: failed to parse task %s: %w", id, err)
	}

	return &twid.Task, nil
}

// writeTask writes a single task file.
func (s *DirStore) writeTask(id string, t task.Task) error {
	path := s.taskPath(id)
	twid := TaskWithID{ID: id, Task: t}
	data, err := json.MarshalIndent(twid, "", "  ")
	if err != nil {
		return fmt.Errorf("store: failed to marshal task: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// deleteTask removes a task file.
func (s *DirStore) deleteTask(id string) error {
	path := s.taskPath(id)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("store: failed to delete task %s: %w", id, err)
	}
	return nil
}

// listTaskIDs returns all task IDs from the tasks directory.
func (s *DirStore) listTaskIDs() ([]string, error) {
	tasksDir := filepath.Join(s.root, TasksDir)
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("store: failed to list tasks: %w", err)
	}

	var ids []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		ids = append(ids, id)
	}
	return ids, nil
}

// Read loads all data into a task.File structure for compatibility.
func (s *DirStore) Read() (*task.File, error) {
	if !s.Exists() {
		return nil, ErrNotInitialized
	}

	f := task.NewFile()
	f.Version = 3

	// Read all tasks
	ids, err := s.listTaskIDs()
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		t, err := s.readTask(id)
		if err != nil {
			return nil, err
		}
		f.Tasks[id] = *t
	}

	// Read learnings
	learningsPath := filepath.Join(s.root, ContextDir, "learnings.json")
	if data, err := os.ReadFile(learningsPath); err == nil {
		if err := json.Unmarshal(data, &f.Context.Learnings); err != nil {
			return nil, fmt.Errorf("store: failed to parse learnings: %w", err)
		}
	}

	// Read decisions
	decisionsPath := filepath.Join(s.root, ContextDir, "decisions.json")
	if data, err := os.ReadFile(decisionsPath); err == nil {
		if err := json.Unmarshal(data, &f.Context.Decisions); err != nil {
			return nil, fmt.Errorf("store: failed to parse decisions: %w", err)
		}
	}

	// Read notes (one file per task)
	notesDir := filepath.Join(s.root, ContextDir, "notes")
	if entries, err := os.ReadDir(notesDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			taskID := strings.TrimSuffix(entry.Name(), ".json")
			notePath := filepath.Join(notesDir, entry.Name())
			if data, err := os.ReadFile(notePath); err == nil {
				var notes []task.Note
				if err := json.Unmarshal(data, &notes); err == nil {
					f.Context.Notes[taskID] = notes
				}
			}
		}
	}

	// Read archive
	archiveDir := filepath.Join(s.root, ArchiveDir)
	if entries, err := os.ReadDir(archiveDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			id := strings.TrimSuffix(entry.Name(), ".json")
			archivePath := filepath.Join(archiveDir, entry.Name())
			if data, err := os.ReadFile(archivePath); err == nil {
				var archived task.ArchivedTask
				if err := json.Unmarshal(data, &archived); err == nil {
					if f.Archive == nil {
						f.Archive = make(map[string]task.ArchivedTask)
					}
					f.Archive[id] = archived
				}
			}
		}
	}

	return f, nil
}

// updateTask reads, modifies, and writes a single task file with locking.
func (s *DirStore) updateTask(id string, fn func(*task.Task) error) error {
	path := s.taskPath(id)

	// Open file for locking (create if doesn't exist for new tasks)
	f, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("store: task %q not found", id)
		}
		return fmt.Errorf("store: failed to open %s: %w", path, err)
	}
	defer f.Close()

	// Acquire exclusive lock
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("store: failed to acquire lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	// Read current state
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("store: failed to read: %w", err)
	}

	var twid TaskWithID
	if err := json.Unmarshal(data, &twid); err != nil {
		return fmt.Errorf("store: failed to parse: %w", err)
	}

	// Apply modification
	if err := fn(&twid.Task); err != nil {
		return err
	}

	// Write back
	return s.writeTask(id, twid.Task)
}

// Update implements the legacy interface by loading all data, modifying, and saving.
// Note: For better performance, use specific methods like AddTask, SetStatus, etc.
func (s *DirStore) Update(fn func(*task.File) error) error {
	// This is a compatibility shim - loads everything, applies fn, saves everything
	// For new code, prefer using the specific methods which only touch one file
	f, err := s.Read()
	if err != nil {
		return err
	}

	if err := fn(f); err != nil {
		return err
	}

	// Write everything back
	return s.writeAll(f)
}

// writeAll writes all data from a task.File to the directory structure.
func (s *DirStore) writeAll(f *task.File) error {
	// Find existing task files to detect deletions
	existingTasks := make(map[string]bool)
	if ids, err := s.listTaskIDs(); err == nil {
		for _, id := range ids {
			existingTasks[id] = true
		}
	}

	// Write tasks
	for id, t := range f.Tasks {
		if err := s.writeTask(id, t); err != nil {
			return err
		}
		delete(existingTasks, id) // Mark as still exists
	}

	// Delete task files that were removed
	for id := range existingTasks {
		os.Remove(s.taskPath(id))
	}

	// Write learnings
	learningsPath := filepath.Join(s.root, ContextDir, "learnings.json")
	if data, err := json.MarshalIndent(f.Context.Learnings, "", "  "); err == nil {
		if err := os.WriteFile(learningsPath, data, 0644); err != nil {
			return fmt.Errorf("store: failed to write learnings: %w", err)
		}
	}

	// Write decisions
	decisionsPath := filepath.Join(s.root, ContextDir, "decisions.json")
	if data, err := json.MarshalIndent(f.Context.Decisions, "", "  "); err == nil {
		if err := os.WriteFile(decisionsPath, data, 0644); err != nil {
			return fmt.Errorf("store: failed to write decisions: %w", err)
		}
	}

	// Write notes and delete orphaned note files
	notesDir := filepath.Join(s.root, ContextDir, "notes")
	os.MkdirAll(notesDir, 0755)

	// First, find existing note files
	existingNotes := make(map[string]bool)
	if entries, err := os.ReadDir(notesDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				taskID := strings.TrimSuffix(entry.Name(), ".json")
				existingNotes[taskID] = true
			}
		}
	}

	// Write notes that exist in f.Context.Notes
	for taskID, notes := range f.Context.Notes {
		notePath := filepath.Join(notesDir, taskID+".json")
		if data, err := json.MarshalIndent(notes, "", "  "); err == nil {
			if err := os.WriteFile(notePath, data, 0644); err != nil {
				return fmt.Errorf("store: failed to write notes: %w", err)
			}
		}
		delete(existingNotes, taskID) // Mark as still exists
	}

	// Delete note files that were removed
	for taskID := range existingNotes {
		notePath := filepath.Join(notesDir, taskID+".json")
		os.Remove(notePath) // Ignore errors
	}

	// Write archive
	for id, archived := range f.Archive {
		archivePath := s.archivePath(id)
		if data, err := json.MarshalIndent(archived, "", "  "); err == nil {
			if err := os.WriteFile(archivePath, data, 0644); err != nil {
				return fmt.Errorf("store: failed to write archived task: %w", err)
			}
		}
	}

	return nil
}

// =============================================================================
// Task Operations (optimized for single-file operations)
// =============================================================================

// AddTask adds a new task.
func (s *DirStore) AddTask(id, description string) error {
	return s.AddTaskWithPriority(id, description, nil)
}

// AddTaskWithPriority adds a new task with optional priority.
func (s *DirStore) AddTaskWithPriority(id, description string, priority *int) error {
	path := s.taskPath(id)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("store: task %q already exists", id)
	}

	t := task.NewTask(description)
	t.Priority = priority
	return s.writeTask(id, t)
}

// AddTaskWithTags adds a new task with tags and optional priority.
func (s *DirStore) AddTaskWithTags(id, description string, priority *int, tags []string) error {
	path := s.taskPath(id)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("store: task %q already exists", id)
	}

	t := task.NewTask(description)
	t.Priority = priority
	t.Tags = tags
	return s.writeTask(id, t)
}

// AddSubtask adds a new task as a subtask of an existing task.
func (s *DirStore) AddSubtask(id, description, parentID string) error {
	// Check task doesn't exist
	if _, err := os.Stat(s.taskPath(id)); err == nil {
		return fmt.Errorf("store: task %q already exists", id)
	}
	// Check parent exists
	if _, err := os.Stat(s.taskPath(parentID)); os.IsNotExist(err) {
		return fmt.Errorf("store: parent task %q not found", parentID)
	}

	t := task.NewTask(description)
	t.ParentID = &parentID
	return s.writeTask(id, t)
}

// SetParent sets or clears the parent of a task.
func (s *DirStore) SetParent(id string, parentID *string) error {
	// Verify parent exists if setting
	if parentID != nil && *parentID != "" {
		if _, err := os.Stat(s.taskPath(*parentID)); os.IsNotExist(err) {
			return fmt.Errorf("store: parent task %q not found", *parentID)
		}
		if *parentID == id {
			return fmt.Errorf("store: task cannot be its own parent")
		}
	}

	return s.updateTask(id, func(t *task.Task) error {
		t.ParentID = parentID
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// GetSubtasks returns all subtasks of a given task.
func (s *DirStore) GetSubtasks(parentID string) (map[string]task.Task, error) {
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

// SetStatus changes a task's status.
func (s *DirStore) SetStatus(id string, status task.Status) error {
	return s.updateTask(id, func(t *task.Task) error {
		if !task.ValidTransition(t.Status, status) {
			return fmt.Errorf("store: invalid transition from %s to %s", t.Status, status)
		}
		t.Status = status
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// SetDescription updates a task's description.
func (s *DirStore) SetDescription(id string, description string) error {
	return s.updateTask(id, func(t *task.Task) error {
		t.Description = description
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// SetPriority changes a task's priority.
func (s *DirStore) SetPriority(id string, priority int) error {
	return s.updateTask(id, func(t *task.Task) error {
		t.Priority = &priority
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// BlockTask marks a task as blocked.
func (s *DirStore) BlockTask(id string, blockers []string) error {
	// Verify blockers exist
	for _, blocker := range blockers {
		if _, err := os.Stat(s.taskPath(blocker)); os.IsNotExist(err) {
			return fmt.Errorf("store: blocker task %q not found", blocker)
		}
	}

	return s.updateTask(id, func(t *task.Task) error {
		t.Status = task.StatusBlocked
		t.BlockedBy = blockers
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// UnblockTask removes all blockers.
func (s *DirStore) UnblockTask(id string) error {
	return s.updateTask(id, func(t *task.Task) error {
		t.Status = task.StatusReady
		t.BlockedBy = []string{}
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// SetOwner sets the owner of a task.
func (s *DirStore) SetOwner(id string, owner string) error {
	return s.updateTask(id, func(t *task.Task) error {
		t.Owner = &owner
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// ClearOwner removes the owner from a task.
func (s *DirStore) ClearOwner(id string) error {
	return s.updateTask(id, func(t *task.Task) error {
		t.Owner = nil
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// ClaimTask claims a task for an agent.
func (s *DirStore) ClaimTask(id string, owner string) error {
	return s.updateTask(id, func(t *task.Task) error {
		if t.Owner != nil && *t.Owner != owner {
			if !t.IsClaimStale(task.DefaultClaimTimeout) {
				return fmt.Errorf("store: task %q is already claimed by %q", id, *t.Owner)
			}
		}
		now := time.Now().UTC()
		t.Owner = &owner
		t.ClaimedAt = &now
		t.UpdatedAt = now
		return nil
	})
}

// ReleaseTask releases a task claim.
func (s *DirStore) ReleaseTask(id string) error {
	return s.updateTask(id, func(t *task.Task) error {
		t.Owner = nil
		t.ClaimedAt = nil
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// AddTag adds a tag to a task.
func (s *DirStore) AddTag(id string, tag string) error {
	return s.updateTask(id, func(t *task.Task) error {
		for _, existing := range t.Tags {
			if existing == tag {
				return nil
			}
		}
		t.Tags = append(t.Tags, tag)
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// RemoveTag removes a tag from a task.
func (s *DirStore) RemoveTag(id string, tag string) error {
	return s.updateTask(id, func(t *task.Task) error {
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
		return nil
	})
}

// SetField sets a custom field on a task.
func (s *DirStore) SetField(id string, key, value string) error {
	return s.updateTask(id, func(t *task.Task) error {
		if t.Fields == nil {
			t.Fields = make(map[string]string)
		}
		t.Fields[key] = value
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// RemoveField removes a custom field from a task.
func (s *DirStore) RemoveField(id string, key string) error {
	return s.updateTask(id, func(t *task.Task) error {
		if len(t.Fields) == 0 {
			return fmt.Errorf("store: task %q has no custom fields", id)
		}
		if _, hasKey := t.Fields[key]; !hasKey {
			return fmt.Errorf("store: task %q does not have field %q", id, key)
		}
		delete(t.Fields, key)
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// StartTimer starts a timer on a task.
func (s *DirStore) StartTimer(id string) error {
	return s.updateTask(id, func(t *task.Task) error {
		if t.TimerStart != nil {
			return fmt.Errorf("store: task %q already has a timer running", id)
		}
		now := time.Now().UTC()
		t.TimerStart = &now
		t.UpdatedAt = now
		return nil
	})
}

// StopTimer stops a running timer.
func (s *DirStore) StopTimer(id string) (time.Duration, error) {
	var elapsed time.Duration
	err := s.updateTask(id, func(t *task.Task) error {
		if t.TimerStart == nil {
			return fmt.Errorf("store: task %q has no timer running", id)
		}
		now := time.Now().UTC()
		elapsed = now.Sub(*t.TimerStart)
		t.Duration = task.Duration(time.Duration(t.Duration) + elapsed)
		t.TimerStart = nil
		t.UpdatedAt = now
		return nil
	})
	return elapsed, err
}

// GetActiveTimers returns all tasks with running timers.
func (s *DirStore) GetActiveTimers() (map[string]task.Task, error) {
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

// DeleteTask removes a task permanently.
func (s *DirStore) DeleteTask(id string) error {
	path := s.taskPath(id)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("store: task %q not found", id)
	}
	if err := os.Remove(path); err != nil {
		return err
	}

	// Also remove notes for this task (if any)
	notePath := filepath.Join(s.root, ContextDir, "notes", id+".json")
	os.Remove(notePath) // Ignore error if notes don't exist

	return nil
}

// =============================================================================
// Context Operations
// =============================================================================

// AddLearning adds a learning to context.
func (s *DirStore) AddLearning(text string) (string, error) {
	id, _, err := s.AddLearningWithRule(text, nil)
	return id, err
}

// AddLearningWithRule adds a learning with explicit rule flag.
func (s *DirStore) AddLearningWithRule(text string, forceRule *bool) (string, bool, error) {
	learningsPath := filepath.Join(s.root, ContextDir, "learnings.json")

	// Lock the file
	f, err := os.OpenFile(learningsPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return "", false, fmt.Errorf("store: failed to open learnings: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return "", false, fmt.Errorf("store: failed to acquire lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	// Read current
	data, _ := os.ReadFile(learningsPath)
	var learnings []task.Learning
	if len(data) > 0 {
		json.Unmarshal(data, &learnings)
	}

	// Determine if rule
	var isRule bool
	if forceRule != nil {
		isRule = *forceRule
	} else {
		isRule = task.IsRuleLearning(text)
	}

	id := task.GenerateShortID()
	learning := task.Learning{
		ID:        id,
		Text:      text,
		IsRule:    isRule,
		CreatedAt: time.Now().UTC(),
	}
	learnings = append(learnings, learning)

	// Write back
	newData, _ := json.MarshalIndent(learnings, "", "  ")
	if err := os.WriteFile(learningsPath, newData, 0644); err != nil {
		return "", false, fmt.Errorf("store: failed to write learnings: %w", err)
	}

	return id, isRule, nil
}

// RemoveLearning removes a learning by ID.
func (s *DirStore) RemoveLearning(id string) (string, error) {
	learningsPath := filepath.Join(s.root, ContextDir, "learnings.json")

	f, err := os.OpenFile(learningsPath, os.O_RDWR, 0644)
	if err != nil {
		return "", fmt.Errorf("store: failed to open learnings: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return "", fmt.Errorf("store: failed to acquire lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	data, _ := os.ReadFile(learningsPath)
	var learnings []task.Learning
	json.Unmarshal(data, &learnings)

	var removedText string
	for i, l := range learnings {
		if l.ID == id {
			removedText = l.Text
			learnings = append(learnings[:i], learnings[i+1:]...)
			break
		}
	}

	if removedText == "" {
		return "", errors.New("learning not found")
	}

	newData, _ := json.MarshalIndent(learnings, "", "  ")
	os.WriteFile(learningsPath, newData, 0644)

	return removedText, nil
}

// FindLearningByText finds a learning by partial text match.
func (s *DirStore) FindLearningByText(query string) (*task.Learning, error) {
	learningsPath := filepath.Join(s.root, ContextDir, "learnings.json")
	data, err := os.ReadFile(learningsPath)
	if err != nil {
		return nil, err
	}

	var learnings []task.Learning
	if err := json.Unmarshal(data, &learnings); err != nil {
		return nil, err
	}

	lowerQuery := strings.ToLower(query)
	for i := range learnings {
		if strings.Contains(strings.ToLower(learnings[i].Text), lowerQuery) {
			return &learnings[i], nil
		}
	}
	return nil, fmt.Errorf("no learning found matching %q", query)
}

// AddDecision adds a decision to context.
func (s *DirStore) AddDecision(d task.Decision) error {
	decisionsPath := filepath.Join(s.root, ContextDir, "decisions.json")

	f, err := os.OpenFile(decisionsPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("store: failed to open decisions: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("store: failed to acquire lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	data, _ := os.ReadFile(decisionsPath)
	var decisions []task.Decision
	if len(data) > 0 {
		json.Unmarshal(data, &decisions)
	}

	decisions = append(decisions, d)

	newData, _ := json.MarshalIndent(decisions, "", "  ")
	return os.WriteFile(decisionsPath, newData, 0644)
}

// AddNote adds a note to a task.
func (s *DirStore) AddNote(taskID, noteText string) (string, error) {
	// Verify task exists
	if _, err := os.Stat(s.taskPath(taskID)); os.IsNotExist(err) {
		return "", fmt.Errorf("store: task %q not found", taskID)
	}

	notesDir := filepath.Join(s.root, ContextDir, "notes")
	os.MkdirAll(notesDir, 0755)
	notePath := filepath.Join(notesDir, taskID+".json")

	f, err := os.OpenFile(notePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return "", fmt.Errorf("store: failed to open notes: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return "", fmt.Errorf("store: failed to acquire lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	data, _ := os.ReadFile(notePath)
	var notes []task.Note
	if len(data) > 0 {
		json.Unmarshal(data, &notes)
	}

	noteID := task.GenerateShortID()
	note := task.Note{
		ID:        noteID,
		Text:      noteText,
		CreatedAt: time.Now().UTC(),
	}
	notes = append(notes, note)

	newData, _ := json.MarshalIndent(notes, "", "  ")
	if err := os.WriteFile(notePath, newData, 0644); err != nil {
		return "", fmt.Errorf("store: failed to write notes: %w", err)
	}

	return noteID, nil
}

// RemoveNote removes a note from a task.
func (s *DirStore) RemoveNote(taskID, noteID string) (string, error) {
	notePath := filepath.Join(s.root, ContextDir, "notes", taskID+".json")

	f, err := os.OpenFile(notePath, os.O_RDWR, 0644)
	if err != nil {
		return "", fmt.Errorf("store: no notes found for task %q", taskID)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return "", fmt.Errorf("store: failed to acquire lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	data, _ := os.ReadFile(notePath)
	var notes []task.Note
	json.Unmarshal(data, &notes)

	var removedText string
	for i, note := range notes {
		if note.ID == noteID {
			removedText = note.Text
			notes = append(notes[:i], notes[i+1:]...)
			break
		}
	}

	if removedText == "" {
		return "", fmt.Errorf("store: note %q not found for task %q", noteID, taskID)
	}

	if len(notes) == 0 {
		os.Remove(notePath)
	} else {
		newData, _ := json.MarshalIndent(notes, "", "  ")
		os.WriteFile(notePath, newData, 0644)
	}

	return removedText, nil
}

// =============================================================================
// Archive Operations
// =============================================================================

// ArchiveTask moves a done task to archive (moves file from tasks/ to archive/).
func (s *DirStore) ArchiveTask(id string, summary string) error {
	t, err := s.readTask(id)
	if err != nil {
		return err
	}

	if t.Status != task.StatusDone {
		return fmt.Errorf("store: task %q must be done to archive (current status: %s)", id, t.Status)
	}

	archived := task.ArchivedTask{
		Task:       *t,
		ArchivedAt: time.Now().UTC(),
		Summary:    summary,
		TotalTime:  t.Duration,
	}

	// Write to archive
	archivePath := s.archivePath(id)
	data, _ := json.MarshalIndent(archived, "", "  ")
	if err := os.WriteFile(archivePath, data, 0644); err != nil {
		return fmt.Errorf("store: failed to write archived task: %w", err)
	}

	// Remove from tasks
	return s.deleteTask(id)
}

// ArchiveDoneTasks archives all done tasks older than the given duration.
func (s *DirStore) ArchiveDoneTasks(olderThan time.Duration) ([]string, error) {
	cutoff := time.Now().Add(-olderThan)
	var archived []string

	ids, err := s.listTaskIDs()
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		t, err := s.readTask(id)
		if err != nil {
			continue
		}

		if t.Status == task.StatusDone && t.UpdatedAt.Before(cutoff) {
			if err := s.ArchiveTask(id, ""); err == nil {
				archived = append(archived, id)
			}
		}
	}

	return archived, nil
}

// RestoreTask moves an archived task back to active tasks.
func (s *DirStore) RestoreTask(id string) error {
	archivePath := s.archivePath(id)
	data, err := os.ReadFile(archivePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("store: archived task %q not found", id)
		}
		return err
	}

	// Check if task already exists
	if _, err := os.Stat(s.taskPath(id)); err == nil {
		return fmt.Errorf("store: cannot restore - task %q already exists in active tasks", id)
	}

	var archived task.ArchivedTask
	if err := json.Unmarshal(data, &archived); err != nil {
		return err
	}

	// Restore with ready status
	archived.Task.Status = task.StatusReady
	archived.Task.UpdatedAt = time.Now().UTC()

	if err := s.writeTask(id, archived.Task); err != nil {
		return err
	}

	return os.Remove(archivePath)
}

// GetArchivedTasks returns all archived tasks.
func (s *DirStore) GetArchivedTasks() (map[string]task.ArchivedTask, error) {
	result := make(map[string]task.ArchivedTask)

	archiveDir := filepath.Join(s.root, ArchiveDir)
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		archivePath := filepath.Join(archiveDir, entry.Name())
		data, err := os.ReadFile(archivePath)
		if err != nil {
			continue
		}
		var archived task.ArchivedTask
		if err := json.Unmarshal(data, &archived); err == nil {
			result[id] = archived
		}
	}

	return result, nil
}

// GetArchivedTask returns a single archived task.
func (s *DirStore) GetArchivedTask(id string) (*task.ArchivedTask, error) {
	archivePath := s.archivePath(id)
	data, err := os.ReadFile(archivePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("store: archived task %q not found", id)
		}
		return nil, err
	}

	var archived task.ArchivedTask
	if err := json.Unmarshal(data, &archived); err != nil {
		return nil, err
	}

	return &archived, nil
}

// ClearArchive removes all archived tasks.
func (s *DirStore) ClearArchive() (int, error) {
	archiveDir := filepath.Join(s.root, ArchiveDir)
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if err := os.Remove(filepath.Join(archiveDir, entry.Name())); err == nil {
			count++
		}
	}

	return count, nil
}
