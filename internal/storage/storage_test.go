package storage

import (
	"os"
	"path/filepath"
	"testing"

	"code-muscle-memory/internal/scheduler"
)

func TestJSONStoreMissingFile(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "progress.json"))
	progress, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if progress.Version != 1 {
		t.Fatalf("Version = %d, want 1", progress.Version)
	}
	if len(progress.Items) != 0 {
		t.Fatalf("Items = %d, want 0", len(progress.Items))
	}
}

func TestJSONStoreSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "progress.json")
	store := NewJSONStore(path)
	want := ProgressFile{
		Items: map[string]scheduler.Progress{
			"go-001": {ExerciseID: "go-001", IntervalDays: 3, EaseFactor: 2.5},
		},
	}

	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Items["go-001"].IntervalDays != 3 {
		t.Fatalf("IntervalDays = %d, want 3", got.Items["go-001"].IntervalDays)
	}
}

func TestJSONStoreRejectsCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewJSONStore(path).Load(); err == nil {
		t.Fatal("Load succeeded with corrupt JSON")
	}
}
