# net/http curriculum

Twelve modules, two cards each — 24 cards, `go-034` … `go-057`. Read only the section for the
module being worked on.

Difficulties continue the Go deck (which ends at `go-033`, difficulty 8) and are **non-decreasing
across modules**, 9 → 15. `cmm` serves new cards sorted by `(difficulty, id)` and ids are monotonic,
so this makes the queue play back in curriculum order.

| # | Module | File | Topic | Diff | Ids | Kinds |
|---|---|---|---|---|---|---|
| 1 | Handlers & ResponseWriter | `http-handlers.json` | `http` | 9 | go-034 | impl ×1   |
| 2 | ServeMux & routing | `http-routing.json` | `routing` | 9, 10 | go-036, 037 | impl ×2   |
| 3 | Reading the request | `http-request.json` | `http` | 10 | go-038, 039 | impl ×2 |
| 4 | Request bodies & JSON APIs | `http-json-api.json` | `json` | 10, 11 | go-040, 041 | impl ×2   |
| 5 | Middleware | `http-middleware.json` | `middleware` | 11 | go-042, 043 | impl ×2 |
| 6 | Handler dependencies | `http-handler-deps.json` | `dependency-injection` | 11, 12 | go-044, 045 | impl ×2   |
| 7 | Context & cancellation | `http-context.json` | `context` | 12 | go-046, 047 | impl ×2 |
| 8 | Streaming & ResponseController | `http-streaming.json` | `streaming` | 12, 13 | go-048, 049 | impl ×2 |
| 9 | Client, Transport, RoundTripper | `http-client.json` | `http-client` | 13 | go-050, 051 | impl ×2 |
| 10 | Server config & lifecycle | `http-server-lifecycle.json` | `servers` | 13, 14 | go-052, 053 | impl ×2 |
| 11 | File serving | `http-file-serving.json` | `file-serving` | 14 | go-054, 055 | impl ×2 |
| 12 | Reverse proxy (capstone) | `http-reverse-proxy.json` | `proxy` | 14, 15 | go-056, 057 | impl ×2   |

**Every card is `implementation`.** The user decided on 2026-08-03 that this roadmap drills writing
`net/http` code, not writing tests for it — the Go deck already has seven `test_writing` cards
covering that skill. The five modules originally planned around a `test_writing` second card (1, 2,
4, 6, 12) need a fresh implementation spec for it; each is marked below. Module 1's second card was
retired rather than replaced, so `go-035` is a permanent gap and must not be recycled.

---

## Module 1 — Handlers & ResponseWriter

**Teach:** the `Handler` interface and `ServeHTTP`; the `HandlerFunc` adapter and why a function can
be a handler; the three `ResponseWriter` methods and their ordering contract (mutate headers →
`WriteHeader` → `Write`); the implicit 200 on first write; that headers set after the status are
silently dropped and a second `WriteHeader` is ignored; content-type sniffing and content-length vs
chunked; `http.Error` and `StatusText`. Mention the optional interfaces (`Flusher`, `Hijacker`,
`io.ReaderFrom`) as a signpost to modules 5 and 8.

- **go-034** *implementation*, diff 9 — a handler that sets a content-type header, an explicit
  status, and a body **in the correct order**, and branches to an error reply on bad input.
*(Module 1 has one card. `go-035` was written as a `test_writing` card and deleted on 2026-08-03
when the user retired that kind from the roadmap; the id is a permanent gap and is not recycled.)*

## Module 2 — ServeMux & routing

**Teach:** `Handle`/`HandleFunc`; Go 1.22 patterns — method-qualified (`GET /x`), `{id}` wildcards,
`{path...}` multi-segment, the `{$}` exact anchor; precedence (most-specific wins, ties panic at
registration); `PathValue`; `r.Pattern` as the low-cardinality metrics label; the mux's own 301
redirects for path cleaning and subtree paths; automatic 404 and 405; why registering `/` is a
catch-all and `/{$}` is usually what was meant; `DefaultServeMux` as global state.

- **go-036** *implementation*, diff 9 — a router with method-qualified routes, one carrying a
  wildcard segment that the handler reads back and echoes.
- **go-037** *implementation*, diff 10 — **spec needed.** The original plan here was a `test_writing` card (405 on the wrong method, 404 on an unknown path, the wildcard round-trip); that kind is retired from this roadmap. Re-plan this as a second implementation card when the module is reached, drawing on whatever the user found hardest while studying it.

## Module 3 — Reading the request

**Teach:** `Request` server-side vs client-side field semantics (read the doc comment properly);
`URL.Query()`; `URL.Path` vs `RawPath` vs `EscapedPath()` vs `RequestURI` and which is safe for
authorization; `Header` canonicalisation and when direct map access is needed; `r.Host` vs
`URL.Host` vs the header map; cookies in (`Cookie`, `CookiesNamed`) and out (`SetCookie`,
`HttpOnly`, `Secure`, `SameSite`, `MaxAge`); form parsing order and that `FormValue` swallows
errors; `r.Clone` vs `r.WithContext` and the shallow-copy trap.

- **go-038** *implementation*, diff 10 — read a query parameter, fall back to a default when absent,
  and reject an unparseable number with 400.
- **go-039** *implementation*, diff 10 — read a request header and a cookie, and set a response
  cookie with lifetime and scope attributes.

## Module 4 — Request bodies & JSON APIs

**Teach:** decoding from `r.Body` with a streaming decoder; `DisallowUnknownFields`; rejecting
trailing data with a second decode; `http.MaxBytesReader` and why an unbounded body is a
vulnerability; encoding replies; setting content-type before the status; 201 and `Location`;
mapping decode errors to 400 vs 422 vs 413; `http.NoBody`.

- **go-040** *implementation*, diff 10 — decode a JSON body into a value, cap how much will be read,
  and reply with JSON, the right content-type, and a created status.
- **go-041** *implementation*, diff 11 — **spec needed.** The original plan here was a `test_writing` card (the created status, the content-type, strict decoding and the size cap); that kind is retired from this roadmap. Re-plan this as a second implementation card when the module is reached, drawing on whatever the user found hardest while studying it.

## Module 5 — Middleware

**Teach:** the `func(Handler) Handler` shape and why it composes; chain ordering and how a chain
constructor reads; panic recovery and `http.ErrAbortHandler`; **the big one** — wrapping
`ResponseWriter` to capture the status destroys `Flusher`, `Hijacker`, and `io.ReaderFrom`, so
streaming and `sendfile` silently break; `Unwrap() http.ResponseWriter` and
`http.NewResponseController` as the fix; why `http.TimeoutHandler` buffers and breaks streaming.

- **go-042** *implementation*, diff 11 — a chain constructor that applies middleware in a stated
  order, exercised with a header-stamping middleware and a panic-recovering one that replies 500.
- **go-043** *implementation*, diff 11 — a reply-writer wrapper that records the status and byte
  count and still lets the standard flush and deadline controls reach the real writer underneath.

## Module 6 — Handler dependencies & structure

**Teach:** a struct holding dependencies that serves requests through its own method; closures
returning a handler as the alternative; a routes method listing every route in one readable block;
an error-returning handler adapter so handlers can `return err`; mapping a sentinel error to a
status with `errors.Is`; no globals, no DI framework.

- **go-044** *implementation*, diff 11 — a type holding a store dependency that serves requests
  through its own method, replying 404 when the record is missing.
- **go-045** *implementation*, diff 12 — **spec needed.** The original plan here was a `test_writing` card (exercising a handler against a stub store); that kind is retired from this roadmap. Re-plan this as a second implementation card when the module is reached, drawing on whatever the user found hardest while studying it.

## Module 7 — Context & cancellation

**Teach:** `r.Context()` and that it is cancelled when the client disconnects — so a DB write can
die because someone closed a tab; `context.WithoutCancel` as the escape hatch for must-complete
work; `WithTimeout`/`WithDeadlineCause` and `context.Cause`; `AfterFunc`; typed context keys and why
a bare string key is a collision; request-scoped values only, never optional parameters.

- **go-046** *implementation*, diff 12 — middleware that stores a request id under a collision-proof
  key, read back by the handler and echoed.
- **go-047** *implementation*, diff 12 — a handler that honours a deadline and replies 503 when it
  expires instead of blocking. Keep waits in the tens of milliseconds; must be deterministic
  under `-race`.

## Module 8 — Streaming & ResponseController

**Teach:** when Go picks content-length vs chunked; why a handler's output can sit in a buffer until
it returns; `Flusher` and reaching it through the response controller; per-request write deadlines
via `SetWriteDeadline` (and why `Server.WriteTimeout` otherwise caps total handler time);
`EnableFullDuplex` for reading the body while writing the reply on HTTP/1.1; SSE framing and headers;
why proxies buffer.

- **go-048** *implementation*, diff 12 — stream several chunks, flushing each so they arrive before
  the handler returns.
- **go-049** *implementation*, diff 13 — a server-sent-events endpoint with the right headers and
  event framing.

Note: a no-flush fault is only catchable against a live server with a read deadline, never a
recorder — which is why there is no `test_writing` card here.

## Module 9 — Client, Transport, RoundTripper

**Teach:** `NewRequestWithContext` and never `http.Get` in library code; `Client.Timeout` covering
the body read too, so it is wrong for streaming; draining **and** closing the body being what makes
connection reuse work; `Transport` knobs, especially `MaxIdleConnsPerHost` defaulting to 2;
`ForceAttemptHTTP2`; the `RoundTripper` contract (don't mutate the request, always close the body,
don't retry non-idempotent calls) and client middleware as a wrapped round tripper; `GetBody` for
redirects and retries; `CheckRedirect` and `ErrUseLastResponse`; `httptrace` as the tool for "why is
this slow".

- **go-050** *implementation*, diff 13 — a typed API client over a base URL whose method builds a
  context-carrying request, rejects non-2xx, decodes JSON, and closes the body properly.
- **go-051** *implementation*, diff 13 — a round tripper that stamps an auth header on every outbound
  request without mutating the caller's request.

## Module 10 — Server config & lifecycle

**Teach:** why `http.ListenAndServe` is never production — no timeouts; the full set
(`ReadHeaderTimeout` as slowloris defense, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`,
`MaxHeaderBytes`); `WriteTimeout` measured from end-of-header-read so it caps handler duration;
serving on an ephemeral listener; `Shutdown` vs `Close`, `ErrServerClosed`, `RegisterOnShutdown`,
and that hijacked connections are not tracked; readiness-fails-first draining; `ConnState` for
connection metrics; `BaseContext`/`ConnContext`.

- **go-052** *implementation*, diff 13 — a constructor returning a fully configured server
  (read-header, read, write and idle timeouts, handler, address).
- **go-053** *implementation*, diff 14 — start on an ephemeral listener, shut down gracefully, and
  return no error rather than the closed-server sentinel. Tens of milliseconds; deterministic.

## Module 11 — File serving

**Teach:** `ServeContent` as the one that matters — range requests, `If-Modified-Since`,
`If-None-Match`, content-type detection — with `ServeFile`, `FileServer`, `http.FS` and `http.Dir`
as thin layers over it; `StripPrefix`; why `filepath.Join(base, userInput)` is not traversal-safe
and what `os.Root` gives you. Cards use `testing/fstest` so no real files are touched.

- **go-054** *implementation*, diff 14 — serve an in-memory filesystem under a path prefix that gets
  stripped first.
- **go-055** *implementation*, diff 14 — serve a byte payload so partial-content and conditional
  requests work.

## Module 12 — Reverse proxy (capstone)

**Teach:** `httputil.ReverseProxy` as the thing that exercises the whole package; the `Rewrite` hook
(and why it replaced `Director`); `ProxyRequest.SetURL` and `SetXForwarded`; `ModifyResponse`;
`ErrorHandler`; `FlushInterval` of -1 for streaming; hop-by-hop header stripping; `BufferPool`.

- **go-056** *implementation*, diff 14 — a proxy that rewrites the outbound URL to a target, sets
  forwarded headers, and stamps a header onto the reply on its way back.
- **go-057** *implementation*, diff 15 — **spec needed.** The original plan here was a `test_writing` card (an end-to-end proxy test between two throwaway local servers); that kind is retired from this roadmap. Re-plan this as a second implementation card when the module is reached, drawing on whatever the user found hardest while studying it.

---

## Teaching-only topics

Not exercisable through `go test` in a throwaway module, so these get covered in chat during the
relevant module and never become cards: `pprof` and observability wiring, `httptrace` timing
breakdowns, HTTP/2 and h2c specifics, connection hijacking and WebSocket handshakes, `GODEBUG`
knobs, and reading `server.go` / `transport.go` source (worth doing — roughly 6k unusually readable
lines that turn all of the above from memorised rules into understood behaviour).

Also folded into teaching only, to respect the two-card cap: form parsing, strict-decoding as its
own card (module 4's mutants teach it), and transport pool tuning.

**Third-party imports are impossible** — the graded module has no `go.sum` and no network. Every
card is stdlib-only. `net/http`, `net/http/httptest`, `net/http/httputil`, `context`, `encoding/json`,
`testing/fstest` and `sync` are all available. The graded module declares `go 1.22`, which gates
language features only — post-1.22 stdlib symbols (`r.Pattern`, `os.Root`,
`http.CrossOriginProtection`) compile fine; verified 2026-08-03.

---

## Authoring rules

`EXERCISE_AUTHORING_GUIDE.md` at the repo root is binding and must be re-read in full before writing
any card. Two rules bite unusually hard here.

### Never spell the identifier being recalled

These cards exist so the user can reproduce stdlib spellings from memory. Describe the role, not the
name — the way `go-032` says "a mutual exclusion lock" and never `sync.Mutex`.

| Recall target | Prose |
|---|---|
| `http.ResponseWriter` | "the writer the server hands the handler for building the reply" |
| `http.Request` | "a pointer to the incoming request" |
| `http.HandlerFunc` | "the adapter that lets a plain function act as a handler" |
| `http.ServeMux` | "the standard request router" |
| `r.PathValue("id")` | "read back the segment the pattern captured" |
| `httptest.NewRecorder` | "a stand-in reply writer that records what was written" |
| `httptest.NewServer` | "a throwaway local server" (the wording already used in `go-031`) |
| `http.NewResponseController` | "the standard way to reach the flush and deadline controls" |

Method and field names the user must type (`ServeHTTP`, `Shutdown`, `ModifyResponse`) may be named
directly — that matches how the guide names `Area` and `Hello`. Package-qualified type names must
not. Examples stay generic pseudocode (`Handler(method: GET, path: /users/42) -> 200, "alice"`), with
no `:=` and no Go struct literals.

### Mutant blind spots — no longer applicable

Retired with the `test_writing` kind on 2026-08-03. Kept only as a pointer: if this roadmap ever
takes mutants back, the rule is in `EXERCISE_AUTHORING_GUIDE.md`, and `cmm validate` cannot check it.

### Mechanics

- Every card needs a `solution` (validate requires it) and `starter_code` that
  compiles but fails the hidden tests (validate enforces both).
- `starter_code` is never shown to the user, but its declaration names, parameter names and types
  generate the "Implement:" prose — so it carries the API contract.
- Set `metadata.source` to `"Go net/http roadmap — <module name>"`.
- Time-dependent cards (modules 7, 8, 10) use tens-of-milliseconds waits and must be deterministic
  under `-race`. Run `validate` a few times in a row on those.
