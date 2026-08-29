---
name: resume-http-learning
description: Resume the user's Go net/http learning journey - a long-running, module-by-module curriculum for mastering the net/http package, taught in chat and drilled with cards in the Go deck. Reads where the last session stopped and continues from there. Use when the user wants to continue the http curriculum, learn the next net/http topic, or asks to pick up their net/http learning.
---

# Resume the net/http learning journey

A twelve-module curriculum taking the user from "can write a handler" to advanced `net/http`. It
runs as a loop: teach a module in chat, the user studies it, and only once they say the chapter is
finished do two practice cards get written into the Go deck.

Everything this needs lives in this directory:

| File | Holds | Changes |
| --- | --- | --- |
| `PROGRESS.md` | where we stopped: the Next-up line and the tracker table | every session |
| `curriculum.md` | all twelve modules — what to teach, and the two card specs each | only if the user reshapes the curriculum |

## 1. Work out where we are

1. Read `PROGRESS.md`. It is the source of truth; this skill keeps no state of its own.
2. Read only the current module's section of `curriculum.md` — the file is long.
3. Confirm the deck matches: `go run ./cmd/cmm list`.

If the user passed an argument (`module 7`, `re-teach`, `feedback`), it overrides the resume point.

## 2. Branch on the first empty checkbox

Find the first module row that is not fully ticked, then do **only** the branch its first empty box
names. Each branch ends the session's work on this module.

### Taught is empty → teach it

Explain the module in chat from its concept list in `curriculum.md`: the API, the mental model, and
the traps that separate intermediate from advanced. Answer questions until the user is satisfied.

Tick **Taught**. **Stop.** Do not write cards in the same breath — the cards come after the user has
worked with the material, not after hearing about it.

### Chapter done is empty → wait for the gate

Write nothing. Ask whether they have finished the chapter.

- Yes → tick **Chapter done** and fall through to the next branch in the same session.
- Not yet → answer questions, re-explain, work through examples. Stop there.

This is the one real gate. A module whose Chapter-done box is empty must never get cards.

### Carded is empty → author the two cards

1. Re-read `EXERCISE_AUTHORING_GUIDE.md` in full first. Its rules are binding.
2. Ask what they actually struggled with while studying, and let that shape the two cards. The specs
   in `curriculum.md` are the plan, not a straitjacket — a card that targets a real gap beats one
   that targets the plan.
3. Write both cards as a JSON array to the module's file, `exercises/go/<file from curriculum.md>`.
   Never edit `seed.json`. Never write more than two cards for a module.
4. Verify with `go run ./cmd/cmm validate`, then hand-check the two things it cannot: the mutant
   blind-spot rule, and that no identifier the card is drilling is spelled out in its prose (see
   "Authoring rules" in `curriculum.md`). `go run ./cmd/cmm describe <id>` shows the instructions as
   the user will see them.
5. Report the ids and one line each on what they drill.

Tick **Carded**. **Stop there** — the cards are the user's to practise whenever they like.

## 3. Before starting a new module

Ask once whether they have feedback on the previous module's cards. If they do, revise and tick
**Tuned**. If they don't, or they would rather keep going, proceed — Tuned is a record, not a gate,
and often stays empty.

Revising a card, `id` must stay stable — it is the key the SRS schedule hangs off, so renaming one
silently orphans its review history:

- **Reworded or retuned** → edit the JSON in place, keep the id, re-run `validate`. If they have a
  half-finished draft saved against it, tell them `./cmm reset <id>` discards the stale attempt so
  the new instructions are what they see.
- **Replaced by a genuinely different exercise** → `./cmm delete <id>`, then author the replacement
  under the next free id. Do not recycle the retired number; `go-016` and `go-025` are precedents.

## 4. Always update PROGRESS.md

Before finishing, update the table and rewrite the Next-up line to name the concrete next action —
not just a module number. Do this even if the session accomplished nothing else, so the next
invocation resumes correctly.

## Guardrails

- Never work more than one module ahead.
- Never author cards for a module whose Chapter-done box is empty.
- Never author more than two cards for a module.
- **Never run `cmm next` or `cmm try`.** Both open the user's editor, and `next` writes SRS
  progress. Drilling is the user's alone, on their own schedule — do not run it, prompt for it, or
  gate anything on it. The only commands this skill runs are `cmm validate`, `cmm list`, and
  `cmm describe`.
- No helper scripts. If a new check is ever needed, it belongs in the `cmm` CLI.
