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

type Exercise struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Objective   string            `json:"objective"`
	Difficulty  int               `json:"difficulty"`
	Language    string            `json:"language"`
	Topic       string            `json:"topic"`
	StarterCode string            `json:"starter_code"`
	Tests       string            `json:"tests"`
	Solution    string            `json:"solution,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
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
	if strings.TrimSpace(e.Tests) == "" {
		missing = append(missing, "tests")
	}
	if e.Difficulty <= 0 {
		missing = append(missing, "difficulty")
	}
	if len(missing) > 0 {
		return fmt.Errorf("exercise %q missing required fields: %s", e.ID, strings.Join(missing, ", "))
	}
	if e.Language != "go" {
		return errors.New("v1 supports only go exercises")
	}
	return nil
}
