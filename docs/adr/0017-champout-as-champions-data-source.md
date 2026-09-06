# AD-17: Champions reference data from the projectpokemon/champout game dump

**Date**: 2026-09-05
**Status**: accepted

## Context

Competitive play runs on Pokemon Champions, modelled as the `champions` Data Set layered on `gen-9` (AD-15). AD-14's source, PokeAPI, carries no Champions data at all, and Champions changes move values in bulk: PP is recomputed and capped at 20, and power, accuracy and type differ on dozens of moves. The move, ability and item rosters are also a subset of mainline's. Until a Champions Data Set is loaded, every read defaults to Scarlet/Violet values for a game that does not use them.

Candidates evaluated: Pokemon Showdown's `data/mods/champions` (delta files in TypeScript mixing data with engine code; needs a parser or a Node toolchain), community JSON dexes (`otterlyclueless/pokemon-champions-data`, stale since launch; `pmwl0128/pokemon_champion_agent`, a single-regulation snapshot), championsbattledata.com (usage statistics and base stats only; no move mechanics), and `projectpokemon/champout`, Project Pokemon's dump of the game's own downloadable data tables and localized text.

## Decision

The Champions Data Set is loaded from the `projectpokemon/champout` dump, pinned to a commit SHA and vendored like the PokeAPI dump (`backend/internal/etl/data/champout/`). It is the game's own data rather than a simulator's or a community's transcription of it, it is plain JSON, and it has been re-dumped on each game data update.

The dump supplies what it has: move type, damage class, power, accuracy, PP, priority, availability and inflicted ailment; the ability roster via species ability slots; the item roster. Columns it does not carry (generation, target, effect categories and chances, stat changes) are inherited from the parent Data Set row with the same identifier, per AD-15's materialization. Identifiers come from the dump's English names through a fixed slug rule; the id and name alignment with PokeAPI is pinned by tests.

Species are not loaded from the dump: `species` is not Data Set versioned (AD-15) and Champions does not alter base stats or typing. `available = 0` moves are not loaded anywhere; regulation legality stays in the deferred policy layer (NFR2).

## Considered Options

### Showdown `data/mods/champions` deltas
- **Cons**: TypeScript delta files with function-valued effects; either a fragile hand parser or a Node toolchain for about sixty rows; encodes what the simulator implements, not what the game ships.
- **Why not**: the game dump is upstream of it.

### Community JSON dexes
- **Cons**: unmaintained or single-snapshot; no lineage to the game data; unverified corrections.
- **Why not**: no way to know what a value's provenance is.

### championsbattledata.com
- **Cons**: usage statistics only; no move mechanics.
- **Why not**: wrong kind of data for this decision. It remains the leading candidate for the CAP-4/CAP-7 usage source.

## Consequences

### Positive
- Champions values are the game's, with provenance to a pinned dump.
- One loader, one transaction, one re-pin procedure for both sources.

### Negative
- Stat-change magnitudes and effect chances are inherited from mainline, since the buff table the dump references is not part of it. A Champions-only change to one of those is invisible until the dump grows that table.
- A new regulation needs a new dump from Project Pokemon and a re-pin; a roster entry with no PokeAPI parent row fails the load and needs a deliberate decision, not a fallback.
