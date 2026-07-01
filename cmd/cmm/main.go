package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"code-muscle-memory/internal/app"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	fs := flag.NewFlagSet("cmm", flag.ExitOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		printUsage(os.Stdout)
	}
	exerciseDir := fs.String("exercises", "exercises/go", "directory containing exercise JSON files")
	progressPath := fs.String("progress", filepath.Join(home, ".code-muscle-memory", "progress.json"), "progress JSON file")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	command := "next"
	if fs.NArg() > 0 {
		command = fs.Arg(0)
	}

	a := app.New(app.Config{
		ExerciseDir:  *exerciseDir,
		ProgressPath: *progressPath,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
	})

	ctx := context.Background()
	switch command {
	case "next":
		return a.Next(ctx)
	case "list":
		return a.List(ctx)
	case "stats":
		return a.Stats(ctx)
	case "help", "-h", "--help":
		return printHelp(os.Stdout, fs.Args()[1:])
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func printHelp(w io.Writer, args []string) error {
	if len(args) == 0 {
		printUsage(w)
		return nil
	}

	switch args[0] {
	case "next":
		fmt.Fprintln(w, `Usage:
  cmm next

Open the next due exercise in $EDITOR, run the hidden Go tests, and update spaced repetition progress.

Environment:
  EDITOR   editor command used to edit the temporary solution file; defaults to vi`)
	case "list":
		fmt.Fprintln(w, `Usage:
  cmm list

List all loaded exercises with their review status, difficulty, and topic.`)
	case "stats":
		fmt.Fprintln(w, `Usage:
  cmm stats

Print a summary of total exercises, reviewed exercises, and exercises currently due.`)
	case "help":
		fmt.Fprintln(w, `Usage:
  cmm help [command]

Show general help or details for a specific command.`)
	default:
		return fmt.Errorf("unknown help topic %q", args[0])
	}
	return nil
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, `cmm - code muscle memory

Usage:
  cmm [flags] [command]
  cmm help [command]

Commands:
  next    open the next due exercise, run tests, and update progress
  list    list exercises and review status
  stats   print progress summary
  help    show this help

Flags:
  -exercises string   directory containing exercise JSON files (default "exercises/go")
  -progress string    progress JSON file (default "$HOME/.code-muscle-memory/progress.json")`)
}
