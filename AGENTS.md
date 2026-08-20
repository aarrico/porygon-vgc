<!-- bmad:context -->
<!-- Verified 2026-08-19, pre-initial-commit (no commits yet). Managed by bmad-project-context; edits inside this block are replaced on refresh. Keep anything you want preserved outside the markers. -->

## porygon-vgc

Pokemon VGC team-building tool: Pokedex/effect search, damage calc, team recommendations, battle simulation, and spaced-repetition coaching, behind a versioned HTTP API. Go backend (modular monolith), Python analytics service, React Native frontend planned but not started. Planning contract: `_bmad-output/specs/spec-porygon-vgc/SPEC.md` (companions: `roadmap.md`, `architecture-notes.md`, `ARCHITECTURE-SPINE.md`). Stories: `_bmad-output/planning-artifacts/epics.md`.

## Policy

- Never commit or push — stage changes only; the user commits.
- Never hand-edit `SPEC.md` or `ARCHITECTURE-SPINE.md` directly — both are derived from their `.memlog.md` via `bmad-spec`/`bmad-architecture`; edit through those skills.
- Never modify `_bmad/` or `.claude/skills/` — vendored tooling, not project code.

## Where things are

- Architecture contract: `_bmad-output/planning-artifacts/architecture/architecture-porygon-vgc-2026-08-19/ARCHITECTURE-SPINE.md` — AD-1 through AD-13 govern every package below.
- Human-readable design writeup: `_bmad-output/planning-artifacts/architecture/architecture-porygon-vgc-2026-08-19/porygon-vgc-solution-design.md`.
- Original project intent: `vision.md` (repo root) — superseded by SPEC.md on scope/sequencing where they conflict, still authoritative on longer-term direction.
- `backend/` — Go module root; each `internal/*` package has its own `README.md` naming the ADs that govern it.
- `analytics/`, `frontend/` — each has a `README.md` stating its current status (analytics: not started, Could-tier; frontend: reserved, Phase 2).

## Running and verifying

- `pre-commit install && pre-commit install --hook-type pre-push` once, after cloning — wires up two local git hooks from `.pre-commit-config.yaml`: `golangci-lint` on commit (scoped to `backend/*.go`), `make verify` on push (build+vet+test+lint+gofmt, same gate CI runs).
- `make verify` — build+vet+test+lint+gofmt, run from repo root.
- `make up` / `make health` / `make down` — bring up the Compose stack, hit `/healthz`, tear down.
- CI: `.github/workflows/ci.yml` runs on every push/PR to `main` — a `verify` job (same as `make verify`) followed by an `integration` job that runs the real Compose stack and checks the health/404/Postgres-down rows from Story 1.1a's I/O matrix. Both are required status checks on `main` (branch protection).

## Conventions that differ from defaults

- Postgres app role is read-only on core reference tables only — normal read-write on operational tables (saved teams, battle logs, review state). Don't assume a blanket read-only rule (AD-6).
- No interfaces between `internal/*` packages by default — plain package imports (AD-3). Add an interface only once direct coupling actually causes a problem.
- Domain logic in each package never imports infrastructure types directly (no `*sql.DB`, `http.Request`, raw AMQP/gRPC types) — infra lives in a thin adapter at the package edge (AD-13).

<!-- /bmad:context -->
