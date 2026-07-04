# Code Muscle Memory

A terminal app for practicing programming syntax through active recall and spaced repetition.

Instead of reading another tutorial, `cmm` gives you one small coding challenge, opens your editor, runs hidden tests, and schedules the exercise for review based on how you did.

The first version focuses on Go.

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

## Features

- Terminal-first workflow
- External editor support through `$EDITOR`
- Hidden Go tests with `go test`
- Local JSON progress file
- SM-2-inspired spaced repetition scheduler
- Exercise pack stored as version-control-friendly JSON
- Clear internal package boundaries for future languages and schedulers

## Installation

### Prerequisites

- Go 1.22 or newer
- A terminal editor configured through `$EDITOR`

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
./cmm help
```

`try` opens a specific exercise for practice without affecting spaced repetition progress: nothing is scheduled, rated, or saved. Useful for previewing an exercise or re-drilling one outside the review queue.

By default, progress is saved to:

```text
~/.code-muscle-memory/progress.json
```

You can override paths:

```sh
./cmm -exercises exercises/go -progress /tmp/cmm-progress.json next
```

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
internal/executor    Go test execution
internal/scheduler   spaced repetition logic
internal/storage     local progress persistence
exercises/go         seed Go exercise pack
```

## Roadmap

- More Go exercises following Learn Go With Tests
- Better review controls after passing an exercise
- Exercise pack validation command
- Daily goal and review queue commands
- Additional scheduling algorithms
- More programming languages

## Status

Early MVP. The core loop works, but the exercise catalog is still small.
