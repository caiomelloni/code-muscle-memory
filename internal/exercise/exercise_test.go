package exercise

import (
	"os"
	"path/filepath"
	"strings"
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

func TestValidateDefaultsKindToImplementation(t *testing.T) {
	ex := Exercise{
		ID:          "go-001",
		Title:       "A",
		Description: "desc",
		Language:    "go",
		Topic:       "topic",
		Difficulty:  1,
	}
	err := ex.Validate()
	if err == nil {
		t.Fatal("Validate() succeeded, want error for missing tests")
	}
	if !strings.Contains(err.Error(), "tests") {
		t.Fatalf("Validate() error = %q, want it to mention missing tests", err)
	}
}

func TestValidateTestWritingRequiresSubjectAndMutants(t *testing.T) {
	ex := Exercise{
		ID:          "go-006",
		Title:       "A",
		Description: "desc",
		Language:    "go",
		Topic:       "topic",
		Difficulty:  1,
		Kind:        KindTestWriting,
	}
	err := ex.Validate()
	if err == nil {
		t.Fatal("Validate() succeeded, want error for missing subject_code/mutants")
	}
	for _, want := range []string{"subject_code", "mutants"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Validate() error = %q, want it to mention %q", err, want)
		}
	}
}

func TestValidateTestWritingMutantHintsLengthMismatch(t *testing.T) {
	ex := Exercise{
		ID:          "go-006",
		Title:       "A",
		Description: "desc",
		Language:    "go",
		Topic:       "topic",
		Difficulty:  1,
		Kind:        KindTestWriting,
		SubjectCode: "package exercise",
		Mutants:     []string{"package exercise"},
		MutantHints: []string{"hint one", "hint two"},
	}
	if err := ex.Validate(); err == nil {
		t.Fatal("Validate() succeeded, want error for mismatched mutant_hints length")
	}
}

func TestValidateRequiredTestFeatures(t *testing.T) {
	ex := Exercise{
		ID:                   "go-008",
		Title:                "A",
		Description:          "desc",
		Language:             "go",
		Topic:                "topic",
		Difficulty:           1,
		Kind:                 KindTestWriting,
		SubjectCode:          "package exercise",
		Mutants:              []string{"package exercise"},
		RequiredTestFeatures: []string{FeatureTable, FeatureSubtests},
	}
	if err := ex.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want success for known features", err)
	}

	ex.RequiredTestFeatures = []string{"bogus"}
	err := ex.Validate()
	if err == nil {
		t.Fatal("Validate() succeeded, want error for unknown required test feature")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Fatalf("Validate() error = %q, want it to name the unknown feature", err)
	}
}

func TestValidateRejectsRequiredTestFeaturesOnImplementation(t *testing.T) {
	ex := Exercise{
		ID:                   "go-001",
		Title:                "A",
		Description:          "desc",
		Language:             "go",
		Topic:                "topic",
		Difficulty:           1,
		Tests:                "package exercise",
		RequiredTestFeatures: []string{FeatureTable},
	}
	if err := ex.Validate(); err == nil {
		t.Fatal("Validate() succeeded, want error for required_test_features on implementation exercise")
	}
}

func TestValidateRejectsUnknownKind(t *testing.T) {
	ex := Exercise{
		ID:          "go-001",
		Title:       "A",
		Description: "desc",
		Language:    "go",
		Topic:       "topic",
		Difficulty:  1,
		Kind:        "bogus",
	}
	if err := ex.Validate(); err == nil {
		t.Fatal("Validate() succeeded, want error for unknown kind")
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
