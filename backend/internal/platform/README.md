# platform

Shared cross-cutting code used by every capability package: DB access (sqlc-generated, role-scoped per AD-6), chi-routed HTTP + middleware chain helper (AD-16, supersedes AD-9), structured logging via `log/slog` + OpenTelemetry trace correlation (AD-10), the gRPC client for the analytics service (contract owned here, generated from the top-level `/proto`, AD-8).

No capability package's domain logic may import infrastructure types directly — that's what this package's adapters are for (AD-13). See [docs/adr/](../../../docs/adr/).
