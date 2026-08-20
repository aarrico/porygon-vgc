# AD-8: Analytics service is stateless against Postgres

**Date**: 2026-08-19
**Status**: accepted

## Context

CAP-7 reads as two capabilities wearing one number. "Suggest teams for the current meta" is, on inspection, the same problem CAP-4 already solves. "Surface coverage gaps for a specific team" is inherently per-request — there's no fixed universe of "all possible teams" to precompute gap analysis against. Two obvious "make CAP-7 stateful" designs were considered and rejected: giving it its own Postgres write role, or having the Go core persist whatever it returns.

## Decision

The Python analytics service never writes to Postgres. "Suggest teams for the current meta" reads CAP-4's precomputed scores table (AD-7) directly — no new computation or storage. "Coverage gaps for a specific team" is computed live per gRPC request. If read latency becomes a real problem, the fix is a cache (candidate: Redis — see Deferred in the PRD), not a new write path.

The gRPC contract has one canonical `.proto` definition, in a top-level `/proto` directory, code-generated into both `internal/platform` (Go client) and `analytics/` (Python server) from that single source — never hand-defined independently on either side. The Go core is the declared owner of the contract, since it's both the caller and the system of record for the shapes involved (team/species representations).

## Consequences

### Positive
- No third write role, no uncontrolled write path into Postgres from a second language runtime.
- One canonical contract definition — the two language runtimes cannot independently invent incompatible wire shapes for the same concept.

### Negative
- Live coverage-gap computation re-runs on every request with no caching layer yet — acceptable at current scale, explicitly flagged as the first thing to revisit if latency becomes a real problem.

### Risks
- If coverage-gap analysis ever needs its own slow-changing precomputed structure (not just live joins), this AD reopens.
