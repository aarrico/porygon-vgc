# Design doc — Pokemon search over `/v1` (backend/internal/pokedex)

Pre-implementation contract for Story 1.2 (issue #1, Epic 1): "Search Pokemon
via Versioned API." `internal/pokedex` currently contains only a README stub —
nothing here is built yet. This is a design doc, not a review guide (compare
`docs/reviews/story-1.1a-runnable-stack.md` and `-1.1b-` for the post-hoc
format this borrows its I/O-matrix convention from).

Governed by AD-2, AD-5, AD-6, AD-13, AD-15, AD-16.

## Scope

Plain name/identifier search across `species`, `move`, `ability`, `item`.
**Not** in scope: Story 1.3's structured effect-predicate search
(`GET /v1/moves/search?effect=...`, `move_stat_change` joins). The route
shapes below are chosen so 1.3 can add `/v1/moves/search` without colliding
with anything here.

## Route table

| Resource | Path | Method | data_set-scoped? |
|---|---|---|---|
| Species | `/v1/species` | GET, HEAD | no (AD-15) |
| Move | `/v1/moves` | GET, HEAD | yes |
| Ability | `/v1/abilities` | GET, HEAD | yes |
| Item | `/v1/items` | GET, HEAD | yes |

All four mount from a single `pokedex.Router() chi.Router`, per
`backend/CLAUDE.md`'s "every handler mounts as a chi sub-router." **Deviation
from that doc's literal example** (`r.Mount("/v1/pokemon", pokedex.Router())`):
pokedex owns four distinct resource nouns, not one "pokemon" umbrella
(CONTEXT.md explicitly says to avoid "Pokemon" as a term — ambiguous with a
species instance on a team). So:

```go
// cmd/api/main.go, inside newHandler
r.Mount("/v1", pokedex.Router())
```

```go
// internal/pokedex/router.go
func Router(deps ...) chi.Router {
    r := chi.NewRouter()
    r.Get("/species", speciesSearchHandler(deps))
    r.Head("/species", speciesSearchHandler(deps))
    r.Get("/moves", movesSearchHandler(deps))
    r.Head("/moves", movesSearchHandler(deps))
    r.Get("/abilities", abilitiesSearchHandler(deps))
    r.Head("/abilities", abilitiesSearchHandler(deps))
    r.Get("/items", itemsSearchHandler(deps))
    r.Head("/items", itemsSearchHandler(deps))
    return r
}
```

Registering GET+HEAD on the same handler is safe and matches the existing
`/healthz` pattern in `main.go`: `net/http`'s server suppresses the body on a
HEAD response but still computes `Content-Length`, so `httpx.WriteJSON`
doesn't need a HEAD-specific branch.

`1.3` later adds `r.Get("/moves/search", ...)` to the same sub-router — a
distinct chi pattern from `/moves`, no conflict.

## `name` query parameter — semantics (needs confirmation, see Open Questions)

All four endpoints take `?name=<string>`, matched case-insensitively as a
**substring** against the `identifier` column (`identifier ILIKE '%' ||
escape(name) || '%'`, with `%`/`_`/`\` escaped so user input can't inject
wildcards).

Rationale: the AC's own example (`?name=pikachu`) is satisfied by an exact
hit, but the AC's phrasing — "the matching record(s)" — reads as anticipating
more than one row, and the schema's own comment block for regional formes
(Kantonian/Alolan/Galarian Meowth, `identifier`s `meowth`/`meowth-alola`/
`meowth-galar`, sharing `national_dex` 52) only groups naturally under a
substring match. Prefix-only or exact-only match would silently drop that
case for a query like `?name=meowth`. Substring gives both behaviors for
free with one query shape, and reads as an actual "search" rather than a
lookup — consistent with CAP-1's verb.

`name` is required; missing or empty (after trim) is `invalid_request` (see
Errors). No minimum length beyond non-empty — a single-character query can
return a large result set, but see Non-goals below on why that's accepted
for this slice.

## `data_set` resolution (moves/abilities/items only) — needs a decision

Per AD-15, `move`/`ability`/`item` carry `data_set_id`; `species` does not,
so the species endpoint has no `data_set` concern at all.

**Nothing upstream has decided what "the current data set" means for a
read request** — `internal/etl/cmd/etl/main.go` has a local, unexported
`defaultDataSet = "gen-9"` used only for *loading*, not read-side. There is
exactly one data set loaded in any real environment today (`gen-9`; no
Champions overlay data is loaded yet per the 1.1c review's "Open for
later").

**Recommendation** (needs architect/human sign-off before implementation,
not something to silently decide inside pokedex):

- Add `Config.DefaultDataSet` to `internal/platform/config` (env
  `DEFAULT_DATA_SET`, `envOr`'d against `"gen-9"`) — same pattern
  `HTTP_ADDR`/`LOG_LEVEL` already use. This makes the constant a single
  cross-cutting value instead of a private const duplicated per package
  (`cmd/etl` already has its own copy; it should probably start reading
  the same config value once this lands, but that's this story's concern
  to touch, not this doc's to mandate for a package it doesn't own).
- The three versioned endpoints accept an optional `?data_set=<identifier>`
  query param. Omitted → resolve against `Config.DefaultDataSet`. Provided
  but not a real row in `data_set` → `unknown_data_set` (see Errors).
- Explicitly rejected: querying across all data sets unfiltered. Once a
  Champions overlay loads, an identifier can exist in two data sets with
  different column values (that's the entire point of AD-15's override
  model) — returning both under one `identifier` would silently produce
  contradictory rows for what a client reasonably expects to be one
  record. A single resolved `data_set_id` predicate is AD-15's own stated
  reason read queries stay simple; this endpoint should not reintroduce
  the overlay ambiguity AD-15 deliberately pushed to load time.

## Success response shape

Object with a `results` key, not a bare array — leaves room to add
pagination metadata later without a breaking shape change. Versioned
resources echo back the resolved `data_set` identifier so a client that
omitted the param can tell which snapshot answered it.

**Species** (`GET /v1/species?name=meowth`):

```json
{
  "results": [
    {
      "id": 52,
      "identifier": "meowth",
      "national_dex": 52,
      "is_default": true,
      "generation": "generation-i",
      "types": ["normal"],
      "base_stats": {
        "hp": 40, "attack": 45, "defense": 35,
        "special_attack": 40, "special_defense": 40, "speed": 90
      }
    },
    { "id": 863, "identifier": "meowth-alola", "national_dex": 52, "is_default": false, "...": "..." },
    { "id": 864, "identifier": "meowth-galar", "national_dex": 52, "is_default": false, "...": "..." }
  ],
  "count": 3
}
```

No `data_set` key — species isn't versioned (AD-15).

**Move** (`GET /v1/moves?name=thunderbolt`):

```json
{
  "data_set": "gen-9",
  "results": [
    {
      "id": 85,
      "identifier": "thunderbolt",
      "generation": "generation-i",
      "type": "electric",
      "damage_class": "special",
      "power": 90,
      "accuracy": 100,
      "pp": 15,
      "priority": 0,
      "target": "selected-pokemon"
    }
  ],
  "count": 1
}
```

Deliberately excludes `move_meta`/`move_stat_change` joins — that's 1.3's
effect-predicate territory, not plain search.

**Ability** (`GET /v1/abilities?name=static`):

```json
{
  "data_set": "gen-9",
  "results": [
    { "id": 9, "identifier": "static", "generation": "generation-iii" }
  ],
  "count": 1
}
```

**Item** (`GET /v1/items?name=leftovers`):

```json
{
  "data_set": "gen-9",
  "results": [
    { "id": 234, "identifier": "leftovers", "category": "held-items", "fling_power": null }
  ],
  "count": 1
}
```

`generation`/`type` on species and move, and `types` (array, one or two
elements), are resolved to their `identifier` strings via join, not left as
opaque FK ids — CAP-2's success criterion is that the API is "fully usable
and demoable via API calls alone," which a bare `generation_id: 3` fails.
`id` (the table's own bigint PK) is included alongside `identifier` since a
later capability (`damagecalc`) will need a stable numeric reference; low
cost to expose now, no known consumer yet — flagged, not hidden.

## Errors

| Case | Status | `code` | Notes |
|---|---|---|---|
| `name` missing or empty/whitespace-only | 400 | `invalid_request` | |
| `data_set` provided, no matching row | 400 | `unknown_data_set` | Client-supplied value is wrong, not a missing path resource — 400 not 404 |
| No rows match `name` | **200** | n/a | `{"results": [], "count": 0}` (+ `data_set` echoed for versioned resources) — a search endpoint returning zero matches is not an error |
| Unrecognized/extra query params | 200 | n/a | Ignored, not rejected — keeps room to add params additively later without breaking existing callers |
| Unexpected DB failure | 500 | `internal_error` | Matches `httpx.CodeInternalError`; log the real error server-side (`slog`), never surface DB error text to the client |

Method-not-allowed on these paths (e.g. `POST /v1/species`) falls through to
chi's existing global `r.MethodNotAllowed` in `main.go`. That handler's
`Allow` header is currently hardcoded to `GET, HEAD` at the router root —
correct for this story only because every route added here is also
GET+HEAD-only. Flagging as a pre-existing limitation (global, not per-route,
so it would misreport `Allow` for a future POST-only capability route) —
out of scope to fix here, not introduced by this story.

## Non-goals for this slice

- **Pagination.** PRD has no pagination requirement, and the loaded row
  counts are small (species 1351, move 919, ability 313, item 2222 per the
  1.1c review) — a full unpaged result set is cheap. Revisit if the
  `data_set` catalog or per-set row counts grow enough that a single
  short-identifier query becomes an unbounded scan (e.g. `?name=e`
  returning hundreds of rows). Not addressed by AD-15's per-data-set
  filtering, since that bounds *which* rows are eligible, not *how many*.
- **Filtering by anything other than `name`** (e.g. `national_dex`,
  `damage_class`, `type`) — not ACd for 1.2, would be scope creep ahead of
  1.3.
- **Auth.** Per PRD's Deferred section, no OAuth/JWT until per-user state
  exists — nothing about a name-search endpoint changes that; not designed
  around here.

## sqlc vs. raw pgx — flagged, not decided here

`backend/CLAUDE.md` already commits the stack to "typed queries via sqlc —
no ORM," but no `sqlc.yaml` or query file exists anywhere in the repo yet.
`internal/etl` is the only precedent, and it uses hand-written SQL directly
against `pgx.Tx` — but the ETL is called out in the PRD's own non-goals as
"necessary plumbing, not evidence this project needs to produce," so it
isn't a strong precedent for the read path a capability package like
`pokedex` actually needs to demo.

**Recommendation**: adopt sqlc starting with this story rather than
deferring — it's the first read-path capability package that needs typed
result-set mapping (species-with-joined-type-identifiers, in particular),
and every additional package built on raw `pgx.Query`/`Scan` before sqlc
exists makes the eventual migration cost strictly larger, the same
asymmetric-migration-cost argument AD-16 already made for chi over stdlib
`ServeMux`. Concretely this means, as part of this story: `sqlc.yaml` at
`backend/`, query files under `internal/pokedex/queries/*.sql`, generated
code into an unexported package the `pokedex` store adapter wraps (AD-13 —
the generated `pgx`-flavored types stay behind that adapter, domain code
never sees them directly), and a `make generate` Makefile target.

This is a toolchain decision bigger than an endpoint contract and should get
explicit sign-off (architect or human) before implementation starts, not be
decided unilaterally inside this story's PR.

## I/O & edge-case matrix

| Scenario | Input | Expected output |
|---|---|---|
| Species exact match | `GET /v1/species?name=pikachu` | `200`, `results` has 1 entry, `identifier: "pikachu"` |
| Species substring, multiple formes | `GET /v1/species?name=meowth` | `200`, `results` has 3 entries (base + Alolan + Galarian) |
| Species no match | `GET /v1/species?name=zzzznotarealname` | `200`, `{"results": [], "count": 0}` |
| Species missing `name` | `GET /v1/species` | `400`, `invalid_request` |
| Species missing `name`, empty value | `GET /v1/species?name=` | `400`, `invalid_request` |
| Move, default data set | `GET /v1/moves?name=thunderbolt` | `200`, `data_set: "gen-9"`, 1 result |
| Move, explicit valid data set | `GET /v1/moves?name=thunderbolt&data_set=gen-9` | `200`, same as above |
| Move, unknown data set | `GET /v1/moves?name=thunderbolt&data_set=champions` | `400`, `unknown_data_set` (until Champions data actually loads, per 1.1c's "Open for later") |
| Ability search | `GET /v1/abilities?name=static` | `200`, `data_set: "gen-9"`, 1 result |
| Item search | `GET /v1/items?name=leftovers` | `200`, `data_set: "gen-9"`, 1 result |
| HEAD species | `HEAD /v1/species?name=pikachu` | `200`, empty body, headers as GET |
| Unknown path under `/v1` | `GET /v1/nope` | `404`, `not_found` (global catch-all, unchanged, not this story's concern) |
| Wrong method | `POST /v1/species` | `405`, `method_not_allowed`, `Allow: GET, HEAD` |
| Postgres unreachable mid-query | any of the above | `500`, `internal_error` |

## Open questions requiring a decision before implementation

1. **`data_set` default and resolution mechanism** — is `Config.DefaultDataSet`
   defaulting to `"gen-9"` acceptable, or does this need a real "current data
   set" concept (e.g. a small `current_data_set` row, or a per-environment
   config) instead of a hardcoded fallback baked into app config? This repo
   has never defined that concept before now.
2. **`name` match semantics** — substring vs. exact vs. prefix. This doc
   recommends substring for the reasons above, but the AC text is genuinely
   ambiguous on whether "matching record(s)" anticipates the regional-forme
   grouping case or is just generic API phrasing.
3. **sqlc adoption timing** — start it in this story (recommended above) or
   explicitly defer with raw `pgx` for this one slice and accept the
   migration-cost tradeoff? This is a toolchain decision, not just an
   endpoint contract, and should get explicit sign-off.
4. **`unknown_data_set` status code** — this doc proposes 400 (bad query
   param value) over 404 (reserved here for path/route-level not-found).
   Worth confirming that convention now since every later versioned
   capability endpoint will reuse it.
