package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iheanyi/tasuku/internal/task"
)

func TestStore_New(t *testing.T) {
	s := New("/tmp/test.json")
	if s.Path() != "/tmp/test.json" {
		t.Errorf("expected path '/tmp/test.json', got %s", s.Path())
	}
}

func TestStore_Exists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)

	if s.Exists() {
		t.Fatal("store should not exist before file is created")
	}

	// Create a valid JSON file
	f := task.NewFile()
	data, _ := json.MarshalIndent(f, "", "  ")
	os.WriteFile(path, data, 0644)

	if !s.Exists() {
		t.Fatal("store should exist after file is created")
	}
}

func TestStore_Read(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)

	// Create a V2 file with some data
	f := task.NewFile()
	f.Tasks["test-task"] = task.NewTask("Test description")
	f.Context.Learnings = append(f.Context.Learnings, task.Learning{
		ID:   "l1",
		Text: "Test learning",
	})
	f.Context.Decisions = append(f.Context.Decisions, task.Decision{
		ID:    "d1",
		Chose: "Option A",
	})
	data, _ := json.MarshalIndent(f, "", "  ")
	os.WriteFile(path, data, 0644)

	// Read it back
	result, err := s.Read()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if len(result.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result.Tasks))
	}

	tk, ok := result.Tasks["test-task"]
	if !ok {
		t.Fatal("task not found")
	}
	if tk.Description != "Test description" {
		t.Errorf("wrong description: %s", tk.Description)
	}
	if tk.Status != "ready" {
		t.Errorf("wrong status: %s", tk.Status)
	}

	if len(result.Context.Learnings) != 1 {
		t.Fatalf("expected 1 learning, got %d", len(result.Context.Learnings))
	}
	if result.Context.Learnings[0].Text != "Test learning" {
		t.Errorf("wrong learning: %s", result.Context.Learnings[0].Text)
	}

	if len(result.Context.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(result.Context.Decisions))
	}
	if result.Context.Decisions[0].Chose != "Option A" {
		t.Errorf("wrong decision: %s", result.Context.Decisions[0].Chose)
	}
}

func TestStore_Read_NotExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)

	_, err := s.Read()
	if err == nil {
		t.Fatal("expected error when reading non-existent file")
	}
	if err != ErrNotInitialized {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}
}

func TestStore_Read_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	os.WriteFile(path, []byte("not valid json"), 0644)

	s := New(path)
	_, err := s.Read()
	if err == nil {
		t.Fatal("expected error when reading invalid JSON")
	}
}

func TestStore_Read_WithArchive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)

	// Create file with archived tasks
	f := task.NewFile()
	f.Archive = map[string]task.ArchivedTask{
		"archived-1": {
			Task:    task.NewTask("Archived task"),
			Summary: "Done",
		},
	}
	data, _ := json.MarshalIndent(f, "", "  ")
	os.WriteFile(path, data, 0644)

	result, err := s.Read()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if len(result.Archive) != 1 {
		t.Fatalf("expected 1 archived task, got %d", len(result.Archive))
	}
	if result.Archive["archived-1"].Summary != "Done" {
		t.Errorf("wrong summary: %s", result.Archive["archived-1"].Summary)
	}
}

func TestStore_MigrationReader(t *testing.T) {
	// Verify Store implements MigrationReader
	var _ MigrationReader = New("/tmp/test.json")
}

func TestStore_ErrNotInitialized(t *testing.T) {
	if !strings.Contains(ErrNotInitialized.Error(), "no Tasuku storage found") {
		t.Errorf("unexpected error message: %s", ErrNotInitialized.Error())
	}
	if !strings.Contains(ErrNotInitialized.Error(), "tk init") {
		t.Errorf("error should mention tk init: %s", ErrNotInitialized.Error())
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
