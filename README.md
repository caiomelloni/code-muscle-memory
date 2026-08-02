# Code Muscle Memory

A terminal app for practicing programming syntax through active recall and spaced repetition.

Instead of reading another tutorial, `cmm` gives you one small challenge, opens your editor, grades it, and schedules the exercise for review based on how you did.

There are two decks. The Go deck has you write Go from memory against hidden tests. The shell deck has you write Linux command lines against a seeded directory and a hidden check.

## Why

Most programming practice tools give you too much scaffolding. That feels productive, but it does not always build recall.

`code-muscle-memory` is built around a stricter idea:

- no starter code
- no syntax-shaped hints
- small focused exercises
- immediate test feedback
- spaced repetition for review

The goal is to practice remembering the shape of code, not just recognizing it.

## What It Looks Like

Run the next due exercise:

```sh
./cmm next
```

The app opens a temporary solution file in `$EDITOR`:

```go
// Return a greeting
//
// What to do: Implement Hello so it returns a friendly greeting for the provided name. If the name is empty, greet World.
// Objective: Practice defining a function, using strings, and returning values.
//
// Implement: a function called "Hello", receives a parameter called "name" of type string, and returns a string

package exercise
```

You write the code from memory. Then `cmm` runs hidden Go tests and reports whether the result was a compile error, test failure, or pass.

The shell deck works the same way, with `./cmm -lang shell next`:

```sh
# Hunting for files anywhere below you
#
# What to do: Somewhere under the current directory there are files whose names end in
# .log, at different depths. Print the path of every one of them, however deep it is buried.
# Objective: Practise searching a whole directory tree for files by name pattern.
#
# Write: the command line that does this
# Use: the find command
#
# The working directory contains:
#     boot.log
#     services/api/logs/api.log
#     var/log/system.log
```

You write the command line. `cmm` runs it in a throwaway copy of that directory and hands the result to a hidden check, which decides whether it did what was asked. The command's own exit status does not decide anything — a `grep` that matches nothing reports failure and may still be the right answer.

## Features

- Terminal-first workflow
- Separate decks for Go and the Linux command line, with a remembered default
- External editor support through `$EDITOR`
- Hidden Go tests with `go test`
- Sandboxed command lines graded by hidden check scripts
- Local JSON progress file
- SM-2-inspired spaced repetition scheduler
- Exercise pack stored as version-control-friendly JSON
- Clear internal package boundaries for future languages and schedulers

## Installation

### Prerequisites

- Go 1.22 or newer
- A terminal editor configured through `$EDITOR`
- `bash`, for the shell deck

If `$EDITOR` is not set, `cmm next` falls back to `vi`.

### Compile From Source

Clone the repository:

```sh
git clone <repository-url>
cd code-muscle-memory
```

Download module dependencies:

```sh
go mod download
```

Build the CLI:

```sh
go build -o ./cmm ./cmd/cmm
```

This creates a local `./cmm` binary in the project root. You can verify it with:

```sh
./cmm help
```

### Install On Your PATH

To install the command into your Go binary directory:

```sh
go install ./cmd/cmm
```

Make sure the Go binary directory is on your `PATH`. It is usually:

```text
$(go env GOPATH)/bin
```

After that, you can run:

```sh
cmm help
```

## Usage

```sh
./cmm next
./cmm list
./cmm stats
./cmm describe <exercise-id>
./cmm try <exercise-id>
./cmm validate
./cmm help
```

Pick the deck you are working through once, and every later command uses it:

```sh
./cmm deck shell   # remembered from now on
./cmm next         # serves a shell exercise
./cmm list
./cmm describe sh-009-find-by-name
```

`./cmm deck` with no name prints the current default and the decks available. To work in the
other deck for a single run without changing the default, pass `-lang`:

```sh
./cmm -lang go next
```

The decks are practised separately: each serves only its own exercises and keeps its own
in-progress card, so switching to the shell deck for an evening does not disturb a Go card
you left half answered. They share one progress file, so review history survives switching.

`try` opens a specific exercise for practice without affecting spaced repetition progress: nothing is scheduled, rated, or saved. Useful for previewing an exercise or re-drilling one outside the review queue.

`validate` runs authoring checks on every exercise in the deck: each implementation exercise's solution must pass its hidden tests and its starter code must not, each test-writing exercise's subject and mutants must compile, and each shell exercise's setup must run, its solution must satisfy its check, and a command that does nothing must not. Run it after adding or revising exercises.

By default, progress is saved to:

```text
~/.code-muscle-memory/progress.json
```

and settings, currently just the default deck, to:

```text
~/.code-muscle-memory/config.json
```

Settings are kept apart from progress on purpose: a choice you made deliberately should
survive throwing your progress away.

You can override paths:

```sh
./cmm -exercises exercises/go -progress /tmp/cmm-progress.json next
```

`-exercises` defaults to `exercises/<lang>`, and `-config` overrides the settings file.

## Exercise Philosophy

Exercises should train recall, not recognition.

Good exercise instructions describe the goal in prose:

```text
Implement a function called Add that receives two integer parameters and returns their sum.
```

They should not give Go syntax away:

```text
func Add(a, b int) int
```

The same holds for the shell deck. An exercise says what has to end up true, not which
command does it:

```text
Print how many lines access.log has. Print the number on its own.
```

```text
Run wc -l < access.log
```

See [EXERCISE_AUTHORING_GUIDE.md](EXERCISE_AUTHORING_GUIDE.md) for the authoring rules.

## Development

Run tests:

```sh
go test ./...
```

Project structure:

```text
cmd/cmm              CLI entrypoint
internal/app         workflow orchestration
internal/exercise    exercise loading and validation
internal/executor    grading: Go test runs and sandboxed shell runs
internal/scheduler   spaced repetition logic
internal/config      local settings persistence
internal/storage     local progress persistence
exercises/go         seed Go exercise pack
exercises/shell      seed Linux command line exercise pack
```

## Roadmap

- More Go exercises following Learn Go With Tests
- More command line exercises: processes, archives, permissions, text processing
- Better review controls after passing an exercise
- Exercise pack validation command
- Daily goal and review queue commands
- Additional scheduling algorithms
- More programming languages

## Status

Early MVP. The core loop works, but the exercise catalog is still small.
