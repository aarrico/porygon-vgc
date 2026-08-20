# analytics

CAP-7 — Python gRPC service. Could-tier, not yet started.

Stateless against Postgres: read-only access, same shape as the Go core's app role — no write role exists for this service's output (AD-6, AD-8). "Suggest teams for the current meta" reads `teambuilder`'s precomputed scores table directly; "coverage gaps for a specific team" is computed live per request. Consumes the battle-event stream (fed via RabbitMQ from `battlesim`) as offline training input, not a live query path (AD-4).

Python + dependency version pins are deliberately unpinned until this capability's implementation starts — see the [Stack table](../docs/adr/README.md#stack-current-pins) in the ADR index.
