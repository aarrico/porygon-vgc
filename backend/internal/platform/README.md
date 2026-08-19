# platform

Shared cross-cutting code used by every capability package: DB access (sqlc-generated, role-scoped per AD-6), stdlib `net/http` routing + middleware chain helper (AD-9), structured logging via `log/slog` + OpenTelemetry trace correlation (AD-10), the gRPC client for the analytics service (contract owned here, generated from the top-level `/proto`, AD-8).

No capability package's domain logic may import infrastructure types directly — that's what this package's adapters are for (AD-13). See [ARCHITECTURE-SPINE.md](../../../_bmad-output/planning-artifacts/architecture/architecture-porygon-vgc-2026-08-19/ARCHITECTURE-SPINE.md).
