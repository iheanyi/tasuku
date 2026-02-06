// Package store provides file-based storage with locking.
// This file implements the V3 directory-based storage reader for migration purposes only.
// V3 stores are NOT usable as full storage backends — use V4 (internal/store/v4) instead.

package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// DirStore is the legacy V3 directory-based storage reader.
// It only supports Read(), Exists(), and Path() for migration purposes.
// It implements the MigrationReader interface.
type DirStore struct {
	root string // Path to .tasuku directory
}

// NewDirStore creates a new V3 directory-based store reader.
// This should only be used for migration (reading V3 data to migrate to V4).
func NewDirStore(root string) *DirStore {
	return &DirStore{root: root}
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

// TaskWithID wraps a Task with its ID for storage.
type TaskWithID struct {
	ID string `json:"id"`
	task.Task
}

// taskPath returns the path to a task file.
func (s *DirStore) taskPath(id string) string {
	return filepath.Join(s.root, TasksDir, id+".json")
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

// Read loads all data into a task.File structure for migration.
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
