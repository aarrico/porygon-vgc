# PRD — porygon-vgc

Pokemon VGC (Video Game Championships) team-building tool. Stack/module-level decisions live in [docs/adr/](adr/); domain vocabulary lives in [CONTEXT.md](../CONTEXT.md); build order lives in [docs/roadmap.md](roadmap.md).

## Why

Alex has been unemployed for close to three years; resume.yaml records ten technologies (Go, Kubernetes, Terraform, Nginx, Django, WebSockets, OAuth/JWT, RabbitMQ, pandas, DynamoDB) cut on 2026-07-31 for lack of evidence. This project closes that evidence gap with a real, personally-used tool rather than a resume-bingo exercise — and, independently, because Alex wants a Pokemon VGC team-building tool that helps break the cookie-cutter meta instead of just another dex clone. Both forces hold at once: a feature that doesn't serve real use isn't worth building for the portfolio, and a technology that doesn't serve the tool isn't worth adding for the resume.

## Capabilities

- **CAP-1**
  - **Intent**: Search Pokemon species, moves, abilities, and items across all generations by structured effect data, not just text.
  - **Success**: A query for a specific effect predicate (e.g. "Attack +2 stages") returns all matching entries correctly for every supported generation.

- **CAP-2**
  - **Intent**: All Pokedex functionality is reachable via a versioned HTTP API with zero UI required.
  - **Success**: Search and effect-query are fully usable and demoable via API calls alone; the API carries an explicit version from its first release.

- **CAP-3**
  - **Intent**: Calculate the outcome of a move used by an attacker against a defender under given battle conditions.
  - **Success**: Given attacker/defender/move/field state, returns a damage range/roll matching the correct Pokemon damage formula, exposed via the same versioned API.

- **CAP-4**
  - **Intent**: Team-building recommendations that either fit a target Regulation, or are deliberately unconventional-but-statistically-viable, sourced from official VGC tournament data.
  - **Success**: Recommendations are backed by sample-size-adjusted performance scoring (not raw usage rate); off-meta-but-viable suggestions are visibly distinguished from meta-conforming ones.

- **CAP-5**
  - **Intent**: Simulate a turn-based battle, with effects resolved as an ordered event stream (priority, speed ties, triggered abilities).
  - **Success**: Given two teams and a chosen action sequence, produces a deterministic, correctly-ordered event log of the turn's resolution.

- **CAP-6**
  - **Intent**: Review decision points from your own battle history and drill the ones you got wrong via spaced repetition.
  - **Success**: A decision point flagged as suboptimal reappears on a spaced-repetition schedule and stops reappearing once answered correctly enough times.

- **CAP-7**
  - **Intent**: A separate analytics/ML service consumes the battle event stream plus official tournament data to suggest teams for the current meta and surface team coverage gaps / bad matchups.
  - **Success**: Given the current dataset, produces team suggestions and coverage-gap findings consumable by the core Go service over a defined interface.

## Constraints

- No frontend exists in v1; this forces API-first design and means API versioning must be decided early (AD-5), while auth can wait.
- Core schema excludes Regulations/formats entirely (root cause: two prior attempts failed by front-loading format/regulation/mainline-to-Champions modeling before basic species data existed). Regulation legality is a separate, later, swappable ruleset/policy layer, not a schema column.
- Postgres is the system of record for species/moves/abilities/items and for anything referencing them (e.g. saved teams use real FK constraints). Write access is role-scoped (AD-6), not blanket read-only.
- Async/event-queue processing is scoped only to the battle simulator (CAP-5, AD-4); the damage calculator (CAP-3) and Pokedex API (CAP-1/CAP-2) remain synchronous request/response.
- Named resume-gap technologies are illustrative of the underlying pattern to demonstrate (message queuing, IaC, event-driven design, vendor-neutral tracing), not mandatory specific products — current-market substitutes are acceptable. See [docs/roadmap.md](roadmap.md) for the mapping.

## Non-goals

- Kubernetes / full container orchestration — rejected as overkill for this project's actual scale (AD-1).
- Django and DynamoDB — no identified fit in this project's architecture (Go/Postgres cover their roles).
- Multimodal broadcast-footage + commentary training data — too heavy a lift (video/audio processing, ASR) for this project's scope.
- Real-time, live in-match AI coaching — requires real-time vision/audio inference out of reach for a solo build; downgraded to post-hoc battle-history review (CAP-6).
- Treating the ETL/import pipeline itself as a resume-gap-filling deliverable — Alex already has professional experience here; it is necessary plumbing, not evidence this project needs to produce.

## Success signal

The Must+Should tier (CAP-1, CAP-2, CAP-3, CAP-4) is deployed and usable via API without a UI, replaces Alex's current multi-tool workflow (dex site + damage calculator + usage stats + spreadsheet) for building at least one real VGC team, and at least 3 of the mapped resume-gap technologies are demonstrably implemented with evidence Alex can cite in an interview.

## Deferred

Carried forward from the architecture spine's Deferred section — not blocking, tracked so nothing here is assumed silently resolved:

- **Auth (OAuth/JWT).** No auth in v1 — wait until per-user saved state (saved teams, battle history) actually needs it.
- **Regulation/format ruleset layer.** Explicitly out of core schema (AD-15) — a separate, later, swappable policy layer.
- **WebSockets / real-time push.** Implied by future real-time-during-training use cases; nothing in Must/Should needs it yet.
- **Redis caching layer.** Good fit on top of CAP-4's precomputed scores table and CAP-7's live coverage-gap computation (AD-8) — revisit if read latency or cache invalidation becomes a real problem.
- **Managed Postgres / managed hosting platform.** Rejected for cost reasons now (AD-11); revisit if funding or automated-backup needs change.
- **Concrete official VGC tournament data source.** Which official source (RK9 Labs, Play! Pokemon results, an official stats aggregator) is still open. Settled and non-negotiable: it is **not** Pokemon Showdown ladder replays.
- **VM provider and sizing, deploy mechanism, metrics/alerting.** Low-stakes, genuinely undecided.
- **Frontend (React Native, web & mobile).** Real Phase 2 scope per [docs/vision.md](vision.md), not a non-goal — but no capability here requires it yet (v1 is API-only).

## Open Questions

- Which official VGC tournament data source will CAP-4/CAP-7 actually ingest from? Not blocking downstream planning.
