# AD-10: Logging is structured, trace-correlated, stdout-only

**Date**: 2026-08-19
**Status**: accepted

## Context

Modules built independently risk inventing ad hoc log formats per module, and logs that can't be tied back to the trace that produced them. A separate log-aggregation service (ELK, Loki) is a real operational cost with no payoff at solo scale.

## Decision

Structured JSON logs via stdlib `log/slog`, written to stdout (captured by Docker/Compose). Every log line carries the active OpenTelemetry trace/span ID as an attribute. Spans export to an OpenTelemetry Collector container running in the same Compose stack — the single fixed export target for all modules, so no module invents its own tracing backend. No separate log-aggregation service in v1 — logs stay stdout-only, traces go through the Collector.

## Consequences

### Positive
- Every request's logs and its trace can always be correlated, with no extra infrastructure beyond the Collector.

### Negative
- No log search/aggregation UI in v1 — `docker compose logs` and the Collector's trace view are the only observability surfaces.

### Risks
- Metrics/alerting (Prometheus-style) remain fully open — not addressed by this AD, tracked separately.
