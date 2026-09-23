# Review guide — Champions Data Set load (backend/internal/etl)

Loads the `champions` Data Set as a child of `gen-9` from the vendored
projectpokemon/champout game dump (AD-17), inside the same ETL transaction
as the PokeAPI load. After this, `data_set=champions` on any Pokedex route
answers with the game's values, and the API container defaults to it.

## What lands

| Table | Rows | From |
|---|---|---|
| `data_set` | 1, `parent_id` → gen-9 | |
| `move` | 497 | `waza.json` where `available = 1`; generation and target inherited |
| `move_meta` | 445 | copied from the parent row; `ailment` overwritten from `con_ref` when single-valued |
| `move_stat_change` | 151 | copied from the parent row |
| `ability` | 200 | ability ids referenced by `personal.json`, named via `tokusei.json`; generation inherited |
| `item` | 148 | `item.json` named via `itemname.json`; category and fling power inherited |

## Code

- `backend/internal/etl/champout.go` — embed, JSON readers sharing the CSV
  loader's `record`/`parser`, and `slug` (display name → PokeAPI
  identifier).
- `backend/internal/etl/champions.go` — `runChampions` and one loader per
  table. Copies are single `INSERT ... SELECT ... ON CONFLICT` statements
  joined on identifier across Data Sets; per-row inserts follow the
  existing upsert shape.
- `backend/internal/etl/load.go` — `Load` runs the base loader, then a
  second loader sharing the transaction and the reference-table maps, and
  returns one `Summary` per Data Set.
- `backend/cmd/etl/main.go` — one log line per Data Set.

## Rules the loader enforces

- A roster entry whose slug has no parent row fails the load with the
  identifier in the error. No fallback generation, target or category.
- `available = 0` moves are loaded nowhere. Nihil Light (920) is the only
  move new to Champions in this dump and is unavailable.
- `accuracy = 101` → NULL; `power = 0` → NULL; `pp` stored as the game's
  value.
- `con_ref` with several ailments keeps the parent's `ailment`; an empty
  `con_ref` keeps it too, since the dump does not encode volatile ailments.
- A move with a single `con_ref` but no parent effect row gets no effect
  row: the chances that row needs are not in the dump. It is listed in
  `Summary.Skipped` and logged as a warning. In this dump that is Infernal
  Parade (burn), Mortal Spin (poison) and Matcha Gotcha (burn).

## Tests

- `champout_test.go` — slug table; every available move's slug equals the
  PokeAPI identifier at the same id and its category and `con_ref` values
  are known; every referenced ability and every item exists in PokeAPI;
  pinned value checks (Pound PP 20, Beak Blast PP 8, Moonblast Fairy).
- CI `integration` — after `make etl` twice: `champions` has parent
  `gen-9` and 497/200/148 rows; Beak Blast is 120/8; the API's default
  Data Set is `champions` and Attack +2 returns 5 moves there.

## Verified locally 2026-09-05

Two consecutive `make etl` runs produced identical row counts. Spot checks
against gen-9: Pound PP 35 → 20, Beak Blast 100/15 → 120/8, Moonblast PP
15 → 16, Will-O-Wisp burn and Thunder Wave paralysis carried, Tri Attack
kept `unknown`.

## Not in this slice

- Species and learnsets from the dump. `species` is not versioned; there
  is no learnset table yet (1.1c "Open for later").
- Stat-change magnitudes and effect chances from the game. The buff table
  `buf_ref` points at is not in the dump.
- Regulation legality. Policy layer, deferred (NFR2).
