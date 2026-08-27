# Review guide — core reference schema and AD-6 grants (backend/migrations, deploy/postgres)

Covers Story 1.1b (issue #7): the 11-table core reference schema as `golang-migrate` migrations, AD-6's role boundary as live Postgres grants, and the one-shot `migrate` service that applies them. Closes Epic 1 / Story 1.1 AC-2 and AC-3.

Governed by AD-6, AD-12, AD-14, AD-15.

## What this slice delivers

Two migrations and a role-bootstrap script. `docker compose up` now creates four roles, applies the schema as the ETL role, and refuses to start the API until that has succeeded. No rows are loaded — the AD-6 rejection test needs a table, not data, which is what makes this shippable ahead of 1.1c.

## Constraints this slice was built under

- Only tables a Champions regulation demonstrably overrides carry `data_set_id` (AD-15).
- No legality or Regulation column anywhere in the core schema (NFR2, PRD Constraints).
- Application code neither runs nor contains a migration (AD-12).
- Grants are per table; no `ALTER DEFAULT PRIVILEGES` (AD-6's own consequence note).
- No ETL, no rows, no sqlc, no `/v1` endpoint over any of this yet.

## Table set

| Table | Versioned | Notes |
|---|---|---|
| `data_set` | — | Self-referencing `parent_id`; the `gen-9` → `champions` → `champions-reg-ma` chain (AD-15) |
| `generation` | no | `number` exists as a sort key; `'generation-ix'` does not order |
| `type` | no | The type *chart* is damagecalc's, not here |
| `stat` | no | Exists for `move_stat_change` and `nature`, not for species base stats |
| `nature` | no | Neutral natures carry both modifiers NULL |
| `species` | no | Base stats as six columns, typing as `type_1_id`/`type_2_id` |
| `move` | yes | |
| `move_meta` | yes | 1:1 with `move` |
| `move_stat_change` | yes | FR1's "Attack +2 stages" predicate |
| `ability` | yes | |
| `item` | yes | |

## I/O & edge-case matrix

| Scenario | Input / State | Expected behavior |
|---|---|---|
| Fresh volume | `make up` | initdb creates four roles, `migrate` exits 0, `api` starts and answers `/healthz` 200 |
| Migrations already applied | `make migrate` | `no change`, exit 0 |
| ETL role writes core reference data | `INSERT INTO data_set …` as `porygon_etl` | Succeeds (AC-3, positive half) |
| App role writes core reference data | `INSERT INTO data_set …` as `porygon_app` | `ERROR: permission denied for table data_set` (AC-3, negative half) |
| App role reads core reference data | `SELECT … FROM data_set` as `porygon_app` | Succeeds — proves the rejection above is the grant, not a broken connection |
| Migration fails | any SQL error | `migrate` exits non-zero; `api` never starts, because `service_completed_successfully` is unmet |
| Pre-existing volume, no roles | `make up` on a volume created before this slice | `migrate` fails on `000002` with "role porygon_app does not exist"; `make clean` is the fix |
| Child row from another Data Set | `move_meta` row whose `data_set_id` differs from its move's | Rejected by the composite FK, not by application code |

## Suggested review order

**Versioning model**
- `data_set.parent_id` self-reference and the one-step cycle guard.
- Which tables carry `data_set_id`, checked against AD-15's list.
- The redundant-looking `UNIQUE (id, data_set_id)` on `move`: it exists solely as the composite FK target, so a child row cannot silently reference a move from another Data Set.

**Species identity**
- `national_dex` is deliberately not a key — Kantonian, Alolan and Galarian Meowth are all 52.
- `species_default_per_national_dex`, a partial unique index, gives exactly one default forme per dex number.
- Ability slots are absent. They are a `data_set`-versioned relationship onto an unversioned `species` — the same problem learnsets have, and both land in one later slice rather than being solved twice.

**Effect search**
- `move_stat_change (stat_id, change)` is FR1's access path, and the only non-constraint index in the slice.
- `move_meta`'s percent columns are `NOT NULL` with 0–100 CHECKs; PokeAPI supplies all of them.

**Role boundary**
- Roles are created in initdb, not a migration: they are cluster-level and the ETL role cannot create itself.
- Passwords reach SQL as psql variables (`:'name'`), never string-interpolated.
- `REVOKE ALL ON DATABASE … FROM PUBLIC` is what makes the four `CONNECT` grants mean anything.
- Per-table `GRANT SELECT`, no default privileges — a table added later has no access until a migration grants it.
- `DATABASE_URL` now authenticates as `porygon_app`, so an AD-6 violation fails at the database rather than in review.

**Orchestration**
- `migrate` is one-shot with `restart: "no"` — a failed migration must stay failed.
- `api`'s `service_completed_successfully` dependency is AD-12's "migrate before start" made structural.
- CI proves both halves of AC-3 and asserts all 11 tables exist. `.github/workflows/ci.yml`

## Known risks to settle at 1.1c

- `move.pp` is `NOT NULL`. PokeAPI's `moves.csv` includes Colosseum/XD shadow moves with null `pp`; if the loader's filter lets one through, the insert fails. Relax the column or tighten the filter — decide then, with the real file in hand.
- PokeAPI stores neutral natures as `increased_stat_id = decreased_stat_id`, which `nature_modifiers_differ` rejects. The loader must normalize both to NULL.
- Every identity column is `GENERATED ALWAYS`, so the loader cannot carry PokeAPI's own ids across. That is deliberate — ids are Data-Set-scoped and resolution goes through `identifier` — but it is a real constraint on how 1.1c gets written.
