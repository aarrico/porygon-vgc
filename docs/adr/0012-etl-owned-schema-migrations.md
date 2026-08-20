# AD-12: Schema migrations are a named, ETL-owned tool

**Date**: 2026-08-19
**Status**: accepted

## Context

The ETL pipeline and application code, if allowed to independently assume different migration mechanisms (hand-rolled SQL scripts vs. a migration framework), leave no reliable way to know what schema state either side expects.

## Decision

Schema migrations live under `migrations/` and are applied exclusively via `golang-migrate`, run only by the ETL role (AD-6). Application code never runs migrations itself.

## Consequences

### Positive
- One tool, one owner, one place migrations run from — no ambiguity about current schema state between the app and the ETL pipeline.

### Negative
- The application can never self-migrate on boot — deploys must sequence "run migrations" before "start the app" as a separate explicit step.
