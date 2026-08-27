## porygon-vgc

Pokemon VGC team-building tool: Pokedex/effect search, damage calc, team recommendations, battle simulation, and spaced-repetition coaching, behind a versioned HTTP API.

Three deployable pieces, each its own stack:

- `backend/` — Go modular monolith. Active. See [backend/CLAUDE.md](backend/CLAUDE.md) for Go-specific conventions.
- `analytics/` — Python gRPC service. Not started, furthest out on the roadmap.
- `frontend/` — React Native (web & mobile). Reserved, Phase 2, not started.

## Policy

- Never commit or push — stage changes only; the user commits.
- `main` is protected: PRs required, `verify` + `integration` CI checks must pass before merge.

## Where things are

- Product intent and scope: [docs/PRD.md](docs/PRD.md) — capabilities CAP-1 through CAP-7, constraints, non-goals, deferred items.
- Architecture decisions: [docs/adr/](docs/adr/) — AD-1 through AD-16 govern the codebase; each is a standalone, numbered file (`docs/adr/NNNN-*.md`), index and current stack pins in `docs/adr/README.md`.
- Domain vocabulary: [CONTEXT.md](CONTEXT.md) — canonical terms (Species, Data Set, Regulation, Battle Event, etc.) and what to avoid calling them instead.
- Build order: [docs/roadmap.md](docs/roadmap.md). Epic/story backlog with acceptance criteria lives as GitHub Issues (`epic`/`backlog` labels), not a markdown file.
- Original project intent: [docs/vision.md](docs/vision.md) — superseded by docs/PRD.md on scope/sequencing where they conflict, still authoritative on longer-term direction.
- `docs/research/` — domain research backing a specific ADR (e.g. the Champions-regulation data model behind AD-15).
- `docs/reviews/` — review guides for specific chunks of code pending review, cross-referenced by file:line.
- Repo-wide subagents live in `.claude/agents/` (this directory). They're
  process definitions (review, planning, security) — stack-specific
  knowledge comes from AGENTS.md, the package READMEs, and the ADRs, not
  from duplicating agents per subsystem.
- `backend/` has its own `.claude/settings.json` and `.claude/skills/`,
  self-contained since project settings don't inherit from this file's
  directory. Launch Claude from `backend/` for backend-only work — you'll
  get this file plus `backend/CLAUDE.md`, and none of `frontend/`'s or
  `analytics/`'s context.
- `frontend/` and `analytics/` are reserved/not-started (see their
  READMEs) and deliberately have no Claude Code config yet — add it when
  those phases actually start, matching the pattern in `backend/`.



## Running and verifying

- `pre-commit install && pre-commit install --hook-type pre-push` once, after cloning — wires up two local git hooks from `.pre-commit-config.yaml`: `golangci-lint` on commit (scoped to `backend/*.go`), `make verify` on push (build+vet+test+lint+gofmt, same gate CI runs).
- `make verify` — build+vet+test+lint+gofmt for the backend, run from repo root.
- `make up` / `make health` / `make down` — bring up the Compose stack, hit `/healthz`, tear down.
- CI: `.github/workflows/ci.yml` runs on every push/PR to `main` — a `verify` job (same as `make verify`) followed by an `integration` job that runs the real Compose stack against the health/404/Postgres-down cases. Both are required status checks on `main` (branch protection).


