# etl

Loads the pinned PokeAPI CSV dump (AD-14) into the core reference schema
(species/moves/abilities/items) in one transaction, run by `cmd/etl` under
the ETL role — the only role permitted to write these tables (AD-6).
Migrations are a separate, also ETL-owned concern applied via
`golang-migrate` (AD-12); this package never runs one.

Not a capability package — no dependency on `pokedex` or any other
`internal/*` package. Governed by AD-6, AD-12, AD-14 — see
[docs/adr/](../../../docs/adr/).
