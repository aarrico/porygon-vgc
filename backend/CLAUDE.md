Read the root `CLAUDE.md` first — this
file adds backend-specific detail on top, it doesn't repeat repo-wide policy
(never commit/push — stage only) or doc pointers already covered there.

## Stack

- Go 1.27, module `github.com/aarrico/porygon-vgc/backend`
- HTTP: `chi` v5 (AD-16, supersedes AD-9 — don't reintroduce stdlib
  `ServeMux` routing, that migration already happened once)
- DB: PostgreSQL 18.x via `pgx/v5` (`pgxpool`), typed queries via `sqlc` —
  no ORM. Migrations via `golang-migrate`, applied only by the ETL role;
  application code never runs a migration itself (AD-12)
- Logging: `log/slog`, structured JSON to stdout, OTel trace-correlated
  (AD-10) — no log file, no external logging service
- Inter-service: gRPC to the future Python analytics service, contract
  owned in `/proto`, generated into `internal/platform` (AD-8)

## Commands

Run from `backend/`, or via the root `Makefile` (`cd backend && ...` under
the hood — the Makefile is the single entrypoint per its own header
comment, don't add a second one):

```bash
make build     # go build ./...
make vet       # go vet ./...
make test      # go test ./...          (CI runs `go test -race ./...`)
make lint      # golangci-lint run
make fmt       # gofmt -w .
make tidy      # go mod tidy
make verify    # gofmt check + build + vet + test + lint — same gate as CI and the pre-push hook
make up        # docker compose up, build+wait-healthy
make health    # curl /healthz
make down      # docker compose down
```

`pre-commit install && pre-commit install --hook-type pre-push` once per
clone wires up `golangci-lint --fix` on commit (scoped to `backend/*.go`)
and `make verify` on push — already configured in `.pre-commit-config.yaml`,
don't duplicate this logic in a Claude Code hook.

## Module boundaries (AD-1, AD-2, AD-3)

Single deployable, one package per capability under `internal/`:

```
pokedex     (CAP-1, CAP-2) — base layer, depends on nothing else here
damagecalc  (CAP-3)        — depends on pokedex; owns FieldConditions
                              and type-effectiveness data
teambuilder (CAP-4)        — depends on pokedex + ETL tournament data;
                              never damagecalc or battlesim
battlesim   (CAP-5)        — depends on damagecalc, pokedex
coaching    (CAP-6)        — depends on battlesim's event log as a
                              read model, never the reverse
platform    (shared)       — DB access, HTTP/middleware, logging, gRPC
                              client; every capability package may
                              depend on this
```

Dependencies flow one direction only (AD-2) — a capability needing
something "backwards" means the boundary is wrong for that case, not
something to route around locally. Each `internal/*` package's own
`README.md` names the exact ADs governing it; read that package's README
before making a structural change to it.

**No interfaces between packages by default** (AD-3). Add one only at a
specific edge once direct coupling actually causes a real problem there —
`postgres.Pinger` is the one existing exception, added because the
health-check failure path needed to be unit-testable without a live DB.
Don't pre-emptively wrap a new dependency in an interface "for
testability."

**Domain logic never imports infrastructure types directly** (AD-13) — no
`*pgxpool.Pool`, `http.Request`, or raw gRPC types inside a capability
package's domain logic. Infrastructure lives in `internal/platform`'s
adapters at the package edge.

## HTTP conventions

- Every handler mounts as a chi sub-router: `r.Mount("/v1/pokemon",
  pokedex.Router())` — see AD-16. Don't register routes directly on a
  shared mux.
- Middleware is `httpx.Middleware` (`func(http.Handler) http.Handler`),
  composed with `httpx.Chain` or chi's `r.Use(...)` — not a third-party
  middleware framework. `httpx.RequestLogger` and `httpx.Recoverer` are
  the existing ones; extend this package rather than reaching for chi's
  built-in middleware, since chi's versions don't produce the AD-5 JSON
  envelope or AD-10's structured slog output.
- Responses go through `httpx.WriteJSON`/`httpx.WriteError` — the error
  envelope is `{"error": {"code": "...", "message": "..."}}` (AD-5). Don't
  hand-roll a different error shape.
- API versioning is URL-path-based (AD-5): `/v1/...`.
- A GET route that should also answer HEAD needs an explicit `r.Head(...)`
  registration — chi doesn't infer it from GET the way stdlib `ServeMux`
  did.

## Testing

- `go test -race ./...` is what CI actually runs — a change that passes
  `go test ./...` locally but has a data race will still fail CI; prefer
  running with `-race` locally too when touching anything concurrent
  (the server's shutdown path, anything touching the pgx pool).
- Table-driven tests are the existing style (see `main_test.go`,
  `httpx_test.go`). Match it.
- The Story 1.1a review (`docs/reviews/story-1.1a-runnable-stack.md`) has
  the I/O matrix CI's integration job checks against — health/404/
  Postgres-down behavior. A change to `/healthz` or the error envelope
  shape should keep that matrix passing, and CI's `integration` job will
  catch a regression there directly (it runs the real Compose stack).

## Do not touch

- `backend/migrations/` — schema migrations are ETL-owned (AD-12), so
  nothing applies one except the `migrate` service in the Compose stack
  (or `make migrate`). No `internal/*` package runs a migration, embeds
  one, or reaches for `golang-migrate` as a library. Authoring a
  migration is a different activity from running one and belongs to
  whichever story owns the schema change. An already-applied migration is
  immutable: correct it with a new version, never by editing the old one.
- Anything in `docs/adr/` — ADRs are a historical record. A changed
  decision gets a *new* numbered ADR that supersedes the old one (see how
  AD-16 supersedes AD-9), never an edit to an existing accepted ADR's
  Decision section.

## Other notes

- Code comments should be minimal and only when absolutely necessary.
