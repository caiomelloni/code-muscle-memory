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
