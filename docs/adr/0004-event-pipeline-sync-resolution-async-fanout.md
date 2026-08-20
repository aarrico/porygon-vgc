# AD-4: Event pipeline — synchronous resolution, async fan-out only after

**Date**: 2026-08-19
**Status**: accepted

## Context

The battle simulator (CAP-5) resolves a turn deterministically (priority brackets, speed ties, triggered abilities). That resolution is not concurrent and not distributed — putting a message broker in that hot path would only add latency and a new failure mode (a turn stuck half-resolved because a broker hiccuped) for zero upside. But once a turn finishes, its event log needs to reach two independent consumers: Postgres persistence (feeding CAP-6's replay) and the analytics service (CAP-7's training input) — without either consumer's build assuming a different delivery/ordering guarantee off the same queue.

## Decision

Turn resolution is synchronous, in-process, deterministic Go logic — no queue involved. Each `BATTLE_EVENT` carries an explicit monotonic `sequence` integer, scoped per battle; UUIDs on events are identity only, never an ordering signal. On completing a turn, the simulator emits one finished ordered event log and publishes it once to a fanout exchange, with one durable queue per consumer (persistence, analytics) — never a single shared queue. Publishing uses publisher confirms; each consumer acks manually and dedupes on the event's `sequence`/battle-ID pair, so redelivery after a consumer outage is safe (at-least-once, not at-most-once, on both sides). RabbitMQ's only job is this post-turn fan-out — it does not block the simulator.

The analytics consumer (CAP-7) treats this feed as offline/batch training-analysis input — the same log that also persists to Postgres for CAP-6 replay — not a live query path. This is a deliberate reassignment from the original architecture-notes.md sketch (which cast gRPC as the CAP-5↔CAP-7 connector): RabbitMQ carries the event stream, gRPC (AD-8) is reserved for live per-request queries.

## Considered Options

### Kafka
- **Pros**: distributed, partitioned, high-throughput log streaming across many consumers.
- **Why not**: this project's actual event volume — one solo user's play sessions — never approaches the scale where Kafka's value proposition pays for itself. Same shape of overkill as the Kubernetes exclusion (AD-1).

### NATS JetStream
- **Pros**: genuinely a better raw technical fit at this scale — single binary, sub-millisecond latency, trivial to run on the same cheap VM as everything else.
- **Cons**: no comparable management UI to RabbitMQ's; less resume-line recognition.
- **Why not**: RabbitMQ's own throughput ceiling is well above what this project will ever produce, so it isn't the wrong-scale choice Kafka would have been — and RabbitMQ was the named resume-gap technology being demonstrated.

## Consequences

### Positive
- Turn resolution has zero broker dependency — a RabbitMQ outage cannot corrupt or block an in-flight battle.
- The same event log serves both persistence and ML training input; no second "collect ML data" build.

### Negative
- Two durable queues, publisher confirms, and manual-ack-with-dedupe logic are real operational surface for a single-user application's actual load — this is the one place complexity is being taken on for resume-demonstration reasons rather than a load the project will ever see.

### Risks
- Because this fan-out has no organic load to shake out bugs under, its correctness (dedup-on-sequence, redelivery safety) needs to be proven by test, not by production traffic.
