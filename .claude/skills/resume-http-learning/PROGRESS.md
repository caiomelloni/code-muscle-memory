# net/http learning progress

**Next up:** module 2 is complete — `go-036` and `go-037` are written and validated. Next session:
ask for feedback on those two cards first (tick **Tuned** if any is applied, skip if not), then teach
module 3, Reading the request, from `curriculum.md`. Do not write `go-038`/`go-039` until module 3's
Chapter-done box is ticked.

**Roadmap-wide:** every card is `implementation`. No `test_writing` cards, no mutants.

| # | Module | File | Ids | Taught | Chapter done | Carded | Tuned |
|---|---|---|---|---|---|---|---|
| 1 | Handlers & ResponseWriter | `http-handlers.json` | go-034 | [x] | [x] | [x] | [x] |
| 2 | ServeMux & routing | `http-routing.json` | go-036, 037 | [x] | [x] | [x] | [ ] |
| 3 | Reading the request | `http-request.json` | go-038, 039 | [ ] | [ ] | [ ] | [ ] |
| 4 | Request bodies & JSON APIs | `http-json-api.json` | go-040, 041 | [ ] | [ ] | [ ] | [ ] |
| 5 | Middleware | `http-middleware.json` | go-042, 043 | [ ] | [ ] | [ ] | [ ] |
| 6 | Handler dependencies | `http-handler-deps.json` | go-044, 045 | [ ] | [ ] | [ ] | [ ] |
| 7 | Context & cancellation | `http-context.json` | go-046, 047 | [ ] | [ ] | [ ] | [ ] |
| 8 | Streaming & ResponseController | `http-streaming.json` | go-048, 049 | [ ] | [ ] | [ ] | [ ] |
| 9 | Client, Transport, RoundTripper | `http-client.json` | go-050, 051 | [ ] | [ ] | [ ] | [ ] |
| 10 | Server config & lifecycle | `http-server-lifecycle.json` | go-052, 053 | [ ] | [ ] | [ ] | [ ] |
| 11 | File serving | `http-file-serving.json` | go-054, 055 | [ ] | [ ] | [ ] | [ ] |
| 12 | Reverse proxy (capstone) | `http-reverse-proxy.json` | go-056, 057 | [ ] | [ ] | [ ] | [ ] |

## What the columns mean

- **Taught** — the module has been explained in chat. Claude ticks it.
- **Chapter done** — the *user* has studied it and said so. **This is the gate: no cards get written
  for a module until this is ticked.**
- **Carded** — the module's two cards are written and `cmm validate` passes. Claude ticks it.
- **Tuned** — the user's feedback on the cards has been applied. Often stays empty forever; it is a
  record, not a gate.

There is deliberately no "drilled" column. Practice happens on the user's own schedule with
`./cmm next`, and `./cmm stats` already knows the answer.

Ids are *planned* until Carded is ticked. If feedback retires a card, record the replacement id and
leave the gap, matching what is actually in `exercises/go/`.

## Log

- **2026-08-03** — Roadmap set up. Prerequisite checked: the graded module's `go 1.22` directive
  gates language features only, so post-1.22 stdlib symbols compile; no change needed to
  `internal/executor/go.go`. Module 1 taught.
- **2026-08-03** — Module 1 actually delivered: `Handler`/`ServeHTTP`, the `HandlerFunc` adapter, the
  header → status → body ordering contract, the implicit 200, superfluous `WriteHeader`, sniffing vs
  `application/json`, content-length vs chunked, `http.Error`/`StatusText`, and the optional
  interfaces as a signpost to modules 5 and 8. Awaiting the Chapter-done gate.
- **2026-08-03** — Chapter done. User named all three of the ordering contract, the implicit 200 /
  double `WriteHeader`, and content-type & sniffing as the slippery parts, so both cards cover all
  three rather than splitting them. `go-034` and `go-035` written; `validate` passes on all 33.
  Blind spots checked empirically — every input has at least one disagreeing mutant. Note: the
  auto-generated `Implement:`/`Given:` line names `http.ResponseWriter` and `http.Request`, since it
  is derived from `starter_code`/`subject_code`; unavoidable for every handler-shaped card in this
  roadmap, so those two spellings are not recall targets from module 1 onward.
- **2026-08-03** — `go-034` restated to spell out how the request arrives (a GET carrying the name in
  the query string) and how the reply is taken (assembled on the writer, read back as a client sees
  it), after the user said the mechanics of reading a request and writing a reply were the gap. Same
  id. Supplement written to `~/go-http/net-http-module-01a-request-in-reply-out.md`.
- **2026-08-03** — `go-035` deleted at the user's request: **this roadmap drills implementation
  only, never test writing.** `curriculum.md` updated — all Kinds are now `impl`, the blind-spot
  rule is retired, and modules 2, 4, 6 and 12 are marked as needing a fresh implementation spec for
  their second card. `go-035` is a permanent id gap; do not recycle it. Module 1 keeps one card.
- **2026-08-04** — Module 2 taught: `Handle`/`HandleFunc`; the `[METHOD ][HOST]/[PATH]` grammar and
  the silent failure of a lowercase method; GET matching HEAD; subtree vs exact patterns; wildcard
  segments, `{path...}`, `{$}`; specificity-not-registration-order precedence and the registration
  panic on overlapping non-subset patterns; `PathValue` returning `""` for a typo'd name; decoded
  segments meaning a wildcard can contain a slash; `r.Pattern` / `mux.Handler(r)` as the bounded
  metrics label and why it is only visible inside the mux; the mux's own 301s for path cleaning and
  the trailing-slash subtree redirect; automatic 404 vs 405-with-`Allow` and why that needs
  method-qualified routes; `/` as an accidental catch-all vs `/{$}`; `DefaultServeMux` as global
  state and the `net/http/pprof` side-effect import; nesting with `StripPrefix`; the
  `httpmuxgo121=1` escape hatch. Awaiting the Chapter-done gate.
- **2026-08-04** — Module 2 written up to `~/go-http/net-http-module-02-servemux-routing.md`, with
  every claim verified by running it against go1.26.4. Three things taught in chat were wrong and
  are corrected in the file: the mux's redirects are **307, not 301** (so no method rewriting); a
  lowercase method like `"get /x"` parses as a *method*, not a host, and shows up as a 405 with a
  lowercase `Allow`; and registering a more specific third pattern does **not** rescue a conflicting
  pair — it panics in every registration order, and the only documented exception is
  host-qualified-beats-host-agnostic. Also found and documented: a bare `"/"` suppresses the 405 as
  well as the 404, `Allow` reports `GET, HEAD` together, `r.Pattern` inside a nested mux is the
  inner pattern only, and the conflict panic message names both registration sites plus witness
  paths.
- **2026-08-04** — Module 2 cards planned in chat, **not authored** (Chapter-done still open). Agreed
  shape: both cards are `func New...Server(addr string) *http.Server`, building the router inside and
  returning the server wired to it. Rationale: the mux stays internal so `http.NewServeMux` remains a
  full recall target (a `*http.ServeMux` return type would leak it into the auto-generated
  "Implement:" line), and the hidden tests get both halves — assert `Addr`, then drive `Handler` with
  a recorder. `go-036` = route table (`GET /items`, `POST /items`, `GET /items/{id}`,
  `GET /items/{id}/comments`, `GET /{$}`), asserting the free 404/405/`Allow` and the implicit HEAD,
  with the solution registering least-specific-first to prove precedence is not registration order.
  `go-037` = `/api/` sub-router mounted with `StripPrefix`, tail wildcard, and the 307. Both
  candidate solutions were written and run green against the proposed assertions.
  **Decision (user, 2026-08-04): "construct", not "run"** — cards build and return a `*http.Server`
  and never start it. A hidden test cannot grade `ListenAndServe` because it blocks; running and
  stopping a server on a real listener stays module 10's job. Note for module 10: the basic
  `&http.Server{Addr, Handler}` wiring is now pulled forward into module 2, so teach it there as
  revision, not as new material — module 10 still owns timeouts, `Shutdown` and `BaseContext`.
  Also found while verifying, and now written into §7 of the module 2 file: a **subtree registration
  lends its methods to the slash-less path's `Allow` header**, so `GET /items/` alone makes
  `DELETE /items` reply `405 Allow: GET, HEAD` even though nothing is registered at `/items` — and
  those advertised methods 307 rather than serve. Same for `{path...}`. This is why `go-036` uses the
  exact `GET /items` for the collection: with the subtree form, `GET /items` redirects instead of
  returning the list.
- **2026-08-04** — Chapter done. `go-036-router-route-table` (diff 9) and
  `go-037-mounted-sub-router` (diff 10) authored to `exercises/go/http-routing.json` exactly as
  agreed above; `validate` passes on all 34. Authored straight from the agreed spec rather than
  re-asking what fought them, since the spec had already been settled over three rounds and the user
  asked to proceed — feedback still welcome via the Tuned column. Both solutions and both
  `starter_code` stubs were run before writing the JSON: solutions green, starters failing cleanly
  (the stub is `&http.Server{}`, which fails the address assertion first rather than nil-panicking,
  and does not leak the router constructor). Hand-checked what `validate` cannot: `describe` output
  for both cards is free of `ServeMux`, `NewServeMux`, `PathValue`, `StripPrefix`, `HandleFunc`, and
  of every pattern spelling (`{id}`, `{$}`, `{path...}`). The auto-generated line does say "returns a
  pointer to http.Server" — same unavoidable leak as module 1's, and `*http.Server` is deliberately
  not a recall target because of it; the mux stays entirely internal so its constructor still is.
