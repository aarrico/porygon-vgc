# AD-9: HTTP transport is stdlib, no router library

**Date**: 2026-08-19
**Status**: superseded by [AD-16](0016-chi-router-for-multi-capability-routing.md)

## Context

A service with no exotic routing needs doesn't need a router library's main value proposition (route-grouped middleware ergonomics) badly enough to justify the dependency, especially now that Go 1.22+'s `ServeMux` supports pattern-based routing (`GET /v1/pokemon/{id}`) natively.

## Decision

`net/http` with Go 1.22+ pattern-based `ServeMux` routing. Middleware is plain `func(http.Handler) http.Handler` composition via a shared chain helper in `internal/platform` — not a third-party router/middleware framework (e.g. `chi`).

## Consequences

### Positive
- One fewer dependency; middleware has been trivial in stdlib Go since long before 1.0 — nothing a router library adds is capability, only ergonomics.

### Negative
- No route-grouped `Use()` calls or other router-library conveniences — middleware composition is explicit at each mount point via the shared `Chain` helper.

## Supersession (2026-08-20)

Reversed by AD-16 once the PRD's full route surface (~20 endpoints across 6 capability packages, see docs/PRD.md and issues #1-#6) made the migration cost asymmetry clear: doing this at 2 routes is cheap, doing it after 5 more capability packages have each wired their own routes onto a shared `ServeMux` is not. The original reasoning here (no exotic routing need, one fewer dependency) held at the time — Story 1.1a shipped exactly 2 routes — but didn't account for already-committed future scope, which isn't a YAGNI case.
