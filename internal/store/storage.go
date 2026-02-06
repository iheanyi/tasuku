// Package store provides file-based storage with locking.
// This file defines the Storage interface and auto-detection logic.

package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/iheanyi/tasuku/internal/store/v4"
	"github.com/iheanyi/tasuku/internal/task"
)

// MigrationReader defines the minimal interface for legacy storage backends (V2/V3).
// These backends are only used to read data for migration to the current format (V4).
type MigrationReader interface {
	Read() (*task.File, error)
	Exists() bool
	Path() string
}

// Storage defines the interface for task storage backends.
// Only the V4 store implements this interface.
type Storage interface {
	// Core operations
	Path() string
	Exists() bool
	Init() error
	Read() (*task.File, error)
	Update(fn func(*task.File) error) error

	// Index-based fast reads (reads index.json instead of all task files)
	ListFromIndex() ([]task.TaskSummary, error)                   // Returns task summaries from index
	CountByStatus() (map[string]int, error)                       // Returns status counts from index
	GetSubtaskIDs(parentID string) ([]string, error)              // Returns subtask IDs from index
	ContextCounts() (learnings int, decisions int, err error)     // Returns learnings/decisions counts from index

	// Task operations
	AddTask(id, description string) error
	AddTaskWithPriority(id, description string, priority *int) error
	AddTaskWithTags(id, description string, priority *int, tags []string) error
	AddSubtask(id, description, parentID string) error
	SetStatus(id string, status task.Status) error
	SetStatusAndRead(id string, status task.Status) (*task.File, error) // Sets status and returns updated file (avoids redundant read)
	MarkDoneAndUnblock(id string) ([]string, error)                     // Marks task done and auto-unblocks dependent tasks
	SetDescription(id string, description string) error
	SetPriority(id string, priority int) error
	SetParent(id string, parentID *string) error
	GetSubtasks(parentID string) (map[string]task.Task, error)

	// Blocking
	BlockTask(id string, blockers []string) error
	UnblockTask(id string) error
	RemoveBlocker(id string, blocker string) error

	// Task management
	DeleteTask(id string) error
	EditTask(id string, description string) error

	// Ownership
	SetOwner(id string, owner string) error
	ClearOwner(id string) error
	ClaimTask(id string, owner string) error
	ReleaseTask(id string) error

	// Tags
	AddTag(id string, tag string) error
	RemoveTag(id string, tag string) error

	// Fields
	SetField(id string, key, value string) error
	RemoveField(id string, key string) error

	// Timer
	StartTimer(id string) error
	StopTimer(id string) (time.Duration, error)
	StopTimerIfRunning(id string) (time.Duration, bool, error) // Returns elapsed time, whether timer was running, error
	GetActiveTimers() (map[string]task.Task, error)

	// Context
	AddLearning(text string) (string, error)
	AddLearningWithRule(text string, forceRule *bool) (string, bool, error)
	AddLearningWithScope(text, scope string, forceRule *bool) (string, bool, error)
	RemoveLearning(id string) (string, error)
	FindLearningByText(query string) (*task.Learning, error)
	AddDecision(d task.Decision) error
	AddNote(taskID, noteText string) (string, error)
	RemoveNote(taskID, noteID string) (string, error)

	// Archive
	ArchiveTask(id string, summary string) error
	ArchiveDoneTasks(olderThan time.Duration) ([]string, error)
	RestoreTask(id string) error
	GetArchivedTasks() (map[string]task.ArchivedTask, error)
	GetArchivedTask(id string) (*task.ArchivedTask, error)
	ClearArchive() (int, error)
}

// StorageType indicates which storage backend is in use.
type StorageType int

const (
	StorageTypeNone StorageType = iota
	StorageTypeFile             // .tasuku.json (V1/V2)
	StorageTypeDir              // .tasuku/ (V3 JSON)
	StorageTypeDirV4            // .tasuku/ (V4 Markdown)
)

// LatestStorageType is the current default storage format for new projects.
// Update this when adding a new storage version so AutoDetect, the test
// harness, and any assertions stay in sync automatically.
const LatestStorageType = StorageTypeDirV4

// DetectStorageType checks which storage format exists in the given directory.
func DetectStorageType(dir string) StorageType {
	// Check for directory format first (V3 or V4)
	dirPath := filepath.Join(dir, DirName)
	if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
		// Check if V4 (has config.json with version: 4)
		if detectV4Format(dirPath) {
			return StorageTypeDirV4
		}
		return StorageTypeDir
	}

	// Check for V1/V2 file format
	filePath := filepath.Join(dir, DefaultFileName)
	if _, err := os.Stat(filePath); err == nil {
		return StorageTypeFile
	}

	return StorageTypeNone
}

// detectV4Format checks if a .tasuku directory uses V4 format.
// V4 is distinguished by a config.json with "version": 4.
func detectV4Format(dirPath string) bool {
	configPath := filepath.Join(dirPath, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}

	var config struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return false
	}

	return config.Version == 4
}

// DetectStorageTypeUp searches up directories to detect storage type.
// Returns the storage type and the directory where storage was found.
func DetectStorageTypeUp() (StorageType, string) {
	dir, err := os.Getwd()
	if err != nil {
		return StorageTypeNone, ""
	}

	for {
		// Check for directory format first (V3 or V4)
		dirPath := filepath.Join(dir, DirName)
		if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
			// Check if V4 (has config.json with version: 4)
			if detectV4Format(dirPath) {
				return StorageTypeDirV4, dir
			}
			return StorageTypeDir, dir
		}

		// Check for V1/V2 file format
		filePath := filepath.Join(dir, DefaultFileName)
		if _, err := os.Stat(filePath); err == nil {
			return StorageTypeFile, dir
		}

		// Stop at git root
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return StorageTypeNone, dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return StorageTypeNone, ""
		}
		dir = parent
	}
}

// AutoDetect returns the appropriate storage backend based on what exists.
// Only V4 (.tasuku/ with config.json version=4) is supported as a full storage backend.
// Legacy formats (V2 .tasuku.json, V3 .tasuku/) return nil and require migration.
// If no storage exists, returns a V4 Markdown store (the default for new projects).
func AutoDetect() Storage {
	storageType, dir := DetectStorageTypeUp()

	switch storageType {
	case StorageTypeDirV4:
		return v4.New(filepath.Join(dir, DirName))
	case StorageTypeDir:
		// V3 detected - return nil to signal migration needed
		return nil
	case StorageTypeFile:
		// V2 detected - return nil to signal migration needed
		return nil
	default:
		// Default to V4 Markdown-based storage for new projects
		return v4.New(DirName)
	}
}

// AutoDetectWithWarning returns the storage backend and an error if
// a legacy format (V2 or V3) is detected and must be migrated.
func AutoDetectWithWarning() (Storage, error) {
	storageType, dir := DetectStorageTypeUp()

	switch storageType {
	case StorageTypeDirV4:
		return v4.New(filepath.Join(dir, DirName)), nil
	case StorageTypeDir:
		// V3 detected - return error requiring migration
		return nil, fmt.Errorf("legacy V3 format detected at %s - run 'tk migrate v4' to upgrade", filepath.Join(dir, DirName))
	case StorageTypeFile:
		// V2 detected - return error requiring migration
		return nil, fmt.Errorf("legacy .tasuku.json format detected at %s - run 'tk migrate v4' to upgrade", filepath.Join(dir, DefaultFileName))
	default:
		// Default to V4 Markdown-based storage for new projects
		return v4.New(DirName), nil
	}
}

// GetV2StoreForMigration returns the V2 file store for migration purposes.
// Returns nil if no V2 storage is detected.
// This is the ONLY function that should access V2 storage directly.
// Note: This directly checks for .tasuku.json file regardless of whether
// .tasuku/ directory exists, so migration can detect "already migrated" state.
func GetV2StoreForMigration() MigrationReader {
	dir, err := os.Getwd()
	if err != nil {
		return nil
	}

	for {
		// Directly check for V2 file (don't use DetectStorageTypeUp which prioritizes dir)
		filePath := filepath.Join(dir, DefaultFileName)
		if _, err := os.Stat(filePath); err == nil {
			return New(filePath)
		}

		// Stop at git root
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}

// GetV3StoreForMigration returns the V3 directory store for migration purposes.
// Returns nil if no V3 storage is detected.
// This is the ONLY function that should access V3 storage directly.
func GetV3StoreForMigration(root string) MigrationReader {
	return NewDirStore(root)
}

// DefaultStorageWithWarning returns the auto-detected storage backend.
// If legacy format is detected, returns an error requiring migration.
// This is the recommended function for CLI commands.
func DefaultStorageWithWarning() (Storage, error) {
	return AutoDetectWithWarning()
}
