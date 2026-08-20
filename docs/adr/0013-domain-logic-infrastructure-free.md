# AD-13: Domain logic stays infrastructure-free

**Date**: 2026-08-19
**Status**: accepted

## Context

Left unchecked, a package's core domain logic (a formula, a resolution algorithm, a scheduling rule) can take a direct dependency on an infrastructure primitive — a specific DB driver type, `http.Request`/`ResponseWriter`, an AMQP client type, a generated gRPC struct used as a domain type. This makes domain logic untestable without real infrastructure running, and lets independently-built packages disagree on where the infra/domain line sits. This surfaced from `docs/vision.md`, which named it as a hard non-negotiable but had never been formalized into an enforceable rule until this AD.

Distinct from AD-3: AD-3 governs package-to-package boundaries; this AD governs the infra-to-domain boundary *inside* each package.

## Decision

Each package's domain logic (e.g. `damagecalc`'s formula, `battlesim`'s resolution logic, `coaching`'s SM-2 scheduler) takes and returns only plain Go types and its own package's exported domain types. Infrastructure concerns — Postgres access via sqlc-generated code, HTTP handlers, the RabbitMQ publisher/consumer, the gRPC client/server — live in a thin adapter at the edge of the package (e.g. `store.go`, `handler.go`) that translates between infra shapes and domain calls. No infra call is made inline from within domain logic.

## Consequences

### Positive
- Domain logic is unit-testable with plain Go values — no database, no HTTP server, no message broker needed to test a formula or a scheduling rule.
- The infra/domain boundary is the same shape in every package, so it doesn't need re-deriving per capability.

### Negative
- Every package needs its own thin adapter layer, even for simple cases — a small amount of boilerplate traded for the testability and boundary clarity.
