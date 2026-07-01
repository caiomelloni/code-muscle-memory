package exercise

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDir(t *testing.T) {
	dir := t.TempDir()
	data := `[
		{
			"id": "go-002",
			"title": "B",
			"description": "desc",
			"objective": "obj",
			"difficulty": 2,
			"language": "go",
			"topic": "topic",
			"tests": "package exercise"
		},
		{
			"id": "go-001",
			"title": "A",
			"description": "desc",
			"objective": "obj",
			"difficulty": 1,
			"language": "go",
			"topic": "topic",
			"tests": "package exercise"
		}
	]`
	if err := os.WriteFile(filepath.Join(dir, "exercises.json"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	exercises, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(exercises), 2; got != want {
		t.Fatalf("len(exercises) = %d, want %d", got, want)
	}
	if got, want := exercises[0].ID, "go-001"; got != want {
		t.Fatalf("first exercise = %q, want %q", got, want)
	}
}

func TestLoadDirRejectsDuplicates(t *testing.T) {
	dir := t.TempDir()
	data := `[
		{"id":"same","title":"A","description":"desc","difficulty":1,"language":"go","topic":"topic","tests":"package exercise"},
		{"id":"same","title":"B","description":"desc","difficulty":1,"language":"go","topic":"topic","tests":"package exercise"}
	]`
	if err := os.WriteFile(filepath.Join(dir, "exercises.json"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadDir(dir); err == nil {
		t.Fatal("LoadDir succeeded with duplicate IDs")
	}
}
