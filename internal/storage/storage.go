package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"code-muscle-memory/internal/scheduler"
)

type ProgressFile struct {
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`

	// CurrentExerciseID is the pre-deck form of Current, kept so progress files
	// written before shell exercises existed still resume their Go card.
	CurrentExerciseID string `json:"current_exercise_id,omitempty"`

	// Current maps a language to the exercise being worked on in that deck.
	// Decks are practised separately, so switching to shell for an evening must
	// not lose the Go card that was half answered.
	Current map[string]string `json:"current,omitempty"`

	Items    map[string]scheduler.Progress `json:"items"`
	Attempts map[string]string             `json:"attempts,omitempty"`
}

// CurrentFor returns the exercise in progress in one deck, if any.
func (p ProgressFile) CurrentFor(language string) string {
	if id, ok := p.Current[language]; ok {
		return id
	}
	if language == goLanguage {
		return p.CurrentExerciseID
	}
	return ""
}

// SetCurrent records the exercise being worked on in one deck. An empty id
// clears it.
func (p *ProgressFile) SetCurrent(language, id string) {
	if language == goLanguage {
		p.CurrentExerciseID = ""
	}
	if id == "" {
		delete(p.Current, language)
		return
	}
	if p.Current == nil {
		p.Current = map[string]string{}
	}
	p.Current[language] = id
}

// goLanguage is spelled out here rather than imported from the exercise package
// to keep storage free of a dependency on it; only the legacy field needs it.
const goLanguage = "go"

type Store interface {
	Load() (ProgressFile, error)
	Save(progress ProgressFile) error
}

type JSONStore struct {
	Path string
}

func NewJSONStore(path string) JSONStore {
	return JSONStore{Path: path}
}

func (s JSONStore) Load() (ProgressFile, error) {
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return ProgressFile{Version: 1, Items: map[string]scheduler.Progress{}}, nil
	}
	if err != nil {
		return ProgressFile{}, err
	}

	var progress ProgressFile
	if err := json.Unmarshal(data, &progress); err != nil {
		return ProgressFile{}, fmt.Errorf("parse progress file %s: %w", s.Path, err)
	}
	if progress.Version == 0 {
		progress.Version = 1
	}
	if progress.Items == nil {
		progress.Items = map[string]scheduler.Progress{}
	}
	return progress, nil
}

func (s JSONStore) Save(progress ProgressFile) error {
	progress.Version = 1
	progress.UpdatedAt = time.Now()

	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(progress, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(s.Path), ".progress-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.Path)
}
