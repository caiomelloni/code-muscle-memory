---
name: plan-exercises
description: Scrape a URL (tutorial, docs page, blog post) and plan new exercise cards from it together with the user. Works for either deck - Go or the Linux command line. Proposes a card plan, iterates on the user's feedback (change, drop, add cards), then authors the exercises as JSON and verifies them with cmm validate. Use when the user shares a URL and wants exercises or cards created from it.
---

# Plan exercises from a URL

Turn a web page into new exercise cards. The user approves the plan before any card is written. The URL is the skill argument or appears in the user's message; if none was given, ask for one.

First settle which deck the page belongs to, since it decides the directory, the id prefix, and the shape of every card:

| Deck | Page teaches | Directory | Ids | Cards ask for |
| --- | --- | --- | --- | --- |
| `go` | Go | `exercises/go/` | `go-NNN-slug` | Go code, or a Go test |
| `shell` | the Linux command line | `exercises/shell/` | `sh-NNN-slug` | a command line |

Pick it from the page's subject and say which you picked; ask only if the page genuinely straddles both.

## 1. Scrape the page

Fetch the URL with WebFetch (load it via ToolSearch first). Prompt for: the concepts, idioms, commands, and APIs the page teaches, in teaching order, each with any concrete examples and edge cases the page mentions. If the fetch fails or returns boilerplate (paywall, JS-rendered), say so and ask the user to paste the relevant content — do not guess the page's content. Do not follow links to other pages unless the user asks for the whole series.

## 2. Survey the existing deck

Get every existing card's id, difficulty, and topic without reading the JSON files wholesale:

```sh
go run ./cmd/cmm list              # go deck
go run ./cmd/cmm -lang shell list  # shell deck
```

Note the highest number in the deck you are adding to (new ids continue from it, zero-padded) and which topics are already covered.

## 3. Propose a plan — do not write files yet

Present a compact plan as a markdown table, one row per proposed card:

| id | title | kind | topic | difficulty | concept (one line) |

Guidelines for the plan:
- One primary concept per card; order by difficulty continuing the deck's progression.
- Go deck: mix kinds where the page warrants it, `implementation` for syntax recall and `test_writing` when the page teaches testing. Shell deck cards are all `command`; leave `kind` out.
- Below the table, list page concepts you deliberately skipped and why (already covered by `go-NNN-…`, or not exercisable).

End by inviting edits: the user can change, drop, or add cards, adjust difficulty, or ask for more/fewer. Iterate on the table until they approve. Use AskUserQuestion only for a genuine fork (e.g. the page covers two distinct areas and scope is unclear); otherwise iterate conversationally.

## 4. Author the approved cards

Read `EXERCISE_AUTHORING_GUIDE.md` in full before writing any card. Its rules are binding: the instruction rules (no syntax or command names in descriptions, generic pseudocode examples, concrete input -> output pairs), the mutant rules for `test_writing` cards (single fault, must compile, no shared blind spot, plain-English hints), and the "Authoring Shell Exercises" section for `shell` cards (deterministic `setup`, a `check` that explains itself and accepts every reasonable answer, `required_commands` only when the tool is the point).

- Write all cards as one JSON array to a new file `exercises/<deck>/<page-slug>.json` (the loader picks up every `.json` in the directory; never edit `seed.json` for this).
- Every card except `test_writing` must include a `solution` — `cmm validate` requires it.
- Set `metadata.source` to the page title and URL.

## 5. Verify

```sh
go run ./cmd/cmm validate              # go deck
go run ./cmd/cmm -lang shell validate  # shell deck
```

Validation is per-deck, so run the one you added to. It covers schema validation (required fields, unique ids, mutant/hint counts) plus authoring checks: solutions pass their hidden tests, starter code does not, test-writing subjects and mutants compile, and shell setups run while their checks reject a do-nothing command. Fix and rerun until everything passes.

Two things it cannot check, so re-check them yourself against the guide: the blind-spot rule for mutants, and whether a shell check accepts reasonable answers other than the one in `solution`.

## 6. Report

Summarize the created cards (ids, topics, difficulties) and point the user at `./cmm describe <id>` and `./cmm try <id>` to preview them, with `-lang shell` when they belong to the shell deck.
