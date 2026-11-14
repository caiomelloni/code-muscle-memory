package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJSONStoreRoundTrip(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "config.json"))

	if err := store.Save(Config{DefaultDeck: "shell"}); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.DefaultDeck != "shell" {
		t.Fatalf("DefaultDeck = %q, want shell", got.DefaultDeck)
	}
}

// Nothing has been configured before the first run, which is not a problem to
// report to someone who has just installed the thing.
func TestJSONStoreTreatsAMissingFileAsUnconfigured(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "config.json"))

	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got != (Config{}) {
		t.Fatalf("Load() = %+v, want the zero config", got)
	}
}

func TestJSONStoreSaveCreatesTheDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")

	if err := NewJSONStore(path).Save(Config{DefaultDeck: "go"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestJSONStoreRejectsACorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := NewJSONStore(path).Load(); err == nil {
		t.Fatal("Load succeeded with corrupt JSON")
	}
}
