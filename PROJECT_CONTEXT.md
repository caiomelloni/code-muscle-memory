# Project Context

`code-muscle-memory` is a terminal app for practicing programming language syntax through active recall and spaced repetition. It has two decks: Go, and the Linux command line.

## Product Intent

- Users should practice writing code from memory, not read tutorials or fill in blanks.
- `./cmm next` opens one exercise in the user's editor, grades it, reports feedback, and updates review progress.
- Exercises are short and focused. Each exercise should teach one concept when possible.
- The Go curriculum should roughly follow Learn Go With Tests, then expand with idiomatic Go exercises.
- The shell curriculum should follow what someone actually does at a terminal, in rough order of how early they meet it.

## Current Design Decisions

- No starter code is shown to the user.
- The editable solution file may include high-level comments and `package exercise`.
- Instructions must not reveal programming language syntax the user should recall.
- Hidden tests define the required API and correctness.
- Exercise files are JSON under `exercises/<language>`.
- Decks are practised separately and share one progress file. The deck in use comes from the saved default (`cmm deck <name>`), which `-lang` overrides for a single run. Each deck keeps its own in-progress exercise, so switching decks does not disturb a half-answered card in the other.
- Shell exercises are graded by running the user's command line in a throwaway sandbox directory seeded by the exercise's `setup`, then handing the result to its hidden `check` script. The command's own exit status decides nothing; the check does.
- Adding a language means adding an executor behind `executor.Executor` and, if its exercises start from a prepared environment, `executor.Previewer`. Nothing else should need to know the language.
- User progress is stored as local JSON behind a storage interface. Settings the user chose on purpose live in a separate config file behind their own interface, so wiping progress does not reset them.
- Scheduling is isolated behind an interface so the algorithm can be replaced later.
- Practice runs via `cmm try` never touch stored progress; scheduling is only updated by rated reviews in `cmm next`.

## Assistant Guidance

- Prefer small, incremental changes with tests.
- Keep the CLI simple before adding a full TUI.
- Keep package boundaries clean: app orchestration, exercise loading, scheduling, storage, and execution should stay separate.
- When changing exercise generation or instructions, preserve the active recall goal: avoid syntax hints and prefilled code. For shell exercises this means describing what has to end up true, never the command that does it.
