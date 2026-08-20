# AD-15: Data Set snapshot/lineage model for versioned reference tables

**Date**: 2026-08-19
**Status**: accepted

## Context

Reference data needs a generation/versioning axis, and — per the Champions regulation research (`docs/research/champions-regulation-data-model.md`) — a Regulation-level axis too: Pokemon Showdown's `champions` mod overrides 259 moves, 13 abilities, and 258 items on stock Gen 9; its `championsregma` child overrides a further 1 ability, 1 move, and 31 item availabilities. The versioning model has to represent that inheritance chain without duplicating unaffected rows at every level.

## Decision

A `data_set` table with a nullable self-referencing `parent_id`. A Data Set is a data snapshot, never a legality policy (NFR2 stays intact). Lineage resolves at **load time** — child = parent rows + deltas, materialized — so read queries stay a single `data_set_id` predicate with no overlay logic at query time. Chains run at least three deep in practice: `gen-9` → `champions` → `champions-reg-ma`.

Only some tables carry `data_set_id`, narrowed by the Champions research:

- **Versioned**: `move`, `move_meta`, `move_stat_change`, `ability`, `item`, and later learnsets — the tables Champions regulations demonstrably override.
- **Not versioned**: `species`, `type`, `stat`, `generation`, `nature` — no Champions regulation changes base stats, typing, ability slots, natures, or the type chart. Adding `data_set_id` here would duplicate 1025 species per regulation for a zero-row delta.

## Considered Options

### Overlay resolution at query time (no materialization)
- **Cons**: every read query needs to walk the parent chain and apply deltas live.
- **Why not**: materializing at load time keeps every read a single `data_set_id` predicate — the complexity is paid once, at load, not on every query.

### `data_set_id` on every core table
- **Cons**: duplicates unaffected rows (e.g. all 1025 species) at every regulation level for zero actual change.
- **Why not**: narrowed by the Champions research to only the tables that are demonstrably regulation-versioned.

## Consequences

### Positive
- Read queries never need overlay/inheritance logic — a single `data_set_id` filter is always correct.

### Negative
- The remaining Champions per-regulation axis — species/Mega legality, expressed in Showdown as `isNonstandard`/`tier` — is *legality*, not a reference-data delta, and NFR2 keeps legality out of the core schema. This means the deferred ruleset/policy layer is load-bearing for Champions support, not optional polish — a legality flag must not leak into a core table as a shortcut when Champions data arrives.
