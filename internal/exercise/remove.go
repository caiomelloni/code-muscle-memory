package exercise

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// RemoveResult describes what removing an exercise did to the deck on disk.
type RemoveResult struct {
	// Path is the JSON file the exercise was defined in.
	Path string
	// FileDeleted reports that the exercise was the file's last one, so the
	// now-empty file was deleted rather than left holding an empty list.
	FileDeleted bool
}

// Remove deletes the exercise with the given id from whichever JSON file in
// dir defines it. The remaining exercises are written back verbatim: each is
// preserved as its original JSON text, so removing one card never reorders or
// rewrites the fields of its neighbours.
func Remove(dir, id string) (RemoveResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return RemoveResult{}, err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		raw, err := readRawExercises(path)
		if err != nil {
			return RemoveResult{}, err
		}

		kept := make([]json.RawMessage, 0, len(raw))
		found := false
		for _, message := range raw {
			var header struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(message, &header); err != nil {
				return RemoveResult{}, fmt.Errorf("parse %s: %w", path, err)
			}
			if header.ID == id {
				found = true
				continue
			}
			kept = append(kept, message)
		}
		if !found {
			continue
		}

		if len(kept) == 0 {
			if err := os.Remove(path); err != nil {
				return RemoveResult{}, err
			}
			return RemoveResult{Path: path, FileDeleted: true}, nil
		}

		encoded, err := json.MarshalIndent(kept, "", "  ")
		if err != nil {
			return RemoveResult{}, err
		}
		if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
			return RemoveResult{}, err
		}
		return RemoveResult{Path: path}, nil
	}

	return RemoveResult{}, fmt.Errorf("exercise %q not found in %s", id, dir)
}

// readRawExercises returns a file's exercises as their untouched JSON text,
// accepting both shapes LoadFile does: a list of exercises or a single one.
func readRawExercises(path string) ([]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var many []json.RawMessage
	if err := json.Unmarshal(data, &many); err == nil {
		return many, nil
	}

	var one json.RawMessage
	if err := json.Unmarshal(data, &one); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return []json.RawMessage{one}, nil
}
