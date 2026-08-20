# AD-1: Single deployable, capability-per-module

**Date**: 2026-08-19
**Status**: accepted

## Context

porygon-vgc has seven capabilities (Pokedex/effect search, damage calc, team building, battle simulation, coaching, analytics) built and deployed by one person. Left unconstrained, capabilities built independently tend to drift into separate services/binaries before there's a real scaling or team reason to split them.

## Decision

One deployable Go binary (`cmd/api`). Each capability lives as its own package under `internal/` (`pokedex`, `damagecalc`, `teambuilder`, `battlesim`, `coaching`), boundaries enforced by Go's own `internal/` package visibility, not a network hop. A new capability gets a new package, not a new service. The one exception is the analytics service (CAP-7) — a separate Python process, because it's a different language runtime, not because it needs independent scaling.

## Considered Options

### Kubernetes / microservices per capability
- **Pros**: independent scaling and deployment per capability; a genuine resume-gap line item.
- **Cons**: network calls, service discovery, and distributed debugging between capabilities that are all read by the same user, deployed by the same person, on the same box.
- **Why not**: no capability here has an isolation or scaling need a process boundary would actually serve. The boundary that matters — module A's internals leaking into module B — is fully addressed by `internal/` visibility.

## Consequences

### Positive
- No network calls, service discovery, or distributed tracing overhead between capabilities.
- One build, one deploy, one thing to keep running.

### Negative
- If a capability ever does need independent scaling, splitting it out later is real work (extracting the package, defining a wire contract, standing up a second deployable).

### Risks
- None currently — no capability's projected load approaches a scale where this trade-off reverses.
