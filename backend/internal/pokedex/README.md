# pokedex

CAP-1, CAP-2 — structured effect search across species/moves/abilities/items, versioned HTTP API.

Base layer: no dependency on other capability packages (AD-2). Governed by AD-1, AD-2, AD-5, AD-6, AD-13, AD-16 — see [docs/adr/](../../../docs/adr/).

`Router` mounts name search over species/moves/abilities/items (Story 1.2) and structured effect search over moves at `/moves/search` (Story 1.3); `Store` is the AD-13 adapter wrapping sqlc-generated queries under `queries/` (generated code in `internal/sqlcgen/`, regenerate with `make generate` after editing a query file).

Effect search takes one or more `effect=<kind>:<value>[:<arg>]` predicates, combined as a conjunction: `stat:<stat identifier>:<stages>` (stages -6..6, non-zero, `+` optional) matches `move_stat_change`; `ailment:<ailment identifier>` matches `move_meta.ailment`. Grammar, stage range, and limits (8 predicates per request, 64 characters each) are checked in `ParseEffects` before any query runs; stat and ailment identifiers are checked against the loaded vocabulary only when a query returns no rows, the same on-empty pattern `data_set` uses. Effect data exists only for moves (`move_meta`, `move_stat_change`); abilities and items carry no structured effect columns in PokeAPI, so there is no effect route for them.
