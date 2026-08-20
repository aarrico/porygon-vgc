# AD-11: Deployment topology — single VM, no staging tier

**Date**: 2026-08-19
**Status**: accepted

## Context

This is a cost decision, stated plainly rather than dressed up as a technical preference: the project is being built during unemployment, without unemployment benefits. A managed platform (ECS, Fargate, Cloud Run) was the preferred option on pure technical merits but isn't affordable right now.

## Decision

One Terraform-provisioned VM runs the full Docker Compose stack: nginx, the Go core, Postgres, RabbitMQ, an OpenTelemetry Collector, and the Python analytics service once CAP-7 is built. Postgres is self-hosted in Compose, not a managed DB add-on. nginx does reverse-proxy + TLS termination (Let's Encrypt) only — no routing/splitting logic at that layer, all routing stays inside the Go service. Only two environments exist: local dev (Compose) and the one prod VM — no staging tier.

## Considered Options

### Managed platform (ECS/Fargate/Cloud Run) + managed Postgres
- **Pros**: automated backups/failover, no VM/OS maintenance, better technical fit.
- **Why not**: cost-blocked right now. Explicitly the preferred option if circumstances change — see the PRD's Deferred section.

## Consequences

### Positive
- Terraform provisioning one real box, with real state and a real apply/plan cycle, still demonstrates the actual resume-gap pattern (declarative infra) — the gap being closed is "have you used Terraform," not "did you provision the most expensive possible target for it."

### Negative
- No staging tier means local Compose parity is the only pre-prod signal available.
- Self-hosted Postgres means no automated backup/failover unless built separately — not addressed by this AD.

### Risks
- Explicitly marked low-confidence and cheap to revisit if circumstances change (funding, need for automated backups/failover).
