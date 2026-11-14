package exercise

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeDeck(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const twoCards = `[
  {
    "id": "go-001",
    "title": "First",
    "description": "d",
    "objective": "o",
    "difficulty": 1,
    "language": "go",
    "topic": "t",
    "starter_code": "package exercise\n",
    "tests": "package exercise\n"
  },
  {
    "id": "go-002",
    "title": "Second",
    "description": "d",
    "objective": "o",
    "difficulty": 2,
    "language": "go",
    "topic": "t",
    "starter_code": "package exercise\n",
    "tests": "package exercise\n"
  }
]
`

func TestRemoveDropsOnlyTheNamedExercise(t *testing.T) {
	dir := writeDeck(t, map[string]string{"deck.json": twoCards})

	result, err := Remove(dir, "go-001")
	if err != nil {
		t.Fatal(err)
	}
	if result.FileDeleted {
		t.Error("FileDeleted = true, want false: the file still holds another exercise")
	}
	if want := filepath.Join(dir, "deck.json"); result.Path != want {
		t.Errorf("Path = %q, want %q", result.Path, want)
	}

	remaining, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0].ID != "go-002" {
		t.Fatalf("remaining exercises = %v, want just go-002", ids(remaining))
	}
	if remaining[0].Title != "Second" {
		t.Errorf("surviving exercise Title = %q, want %q: its fields must be untouched", remaining[0].Title, "Second")
	}
}

func TestRemoveDeletesTheFileWhenItsLastExerciseGoes(t *testing.T) {
	soloList := `[
  {
    "id": "go-003",
    "title": "Only one here",
    "description": "d",
    "objective": "o",
    "difficulty": 3,
    "language": "go",
    "topic": "t",
    "starter_code": "package exercise\n",
    "tests": "package exercise\n"
  }
]
`
	dir := writeDeck(t, map[string]string{"deck.json": twoCards, "solo.json": soloList})

	result, err := Remove(dir, "go-003")
	if err != nil {
		t.Fatal(err)
	}
	if !result.FileDeleted {
		t.Error("FileDeleted = false, want true: that was the file's last exercise")
	}
	if _, err := os.Stat(result.Path); !os.IsNotExist(err) {
		t.Errorf("file %s still exists, want it deleted", result.Path)
	}
}

func TestRemoveHandlesAFileHoldingASingleExercise(t *testing.T) {
	single := `{
  "id": "go-050",
  "title": "Alone",
  "description": "d",
  "objective": "o",
  "difficulty": 1,
  "language": "go",
  "topic": "t",
  "starter_code": "package exercise\n",
  "tests": "package exercise\n"
}
`
	dir := writeDeck(t, map[string]string{"deck.json": twoCards, "single.json": single})

	result, err := Remove(dir, "go-050")
	if err != nil {
		t.Fatal(err)
	}
	if !result.FileDeleted {
		t.Error("FileDeleted = false, want true for a file defining one exercise")
	}
}

func TestRemoveReportsAnUnknownID(t *testing.T) {
	dir := writeDeck(t, map[string]string{"deck.json": twoCards})

	if _, err := Remove(dir, "go-999"); err == nil {
		t.Fatal("Remove() returned no error for an unknown id")
	}

	remaining, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 2 {
		t.Fatalf("remaining exercises = %v, want both left untouched", ids(remaining))
	}
}

func TestRemovePreservesFieldOrderOfSurvivors(t *testing.T) {
	dir := writeDeck(t, map[string]string{"deck.json": twoCards})

	if _, err := Remove(dir, "go-001"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "deck.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("rewritten file is not a JSON list: %v", err)
	}
	if len(raw) != 1 {
		t.Fatalf("rewritten file holds %d exercises, want 1", len(raw))
	}
	// "id" was written first originally and must still come first: the
	// survivor's own JSON text is copied over, not re-encoded from a struct.
	if got := string(raw[0]); !strings.HasPrefix(strings.TrimSpace(got), `{`) || strings.Index(got, `"id"`) > strings.Index(got, `"title"`) {
		t.Errorf("survivor's field order changed:\n%s", got)
	}
}

func ids(exercises []Exercise) []string {
	out := make([]string, 0, len(exercises))
	for _, ex := range exercises {
		out = append(out, ex.ID)
	}
	return out
}
