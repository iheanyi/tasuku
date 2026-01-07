// Package v4 implements the V4 Markdown-based storage format.
package v4

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
	IndexFileName  = "index.json"
	TasksDir       = "tasks"
	ArchiveDir     = "archive"
	ContextDir     = "context"
)

// Validation errors
var (
	ErrEmptyID          = errors.New("store: id cannot be empty")
	ErrEmptyDescription = errors.New("store: description cannot be empty")
	ErrEmptyTag         = errors.New("store: tag cannot be empty")
	ErrEmptyKey         = errors.New("store: field key cannot be empty")
	ErrEmptyOwner       = errors.New("store: owner cannot be empty")
	ErrEmptyNoteText    = errors.New("store: note text cannot be empty")
	ErrEmptyLearning    = errors.New("store: learning text cannot be empty")
	ErrEmptyDecisionID  = errors.New("store: decision id cannot be empty")
	ErrEmptyChose       = errors.New("store: decision 'chose' cannot be empty")
	ErrEmptyBecause     = errors.New("store: decision 'because' cannot be empty")
)

// isEmptyString checks if a string is empty or whitespace-only.
func isEmptyString(s string) bool {
	return strings.TrimSpace(s) == ""
}

// filterEmptyStrings removes empty/whitespace-only strings from a slice.
func filterEmptyStrings(slice []string) []string {
	var result []string
	for _, s := range slice {
		if !isEmptyString(s) {
			result = append(result, s)
		}
	}
	return result
}

// Config holds the V4 configuration.
type Config struct {
	Version int `json:"version"`
}

// Store implements Markdown-based storage for V4.
type Store struct {
	root string
}

// New creates a new V4 store.
func New(root string) *Store {
	return &Store{root: root}
}

// Path returns the root directory path.
func (s *Store) Path() string {
	return s.root
}

// Exists checks if the .tasuku directory exists.
func (s *Store) Exists() bool {
	info, err := os.Stat(s.root)
	return err == nil && info.IsDir()
}

// Init creates a new .tasuku directory structure with V4 format.
func (s *Store) Init() error {
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
	config := Config{Version: 4}
	configPath := filepath.Join(s.root, ConfigFileName)
	data, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("store: failed to write config: %w", err)
	}

	// Initialize empty context files (Markdown format)
	learningsPath := filepath.Join(s.root, ContextDir, "learnings.md")
	decisionsPath := filepath.Join(s.root, ContextDir, "decisions.md")

	if err := os.WriteFile(learningsPath, []byte("# Learnings\n\n"), 0644); err != nil {
		return fmt.Errorf("store: failed to write learnings: %w", err)
	}
	if err := os.WriteFile(decisionsPath, []byte("# Decisions\n\n"), 0644); err != nil {
		return fmt.Errorf("store: failed to write decisions: %w", err)
	}

	// Initialize empty index
	idx := NewIndex()
	idxData, _ := idx.Marshal()
	if err := os.WriteFile(filepath.Join(s.root, IndexFileName), idxData, 0644); err != nil {
		return fmt.Errorf("store: failed to write index: %w", err)
	}

	return nil
}

// taskPath returns the path to a task file.
func (s *Store) taskPath(id string) string {
	return filepath.Join(s.root, TasksDir, id+".md")
}

// archivePath returns the path to an archived task file.
func (s *Store) archivePath(id string) string {
	return filepath.Join(s.root, ArchiveDir, id+".md")
}

// notesForTask returns notes stored within a task's .md file.
func (s *Store) notesForTask(id string) ([]task.Note, error) {
	data, err := os.ReadFile(s.taskPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	parsed, err := ParseTaskFile(id, data)
	if err != nil {
		return nil, err
	}

	return parsed.ToNotes(), nil
}

// readTask reads a single task file.
func (s *Store) readTask(id string) (*task.Task, []task.Note, error) {
	path := s.taskPath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("store: task %q not found", id)
		}
		return nil, nil, fmt.Errorf("store: failed to read task %s: %w", id, err)
	}

	parsed, err := ParseTaskFile(id, data)
	if err != nil {
		return nil, nil, fmt.Errorf("store: failed to parse task %s: %w", id, err)
	}

	t := parsed.ToTask()
	notes := parsed.ToNotes()
	return &t, notes, nil
}

// writeTask writes a single task file and updates the index.
func (s *Store) writeTask(id string, t task.Task, notes []task.Note) error {
	content, err := WriteTaskFile(id, t, notes)
	if err != nil {
		return fmt.Errorf("store: failed to generate task content: %w", err)
	}

	path := s.taskPath(id)
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("store: failed to write task %s: %w", id, err)
	}

	// Update index
	return s.updateIndex(func(idx *Index) {
		idx.AddTask(id, TaskFrontmatter{
			Status:    string(t.Status),
			Priority:  t.Priority,
			Tags:      t.Tags,
			BlockedBy: t.BlockedBy,
			ParentID:  ptrToStr(t.ParentID),
			Owner:     ptrToStr(t.Owner),
			UpdatedAt: t.UpdatedAt,
		})
	})
}

// deleteTask removes a task file and updates the index.
func (s *Store) deleteTask(id string) error {
	path := s.taskPath(id)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("store: failed to delete task %s: %w", id, err)
	}

	return s.updateIndex(func(idx *Index) {
		idx.RemoveTask(id)
	})
}

// listTaskIDs returns all task IDs from the tasks directory.
func (s *Store) listTaskIDs() ([]string, error) {
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
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".md")
		ids = append(ids, id)
	}
	return ids, nil
}

// updateIndex loads, modifies, and saves the index.
func (s *Store) updateIndex(fn func(*Index)) error {
	idxPath := filepath.Join(s.root, IndexFileName)

	// Open file for locking
	f, err := os.OpenFile(idxPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("store: failed to open index: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("store: failed to acquire index lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	// Read current
	data, _ := os.ReadFile(idxPath)
	var idx *Index
	if len(data) > 0 {
		idx, _ = ParseIndex(data)
	}
	if idx == nil {
		idx = NewIndex()
	}

	// Apply modification
	fn(idx)

	// Write back
	newData, _ := idx.Marshal()
	return os.WriteFile(idxPath, newData, 0644)
}

// regenerateIndex fully regenerates the index from all task files.
func (s *Store) regenerateIndex() error {
	idx := NewIndex()

	// Read all tasks
	ids, err := s.listTaskIDs()
	if err != nil {
		return err
	}

	for _, id := range ids {
		data, err := os.ReadFile(s.taskPath(id))
		if err != nil {
			continue // Skip unreadable files
		}

		parsed, err := ParseTaskFile(id, data)
		if err != nil {
			continue // Skip malformed files
		}

		idx.AddTask(id, parsed.Frontmatter)
	}

	// Count archived
	archiveDir := filepath.Join(s.root, ArchiveDir)
	if entries, err := os.ReadDir(archiveDir); err == nil {
		count := 0
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				count++
			}
		}
		idx.ArchivedCount = count
	}

	// Count learnings
	learningsPath := filepath.Join(s.root, ContextDir, "learnings.md")
	if data, err := os.ReadFile(learningsPath); err == nil {
		if lf, err := ParseLearningsFile(data); err == nil {
			idx.LearningsCount = len(lf.Learnings)
		}
	}

	// Count decisions
	decisionsPath := filepath.Join(s.root, ContextDir, "decisions.md")
	if data, err := os.ReadFile(decisionsPath); err == nil {
		if df, err := ParseDecisionsFile(data); err == nil {
			idx.DecisionsCount = len(df.Decisions)
		}
	}

	// Write index
	idxData, _ := idx.Marshal()
	return os.WriteFile(filepath.Join(s.root, IndexFileName), idxData, 0644)
}

// updateTask reads, modifies, and writes a single task file with locking.
func (s *Store) updateTask(id string, fn func(*task.Task, *[]task.Note) error) error {
	path := s.taskPath(id)

	// Open file for locking
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

	parsed, err := ParseTaskFile(id, data)
	if err != nil {
		return fmt.Errorf("store: failed to parse: %w", err)
	}

	t := parsed.ToTask()
	notes := parsed.ToNotes()

	// Apply modification
	if err := fn(&t, &notes); err != nil {
		return err
	}

	// Write back
	return s.writeTask(id, t, notes)
}

// ErrNotInitialized is returned when no Tasuku storage exists.
var ErrNotInitialized = errors.New("no Tasuku storage found - run 'tk init' to create one")

// Read loads all data into a task.File structure for compatibility.
func (s *Store) Read() (*task.File, error) {
	if !s.Exists() {
		return nil, ErrNotInitialized
	}

	f := task.NewFile()
	f.Version = 4

	// Read all tasks
	ids, err := s.listTaskIDs()
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		t, notes, err := s.readTask(id)
		if err != nil {
			// Skip malformed files in bulk read (lenient mode)
			continue
		}
		f.Tasks[id] = *t
		if len(notes) > 0 {
			f.Context.Notes[id] = notes
		}
	}

	// Read learnings
	learningsPath := filepath.Join(s.root, ContextDir, "learnings.md")
	if data, err := os.ReadFile(learningsPath); err == nil {
		if lf, err := ParseLearningsFile(data); err == nil {
			f.Context.Learnings = lf.Learnings
		}
	}

	// Read decisions
	decisionsPath := filepath.Join(s.root, ContextDir, "decisions.md")
	if data, err := os.ReadFile(decisionsPath); err == nil {
		if df, err := ParseDecisionsFile(data); err == nil {
			f.Context.Decisions = df.Decisions
		}
	}

	// Read archive
	archiveDir := filepath.Join(s.root, ArchiveDir)
	if entries, err := os.ReadDir(archiveDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			id := strings.TrimSuffix(entry.Name(), ".md")
			archPath := filepath.Join(archiveDir, entry.Name())
			if data, err := os.ReadFile(archPath); err == nil {
				if parsed, err := ParseTaskFile(id, data); err == nil {
					t := parsed.ToTask()
					// Parse archived metadata from frontmatter or use defaults
					if f.Archive == nil {
						f.Archive = make(map[string]task.ArchivedTask)
					}
					f.Archive[id] = task.ArchivedTask{
						Task:       t,
						ArchivedAt: t.UpdatedAt, // Use updated_at as archived_at
						TotalTime:  t.Duration,
					}
				}
			}
		}
	}

	return f, nil
}

// Update implements the legacy interface by loading all data, modifying, and saving.
func (s *Store) Update(fn func(*task.File) error) error {
	f, err := s.Read()
	if err != nil {
		return err
	}

	if err := fn(f); err != nil {
		return err
	}

	return s.writeAll(f)
}

// writeAll writes all data from a task.File to the directory structure.
func (s *Store) writeAll(f *task.File) error {
	// Find existing task files to detect deletions
	existingTasks := make(map[string]bool)
	if ids, err := s.listTaskIDs(); err == nil {
		for _, id := range ids {
			existingTasks[id] = true
		}
	}

	// Write tasks
	for id, t := range f.Tasks {
		notes := f.Context.Notes[id]
		if err := s.writeTask(id, t, notes); err != nil {
			return err
		}
		delete(existingTasks, id)
	}

	// Delete task files that were removed
	for id := range existingTasks {
		os.Remove(s.taskPath(id))
	}

	// Write learnings
	learningsPath := filepath.Join(s.root, ContextDir, "learnings.md")
	if err := os.WriteFile(learningsPath, WriteLearningsFile(f.Context.Learnings), 0644); err != nil {
		return fmt.Errorf("store: failed to write learnings: %w", err)
	}

	// Write decisions
	decisionsPath := filepath.Join(s.root, ContextDir, "decisions.md")
	if err := os.WriteFile(decisionsPath, WriteDecisionsFile(f.Context.Decisions), 0644); err != nil {
		return fmt.Errorf("store: failed to write decisions: %w", err)
	}

	// Write archive
	for id, archived := range f.Archive {
		archPath := s.archivePath(id)
		content, err := WriteTaskFile(id, archived.Task, nil)
		if err != nil {
			continue
		}
		if err := os.WriteFile(archPath, content, 0644); err != nil {
			return fmt.Errorf("store: failed to write archived task: %w", err)
		}
	}

	// Regenerate index
	return s.regenerateIndex()
}

// =============================================================================
// Task Operations
// =============================================================================

// AddTask adds a new task.
func (s *Store) AddTask(id, description string) error {
	return s.AddTaskWithPriority(id, description, nil)
}

// AddTaskWithPriority adds a new task with optional priority.
func (s *Store) AddTaskWithPriority(id, description string, priority *int) error {
	if isEmptyString(id) {
		return ErrEmptyID
	}
	if isEmptyString(description) {
		return ErrEmptyDescription
	}

	path := s.taskPath(id)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("store: task %q already exists", id)
	}

	t := task.NewTask(description)
	t.Priority = priority
	return s.writeTask(id, t, nil)
}

// AddTaskWithTags adds a new task with tags and optional priority.
func (s *Store) AddTaskWithTags(id, description string, priority *int, tags []string) error {
	if isEmptyString(id) {
		return ErrEmptyID
	}
	if isEmptyString(description) {
		return ErrEmptyDescription
	}

	path := s.taskPath(id)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("store: task %q already exists", id)
	}

	t := task.NewTask(description)
	t.Priority = priority
	t.Tags = filterEmptyStrings(tags)
	return s.writeTask(id, t, nil)
}

// AddSubtask adds a new task as a subtask of an existing task.
func (s *Store) AddSubtask(id, description, parentID string) error {
	if isEmptyString(id) {
		return ErrEmptyID
	}
	if isEmptyString(description) {
		return ErrEmptyDescription
	}
	if isEmptyString(parentID) {
		return fmt.Errorf("store: parent id cannot be empty")
	}

	if _, err := os.Stat(s.taskPath(id)); err == nil {
		return fmt.Errorf("store: task %q already exists", id)
	}
	if _, err := os.Stat(s.taskPath(parentID)); os.IsNotExist(err) {
		return fmt.Errorf("store: parent task %q not found (create it first with: tk task add \"description\" --id %s)", parentID, parentID)
	}

	t := task.NewTask(description)
	t.ParentID = &parentID
	return s.writeTask(id, t, nil)
}

// SetParent sets or clears the parent of a task.
func (s *Store) SetParent(id string, parentID *string) error {
	if parentID != nil && *parentID != "" {
		if _, err := os.Stat(s.taskPath(*parentID)); os.IsNotExist(err) {
			return fmt.Errorf("store: parent task %q not found", *parentID)
		}
		if *parentID == id {
			return fmt.Errorf("store: task cannot be its own parent")
		}
	}

	return s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
		t.ParentID = parentID
		t.UpdatedAt = time.Now().UTC()
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

// SetStatus changes a task's status.
func (s *Store) SetStatus(id string, status task.Status) error {
	return s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
		if !task.ValidTransition(t.Status, status) {
			return fmt.Errorf("store: invalid transition from %s to %s", t.Status, status)
		}
		t.Status = status
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// SetStatusAndRead sets a task's status and returns the updated file.
func (s *Store) SetStatusAndRead(id string, status task.Status) (*task.File, error) {
	if err := s.SetStatus(id, status); err != nil {
		return nil, err
	}
	return s.Read()
}

// MarkDoneAndUnblock marks a task as done and auto-unblocks dependent tasks.
func (s *Store) MarkDoneAndUnblock(id string) ([]string, error) {
	if err := s.SetStatus(id, task.StatusDone); err != nil {
		return nil, err
	}

	f, err := s.Read()
	if err != nil {
		return nil, err
	}

	var unblocked []string
	for taskID, t := range f.Tasks {
		if t.Status != task.StatusBlocked {
			continue
		}

		wasBlockedByUs := false
		for _, blockerID := range t.BlockedBy {
			if blockerID == id {
				wasBlockedByUs = true
				break
			}
		}

		if !wasBlockedByUs {
			continue
		}

		allBlockersDone := true
		for _, blockerID := range t.BlockedBy {
			if blocker, exists := f.Tasks[blockerID]; exists && blocker.Status != task.StatusDone {
				allBlockersDone = false
				break
			}
		}

		if allBlockersDone {
			if err := s.UnblockTask(taskID); err == nil {
				unblocked = append(unblocked, taskID)
			}
		}
	}

	return unblocked, nil
}

// SetDescription updates a task's description.
func (s *Store) SetDescription(id string, description string) error {
	if isEmptyString(description) {
		return ErrEmptyDescription
	}
	return s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
		t.Description = description
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// SetPriority changes a task's priority.
func (s *Store) SetPriority(id string, priority int) error {
	return s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
		t.Priority = &priority
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// BlockTask marks a task as blocked.
func (s *Store) BlockTask(id string, blockers []string) error {
	// Filter out empty strings defensively
	validBlockers := filterEmptyStrings(blockers)

	if len(validBlockers) == 0 {
		return fmt.Errorf("store: no valid blockers provided")
	}

	for _, blocker := range validBlockers {
		if blocker == id {
			return fmt.Errorf("store: task %q cannot block itself", id)
		}
		if _, err := os.Stat(s.taskPath(blocker)); os.IsNotExist(err) {
			return fmt.Errorf("store: blocker task %q not found", blocker)
		}
	}

	return s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
		t.Status = task.StatusBlocked
		t.BlockedBy = validBlockers
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// UnblockTask removes all blockers.
func (s *Store) UnblockTask(id string) error {
	return s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
		t.Status = task.StatusReady
		t.BlockedBy = []string{}
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// RemoveBlocker removes a specific blocker from a task.
func (s *Store) RemoveBlocker(id string, blocker string) error {
	return s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
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
		return nil
	})
}

// EditTask updates a task's description.
func (s *Store) EditTask(id string, description string) error {
	return s.SetDescription(id, description)
}

// SetOwner sets the owner of a task. If owner is empty, clears the owner.
func (s *Store) SetOwner(id string, owner string) error {
	if isEmptyString(owner) {
		return s.ClearOwner(id)
	}
	return s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
		t.Owner = &owner
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// ClearOwner removes the owner from a task.
func (s *Store) ClearOwner(id string) error {
	return s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
		t.Owner = nil
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// ClaimTask claims a task for an agent.
func (s *Store) ClaimTask(id string, owner string) error {
	if isEmptyString(owner) {
		return ErrEmptyOwner
	}
	return s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
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
func (s *Store) ReleaseTask(id string) error {
	return s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
		t.Owner = nil
		t.ClaimedAt = nil
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// AddTag adds a tag to a task.
func (s *Store) AddTag(id string, tag string) error {
	if isEmptyString(tag) {
		return ErrEmptyTag
	}
	return s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
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
func (s *Store) RemoveTag(id string, tag string) error {
	return s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
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
func (s *Store) SetField(id string, key, value string) error {
	if isEmptyString(key) {
		return ErrEmptyKey
	}
	return s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
		if t.Fields == nil {
			t.Fields = make(map[string]string)
		}
		t.Fields[key] = value
		t.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// RemoveField removes a custom field from a task.
func (s *Store) RemoveField(id string, key string) error {
	return s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
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
func (s *Store) StartTimer(id string) error {
	return s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
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
func (s *Store) StopTimer(id string) (time.Duration, error) {
	var elapsed time.Duration
	err := s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
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

// StopTimerIfRunning stops a timer only if one is running.
func (s *Store) StopTimerIfRunning(id string) (time.Duration, bool, error) {
	var elapsed time.Duration
	var wasRunning bool
	err := s.updateTask(id, func(t *task.Task, notes *[]task.Note) error {
		if t.TimerStart == nil {
			return nil
		}
		wasRunning = true
		now := time.Now().UTC()
		elapsed = now.Sub(*t.TimerStart)
		t.Duration = task.Duration(time.Duration(t.Duration) + elapsed)
		t.TimerStart = nil
		t.UpdatedAt = now
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

// DeleteTask removes a task permanently.
func (s *Store) DeleteTask(id string) error {
	path := s.taskPath(id)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("store: task %q not found", id)
	}
	return s.deleteTask(id)
}

// =============================================================================
// Context Operations
// =============================================================================

// AddLearning adds a learning to context.
func (s *Store) AddLearning(text string) (string, error) {
	id, _, err := s.AddLearningWithRule(text, nil)
	return id, err
}

// AddLearningWithRule adds a learning with explicit rule flag.
func (s *Store) AddLearningWithRule(text string, forceRule *bool) (string, bool, error) {
	if isEmptyString(text) {
		return "", false, ErrEmptyLearning
	}

	learningsPath := filepath.Join(s.root, ContextDir, "learnings.md")

	f, err := os.OpenFile(learningsPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return "", false, fmt.Errorf("store: failed to open learnings: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return "", false, fmt.Errorf("store: failed to acquire lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	data, _ := os.ReadFile(learningsPath)
	var lf *LearningsFile
	if len(data) > 0 {
		lf, _ = ParseLearningsFile(data)
	}
	if lf == nil {
		lf = &LearningsFile{Learnings: []task.Learning{}}
	}

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
	lf.Learnings = append(lf.Learnings, learning)

	if err := os.WriteFile(learningsPath, WriteLearningsFile(lf.Learnings), 0644); err != nil {
		return "", false, fmt.Errorf("store: failed to write learnings: %w", err)
	}

	// Update index
	s.updateIndex(func(idx *Index) {
		idx.LearningsCount = len(lf.Learnings)
	})

	return id, isRule, nil
}

// AddLearningWithScope adds a learning with explicit scope and rule flag.
func (s *Store) AddLearningWithScope(text, scope string, forceRule *bool) (string, bool, error) {
	if isEmptyString(text) {
		return "", false, ErrEmptyLearning
	}

	learningsPath := filepath.Join(s.root, ContextDir, "learnings.md")

	f, err := os.OpenFile(learningsPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return "", false, fmt.Errorf("store: failed to open learnings: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return "", false, fmt.Errorf("store: failed to acquire lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	data, _ := os.ReadFile(learningsPath)
	var lf *LearningsFile
	if len(data) > 0 {
		lf, _ = ParseLearningsFile(data)
	}
	if lf == nil {
		lf = &LearningsFile{Learnings: []task.Learning{}}
	}

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
		Scope:     scope,
		CreatedAt: time.Now().UTC(),
	}
	lf.Learnings = append(lf.Learnings, learning)

	if err := os.WriteFile(learningsPath, WriteLearningsFile(lf.Learnings), 0644); err != nil {
		return "", false, fmt.Errorf("store: failed to write learnings: %w", err)
	}

	// Update index
	s.updateIndex(func(idx *Index) {
		idx.LearningsCount = len(lf.Learnings)
	})

	return id, isRule, nil
}

// AddLearningFull adds a learning with all fields preserved (for migration).
// Unlike AddLearningWithRule, this preserves the original ID, timestamp, and rule status.
func (s *Store) AddLearningFull(l task.Learning) error {
	learningsPath := filepath.Join(s.root, ContextDir, "learnings.md")

	f, err := os.OpenFile(learningsPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("store: failed to open learnings: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("store: failed to acquire lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	data, _ := os.ReadFile(learningsPath)
	var lf *LearningsFile
	if len(data) > 0 {
		lf, _ = ParseLearningsFile(data)
	}
	if lf == nil {
		lf = &LearningsFile{Learnings: []task.Learning{}}
	}

	// Use the learning as-is, preserving all fields
	// If ID is empty, generate one
	if l.ID == "" {
		l.ID = task.GenerateShortID()
	}
	// If CreatedAt is zero, use now
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now().UTC()
	}

	lf.Learnings = append(lf.Learnings, l)

	if err := os.WriteFile(learningsPath, WriteLearningsFile(lf.Learnings), 0644); err != nil {
		return fmt.Errorf("store: failed to write learnings: %w", err)
	}

	// Update index
	s.updateIndex(func(idx *Index) {
		idx.LearningsCount = len(lf.Learnings)
	})

	return nil
}

// RemoveLearning removes a learning by ID.
func (s *Store) RemoveLearning(id string) (string, error) {
	learningsPath := filepath.Join(s.root, ContextDir, "learnings.md")

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
	lf, _ := ParseLearningsFile(data)
	if lf == nil {
		return "", errors.New("learning not found")
	}

	var removedText string
	for i, l := range lf.Learnings {
		if l.ID == id {
			removedText = l.Text
			lf.Learnings = append(lf.Learnings[:i], lf.Learnings[i+1:]...)
			break
		}
	}

	if removedText == "" {
		return "", errors.New("learning not found")
	}

	os.WriteFile(learningsPath, WriteLearningsFile(lf.Learnings), 0644)

	// Update index
	s.updateIndex(func(idx *Index) {
		idx.LearningsCount = len(lf.Learnings)
	})

	return removedText, nil
}

// FindLearningByText finds a learning by partial text match.
func (s *Store) FindLearningByText(query string) (*task.Learning, error) {
	learningsPath := filepath.Join(s.root, ContextDir, "learnings.md")
	data, err := os.ReadFile(learningsPath)
	if err != nil {
		return nil, err
	}

	lf, err := ParseLearningsFile(data)
	if err != nil {
		return nil, err
	}

	lowerQuery := strings.ToLower(query)
	for i := range lf.Learnings {
		if strings.Contains(strings.ToLower(lf.Learnings[i].Text), lowerQuery) {
			return &lf.Learnings[i], nil
		}
	}
	return nil, fmt.Errorf("no learning found matching %q", query)
}

// AddDecision adds a decision to context.
func (s *Store) AddDecision(d task.Decision) error {
	if isEmptyString(d.ID) {
		return ErrEmptyDecisionID
	}
	if isEmptyString(d.Chose) {
		return ErrEmptyChose
	}
	if isEmptyString(d.Because) {
		return ErrEmptyBecause
	}
	// Filter empty strings from Over
	d.Over = filterEmptyStrings(d.Over)

	decisionsPath := filepath.Join(s.root, ContextDir, "decisions.md")

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
	var df *DecisionsFile
	if len(data) > 0 {
		df, _ = ParseDecisionsFile(data)
	}
	if df == nil {
		df = &DecisionsFile{Decisions: []task.Decision{}}
	}

	df.Decisions = append(df.Decisions, d)

	if err := os.WriteFile(decisionsPath, WriteDecisionsFile(df.Decisions), 0644); err != nil {
		return fmt.Errorf("store: failed to write decisions: %w", err)
	}

	// Update index
	s.updateIndex(func(idx *Index) {
		idx.DecisionsCount = len(df.Decisions)
	})

	return nil
}

// AddNote adds a note to a task (stored within the task's .md file).
func (s *Store) AddNote(taskID, noteText string) (string, error) {
	if isEmptyString(noteText) {
		return "", ErrEmptyNoteText
	}
	if _, err := os.Stat(s.taskPath(taskID)); os.IsNotExist(err) {
		return "", fmt.Errorf("store: task %q not found", taskID)
	}

	noteID := task.GenerateShortID()
	err := s.updateTask(taskID, func(t *task.Task, notes *[]task.Note) error {
		*notes = append(*notes, task.Note{
			ID:        noteID,
			Text:      noteText,
			CreatedAt: time.Now().UTC(),
		})
		t.UpdatedAt = time.Now().UTC()
		return nil
	})

	return noteID, err
}

// AddNoteFull adds a note to a task with all fields preserved (for migration).
// Unlike AddNote, this preserves the original ID and timestamp.
func (s *Store) AddNoteFull(taskID string, note task.Note) error {
	if _, err := os.Stat(s.taskPath(taskID)); os.IsNotExist(err) {
		return fmt.Errorf("store: task %q not found", taskID)
	}

	// Ensure ID and timestamp are set
	if note.ID == "" {
		note.ID = task.GenerateShortID()
	}
	if note.CreatedAt.IsZero() {
		note.CreatedAt = time.Now().UTC()
	}

	return s.updateTask(taskID, func(t *task.Task, notes *[]task.Note) error {
		*notes = append(*notes, note)
		// Don't update task's UpdatedAt to preserve original timeline during migration
		return nil
	})
}

// RemoveNote removes a note from a task.
func (s *Store) RemoveNote(taskID, noteID string) (string, error) {
	var removedText string
	err := s.updateTask(taskID, func(t *task.Task, notes *[]task.Note) error {
		for i, note := range *notes {
			if note.ID == noteID {
				removedText = note.Text
				*notes = append((*notes)[:i], (*notes)[i+1:]...)
				t.UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return fmt.Errorf("store: note %q not found for task %q", noteID, taskID)
	})
	return removedText, err
}

// =============================================================================
// Archive Operations
// =============================================================================

// ArchiveTask moves a done task to archive.
func (s *Store) ArchiveTask(id string, summary string) error {
	t, notes, err := s.readTask(id)
	if err != nil {
		return err
	}

	if t.Status != task.StatusDone {
		return fmt.Errorf("store: task %q must be done to archive (current status: %s)", id, t.Status)
	}

	// Write to archive (include summary in description if provided)
	archiveTask := *t
	if summary != "" {
		archiveTask.Description = t.Description + "\n\n## Summary\n" + summary
	}

	archPath := s.archivePath(id)
	content, err := WriteTaskFile(id, archiveTask, notes)
	if err != nil {
		return err
	}
	if err := os.WriteFile(archPath, content, 0644); err != nil {
		return fmt.Errorf("store: failed to write archived task: %w", err)
	}

	// Remove from tasks
	if err := s.deleteTask(id); err != nil {
		return err
	}

	// Update index
	return s.updateIndex(func(idx *Index) {
		idx.ArchivedCount++
	})
}

// ArchiveDoneTasks archives all done tasks older than the given duration.
func (s *Store) ArchiveDoneTasks(olderThan time.Duration) ([]string, error) {
	cutoff := time.Now().Add(-olderThan)
	var archived []string

	ids, err := s.listTaskIDs()
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		t, _, err := s.readTask(id)
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
func (s *Store) RestoreTask(id string) error {
	archPath := s.archivePath(id)
	data, err := os.ReadFile(archPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("store: archived task %q not found", id)
		}
		return err
	}

	if _, err := os.Stat(s.taskPath(id)); err == nil {
		return fmt.Errorf("store: cannot restore - task %q already exists in active tasks", id)
	}

	parsed, err := ParseTaskFile(id, data)
	if err != nil {
		return err
	}

	t := parsed.ToTask()
	t.Status = task.StatusReady
	t.UpdatedAt = time.Now().UTC()

	if err := s.writeTask(id, t, parsed.ToNotes()); err != nil {
		return err
	}

	if err := os.Remove(archPath); err != nil {
		return err
	}

	// Update index
	return s.updateIndex(func(idx *Index) {
		idx.ArchivedCount--
	})
}

// GetArchivedTasks returns all archived tasks.
func (s *Store) GetArchivedTasks() (map[string]task.ArchivedTask, error) {
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
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".md")
		archPath := filepath.Join(archiveDir, entry.Name())
		data, err := os.ReadFile(archPath)
		if err != nil {
			continue
		}
		parsed, err := ParseTaskFile(id, data)
		if err != nil {
			continue
		}
		t := parsed.ToTask()
		result[id] = task.ArchivedTask{
			Task:       t,
			ArchivedAt: t.UpdatedAt,
			TotalTime:  t.Duration,
		}
	}

	return result, nil
}

// GetArchivedTask returns a single archived task.
func (s *Store) GetArchivedTask(id string) (*task.ArchivedTask, error) {
	archPath := s.archivePath(id)
	data, err := os.ReadFile(archPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("store: archived task %q not found", id)
		}
		return nil, err
	}

	parsed, err := ParseTaskFile(id, data)
	if err != nil {
		return nil, err
	}

	t := parsed.ToTask()
	return &task.ArchivedTask{
		Task:       t,
		ArchivedAt: t.UpdatedAt,
		TotalTime:  t.Duration,
	}, nil
}

// ClearArchive removes all archived tasks.
func (s *Store) ClearArchive() (int, error) {
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
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		if err := os.Remove(filepath.Join(archiveDir, entry.Name())); err == nil {
			count++
		}
	}

	// Update index
	s.updateIndex(func(idx *Index) {
		idx.ArchivedCount = 0
	})

	return count, nil
}

// =============================================================================
// Helpers
// =============================================================================

func ptrToStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
