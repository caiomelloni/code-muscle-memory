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

## JSON Exercise Notes

- Keep `id` stable once created.
- Use increasing `difficulty` to control rough learning order.
- `starter_code` may exist internally so the app can derive prose instructions, but it must not be copied into the user's editable file.
- `solution` is optional and should only be used for debugging or authoring checks.
