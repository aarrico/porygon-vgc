# AD-6: Postgres write ownership is role-scoped, not blanket read-only

**Date**: 2026-08-19
**Status**: accepted

## Context

The original SPEC stated the application role is read-only in the same breath as requiring Postgres to be system-of-record for "anything referencing" core data (e.g. saved teams). Taken completely literally, those two statements conflict — saved teams couldn't be written at all under a blanket read-only rule. This decision is a deliberate disambiguation, not silent scope creep: it's the only reading consistent with the SPEC's own examples (saved teams, battle logs, review state must be writable).

## Decision

Four roles, scoped by table group:

- **ETL role** — write access to core reference tables (species/moves/abilities/items) only.
- **App role** — read-only on core reference tables; normal read-write on operational/user-generated tables (saved teams, battle/event logs, review-schedule state).
- **Batch/scoring-job role** — write access to the CAP-4 precomputed recommendation-scores table only.
- **Analytics-service role** (CAP-7, Python) — read-only, same shape as the app role. No write role exists for CAP-7 output (see AD-8).

## Consequences

### Positive
- A single, real permissions boundary in Postgres — not just documentation — prevents the app role from ever corrupting core reference data, and prevents an uncontrolled third writer (e.g. the analytics service) from touching anything.

### Negative
- Four roles to provision and keep in sync as new operational tables are added — each new table needs a deliberate role-grant decision, not an automatic default.

### Risks
- If a future table's role assignment is ambiguous, default to the most restrictive read-only grant and revisit — not the other way around.
