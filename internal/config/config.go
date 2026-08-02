// Package config holds the user's settings, as opposed to their progress:
// choices they made on purpose and expect to stay made, which must survive
// throwing the progress file away.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	// DefaultDeck is the deck commands use when none is named on the command
	// line. Empty means the built-in default.
	DefaultDeck string `json:"default_deck,omitempty"`
}

type Store interface {
	Load() (Config, error)
	Save(cfg Config) error
}

type JSONStore struct {
	Path string
}

func NewJSONStore(path string) JSONStore {
	return JSONStore{Path: path}
}

// Load reads the settings, treating a missing file as "nothing configured
// yet" rather than an error: running cmm for the first time is not a problem
// to report.
func (s JSONStore) Load() (Config, error) {
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file %s: %w", s.Path, err)
	}
	return cfg, nil
}

func (s JSONStore) Save(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(s.Path), ".config-*.json")
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
