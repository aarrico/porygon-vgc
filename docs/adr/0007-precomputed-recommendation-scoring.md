# AD-7: Recommendation scoring is precomputed, not live

**Date**: 2026-08-19
**Status**: accepted

## Context

CAP-4's differentiating half — surfacing statistically-underused-but-viable picks — requires sample-size-adjusted scoring (shrinkage toward a prior), not a raw usage-rate leaderboard. Running that computation inline on the request path risks divergent per-request scoring logic if built independently of the ingest step, and makes the request path as slow as the scoring computation itself.

## Decision

A scheduled batch job reads ingested tournament data and writes shrinkage-adjusted scores to a dedicated table, on a cadence matching how often tournament data actually changes (not real-time). The API only ever reads that table; it never computes scores at request time.

## Consequences

### Positive
- The request path is cheap — a table read, not a statistics computation.
- "Recompute my scores" becomes a debuggable, rerunnable batch step instead of logic buried inside a request handler.

### Negative
- Recommendations lag the batch job's cadence rather than reflecting the absolute latest ingested data — an accepted trade-off given tournament data doesn't change faster than the batch schedule needs to run.
