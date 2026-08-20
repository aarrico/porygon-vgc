# Roadmap: Build Order & Resume-Gap Mapping

Source: brainstorming session `brainstorm-porygon-vgc-tech-features-2026-08-18`, Affinity Clustering → MoSCoW convergence.

## MoSCoW build order

| Tier | Item | Notes |
|---|---|---|
| Must | Data/domain backbone (CAP-1, CAP-2) | Boring core schema, Postgres source of truth w/ real FKs, versioning-first API |
| Must | Docker Compose baseline | Local dev/deploy parity |
| Should | Damage calculator (CAP-3) | Second backbone piece — needed before anything can reason about a turn |
| Should | Team builder / recommendation engine (CAP-4) | The actual product differentiator |
| Should | Terraform, Nginx, OpenTelemetry | Infra around the calculator/recommendation engine |
| Could | Battle simulator / event architecture (CAP-5) | |
| Could | OAuth/JWT | Deferred deliberately — not needed until per-user saved state matters |
| Could | gRPC | Connects CAP-5's event stream to CAP-7's analytics service |
| Could | Python analytics service, pandas (CAP-7) | |
| Won't this time | Kubernetes | Genuine resume gap, deliberately excluded — see SPEC.md Open Questions |
| Won't this time | Multimodal broadcast/commentary training data | |
| Won't this time | Django | |
| Won't this time | DynamoDB | |

## Resume-gap technology → feature mapping

Source: `resume.yaml`, ten claims cut 2026-07-31 for lack of evidence: Go, Kubernetes, Terraform, Nginx, Django, WebSockets, OAuth/JWT, RabbitMQ, pandas, DynamoDB.

| Technology | Status | Where it lands |
|---|---|---|
| Go | In use | Core backend language, all capabilities |
| Kubernetes | Excluded (non-goal) | Rejected as overkill for project scale |
| Terraform | Should | Infra provisioning on top of Docker Compose |
| Nginx | Should | Fronts the API |
| Django | Excluded (non-goal) | No fit — backend is Go |
| WebSockets | Could | Implied by CAP-5/CAP-6 real-time-during-training use cases |
| OAuth/JWT | Could | Deferred to when per-user state (saved teams, battle history) needs auth |
| RabbitMQ (or current-market substitute) | Could | The event-queue pattern behind CAP-5's ordered effect resolution |
| pandas | Could | CAP-7's analytics/ML service |
| DynamoDB | Excluded (non-goal) | No fit — Postgres is system of record (see SPEC.md Constraints) |

Two additions identified during brainstorming, not on the original cut list but current-market-relevant and slotting into existing decisions rather than adding new scope:

- **gRPC** — connects the Go core to the future Python analytics service (CAP-5 ↔ CAP-7).
- **OpenTelemetry / distributed tracing** — vendor-neutral observability, complementing existing DataDog experience already on the resume.

Note: per SPEC.md Constraints, none of these are mandatory specific products — each stands in for a pattern (message queuing, IaC, event-driven design) that current-market substitutes can satisfy equally well.
