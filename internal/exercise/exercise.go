package exercise

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	KindImplementation = "implementation"
	KindTestWriting    = "test_writing"
)

type Exercise struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Objective   string            `json:"objective"`
	Difficulty  int               `json:"difficulty"`
	Language    string            `json:"language"`
	Topic       string            `json:"topic"`
	Kind        string            `json:"kind,omitempty"`
	StarterCode string            `json:"starter_code"`
	Tests       string            `json:"tests"`
	Solution    string            `json:"solution,omitempty"`
	SubjectCode string            `json:"subject_code,omitempty"`
	Mutants     []string          `json:"mutants,omitempty"`
	MutantHints []string          `json:"mutant_hints,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// EffectiveKind returns the exercise's kind, defaulting to KindImplementation
// for exercises authored before the "kind" field existed.
func (e Exercise) EffectiveKind() string {
	if strings.TrimSpace(e.Kind) == "" {
		return KindImplementation
	}
	return e.Kind
}

func LoadDir(dir string) ([]Exercise, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var exercises []Exercise
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		loaded, err := LoadFile(path)
		if err != nil {
			return nil, err
		}
		exercises = append(exercises, loaded...)
	}
	if len(exercises) == 0 {
		return nil, fmt.Errorf("no exercise JSON files found in %s", dir)
	}

	seen := map[string]struct{}{}
	for _, ex := range exercises {
		if err := ex.Validate(); err != nil {
			return nil, err
		}
		if _, ok := seen[ex.ID]; ok {
			return nil, fmt.Errorf("duplicate exercise id %q", ex.ID)
		}
		seen[ex.ID] = struct{}{}
	}

	sort.Slice(exercises, func(i, j int) bool {
		if exercises[i].Difficulty != exercises[j].Difficulty {
			return exercises[i].Difficulty < exercises[j].Difficulty
		}
		return exercises[i].ID < exercises[j].ID
	})
	return exercises, nil
}

func LoadFile(path string) ([]Exercise, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var many []Exercise
	if err := json.Unmarshal(data, &many); err == nil {
		return many, nil
	}

	var one Exercise
	if err := json.Unmarshal(data, &one); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return []Exercise{one}, nil
}

func (e Exercise) Validate() error {
	var missing []string
	if strings.TrimSpace(e.ID) == "" {
		missing = append(missing, "id")
	}
	if strings.TrimSpace(e.Title) == "" {
		missing = append(missing, "title")
	}
	if strings.TrimSpace(e.Description) == "" {
		missing = append(missing, "description")
	}
	if strings.TrimSpace(e.Language) == "" {
		missing = append(missing, "language")
	}
	if strings.TrimSpace(e.Topic) == "" {
		missing = append(missing, "topic")
	}
	if e.Difficulty <= 0 {
		missing = append(missing, "difficulty")
	}

	switch e.EffectiveKind() {
	case KindImplementation:
		if strings.TrimSpace(e.Tests) == "" {
			missing = append(missing, "tests")
		}
	case KindTestWriting:
		if strings.TrimSpace(e.SubjectCode) == "" {
			missing = append(missing, "subject_code")
		}
		if len(e.Mutants) == 0 {
			missing = append(missing, "mutants")
		}
		if len(e.MutantHints) != 0 && len(e.MutantHints) != len(e.Mutants) {
			return fmt.Errorf("exercise %q has %d mutant_hints but %d mutants", e.ID, len(e.MutantHints), len(e.Mutants))
		}
	default:
		return fmt.Errorf("exercise %q has unknown kind %q", e.ID, e.Kind)
	}

	if len(missing) > 0 {
		return fmt.Errorf("exercise %q missing required fields: %s", e.ID, strings.Join(missing, ", "))
	}
	if e.Language != "go" {
		return errors.New("v1 supports only go exercises")
	}
	return nil
}
