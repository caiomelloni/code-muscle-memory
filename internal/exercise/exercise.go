package exercise

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	LanguageGo    = "go"
	LanguageShell = "shell"
)

// Languages returns every deck the app can serve, in the order they should be
// offered to the user.
func Languages() []string {
	return []string{LanguageGo, LanguageShell}
}

func SupportsLanguage(language string) bool {
	for _, known := range Languages() {
		if known == language {
			return true
		}
	}
	return false
}

const (
	KindImplementation = "implementation"
	KindTestWriting    = "test_writing"
	KindCommand        = "command"
)

// Required test features are language-neutral concepts describing the shape a
// user's test must have. Each language executor decides how to detect them.
const (
	FeatureSubtests = "subtests"
	FeatureTable    = "table"
)

var knownTestFeatures = map[string]struct{}{
	FeatureSubtests: {},
	FeatureTable:    {},
}

type Exercise struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Objective   string   `json:"objective"`
	Difficulty  int      `json:"difficulty"`
	Language    string   `json:"language"`
	Topic       string   `json:"topic"`
	Kind        string   `json:"kind,omitempty"`
	StarterCode string   `json:"starter_code"`
	Tests       string   `json:"tests"`
	Solution    string   `json:"solution,omitempty"`
	SubjectCode string   `json:"subject_code,omitempty"`
	Mutants     []string `json:"mutants,omitempty"`
	MutantHints []string `json:"mutant_hints,omitempty"`

	// RequiredTestFeatures lists structural features the user's test must
	// exhibit, on top of passing the subject and killing every mutant. Only
	// valid for test-writing exercises.
	RequiredTestFeatures []string `json:"required_test_features,omitempty"`

	// Setup is a script that seeds the sandbox directory the user's command
	// runs in. Check is the hidden script that decides whether the command did
	// what the exercise asked. RequiredCommands names utilities the answer must
	// invoke, for exercises whose point is recalling a specific tool. All three
	// are only valid for shell exercises.
	Setup            string   `json:"setup,omitempty"`
	Check            string   `json:"check,omitempty"`
	RequiredCommands []string `json:"required_commands,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// EffectiveKind returns the exercise's kind, defaulting to the only kind its
// language supports out of the box: implementation for Go exercises authored
// before the "kind" field existed, and command for shell exercises, which have
// no second kind to distinguish.
func (e Exercise) EffectiveKind() string {
	if strings.TrimSpace(e.Kind) != "" {
		return e.Kind
	}
	if e.Language == LanguageShell {
		return KindCommand
	}
	return KindImplementation
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

	kind := e.EffectiveKind()
	switch {
	case strings.TrimSpace(e.Language) == "":
		// Already reported as a missing field; the kind rules below are all
		// language-specific and have nothing to say about it.
	case e.Language == LanguageGo && kind == KindImplementation:
		if strings.TrimSpace(e.Tests) == "" {
			missing = append(missing, "tests")
		}
		if len(e.RequiredTestFeatures) > 0 {
			return fmt.Errorf("exercise %q has required_test_features but is not a test-writing exercise", e.ID)
		}
	case e.Language == LanguageGo && kind == KindTestWriting:
		if strings.TrimSpace(e.SubjectCode) == "" {
			missing = append(missing, "subject_code")
		}
		if len(e.Mutants) == 0 {
			missing = append(missing, "mutants")
		}
		if len(e.MutantHints) != 0 && len(e.MutantHints) != len(e.Mutants) {
			return fmt.Errorf("exercise %q has %d mutant_hints but %d mutants", e.ID, len(e.MutantHints), len(e.Mutants))
		}
		for _, feature := range e.RequiredTestFeatures {
			if _, ok := knownTestFeatures[feature]; !ok {
				return fmt.Errorf("exercise %q has unknown required test feature %q", e.ID, feature)
			}
		}
	case e.Language == LanguageShell && kind == KindCommand:
		if strings.TrimSpace(e.Check) == "" {
			missing = append(missing, "check")
		}
		if err := e.rejectGoFields(); err != nil {
			return err
		}
	case !SupportsLanguage(e.Language):
		return fmt.Errorf("exercise %q has unsupported language %q", e.ID, e.Language)
	default:
		return fmt.Errorf("exercise %q has kind %q, which %s exercises do not support", e.ID, kind, e.Language)
	}

	if e.Language != LanguageShell && (strings.TrimSpace(e.Setup) != "" || strings.TrimSpace(e.Check) != "" || len(e.RequiredCommands) > 0) {
		return fmt.Errorf("exercise %q has setup, check, or required_commands but is not a shell exercise", e.ID)
	}
	if len(missing) > 0 {
		return fmt.Errorf("exercise %q missing required fields: %s", e.ID, strings.Join(missing, ", "))
	}
	return nil
}

// rejectGoFields reports Go-only fields carried by a shell exercise. They would
// be silently ignored at grading time, which hides an authoring mistake.
func (e Exercise) rejectGoFields() error {
	var unsupported []string
	if strings.TrimSpace(e.StarterCode) != "" {
		unsupported = append(unsupported, "starter_code")
	}
	if strings.TrimSpace(e.Tests) != "" {
		unsupported = append(unsupported, "tests")
	}
	if strings.TrimSpace(e.SubjectCode) != "" {
		unsupported = append(unsupported, "subject_code")
	}
	if len(e.Mutants) > 0 {
		unsupported = append(unsupported, "mutants")
	}
	if len(e.MutantHints) > 0 {
		unsupported = append(unsupported, "mutant_hints")
	}
	if len(e.RequiredTestFeatures) > 0 {
		unsupported = append(unsupported, "required_test_features")
	}
	if len(unsupported) > 0 {
		return fmt.Errorf("exercise %q has %s, which shell exercises do not support", e.ID, strings.Join(unsupported, ", "))
	}
	return nil
}
