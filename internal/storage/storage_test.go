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

func TestJSONStoreSaveAndLoadAttempts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	store := NewJSONStore(path)
	want := ProgressFile{
		Items:    map[string]scheduler.Progress{},
		Attempts: map[string]string{"go-001": "package exercise\n\nfunc Hello() string { return \"\" }\n"},
	}

	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Attempts["go-001"] != want.Attempts["go-001"] {
		t.Fatalf("Attempts[go-001] = %q, want %q", got.Attempts["go-001"], want.Attempts["go-001"])
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

func TestCurrentIsTrackedPerDeck(t *testing.T) {
	var progress ProgressFile
	progress.SetCurrent("go", "go-002")
	progress.SetCurrent("shell", "sh-004")

	if got, want := progress.CurrentFor("go"), "go-002"; got != want {
		t.Errorf("CurrentFor(go) = %q, want %q", got, want)
	}
	if got, want := progress.CurrentFor("shell"), "sh-004"; got != want {
		t.Errorf("CurrentFor(shell) = %q, want %q", got, want)
	}

	progress.SetCurrent("shell", "")
	if got := progress.CurrentFor("shell"); got != "" {
		t.Errorf("CurrentFor(shell) = %q, want it cleared", got)
	}
	if got, want := progress.CurrentFor("go"), "go-002"; got != want {
		t.Errorf("CurrentFor(go) = %q, want the other deck untouched (%q)", got, want)
	}
}

// Progress files written before shell exercises existed hold a single current
// exercise, which belonged to the only deck there was.
func TestCurrentFallsBackToThePreDeckField(t *testing.T) {
	progress := ProgressFile{CurrentExerciseID: "go-007"}

	if got, want := progress.CurrentFor("go"), "go-007"; got != want {
		t.Errorf("CurrentFor(go) = %q, want %q", got, want)
	}
	if got := progress.CurrentFor("shell"); got != "" {
		t.Errorf("CurrentFor(shell) = %q, want nothing", got)
	}

	progress.SetCurrent("go", "go-008")
	if progress.CurrentExerciseID != "" {
		t.Errorf("CurrentExerciseID = %q, want the pre-deck field retired once the map holds the answer", progress.CurrentExerciseID)
	}
	if got, want := progress.CurrentFor("go"), "go-008"; got != want {
		t.Errorf("CurrentFor(go) = %q, want %q", got, want)
	}
}
