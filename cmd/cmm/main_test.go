package main

import (
	"bytes"
	"strings"
	"testing"
)

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
