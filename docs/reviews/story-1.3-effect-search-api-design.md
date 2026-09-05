# Design doc — Structured effect search over `/v1` (backend/internal/pokedex)

Story 1.3: `GET /v1/moves/search?effect=...` returns every move whose
structured effect data matches a conjunction of predicates. Effects are
typed columns and stat-change rows (CONTEXT.md "Effect Predicate"), never
parsed from prose. Closes CAP-1's success criterion.

Built on Story 1.2's package layout: same sub-router, same `Store` adapter,
same sqlc pipeline, same `data_set` resolution and error envelope.

## Scope

Moves only. PokeAPI carries structured effect data for moves
(`move_meta`, `move_meta_stat_changes`, AD-14) and nothing structured for
abilities or items, so the story's "moves/abilities/items" phrasing has no
data to stand on for the latter two. The AC names only the moves route.

Two predicate kinds, matching the two structured effect tables:

| Kind | Column | Example |
|---|---|---|
| `stat:<identifier>:<stages>` | `move_stat_change (stat_id, change)` | `stat:attack:2` |
| `ailment:<identifier>` | `move_meta.ailment` | `ailment:burn` |

`move_meta`'s other columns (category, chances, drain, healing, hits,
turns) are returned in every result but are not yet queryable. Adding a
kind is one `case` in `ParseEffects` and one clause in the query.

## Route

`backend/internal/pokedex/router.go:34-36` — `GET`/`HEAD /moves/search`,
registered alongside `/moves`; distinct chi patterns, no conflict.
`searcher` gains `SearchMovesByEffect` (`router.go:15`); the interface is
the pre-existing in-package one, not a new cross-package interface (AD-3).

## `effect` parameter — grammar

Repeatable. Each value is `<kind>:<value>[:<arg>]`, split on `:`, trimmed.
Repeats are ANDed: every predicate must hold for a move to be returned.

`backend/internal/pokedex/effect.go:64` `ParseEffects` is pure: no I/O, no
`http` types. It rejects, as `400 invalid_effect`:

- unknown kind
- wrong arity for the kind
- empty identifier
- stages not an integer, zero, or outside -6..6
- more than 8 predicates in one request, or a predicate over 64 characters

The two limits exist because each stat predicate is one more element in a
correlated subquery evaluated per candidate move, and because the error
message echoes the predicate. Measured before the cap: 10,000 predicates
cost ~0.4 s of Postgres CPU per request and 120 concurrent requests
exhausted the pool far enough to fail `/healthz`.

`+2` is accepted (`strconv.ParseInt` takes a leading `+`), but a literal
`+` in a query string decodes to a space; clients send `%2B2` or omit the
sign. Documented in the package README.

Stat and ailment identifiers are not validated against a Go-side list.
The vocabulary is reference data loaded by the ETL role, so the DB is the
source of truth. See "On-empty vocabulary check" below.

Missing `effect` entirely is `400 invalid_request`, mirroring `name` on
the 1.2 routes.

## Query — `queries/effects.sql`

`SearchMovesByEffect` (`effects.sql:1`) is one statement with four
parameters: `data_set`, `ailments text[]`, `stat_identifiers text[]`,
`stat_changes smallint[]`.

**Stat conjunction** (`effects.sql:34-39`). The two arrays are paired by
position with `unnest ... WITH ORDINALITY` joined on ordinal, then joined
to `stat` and `move_stat_change`. A move matches when the count of
satisfied pairs equals the array's cardinality. Because
`move_stat_change`'s primary key is `(move_id, stat_id)`, two predicates
on the same stat with different stages can satisfy at most one pair and
the move drops out, which is the correct answer for a contradiction. Zero
stat predicates give `0 = 0` and impose no filter.

sqlc's analyzer does not know the two-argument `unnest(a, b)` form, which
is why the pairing is done with two single-argument `unnest` calls.

**Ailment** (`effects.sql:31`). `move_meta` is LEFT JOINed because 110 of
the loaded moves have no meta row and 15 of those still have stat
changes. `cardinality(ailments) = 0 OR mm.ailment = ALL(ailments)` is
spelled out rather than relying on `ALL` over an empty array being true
when the left side is NULL. Two different ailments in one request give
an empty result, not an error; a move has one ailment.

**Index use.** `move_stat_change_effect (stat_id, change)` from 1.1b is
the access path for the stat pairs. The ailment filter scans `move_meta`
(827 rows, no index); not worth one at this size.

**Nil slices.** pgx encodes a nil Go slice as SQL `NULL`, and
`cardinality(NULL)` is `NULL`, which would make the whole `WHERE` false.
`SearchMovesByEffect` (`effect.go:108`) always passes non-nil slices.

**Stat changes per result** come from a second query,
`ListMoveStatChanges` (`effects.sql:42`), keyed by the matched move ids
and grouped in Go. One round trip per request, not per row.

## On-empty vocabulary check

`effect.go:165` `checkEffectVocabulary`. When the search returns no rows,
the store distinguishes "valid predicate, no such move" from "unknown
identifier" with two follow-up queries (`ListUnknownStats`,
`ListUnknownAilments`, `effects.sql:49-57`), after the existing
`checkDataSetExists`. An unknown identifier surfaces as
`*InvalidEffectError` naming the offending predicate. This is the same
check-on-empty pattern 1.2 established for `data_set`: the happy path
costs one query, the empty path pays for the diagnosis.

Both checks run across every Data Set on purpose. They answer "is this
a real identifier", not "does it occur in this Data Set"; an identifier
that exists but has no move in the requested Data Set is a valid query
with no matches, so it stays a 200 with `count: 0`.

An unknown stat can never produce a non-empty result (the pair fails to
join, so the count falls short), so checking only on empty is complete,
not a heuristic.

## Response shape

`versionedResponse[MoveWithEffect]`: `{data_set, results, count}`.
`MoveWithEffect` (`effect.go:48`) embeds `Move`, so the 1.2 move fields
are flat, plus `effect`:

```json
{
  "id": 14, "identifier": "swords-dance", "generation": "generation-i",
  "type": "normal", "damage_class": "status", "power": null,
  "accuracy": null, "pp": 20, "priority": 0, "target": "user",
  "effect": {
    "category": "net-good-stats", "ailment": "none",
    "ailment_chance": 0, "stat_chance": 0, "flinch_chance": 0,
    "crit_rate": 0, "drain": 0, "healing": 0,
    "min_hits": null, "max_hits": null, "min_turns": null, "max_turns": null,
    "stat_changes": [{"stat": "attack", "change": 2}]
  }
}
```

Every `move_meta` column is a pointer so a meta-less move serialises with
explicit nulls rather than a missing object; `stat_changes` is always an
array, empty when there are none. Results are ordered by identifier.

## Errors

| Case | Status | Code |
|---|---|---|
| no `effect` param | 400 | `invalid_request` |
| grammar / arity / stage range | 400 | `invalid_effect` |
| unknown stat or ailment identifier | 400 | `invalid_effect` |
| unknown `data_set` | 400 | `unknown_data_set` |
| valid predicate, no matching move | 200 | `count: 0` |
| Postgres failure | 500 | `internal_error` |

`invalid_effect` messages quote the predicate verbatim, JSON-escaped by
`httpx.WriteError`.

## I/O matrix

| Request | Result |
|---|---|
| `?effect=stat:attack:2` | 200, 7 moves (decorate, extreme-evoboost, fillet-away, shell-smash, spicy-extract, swagger, swords-dance) |
| `?effect=stat:attack:%2B2` | same |
| `?effect=stat:attack:2&effect=ailment:confusion` | 200, swagger only |
| `?effect=ailment:burn` | 200, 22 moves, blaze-kick through will-o-wisp |
| `?effect=stat:hp:1` | 200, empty |
| `?effect=stat:atk:2` | 400 `invalid_effect` "unknown stat" |
| `?effect=ailment:sunburn` | 400 `invalid_effect` "unknown ailment" |
| `?effect=weather:rain` | 400 `invalid_effect` "unknown kind" |
| `?effect=stat:attack:9` | 400 `invalid_effect` stage range |
| `?effect=stat:attack:2&data_set=nope` | 400 `unknown_data_set` |
| no `effect` | 400 `invalid_request` |
| `HEAD` any of the above | same status, no body |

Verified against the Compose stack on 2026-09-05. CI's `integration` job
asserts the first, sixth, and the swords-dance stat change
(`.github/workflows/ci.yml:162`).

## Tests

- `effect_test.go:9,38,71` — grammar table, valid and invalid, limits.
- `router_test.go:240-361` — handler status/code/envelope for each error
  class, query passthrough to the store, HEAD.
- `store_test.go:89-188` — live-DB: Attack +2 set, positional pairing of two stat predicates, combined stat+ailment,
  valid-but-empty, unknown stat/ailment/data set. Skip without
  `TEST_DATABASE_URL`; CI runs them after the ETL load.

## Non-goals for this slice

- Predicates over `move_meta`'s numeric columns (`drain`, `crit_rate`,
  `flinch_chance`, hits, turns) and `category`.
- Range or inequality on stages (`>= 2`). Exact match only.
- OR across predicates. Repeat the request.
- Effect search for abilities and items. No structured source data.

## Follow-ups outside this slice

Surfaced by the security review, pre-existing in `internal/platform`:

- No `statement_timeout` on the app role and no per-request context
  deadline; `http.Server.WriteTimeout` does not cancel an in-flight pgx
  query, so a client that has already timed out keeps its pooled
  connection until Postgres finishes.
- No `X-Content-Type-Options: nosniff` on responses.

## Assumption most likely to be wrong

Competitive play runs on Pokemon Champions, and Champions is a Data Set
layered on `gen-9` (AD-15), so the only search that matters to a player
is `data_set=champions` or a regulation child of it. Today the only
loaded Data Set is `gen-9`, and it is the default. Until the Champions
overlay loads, every effect search answers with Scarlet/Violet mechanics
for a game whose move data differs in 259 places.

The assumption is that Champions' effect deltas will arrive as structured
`move_meta` and `move_stat_change` rows under the `champions` Data Set,
so this query stays correct with no change. PokeAPI carries no Champions
data, so those rows have to come from a second source; the research doc
names Showdown's `data/mods/champions` as the only structural one. That
is the ETL story that makes this endpoint useful for its actual purpose,
and it is not this story.
