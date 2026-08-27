# Architecture Decision Records

Numbered to match the `AD-N` labels used throughout the codebase (comments, package READMEs) — e.g. `AD-5` is `0005-url-path-api-versioning.md`. Grep the repo for `AD-<N>` to find every place a decision is enforced in code.

Backfilled 2026-08-20 from the BMAD-produced architecture spine (`ARCHITECTURE-SPINE.md`, dated 2026-08-19) as part of moving off BMAD tooling onto plain ADRs. Content is unchanged; format and location are not.

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [0001](0001-single-deployable-capability-per-module.md) | Single deployable, capability-per-module | accepted | 2026-08-19 |
| [0002](0002-module-dependency-direction.md) | Module dependency direction is fixed | accepted | 2026-08-19 |
| [0003](0003-minimal-module-boundaries.md) | Module boundaries stay minimal — no speculative interfaces | accepted | 2026-08-19 |
| [0004](0004-event-pipeline-sync-resolution-async-fanout.md) | Event pipeline: synchronous resolution, async fan-out only after | accepted | 2026-08-19 |
| [0005](0005-url-path-api-versioning.md) | API versioning is URL path-based | accepted | 2026-08-19 |
| [0006](0006-postgres-role-scoped-write-ownership.md) | Postgres write ownership is role-scoped, not blanket read-only | accepted | 2026-08-19 |
| [0007](0007-precomputed-recommendation-scoring.md) | Recommendation scoring is precomputed, not live | accepted | 2026-08-19 |
| [0008](0008-analytics-service-stateless-against-postgres.md) | Analytics service is stateless against Postgres | accepted | 2026-08-19 |
| [0009](0009-stdlib-http-no-router.md) | HTTP transport is stdlib, no router library | superseded by [0016](0016-chi-router-for-multi-capability-routing.md) | 2026-08-19 |
| [0010](0010-structured-stdout-logging-otel-tracing.md) | Logging is structured, trace-correlated, stdout-only | accepted | 2026-08-19 |
| [0011](0011-single-vm-no-staging-deployment.md) | Deployment topology: single VM, no staging tier | accepted | 2026-08-19 |
| [0012](0012-etl-owned-schema-migrations.md) | Schema migrations are a named, ETL-owned tool | accepted | 2026-08-19 |
| [0013](0013-domain-logic-infrastructure-free.md) | Domain logic stays infrastructure-free | accepted | 2026-08-19 |
| [0014](0014-pokeapi-csv-as-core-data-source.md) | PokeAPI CSV dump as the core reference data source | accepted | 2026-08-19 |
| [0015](0015-data-set-snapshot-versioning-model.md) | Data Set snapshot/lineage model for versioned reference tables | accepted | 2026-08-19 |
| [0016](0016-chi-router-for-multi-capability-routing.md) | chi router for multi-capability routing | accepted | 2026-08-20 |

## Stack (current pins)

Named per the PRD's constraint that resume-gap technologies (queuing, IaC, tracing) demonstrate a pattern, not a mandatory specific product — swappable if circumstances change (see docs/PRD.md Deferred).

| Name | Version | Relevant ADR |
| --- | --- | --- |
| Go | 1.27.x | AD-1 |
| PostgreSQL | 18.x | AD-6 |
| Docker Compose | v2 plugin | AD-11 |
| Terraform | current stable (BSL-licensed; OpenTofu considered, not adopted) | AD-11 |
| chi (`github.com/go-chi/chi/v5`) | v5.3.2 | AD-16 |
| httpsnoop (`github.com/felixge/httpsnoop`) | v1.1.0 | — (httpx middleware, not tied to a specific AD) |
| log/slog (stdlib logging) | Go 1.21+ stdlib | AD-10 |
| grpc-go (`google.golang.org/grpc`) | ~v1.81.x | AD-8 |
| OpenTelemetry Go SDK | ~v1.44.0 | AD-10 |
| OpenTelemetry Collector | ~v0.150.0 | AD-10 |
| RabbitMQ | current stable | AD-4 |
| nginx | ≥1.30.1 | AD-11 |
| sqlc | current (typed Postgres access, no ORM) | AD-6 |
| golang-migrate | current | AD-12 |

Python + pandas version pins for the CAP-7 analytics service are deferred to that capability's implementation (furthest out on the build order) — verify current versions then rather than pin now.
