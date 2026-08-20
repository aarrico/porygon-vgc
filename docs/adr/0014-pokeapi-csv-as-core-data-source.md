# AD-14: PokeAPI CSV dump as the core reference data source

**Date**: 2026-08-19
**Status**: accepted

## Context

No source was named for core Pokemon reference data (species/moves/abilities/items) until Story 1.1 planning. `pokedb.org` was evaluated and rejected: its export carries no structured effect tables (`move_meta`/`move_stat_changes`/`effects` don't exist there), which would force a second data source for structured effect search (CAP-1). This is a distinct decision from AD-4's tournament-data-source open question (CAP-4/CAP-7) — this AD is about species/moves/abilities/items reference data only.

## Decision

Reference data source is the `PokeAPI/pokeapi` CSV dump (`data/v2/csv/`), pinned to a commit SHA. PokeAPI ships `move_meta.csv` (ailment, drain, healing, crit_rate, flinch_chance, stat_chance) and `move_meta_stat_changes.csv` (`move_id, stat_id, change`) — the latter is exactly FR1's "Attack +2 stages" predicate shape. Verified at decision time: 1025 species, gens I–IX.

## Consequences

### Positive
- Structured effect data (CAP-1's core requirement) is available directly from the source with no separate scraping/parsing effort.

### Negative
- Pinned to a specific commit SHA — updates to PokeAPI's data require a deliberate re-pin, not automatic pickup.
