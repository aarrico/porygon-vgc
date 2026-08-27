# Review guide — ETL load of the pinned PokeAPI dump (backend/cmd/etl, backend/internal/etl)

Covers Story 1.1c (issue #8): a `cmd/etl` binary that loads the pinned PokeAPI CSV dump into the core reference schema, in one transaction, idempotent on natural key. Closes Epic 1 / Story 1.1 AC-4.

Governed by AD-6, AD-12, AD-14.

## What this slice delivers

5943 upserts across the 11 tables in ~1s inside the container, from a dump vendored into the binary. The load runs as the ETL role — the only role AD-6 lets write these tables — and is gated behind a compose profile so it never happens as a side effect of `make up`.

## Constraints this slice was built under

- Reference data comes from the PokeAPI CSV dump pinned to a commit SHA (AD-14).
- Only the ETL role writes core reference tables (AD-6).
- The ETL neither runs nor contains a migration (AD-12); the compose service waits on `migrate` completing.
- No `/v1` endpoint over this data yet — that is Story 1.2.

## Why the dump is vendored

All 18 files total ~350 KB, so they are committed under `internal/etl/data/pokeapi/` and pulled into the binary with `go:embed`. A load therefore needs no egress, cannot be broken by a GitHub outage or rate limit, and is byte-reproducible. The alternative — fetching by SHA at runtime — would make every ETL run depend on a network that has nothing to do with the data.

`go:embed` cannot reference parent directories, which is why the CSVs sit under the package rather than at `backend/data/`.

## What the loader normalizes, and why

Each of these is a real mismatch between PokeAPI's encoding and the schema, found by validating the dump before the loader was written:

| Case | PokeAPI | Loaded as |
|---|---|---|
| Shadow moves | 18 moves at `id > 10000`, the only ones with blank `pp` | Skipped — `move.pp` is `NOT NULL` |
| Non-main-series abilities | 60 of 373 with `is_main_series = 0` | Skipped — filtered on the semantic column, not an id range |
| Never-miss moves | Usually blank `accuracy`, but `0` for three | Both become NULL, the schema's single spelling for "cannot miss" |
| Neutral natures | 5 of 25 set `increased_stat_id == decreased_stat_id` | Both NULL, which `nature_modifiers_paired` requires |
| `roseli-berry` | Two rows differing only in `cost` | Upsert collapses them; `cost` is not stored |

`move_meta.healing` ranges −33..50 (Struggle and Clangorous Soul do self-damage). The `CHECK (healing BETWEEN 0 AND 100)` written in 1.1b was wrong and was corrected to `BETWEEN -100 AND 100` before this slice loaded anything.

## I/O & edge-case matrix

| Scenario | Input / State | Expected behavior |
|---|---|---|
| First load | `make etl` on a migrated, empty schema | 5943 upserts, exit 0 |
| Second load | `make etl` again, unchanged dump | Same counts, same rows, exit 0 |
| Row counts | after any load | species 1351, move 919, ability 313, item 2222 (2223 upserts) |
| Any table fails | e.g. a CHECK violation mid-load | Whole transaction rolls back; schema unchanged |
| Re-pin drops a column | `moves.csv` loses `pp` | `missing column "pp"` before any write; caught by `TestEmbeddedDumpHasRequiredColumns` |
| Unmigrated schema | `make etl` before `make migrate` | Compose gates on `migrate` completing; a direct run fails on a missing relation |
| `make up` | stack brought up | ETL does **not** run — it is behind the `etl` profile |

## Suggested review order

**Transaction and idempotency**
- One transaction for the whole dump, so a late failure leaves nothing behind. `load.go`
- Every table upserts on its natural key — `(data_set_id, identifier)` for the versioned five, `identifier` for the rest.
- `Summary.Rows` counts upserts, not resulting rows. The two differ only for `item`, because of the duplicate berry.

**Identity resolution**
- Every primary key is `GENERATED ALWAYS`, so PokeAPI's ids cannot be carried across. `index` reads `identifier -> id` back after each table and `link` turns that into the `PokeAPI id -> id` map the other CSVs need. This is the main structural consequence of the 1.1b key design.

**Failure behavior**
- `readTable` takes the columns the caller needs and validates them against the header, so an upstream rename is a startup error rather than a NULL column.
- `parser` holds the first failure per row, so a table loader checks once instead of after every column.

**Boundaries**
- `cmd/etl` is thin over `internal/etl`, mirroring `cmd/api` over `internal/platform`.
- The `etl` compose service authenticates as `porygon_etl` via `ETL_DATABASE_URL`; `api` still authenticates as `porygon_app` and cannot write any of this.
- The Dockerfile now has two named final stages, so both compose services name a `target`.

## Verified

Against a live stack from an empty volume:

- fresh volume → `migrate` exits 0 → `api` healthy → `etl` loads in 1.0s
- a second `make etl` produces identical row counts
- FR1's structured effect query returns the 7 moves that raise Attack two stages, run as the read-only app role
- Incineroar Fire/Dark 95/115/60; Urshifu-Rapid-Strike Fighting/Water 100/130/97, `is_default = false`

CI's `integration` job runs the load twice and asserts species/move counts plus the FR1 query.

## Open for later

- Ability slots and learnsets are still absent — both are `data_set`-versioned relationships onto an unversioned `species` (see the 1.1b guide). The ETL will need `pokemon_abilities.csv` and `pokemon_moves.csv` when that slice lands; neither is vendored yet.
- `move.target` and `item.category` are free text, not lookup tables, per the 11-table budget. If a query ever needs to filter on them structurally, that is the point to revisit.
