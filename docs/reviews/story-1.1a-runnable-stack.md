# Review guide — runnable stack (backend/cmd/api, internal/platform)

Carried forward from the BMAD-produced story spec for this slice of work (originally `spec-1-1a-runnable-stack.md`, 2026-08-19), trimmed to what's useful for reviewing the code before it's committed. The code itself (`backend/cmd/api/main.go`, `backend/cmd/api/main_test.go`, `backend/internal/platform/{config,logging,httpx,postgres}`) is still untracked/unreviewed as of this doc's creation (2026-08-20).

Updated 2026-08-20 after a comment-density pass (cut restating comments, kept the ones explaining non-obvious "why") and swapping the hand-rolled `statusRecorder`/`Unwrap` in `httpx.go` for `github.com/felixge/httpsnoop` — same observable behavior, less code defending itself. Line numbers below reflect that state.

## What this slice delivers

A two-service Compose stack — Postgres plus a Go API — where `docker compose up` yields a service answering a health endpoint that reflects real Postgres reachability. Establishes `internal/platform` with four cross-cutting pieces (config, logging, httpx, postgres) and the AD-5 error envelope every later handler reuses.

## Constraints this slice was built under

- New Go code lives only under `backend/cmd/api` and `backend/internal/platform` (AD-1).
- stdlib `net/http` `ServeMux` only — no router or web framework (AD-9).
- Every error response body is `{"error":{"code","message"}}`, including the catch-all 404 (AD-5).
- All configuration comes from environment variables; `Load()` fails fast on a missing required var (12-factor).
- `pgx` and `net/http` types stay inside their edge adapters — nothing outside `internal/platform/postgres` constructs or handles a pgx type (AD-13).
- No schema, no migrations, no tables, no ETL in this slice.
- No nginx, RabbitMQ, OTel Collector, gRPC, sqlc, or auth — none has a consumer yet.
- No `/v1` business endpoints — only the health endpoint and the catch-all.

## I/O & edge-case matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Health, Postgres reachable | `GET /healthz` | `200`, body `{"status":"ok"}` | N/A |
| Health, Postgres unreachable | `GET /healthz`, pool ping fails | `503`, error envelope, `code: "dependency_unavailable"` | Ping runs under a 2s context timeout so a hung DB cannot hang the handler |
| Unknown path | `GET /v1/nope` | `404`, error envelope, `code: "not_found"` | N/A |
| Missing required env var | `api` starts without `DATABASE_URL` | Process exits non-zero before binding a port | Message names the missing variable |
| SIGTERM during a request | `docker compose stop` mid-request | In-flight request completes, then the process exits 0 | `Shutdown` under a 10s timeout; force-close after |

## Suggested review order

**Process lifecycle**
- Entry point: boot order is config, logger, pool, listener — validation before any bind. `main.go:50`
- Serve-and-drain split out so the shutdown path is reachable from a test. `main.go:88`
- `stop()` before the drain: a second SIGTERM kills a wedged shutdown instead of being swallowed. `main.go:88`

**HTTP contract every later epic inherits**
- Route table: chi router, `r.NotFound`/`r.MethodNotAllowed` replace the old stdlib shadow-pattern hack (AD-16, supersedes AD-9). `main.go:132`
- Envelope marshalled into a buffer before the status is committed. `httpx.go:50`
- Middleware chain contract: `mw[0]` is outermost. Pinned by test, not just documented. `httpx.go:28`
- Panic recovery keeps a handler fault inside the AD-5 envelope. `httpx.go:100`
- `httpsnoop.Wrap`/`CaptureMetrics` preserve Flusher/Hijacker through the chain — a hand-rolled wrapper implementing only `http.ResponseWriter` would silently break `Flush`. `httpx.go:83`, `httpx.go:110`

**Readiness**
- Health is readiness, not liveness — bounded ping, `503` on an unreachable Postgres. `main.go:152`
- Lazy pool: the process boots with Postgres down, which is what makes the `503` observable. `postgres.go:23`
- The `Pinger` seam — the one interface in this slice, and only because the failure path needs testing. `postgres.go:16`

**Configuration**
- Required vars fail fast and name themselves; the rest default. `config.go:22`
- Unparsable `LOG_LEVEL` falls back to info rather than silencing the only observability sink. `logging.go:20`

**Container and orchestration**
- `stop_grace_period` exceeds the 10s drain budget, so Docker cannot SIGKILL mid-shutdown. `docker-compose.yml:39`
- Both published ports bound to loopback — Postgres is not on the LAN. `docker-compose.yml:35`
- `depends_on: service_healthy` is the gate a later migration step will reuse. `docker-compose.yml:41`
- Non-root runtime user; only the static binary ships in the final stage. `Dockerfile:15`
- `verify` depends on the real targets, so a flag added to `test` reaches it. `Makefile:66`

**Tests**
- Real SIGTERM against a subprocess — the only test that exercises the signal wiring. `main_test.go`
- Status capture across explicit and implicit writes, the case most likely to rot silently. `httpx_test.go`
- Ping against an unreachable port and a closed pool; live-server case is skip-guarded. `postgres_test.go`

## Verification commands

- `make verify` — expected: clean. Wraps `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`.
- `make up` — expected: exits 0 with both services healthy.
- `make health` — expected: `200`, `{"status":"ok"}`.
- `curl -s -o /dev/null -w '%{http_code}' localhost:8080/v1/nope` — expected: `404`.
- `docker compose -f deploy/compose/docker-compose.yml stop postgres && make health` — expected: `503`, API container still running.
- `make down` — expected: stack torn down, named volume retained.
