// Package store provides file-based storage with locking.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/iheanyi/tasuku/internal/task"
)

// ErrNotInitialized is returned when the task file does not exist.
var ErrNotInitialized = errors.New("no .tasuku.json found in current directory - run 'tk init' to create one")

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

		// Verify blockers exist
		for _, blocker := range blockers {
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
