# Project Context

`code-muscle-memory` is a terminal app for practicing programming language syntax through active recall and spaced repetition. The v1 app supports Go only.

## Product Intent

- Users should practice writing code from memory, not read tutorials or fill in blanks.
- `./cmm next` opens one exercise in the user's editor, runs hidden Go tests, reports feedback, and updates review progress.
- Exercises are short and focused. Each exercise should teach one concept when possible.
- The curriculum should roughly follow Learn Go With Tests, then expand with idiomatic Go exercises.

## Current Design Decisions

- No starter code is shown to the user.
- The editable solution file may include high-level comments and `package exercise`.
- Instructions must not reveal programming language syntax the user should recall.
- Hidden tests define the required API and correctness.
- Exercise files are JSON under `exercises/go`.
- User progress is stored as local JSON behind a storage interface.
- Scheduling is isolated behind an interface so the algorithm can be replaced later.
- Practice runs via `cmm try` never touch stored progress; scheduling is only updated by rated reviews in `cmm next`.

## Assistant Guidance

- Prefer small, incremental changes with tests.
- Keep the CLI simple before adding a full TUI.
- Keep package boundaries clean: app orchestration, exercise loading, scheduling, storage, and execution should stay separate.
- When changing exercise generation or instructions, preserve the active recall goal: avoid syntax hints and prefilled code.
