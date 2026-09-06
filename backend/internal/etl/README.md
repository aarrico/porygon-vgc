# etl

Loads the pinned PokeAPI CSV dump (AD-14) into the core reference schema
(species/moves/abilities/items) as the base Data Set, then the Champions
Data Set on top of it from the pinned game dump (AD-17, `champions.go`,
`data/champout/SOURCE.md`), in one transaction, run by `cmd/etl` under
the ETL role — the only role permitted to write these tables (AD-6).

Champions is a child of the base Data Set (AD-15): its move, ability and
item rosters and its move values come from the game; generation, target
and effect rows the dump does not carry are inherited from the parent row
with the same identifier, and a roster entry with no parent row fails the
load rather than guessing.
Migrations are a separate, also ETL-owned concern applied via
`golang-migrate` (AD-12); this package never runs one.

Not a capability package — no dependency on `pokedex` or any other
`internal/*` package. Governed by AD-6, AD-12, AD-14 — see
[docs/adr/](../../../docs/adr/).
