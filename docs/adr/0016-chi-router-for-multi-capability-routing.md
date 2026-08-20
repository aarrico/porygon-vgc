# AD-16: chi router for multi-capability routing

**Date**: 2026-08-20
**Status**: accepted
**Supersedes**: [AD-9](0009-stdlib-http-no-router.md)

## Context

AD-9 chose stdlib `net/http` + `ServeMux` on the reasoning that a service with no exotic routing needs doesn't justify a router dependency. That was evaluated against Story 1.1a's actual surface at the time — 2 routes (`/healthz`, catch-all 404). The PRD and epics (`docs/PRD.md`, issues #1–#6) already commit to roughly 20 endpoints across 6 capability packages (`pokedex`, `damagecalc`, `teambuilder`, `battlesim`, `coaching`, plus the analytics gRPC boundary) — not speculative future scope, but scope already written down. Weighing the 2-route bootstrap slice instead of that committed scope was the wrong frame: the migration cost only grows from here, since each new capability package will otherwise wire its own routes directly onto one shared `ServeMux` in `cmd/api`, making a later switch touch every package instead of one.

## Decision

Route with `github.com/go-chi/chi/v5` instead of stdlib `ServeMux`. `newHandler` builds a `chi.NewRouter()`, registers middleware via `r.Use(...)`, and uses chi's native `r.NotFound`/`r.MethodNotAllowed` instead of the stdlib shadow-pattern workaround AD-9's implementation needed to get a 405 out of `ServeMux` for a method mismatch on an existing path.

The middleware themselves (`httpx.RequestLogger`, `httpx.Recoverer`) are unchanged — chi's own default middleware doesn't produce the AD-5 JSON envelope or AD-10's structured slog output, so there's nothing to gain by swapping them for chi's versions. `httpx.Chain` also stays; it's still used directly in tests and remains available for handler composition outside a chi-routed context.

The forward-looking reason, not just the migration-cost one: each capability package can own and mount its own sub-router (`r.Mount("/v1/pokemon", pokedex.Router())`), which fits AD-1/AD-2's capability-per-package boundary better than every package's handlers registering onto one shared mux.

## Considered Options

### Stay on stdlib ServeMux, revisit later
- **Pros**: no new dependency now.
- **Why not**: the migration cost is asymmetric — cheap at 2 routes, expensive once 5 more capability packages each have routes wired against stdlib patterns. The scope that makes this worth doing is already committed (PRD/epics), not speculative.

### A full framework (gin, echo)
- **Cons**: pulls in its own request/response abstractions, JSON binding, its own middleware ecosystem — replaces more of `httpx` than needed and works against AD-13's infra/domain separation by encouraging framework types into handler signatures.
- **Why not**: chi is a router, not a framework — it implements `http.Handler`/`http.HandlerFunc` throughout, so `httpx`'s existing envelope, logging, and recovery middleware carry over unchanged.

## Consequences

### Positive
- The 404-vs-405 shadow-pattern hack AD-9's implementation needed (a methodless catch-all pattern that only sees non-GET verbs) is gone — chi distinguishes "path exists, wrong method" from "path doesn't exist" natively.
- Future capability packages get `r.Route`/`r.Mount` for grouped, versioned routes (`/v1/...`) and `chi.URLParam`/`r.PathValue` for path params, instead of hand-building stdlib patterns per package.

### Negative
- One more dependency (`go-chi/chi/v5` — small, no transitive deps beyond stdlib).
- HEAD support for a GET route is no longer automatic (stdlib `ServeMux` matches HEAD to a `GET` pattern implicitly; chi requires an explicit `r.Head(...)` registration) — each GET route that needs HEAD support must register it explicitly.
