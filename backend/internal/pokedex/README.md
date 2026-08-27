# pokedex

CAP-1, CAP-2 — structured effect search across species/moves/abilities/items, versioned HTTP API.

Base layer: no dependency on other capability packages (AD-2). Governed by AD-1, AD-2, AD-5, AD-6, AD-13, AD-16 — see [docs/adr/](../../../docs/adr/).

`Router` mounts name search over species/moves/abilities/items (Story 1.2); `Store` is the AD-13 adapter wrapping sqlc-generated queries under `queries/` (generated code in `internal/sqlcgen/`, regenerate with `make generate` after editing a query file). Structured effect-predicate search (`move_stat_change` joins, Story 1.3) is a separate route, not this one.
