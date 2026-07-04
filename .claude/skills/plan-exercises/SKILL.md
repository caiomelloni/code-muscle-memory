---
name: plan-exercises
description: Scrape a URL (tutorial, docs page, blog post) and plan new exercise cards from it together with the user. Proposes a card plan, iterates on the user's feedback (change, drop, add cards), then authors the exercises as JSON and verifies them with cmm validate. Use when the user shares a URL and wants exercises or cards created from it.
---

# Plan exercises from a URL

Turn a web page into new exercise cards under `exercises/go/`. The user approves the plan before any card is written. The URL is the skill argument or appears in the user's message; if none was given, ask for one.

## 1. Scrape the page

Fetch the URL with WebFetch (load it via ToolSearch first). Prompt for: the Go concepts/idioms/APIs the page teaches, in teaching order, each with any concrete examples and edge cases the page mentions. If the fetch fails or returns boilerplate (paywall, JS-rendered), say so and ask the user to paste the relevant content — do not guess the page's content. Do not follow links to other pages unless the user asks for the whole series.

## 2. Survey the existing deck

Get every existing card's id, difficulty, and topic without reading the JSON files wholesale:

```sh
go run ./cmd/cmm list
```

Note the highest `go-NNN` number (new ids continue from it, zero-padded: `go-NNN-slug`) and which topics are already covered.

## 3. Propose a plan — do not write files yet

Present a compact plan as a markdown table, one row per proposed card:

| id | title | kind | topic | difficulty | concept (one line) |

Guidelines for the plan:
- One primary concept per card; order by difficulty continuing the deck's progression.
- Mix kinds where the page warrants it: `implementation` for syntax recall, `test_writing` when the page teaches testing.
- Below the table, list page concepts you deliberately skipped and why (already covered by `go-NNN-…`, or not exercisable).

End by inviting edits: the user can change, drop, or add cards, adjust difficulty, or ask for more/fewer. Iterate on the table until they approve. Use AskUserQuestion only for a genuine fork (e.g. the page covers two distinct areas and scope is unclear); otherwise iterate conversationally.

## 4. Author the approved cards

Read `EXERCISE_AUTHORING_GUIDE.md` in full before writing any card — its instruction rules (no Go syntax in descriptions, generic pseudocode examples, concrete input -> output pairs) and mutant rules (single fault, must compile, no shared blind spot, plain-English hints) are binding.

- Write all cards as one JSON array to a new file `exercises/go/<page-slug>.json` (the loader picks up every `.json` in the directory; never edit `seed.json` for this).
- Every implementation card must include a `solution` — `cmm validate` requires it.
- Set `metadata.source` to the page title and URL.

## 5. Verify

```sh
go run ./cmd/cmm validate
```

This runs schema validation (required fields, unique ids, mutant/hint counts) plus authoring checks: solutions pass their hidden tests, starter code does not, and test-writing subjects and mutants compile. Fix and rerun until everything passes. One thing it cannot check: the blind-spot rule for mutants — re-check that yourself against the guide's instructions.

## 6. Report

Summarize the created cards (ids, topics, difficulties) and point the user at `./cmm describe <id>` and `./cmm try <id>` to preview them.
