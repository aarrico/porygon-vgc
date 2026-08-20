# porygon-vgc

Pokemon VGC (Video Game Championships) team-building tool: structured effect search, damage calculation, team recommendations, turn-based battle simulation, and spaced-repetition battle review, behind a versioned HTTP API.

## Language

### Core reference data

**Species**:
A Pokemon species entry — base stats, typing, ability slots. Not versioned per Regulation; only per generation.
_Avoid_: Pokemon (ambiguous — also used for a species instance on a team), dex entry.

**Move**:
An attack or technique a Pokemon can use. Carries structured effect data (see Effect Predicate), not free text.
_Avoid_: attack, technique (informal, not the schema term).

**Ability**:
A passive trait a Pokemon can have.

**Item**:
A holdable item a Pokemon can carry into battle.

**Effect Predicate**:
A structured, queryable description of what a move/ability/item does mechanically — e.g. "Attack +2 stages." Represented as typed columns and stat-change rows, never parsed from free text at query time.
_Avoid_: effect text, description.

### Versioning

**Data Set**:
A named, versioned snapshot of core reference data (moves, abilities, items — not species/type/stat/generation/nature). A Data Set may inherit from a parent Data Set, with child rows resolved at load time into a materialized snapshot. A Data Set is a data snapshot, never a legality policy.
_Avoid_: generation (a property of a row, not a versioning mechanism), ruleset.

**Regulation**:
A competitive ruleset governing which species, formes, and items are tournament-legal (e.g. Regulation Set M-A, M-B). Regulation/legality data is explicitly excluded from the core schema — it is a separate, later, swappable policy layer, not a schema column.
_Avoid_: format (used loosely elsewhere in project history; Regulation is the canonical term, matching the game's own vocabulary).

### Team building & analytics

**Saved Team**:
A user-persisted team composition — a set of species with optional moves/items — referencing core reference data by real foreign key, not a denormalized copy.

**Recommendation Score**:
A precomputed, sample-size-adjusted score reflecting a species' real competitive viability from official tournament data. Shrunk toward a prior/confidence threshold so a small sample doesn't read as more viable than it is. Always precomputed by a scheduled batch job — never computed live on the request path.
_Avoid_: usage rate (the raw, unadjusted figure this deliberately corrects for).

**Off-meta-but-viable pick**:
A species with a high Recommendation Score despite low raw usage — visibly distinguished from meta-conforming picks in any recommendation result, never silently mixed in.

**Coverage Gap**:
A weakness or bad matchup identified for one specific Saved Team by the analytics service. Computed live per request — there is no fixed universe of teams to precompute this against.

### Battle & review

**Battle Log**:
A persisted record of one simulated battle — an ordered sequence of Battle Events.

**Battle Event**:
One event within a Battle Log's turn resolution (move used, damage dealt, status applied, ability triggered). Carries an explicit monotonic Sequence number.
_Avoid_: event (too generic outside battle-simulation context).

**Sequence**:
The monotonic integer, scoped per battle, that establishes Battle Event order. Never inferred from event UUID — UUIDs on events are identity only.

**Review Item**:
A decision point from a past battle, flagged by the user as suboptimal and tracked for spaced-repetition drilling until answered correctly enough times to graduate.

### Product structure

**Capability**:
One of the seven top-level product capabilities (CAP-1 through CAP-7) defined in [docs/PRD.md](docs/PRD.md) — the unit the architecture, ADRs, and epics are all organized around.
