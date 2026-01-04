// Package store provides file-based storage with locking.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/iheanyi/tasuku/internal/task"
)

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

// AddLearning adds a learning to context.
func (s *Store) AddLearning(learning string) error {
	return s.Update(func(f *task.File) error {
		f.Context.Learnings = append(f.Context.Learnings, learning)
		return nil
	})
}

// AddDecision adds a decision to context.
func (s *Store) AddDecision(d task.Decision) error {
	return s.Update(func(f *task.File) error {
		f.Context.Decisions = append(f.Context.Decisions, d)
		return nil
	})
}

// AddNote adds a note to a specific task.
func (s *Store) AddNote(taskID, note string) error {
	return s.Update(func(f *task.File) error {
		if _, exists := f.Tasks[taskID]; !exists {
			return fmt.Errorf("store: task %q not found", taskID)
		}
		if f.Context.Notes == nil {
			f.Context.Notes = make(map[string][]string)
		}
		f.Context.Notes[taskID] = append(f.Context.Notes[taskID], note)
		return nil
	})
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
