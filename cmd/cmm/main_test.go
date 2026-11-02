package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultExerciseDirIsAbsoluteInMemoryDir(t *testing.T) {
	memoryDir := filepath.Join(t.TempDir(), ".code-muscle-memory")
	got := defaultExerciseDir(memoryDir, "shell")
	if want := filepath.Join(memoryDir, "exercises", "shell"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("%q depends on the current directory", got)
	}
}

func TestAskCommitMessage(t *testing.T) {
	tests := []struct{ input, want string }{
		{"fixed the cards\n", "fixed the cards"},
		{"  no newline  ", "no newline"},
		{"\n", ""},
		{"", ""},
	}
	for _, tt := range tests {
		var out bytes.Buffer
		got, err := askCommitMessage(strings.NewReader(tt.input), &out)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("askCommitMessage(%q) = %q, want %q", tt.input, got, tt.want)
		}
		if !strings.Contains(out.String(), "Commit message:") {
			t.Errorf("no prompt shown: %q", out.String())
		}
	}
}

func TestPrintHelpGeneral(t *testing.T) {
	var out bytes.Buffer
	if err := printHelp(&out, nil); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{"cmm - code muscle memory", "cmm help [command]", "next", "stats"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing %q:\n%s", want, got)
		}
	}
}

func TestPrintHelpCommand(t *testing.T) {
	var out bytes.Buffer
	if err := printHelp(&out, []string{"next"}); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{"Usage:", "cmm next", "$EDITOR"} {
		if !strings.Contains(got, want) {
			t.Fatalf("next help output missing %q:\n%s", want, got)
		}
	}
}

func TestPrintHelpDescribeCommand(t *testing.T) {
	var out bytes.Buffer
	if err := printHelp(&out, []string{"describe"}); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{"Usage:", "cmm describe <exercise-id>"} {
		if !strings.Contains(got, want) {
			t.Fatalf("describe help output missing %q:\n%s", want, got)
		}
	}
}

func TestPrintHelpTryCommand(t *testing.T) {
	var out bytes.Buffer
	if err := printHelp(&out, []string{"try"}); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{"Usage:", "cmm try <exercise-id>", "without affecting", "$EDITOR"} {
		if !strings.Contains(got, want) {
			t.Fatalf("try help output missing %q:\n%s", want, got)
		}
	}
}

func TestPrintHelpResetCommand(t *testing.T) {
	var out bytes.Buffer
	if err := printHelp(&out, []string{"reset"}); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{"Usage:", "cmm reset [exercise-id]", "original instructions"} {
		if !strings.Contains(got, want) {
			t.Fatalf("reset help output missing %q:\n%s", want, got)
		}
	}
}

func TestPrintHelpUnknownTopic(t *testing.T) {
	var out bytes.Buffer
	if err := printHelp(&out, []string{"missing"}); err == nil {
		t.Fatal("printHelp succeeded with unknown topic")
	}
}

func TestPrintHelpDocumentsTheDecks(t *testing.T) {
	var out bytes.Buffer
	if err := printHelp(&out, nil); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{"Decks:", "deck [name]", `cmm deck shell`, "-lang string", "-config string"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing %q:\n%s", want, got)
		}
	}
}

func TestPrintHelpDeckCommand(t *testing.T) {
	var out bytes.Buffer
	if err := printHelp(&out, []string{"deck"}); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{"cmm deck <name>", "cmm -lang go next"} {
		if !strings.Contains(got, want) {
			t.Fatalf("deck help output missing %q:\n%s", want, got)
		}
	}
}
