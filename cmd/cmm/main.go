package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"code-muscle-memory/internal/app"
	"code-muscle-memory/internal/exercise"
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
	language := fs.String("lang", "", "deck to practise for this run, overriding the default deck")
	exerciseDir := fs.String("exercises", "", "directory containing exercise JSON files (default \"exercises/<lang>\")")
	progressPath := fs.String("progress", filepath.Join(home, ".code-muscle-memory", "progress.json"), "progress JSON file")
	configPath := fs.String("config", filepath.Join(home, ".code-muscle-memory", "config.json"), "settings JSON file")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	// An explicit -lang wins for this run only; otherwise the deck the user
	// settled on with "cmm deck" is used, so it needs saying just once.
	if *language == "" {
		if *language, err = app.DefaultDeck(*configPath); err != nil {
			return err
		}
	} else if !exercise.SupportsLanguage(*language) {
		return fmt.Errorf("unknown deck %q: use %s", *language, strings.Join(exercise.Languages(), " or "))
	}
	if *exerciseDir == "" {
		*exerciseDir = filepath.Join("exercises", *language)
	}

	command := "next"
	if fs.NArg() > 0 {
		command = fs.Arg(0)
	}

	a := app.New(app.Config{
		ExerciseDir:  *exerciseDir,
		ProgressPath: *progressPath,
		ConfigPath:   *configPath,
		Language:     *language,
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
	case "describe":
		args := fs.Args()[1:]
		if len(args) == 0 {
			return fmt.Errorf("usage: cmm describe <exercise-id>")
		}
		return a.Describe(ctx, args[0])
	case "try":
		args := fs.Args()[1:]
		if len(args) == 0 {
			return fmt.Errorf("usage: cmm try <exercise-id>")
		}
		return a.Try(ctx, args[0])
	case "delete":
		args := fs.Args()[1:]
		if len(args) == 0 {
			return fmt.Errorf("usage: cmm delete <exercise-id>")
		}
		return a.Delete(ctx, args[0])
	case "reset":
		args := fs.Args()[1:]
		id := ""
		if len(args) > 0 {
			id = args[0]
		}
		return a.Reset(ctx, id)
	case "validate":
		return a.Validate(ctx)
	case "deck":
		args := fs.Args()[1:]
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		return a.Deck(ctx, name)
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
  cmm -lang shell next

Open the next due exercise in $EDITOR, grade it, and update spaced repetition progress.
A Go exercise is graded by hidden tests; a shell exercise is graded by running the command
line you wrote in a seeded working directory and handing the result to a hidden check.

Exercises are served Anki-style: due learning cards first, then due reviews, then new exercises.
After a passing run you rate the review as again, hard, good, or easy; the rating decides when
the exercise comes back. Failed runs are saved so you can retry, and never affect scheduling.

Environment:
  EDITOR   editor command used to edit the temporary solution file; defaults to vi`)
	case "list":
		fmt.Fprintln(w, `Usage:
  cmm list

List all loaded exercises with their review status, difficulty, and topic.`)
	case "stats":
		fmt.Fprintln(w, `Usage:
  cmm stats

Print a summary of exercises by review state: new, in learning, and due now, followed by the
id of the exercise "cmm next" would serve and its review status.`)
	case "describe":
		fmt.Fprintln(w, `Usage:
  cmm describe <exercise-id>

Print an exercise's full instructions and review status without opening $EDITOR. Use "cmm list" to find an exercise id.`)
	case "try":
		fmt.Fprintln(w, `Usage:
  cmm try <exercise-id>

Practice a specific exercise: open it in $EDITOR and run the hidden Go tests, without affecting
review progress. Nothing is saved: no scheduling, no rating prompt, no attempt restore. Use
"cmm list" to find an exercise id.

Environment:
  EDITOR   editor command used to edit the temporary solution file; defaults to vi`)
	case "delete":
		fmt.Fprintln(w, `Usage:
  cmm delete <exercise-id>

Remove an exercise from the deck. Its definition is dropped from the JSON file that holds it,
and that file is deleted if the exercise was the last one in it. Any review history, saved
attempt, and in-progress marker for the exercise are cleared from the progress file too.

You are asked to confirm first; the deletion cannot be undone. Use "cmm list" to find an
exercise id.`)
	case "reset":
		fmt.Fprintln(w, `Usage:
  cmm reset [exercise-id]

Discard a saved attempt so the exercise reopens from the original instructions instead of your
last submission. Useful when a previous attempt went down the wrong path and you want a clean
slate.

With no id it resets the exercise currently in progress (the one "cmm next" is serving); pass
an id to reset a specific exercise instead. Only the in-progress draft is cleared: review
history and scheduling are left untouched. Use "cmm list" to find an exercise id.`)
	case "validate":
		fmt.Fprintln(w, `Usage:
  cmm validate

Run authoring checks on every exercise in the deck, beyond the schema validation all commands
perform: each implementation exercise's solution must pass its hidden tests and its starter
code must not, each test-writing exercise's subject and mutants must compile, and each shell
exercise's setup must run, its solution must satisfy its check, and a do-nothing command must
not. Use it after adding or revising exercises.`)
	case "deck":
		fmt.Fprintln(w, `Usage:
  cmm deck
  cmm deck <name>

With no name, print the deck commands use by default and the decks available. With a name,
make that deck the default, so it no longer has to be named on every run. The setting is
saved and applies to every later command.

Pass -lang to work in the other deck for one run without changing the default:

  cmm deck shell     make shell the default from now on
  cmm next           serves a shell exercise
  cmm -lang go next  serves a Go exercise, just this once`)
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
  next             open the next due exercise, run tests, and update progress
  list             list exercises and review status
  stats            print progress summary
  describe <id>    print an exercise's instructions and review status
  try <id>         practice an exercise without affecting review progress
  delete <id>      remove an exercise from the deck and from stored progress
  reset [id]       discard a saved attempt (defaults to the current exercise)
  validate         run authoring checks on every exercise
  deck [name]      show the default deck, or make one the default
  help             show this help

Decks:
  go       write Go from memory against hidden tests (default)
  shell    write command lines against a seeded directory and a hidden check

Pick the deck you are working through once with "cmm deck shell"; every later command uses
it. Pass -lang to switch deck for a single run without changing that. The two decks are
practised separately, but share one progress file, so each keeps its own review history
and its own in-progress exercise.

Flags:
  -lang string        deck to use for this run (default: the deck set by "cmm deck")
  -exercises string   directory containing exercise JSON files (default "exercises/<lang>")
  -progress string    progress JSON file (default "$HOME/.code-muscle-memory/progress.json")
  -config string      settings JSON file (default "$HOME/.code-muscle-memory/config.json")`)
}
