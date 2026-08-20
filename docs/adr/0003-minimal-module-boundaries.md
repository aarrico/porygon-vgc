# AD-3: Module boundaries stay minimal — no speculative interfaces

**Date**: 2026-08-19
**Status**: accepted

## Context

Modules built independently could pre-emptively wrap every cross-package dependency in an interface "for testability" or "in case of a second implementation." An interface is a real cost — indirection, mocking overhead, a second shape to keep in sync — that only pays for itself once there's an actual second implementation or a concrete testing need.

## Decision

Modules depend on each other's packages directly via Go `internal/` visibility, not through defined interfaces. Introduce an interface at a specific dependency edge only when direct coupling actually causes a problem there — not pre-built across the board. (Story 1.1a's one exception: `postgres.Pinger`, introduced solely because the health-check failure path needed to be unit-testable without a live database — a concrete need, not a speculative one.)

## Consequences

### Positive
- No indirection to read through when tracing a call between two capability packages.

### Negative
- A genuine second implementation, when it arrives, requires introducing the interface at that point rather than having it ready — an accepted, deliberate cost.
