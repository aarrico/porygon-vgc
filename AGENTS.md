## porygon-vgc

Pokemon VGC team-building tool: Pokedex/effect search, damage calc, team recommendations, battle simulation, and spaced-repetition coaching, behind a versioned HTTP API. Go backend (modular monolith), Python analytics service, React Native frontend planned but not started.

## Policy

- Never commit or push — stage changes only; the user commits.

## Where things are

- Product intent and scope: [docs/PRD.md](docs/PRD.md) — capabilities CAP-1 through CAP-7, constraints, non-goals, deferred items.
- Architecture decisions: [docs/adr/](docs/adr/) — AD-1 through AD-15 govern every package below; each is a standalone, numbered file (`docs/adr/NNNN-*.md`), index and current stack pins in `docs/adr/README.md`.
- Domain vocabulary: [CONTEXT.md](CONTEXT.md) — canonical terms (Species, Data Set, Regulation, Battle Event, etc.) and what to avoid calling them instead.
- Build order: [docs/roadmap.md](docs/roadmap.md). Epic/story backlog with acceptance criteria lives as GitHub Issues, not a markdown file.
- Original project intent: [docs/vision.md](docs/vision.md) — superseded by docs/PRD.md on scope/sequencing where they conflict, still authoritative on longer-term direction.
- `backend/` — Go module root; each `internal/*` package has its own `README.md` naming the ADs that govern it.
- `analytics/`, `frontend/` — each has a `README.md` stating its current status (analytics: not started, furthest out on the roadmap; frontend: reserved, Phase 2).
- `docs/research/` — domain research backing a specific ADR (e.g. the Champions-regulation data model behind AD-15).
- `docs/reviews/` — review guides for specific chunks of code pending review, cross-referenced by file:line.

## Running and verifying

- `pre-commit install && pre-commit install --hook-type pre-push` once, after cloning — wires up two local git hooks from `.pre-commit-config.yaml`: `golangci-lint` on commit (scoped to `backend/*.go`), `make verify` on push (build+vet+test+lint+gofmt, same gate CI runs).
- `make verify` — build+vet+test+lint+gofmt, run from repo root.
- `make up` / `make health` / `make down` — bring up the Compose stack, hit `/healthz`, tear down.
- CI: `.github/workflows/ci.yml` runs on every push/PR to `main` — a `verify` job (same as `make verify`) followed by an `integration` job that runs the real Compose stack and checks the health/404/Postgres-down rows from Story 1.1a's I/O matrix. Both are required status checks on `main` (branch protection).

## Conventions that differ from defaults

- Postgres app role is read-only on core reference tables only — normal read-write on operational tables (saved teams, battle logs, review state). Don't assume a blanket read-only rule (AD-6).
- No interfaces between `internal/*` packages by default — plain package imports (AD-3). Add an interface only once direct coupling actually causes a problem.
- Domain logic in each package never imports infrastructure types directly (no `*sql.DB`, `http.Request`, raw AMQP/gRPC types) — infra lives in a thin adapter at the package edge (AD-13).

