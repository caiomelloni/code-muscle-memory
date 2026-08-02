# Exercise Authoring Guide

Use this guide when adding or revising exercises.

There are two decks. Cards for the Go deck live in `exercises/go` with `"language": "go"`;
cards for the Linux command line deck live in `exercises/shell` with `"language": "shell"`.
Everything down to "Authoring Shell Exercises" is about the Go deck, except the JSON notes at
the end, which apply to both.

## Good Exercise Shape

- One primary Go concept per exercise.
- Solvable in a few minutes.
- Clear behavior, small API, hidden tests.
- Gradual progression from beginner syntax to advanced language features.
- Prefer practical, idiomatic Go over puzzle-style problems.

## Instruction Rules

- Do not provide starter code to the user.
- Do not show Go syntax for the required solution.
- Describe the required API in prose.
- State what the return value contains or represents, not just its type. Include at least one concrete example (input -> output), and cover any edge case the hidden tests check.
- Write examples using generic `Name(args) -> result` pseudocode, not Go syntax. Use bracket lists (`[1, 2, 3]`) for collections and `field: value` pairs for constructing values (`Rectangle(width: 12, height: 6)`). Do not use Go-specific shapes like `:=`, Go struct literals (`Rectangle{Width: 12}`), or Go type names in the example — the goal is to show the input/output relationship without leaking the exact Go syntax the user is meant to recall.

Good:

```text
Implement a function called Add that receives two integer parameters and returns their sum.
```

Bad:

```text
Implement: func Add(a, b int) int
```

Good:

```text
Implement a struct type called Rectangle with width and height fields, and a method called Area that returns the rectangle's area.
```

Bad:

```text
type Rectangle struct { ... }
func (r Rectangle) Area() float64
```

Good (concrete return example):

```text
Implement Hello so it returns a greeting built from the provided name: Hello("Chris") -> "Hello, Chris". If the name is empty, greet World instead: Hello("") -> "Hello, World".
```

Bad (return described only by type):

```text
Implement Hello so it returns a string.
```

Bad (leaks Go syntax instead of generic pseudocode):

```text
Implement Hello so that Hello("Chris") returns "Hello, " + name.
```

## Test Rules

- Hidden tests should verify behavior, not exact implementation.
- Include at least one normal case and one useful edge case when appropriate.
- Keep early tests simple; avoid combining multiple new concepts in one exercise.
- Compiler errors are acceptable feedback. They help users recall the required syntax.

## Authoring Test-Writing Exercises

Most exercises have the user write the implementation against a hidden test. A test-writing exercise flips this: the implementation is already correct and hidden, and the user's job is to write the test. Set `"kind": "test_writing"` for these.

- Omit `"tests"` — it is unused for this kind. Grading instead uses `subject_code` and `mutants`.
- `subject_code` must be a correct, compilable reference implementation of the thing being tested.
- Each entry in `mutants` must be a single-fault variant of `subject_code`: change exactly one behavior, and keep it compiling. Mutants are graded by running the user's test against them and expecting a test *failure*, not a compile error — a mutant that fails to compile never gets exercised.
- Provide `mutant_hints`, the same length and order as `mutants`: one plain-English sentence per mutant describing the missed behavior (e.g. "misreports zero as positive"). Never describe a mutant with Go code or a diff — this sentence is shown to the user when their test lets that mutant slip through.
- Keep early test-writing exercises catchable by one obvious case (no table required). Reserve mutants that are only distinguishable from the correct implementation by a specific edge-case input for later, harder exercises — this is what naturally pushes the user toward table-driven tests and subtests, without ever telling them to use one.
- Use `required_test_features` when the exercise's stated objective *is* the structure of the test, not just its coverage. It lists structural features the user's test must exhibit; grading checks them after the test passes the correct implementation and before mutants run. Valid values: `"table"` (cases driven from a data table by a loop; detected heuristically as a range loop inside a test function) and `"subtests"` (each case runs as a named subtest via the testing.T `Run` method). The prose description must still state the requirement — the check enforces it, the description teaches it. Do not add features an exercise's description never asks for.
- Check that no single input value is a "blind spot" shared by every mutant. A mutant is blind at any input where it happens to agree with `subject_code`; if all mutants share one, a test that only checks that one input passes every mutant and gets Success without exercising anything else. Concretely: for every input, at least one mutant must disagree with `subject_code` there. When one mutant is only wrong at a single edge case, make sure the *other* mutants are wrong on ordinary inputs (and not also coincidentally correct at that same edge case), so testing only the edge case still gets caught.

## Authoring Shell Exercises

A shell exercise asks for a command line instead of code. Set `"language": "shell"`, use ids
of the form `sh-NNN-slug`, and put the card in `exercises/shell`. These cards have their own
fields; the Go ones (`starter_code`, `tests`, `subject_code`, `mutants`, `required_test_features`)
are rejected.

- `setup` seeds the directory the user's command runs in. Optional: leave it out for an
  exercise that starts from an empty directory.
- `check` is the hidden script that decides whether the command did what was asked. Required.
- `solution` is a reference command line. Required in practice, because `cmm validate` uses it.
- `required_commands` names utilities the answer has to invoke. Optional; see below.

### How Grading Works

Each run gets a fresh throwaway directory. `setup` runs in it, then the user's command line,
then `check` — all three in that same directory, all three with `bash -c`. The command runs
with empty input, `HOME` pointing inside the sandbox, `LC_ALL=C` so sorting and collation are
the same everywhere, and a fifteen second limit. Nothing outside the sandbox is touched.

`check` sees the run through four environment variables:

| Variable      | Holds                                                       |
| ------------- | ----------------------------------------------------------- |
| `CMM_STDOUT`  | path to a file with everything the command printed           |
| `CMM_STDERR`  | path to a file with everything it reported                   |
| `CMM_EXIT`    | its exit status                                              |
| `CMM_COMMAND` | path to a file with the command line, instruction header stripped |

The check passes when it exits zero. `set -e` is prepended to both `setup` and `check`, so a
list of assertions stops at the first one that fails.

The command's own exit status decides nothing — a `grep` that matches nothing reports failure
and can still be the right answer, and a wrong command can succeed. Read `CMM_EXIT` in the
check when the status is genuinely part of what is being taught.

### Check Rules

- Say why it failed. The check's output is shown to the user, so write it as feedback:
  `echo "Expected the last three lines of the file, in file order."; exit 1`. A check that
  fails silently gives the user a generic message and nothing to work with.
- Accept every reasonable answer, not just the one in `solution`. Compare what matters and
  normalise the rest: trim whitespace before comparing a number (`wc -l` pads differently on
  macOS and Linux), sort before comparing an unordered list, strip a leading `./` from paths
  `find` prints.
- Check the effect, not the spelling of the command. Assert on the directory and on the
  output; do not grep `CMM_COMMAND` for the answer. The one fair use of `CMM_COMMAND` is a
  structural requirement the exercise states out loud, such as a card about pipes checking
  that a pipe was used.
- Never let a do-nothing command pass. `cmm validate` runs `:` against every check and fails
  the card if it succeeds, the same rule as hidden tests a blank stub already passes.

### Setup Rules

- Keep it deterministic. It runs twice — once to show the user the starting directory, once to
  grade — and the two have to match. No timestamps, no random names, nothing machine-specific.
- Keep it small. The user is shown every entry in the directory and the contents of every
  small text file in it, so a file large enough not to be shown is a file they are answering
  blind about.
- Stay portable. macOS and Linux disagree about `sed -i`, `readlink -f`, and `stat` flags.

### Instruction Rules

The recall rule carries over: describe what has to end up true, never the command that does it.

Good:

```text
Print how many lines access.log has. Print the number on its own: no file name beside it, no other text.
```

Bad:

```text
Run wc -l < access.log
```

- No command names, flags, or shell operators in the description. Say "add a line to the end
  of the file without losing what is there", not "append with `>>`".
- Say precisely what the output must look like, since the check will hold the user to it:
  which lines, in what order, and whether anything else may be printed.
- Where a command has to be told to do something it refuses to do by default — descend into a
  directory, create missing parents — say so as a requirement, not as a flag: "copying a
  directory rather than a plain file has to be asked for explicitly, or the command refuses".
- The user cannot look around before answering, so the directory they get is shown to them
  automatically. Do not re-list it in the description; describe only what matters about it.

### Required Commands

`required_commands` makes a card fail with WRONG TOOL when the answer gets the right result
without the utility being practised. It is checked only after the check has already passed.

Use it when the utility is the point of the card and other utilities would otherwise pass —
a card about `find` that `ls` in a loop would satisfy. Do not use it to force one particular
solution among equals: a card about counting lines should accept `wc -l` or `grep -c`, and
lines up as an exercise about the goal, not the tool.

Naming a required command puts "Use: the find command" in the instructions. That is deliberate
— the card is about that tool — but it does give the tool away, so it is the wrong choice for
any card where recalling *which* command to reach for is the exercise.

## JSON Exercise Notes

- Keep `id` stable once created.
- Use increasing `difficulty` to control rough learning order.
- `starter_code` may exist internally so the app can derive prose instructions, but it must not be copied into the user's editable file. Test-writing exercises (`kind: "test_writing"`) use `subject_code` the same way, describing what already exists rather than what to implement.
- `solution` is optional and should only be used for debugging or authoring checks. Shell exercises are the exception: their `solution` is what `cmm validate` checks the card against, so every one needs it.
- `kind` can be left out. It defaults to `implementation` for Go exercises and `command` for shell ones.
- Run `cmm validate` on the deck you touched — it is per-deck, so `./cmm validate` and `./cmm -lang shell validate` are separate runs.
