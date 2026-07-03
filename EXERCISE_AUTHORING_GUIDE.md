# Exercise Authoring Guide

Use this guide when adding or revising exercises.

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
- Check that no single input value is a "blind spot" shared by every mutant. A mutant is blind at any input where it happens to agree with `subject_code`; if all mutants share one, a test that only checks that one input passes every mutant and gets Success without exercising anything else. Concretely: for every input, at least one mutant must disagree with `subject_code` there. When one mutant is only wrong at a single edge case, make sure the *other* mutants are wrong on ordinary inputs (and not also coincidentally correct at that same edge case), so testing only the edge case still gets caught.

## JSON Exercise Notes

- Keep `id` stable once created.
- Use increasing `difficulty` to control rough learning order.
- `starter_code` may exist internally so the app can derive prose instructions, but it must not be copied into the user's editable file. Test-writing exercises (`kind: "test_writing"`) use `subject_code` the same way, describing what already exists rather than what to implement.
- `solution` is optional and should only be used for debugging or authoring checks.
