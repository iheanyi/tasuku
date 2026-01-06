// Package store provides file-based storage with locking.
// This file defines the Storage interface and auto-detection logic.

package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/iheanyi/tasuku/internal/task"
)

// Storage defines the interface for task storage backends.
// Both Store (single file) and DirStore (directory) implement this interface.
type Storage interface {
	// Core operations
	Path() string
	Exists() bool
	Init() error
	Read() (*task.File, error)
	Update(fn func(*task.File) error) error

	// Task operations
	AddTask(id, description string) error
	AddTaskWithPriority(id, description string, priority *int) error
	AddTaskWithTags(id, description string, priority *int, tags []string) error
	AddSubtask(id, description, parentID string) error
	SetStatus(id string, status task.Status) error
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
	StorageTypeFile             // .tasuku.json
	StorageTypeDir              // .tasuku/
)

// DetectStorageType checks which storage format exists in the given directory.
func DetectStorageType(dir string) StorageType {
	// Check for V3 directory format first (preferred)
	dirPath := filepath.Join(dir, DirName)
	if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
		return StorageTypeDir
	}

	// Check for V1/V2 file format
	filePath := filepath.Join(dir, DefaultFileName)
	if _, err := os.Stat(filePath); err == nil {
		return StorageTypeFile
	}

	return StorageTypeNone
}

// DetectStorageTypeUp searches up directories to detect storage type.
// Returns the storage type and the directory where storage was found.
func DetectStorageTypeUp() (StorageType, string) {
	dir, err := os.Getwd()
	if err != nil {
		return StorageTypeNone, ""
	}

	for {
		// Check for V3 directory format first
		dirPath := filepath.Join(dir, DirName)
		if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
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

// ErrV2Detected is returned when legacy V2 format is detected.
// Users should migrate to V3 using 'tk migrate v3'.
var ErrV2Detected = fmt.Errorf("legacy .tasuku.json format detected - run 'tk migrate v3' to upgrade")

// AutoDetect returns the appropriate storage backend based on what exists.
// It only supports V3 (.tasuku/) format. If V2 (.tasuku.json) is detected,
// it returns nil and users should migrate using 'tk migrate v3'.
// If neither exists, returns a V3 directory store (the default for new projects).
func AutoDetect() Storage {
	storageType, dir := DetectStorageTypeUp()

	switch storageType {
	case StorageTypeDir:
		return NewDirStore(filepath.Join(dir, DirName))
	case StorageTypeFile:
		// V2 detected - return nil to signal error
		return nil
	default:
		// Default to V3 directory-based storage for new projects
		return NewDirStore(DirName)
	}
}

// AutoDetectWithWarning returns the storage backend and an error if
// the V1/V2 format is detected and must be migrated.
func AutoDetectWithWarning() (Storage, error) {
	storageType, dir := DetectStorageTypeUp()

	switch storageType {
	case StorageTypeDir:
		return NewDirStore(filepath.Join(dir, DirName)), nil
	case StorageTypeFile:
		// V2 detected - return error requiring migration
		return nil, fmt.Errorf("legacy .tasuku.json format detected at %s - run 'tk migrate v3' to upgrade", filepath.Join(dir, DefaultFileName))
	default:
		// Default to V3 directory-based storage for new projects
		return NewDirStore(DirName), nil
	}
}

// NeedsMigration returns true if a V1/V2 file exists and should be migrated.
func NeedsMigration() bool {
	storageType, _ := DetectStorageTypeUp()
	return storageType == StorageTypeFile
}

// GetV2StoreForMigration returns the V2 file store for migration purposes.
// Returns nil if no V2 storage is detected.
// This is the ONLY function that should access V2 storage directly.
// Note: This directly checks for .tasuku.json file regardless of whether
// .tasuku/ directory exists, so migration can detect "already migrated" state.
func GetV2StoreForMigration() Storage {
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

// DefaultStorage returns the auto-detected storage backend.
// This is the recommended way to get a storage instance in CLI commands.
func DefaultStorage() Storage {
	return AutoDetect()
}

// DefaultStorageWithWarning returns the auto-detected storage backend.
// If legacy V2 format is detected, prints an error and exits.
// This is the recommended function for CLI commands.
func DefaultStorageWithWarning() Storage {
	storage, err := AutoDetectWithWarning()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	return storage
}
