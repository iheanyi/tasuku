// Package store provides file-based storage with locking.
// This file implements the V2 single-file storage reader for migration purposes only.
// V2 stores are NOT usable as full storage backends — use V4 (internal/store/v4) instead.

package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/iheanyi/tasuku/internal/task"
)

// ErrNotInitialized is returned when no Tasuku storage exists.
var ErrNotInitialized = errors.New("no Tasuku storage found - run 'tk init' to create one (MCP users: set cwd to project root or use tk serve mcp --dir <path>)")

const (
	DefaultFileName = ".tasuku.json"
)

// Store is the legacy V2 single-file storage reader.
// It only supports Read(), Exists(), and Path() for migration purposes.
// It implements the MigrationReader interface.
type Store struct {
	path string
}

// New creates a new V2 store reader for the given path.
// This should only be used for migration (reading V2 data to migrate to V4).
func New(path string) *Store {
	return &Store{path: path}
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
