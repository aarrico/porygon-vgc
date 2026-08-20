# Pokémon Champions: how regulations alter reference data

Research date: 2026-08-19. Scope: Pokémon Champions (launched 2026-04-08), Regulation Sets M-A and M-B.

## Answer

Inheritance from a mainline generation is real, and it is two layers deep.

Layer 1 is Scarlet/Violet (Gen 9). Serebii's Champions pages are published explicitly as diffs against S/V, listing per-move old-vs-new power, accuracy, PP and type ([Updated Attacks](https://www.serebii.net/pokemonchampions/updatedattacks.shtml)). Pokémon Showdown implements Champions as a Gen 9 mod (`data/mods/champions`, `gen: 9`, no `pokedex.ts`) that overrides 259 moves, 13 abilities and 258 items on top of stock Gen 9 data ([source](https://github.com/smogon/pokemon-showdown/tree/master/data/mods/champions)).

Layer 2 is the regulation. Showdown models Reg M-A as a child mod `championsregma` with `inherit: 'champions'` ([scripts.ts](https://raw.githubusercontent.com/smogon/pokemon-showdown/master/data/mods/championsregma/scripts.ts)), overriding only: one ability behaviour (Spicy Spray), one move behaviour (Dire Claw), 31 item availabilities, and species legality flags.

Entity types that demonstrably need per-regulation versioning: **learnsets** (Incineroar lost U-turn and Knock Off in M-B — [pokemon.com](https://www.pokemon.com/us/features/pokemon-champions-regulation-m-b-double-battles-overview)), **moves** (Rage Fist now resets on switch-out), **abilities** (Eelevate and Fire Mane added), **items** (31 newly legal), and **species/forme legality** (21 species + 16 Megas added).

Entity types with no evidence of per-regulation change: **base stats, typing, ability slots, natures, type chart**. Showdown's Champions mods contain no `pokedex.ts`, `natures.ts` or `typechart.ts`; species deltas are expressed purely as `isNonstandard`/`tier` in `formats-data.ts`. A claim that Champions raises all base stats by +75 HP / +20 ([issue #2](https://github.com/otterlyclueless/pokemon-champions-data/issues/2)) is a misreading of the level-50 stat formula, not a data change.

So: version legality, learnsets, moves, abilities and items per regulation; treat species identity data as generation-scoped, not regulation-scoped.

---

## Findings

### 1. Entity types actually changed by a Champions regulation update

The only regulation transition to date is M-A → M-B (2026-06-17), shipped with game patch v1.1.0 ([Serebii, Patches & Updates](https://www.serebii.net/pokemonchampions/patch.shtml)).

| Entity type | Changed by a regulation? | Concrete examples (regulation) |
|---|---|---|
| Learnsets | Yes | M-B: Incineroar loses U-turn and Knock Off ([pokemon.com](https://www.pokemon.com/us/features/pokemon-champions-regulation-m-b-double-battles-overview)); Scrafty loses Parting Shot; Annihilape loses Final Gambit; Grimmsnarl loses False Surrender and Thunder Wave; Metagross loses Heavy Slam; Overqwil loses Mortal Spin; Sceptile gains Earth Power; Scolipede gains Leech Life and Trailblazer ([Insider Gaming patch notes](https://insider-gaming.com/pokemon-champions-june-17-update-patch-notes/)) |
| Moves | Yes | M-B: Rage Fist's accumulated power now resets when Annihilape switches out ([Insider Gaming](https://insider-gaming.com/pokemon-champions-june-17-update-patch-notes/)). Showdown encodes a Dire Claw secondary-effect difference between M-A and M-B ([championsregma/moves.ts](https://raw.githubusercontent.com/smogon/pokemon-showdown/master/data/mods/championsregma/moves.ts)) |
| Abilities | Yes | M-B introduces two brand-new abilities: **Eelevate** (Mega Eelektross) and **Fire Mane** (Mega Pyroar) ([The Game Haus](https://thegamehaus.com/pokemon-champions/all-new-pokemon-champions-mega-abilities/2026/06/17/)). Showdown encodes a Spicy Spray behavioural difference between M-A and M-B ([championsregma/abilities.ts](https://raw.githubusercontent.com/smogon/pokemon-showdown/master/data/mods/championsregma/abilities.ts)) |
| Items | Yes | 31 items become legal in M-B, per the M-A mod that gates them off: 16 Mega Stones (Sceptilite, Blazikenite, Swampertite, Mawilite, Metagrossite, Staraptite, Scolipite, Scraftinite, Eelektrossite, Pyroarite, Malamarite, Barbaracite, Dragalgite, Falinksite, Raichunite X, Raichunite Y) and 15 held items (Life Orb, Wide Lens, Zoom Lens, Muscle Band, Wise Glasses, Expert Belt, Light Clay, Metronome, Shed Shell, Iron Ball, Big Root, Damp Rock, Heat Rock, Icy Rock, Smooth Rock) ([championsregma/items.ts](https://raw.githubusercontent.com/smogon/pokemon-showdown/master/data/mods/championsregma/items.ts)) |
| Species availability | Yes | M-B adds 21 base-form species (Vileplume, Qwilfish, Sceptile, Blaziken, Swampert, Mawile, Metagross, Staraptor, …) and 16 new Mega Evolutions ([pokemon.com announcement](https://www.pokemon.com/us/pokemon-news/regulation-set-m-b-kicks-off-a-new-ranked-battles-season-and-battle-pass-in-pokemon-champions); [Victory Road](https://victoryroad.pro/champions-regulations/)) |
| Species base stats | **No evidence** | See §2 |
| Species typing | **No evidence** | See §2 |
| Ability slots | **No evidence** | See §2 |
| Natures | **No evidence** | No source documents a nature change. Showdown's Champions mods ship no `natures.ts`; the community dataset counts 25 natures, matching mainline ([version.json](https://raw.githubusercontent.com/otterlyclueless/pokemon-champions-data/main/meta/version.json)) |
| Type chart | **No evidence** | Showdown's Champions mods ship no `typechart.ts` ([mod contents](https://github.com/smogon/pokemon-showdown/tree/master/data/mods/champions)) |

Reconciling the conflicting item counts in secondary coverage: Victory Road's "16 items" is the Mega Stones, Game8's "15 new items" is the held items, and Showdown's 31 is the union. Both partial figures are consistent with the mod file.

Caution: Game8's [List of Changes](https://game8.co/games/Pokemon-Champions/archives/593893) presents Champions-vs-S/V move rebalances (Beak Blast 100→120, Dire Claw 50%→30%, Moonblast 30%→10%, Make It Rain accuracy 100→95, Unseen Fist nerf) under a Regulation M-B heading. Those same entries appear on Serebii's [Updated Attacks](https://www.serebii.net/pokemonchampions/updatedattacks.shtml) and [Updated Abilities](https://www.serebii.net/pokemonchampions/updatedabilities.shtml) pages as launch-era Champions-vs-Scarlet/Violet deltas, i.e. they predate M-B. Treat Game8's attribution as unreliable; see Open/unverified.

### 2. Has any regulation changed base stats, typing, or ability slots?

No, on the evidence available. Changes are confined to moves, abilities, items, learnsets, and legality.

- Showdown's `data/mods/champions` directory contains `abilities.ts`, `conditions.ts`, `formats-data.ts`, `items.ts`, `learnsets.ts`, `moves.ts`, `rulesets.ts`, `scripts.ts` — and **no `pokedex.ts`** ([directory listing](https://github.com/smogon/pokemon-showdown/tree/master/data/mods/champions)). Base stats, types and ability slots are therefore inherited verbatim from Gen 9.
- `data/mods/championsregma` contains only `abilities.ts`, `formats-data.ts`, `items.ts`, `moves.ts`, `scripts.ts` ([directory listing](https://github.com/smogon/pokemon-showdown/tree/master/data/mods/championsregma)). Diffing its `formats-data.ts` against the parent mod's shows the entire species-level M-A/M-B delta is `isNonstandard: "Past"` + `tier: "Illegal"` flips — a legality gate, never a stat or type edit.
- Independent cross-check: the community Champions dex records Abomasnow as HP 90 / Atk 92 / Def 75 / SpA 92 / SpD 85 / Spe 60, Grass/Ice — identical to Scarlet/Violet ([champions_dex.json](https://github.com/pmwl0128/pokemon_champion_agent/blob/main/.claude/skills/pokemon-champions-dex/data/champions_dex.json)).
- New Mega formes with novel typings (e.g. Mega Staraptor as Fighting/Flying, [The Game Haus](https://thegamehaus.com/pokemon-champions/all-new-pokemon-champions-mega-abilities/2026/06/17/)) are **new entities**, not mutations of an existing species row. The base species keeps its Normal/Flying typing.

On the "+75 HP / +20" claim: [issue #2](https://github.com/otterlyclueless/pokemon-champions-data/issues/2) and [PR #3](https://github.com/otterlyclueless/pokemon-champions-data/issues/3) on the otterlyclueless dataset assert Champions raises every Pokémon's base stats by +75 HP and +20 to all other stats. Showdown implements the same numbers, but as a `statModify` override in [`champions/scripts.ts`](https://raw.githubusercontent.com/smogon/pokemon-showdown/master/data/mods/champions/scripts.ts):

```ts
if (statName === 'hp') return stat + evs + 75;
stat = stat + evs + 20;
```

Derivation (mine, not quoted from a source): the mainline level-50 formula with 31 IVs is `floor((2*base + 31 + ev/4) * 50/100) + 50 + 10` for HP, which reduces to `base + ev/8 + 75`; for other stats `floor((2*base + 31 + ev/4) * 50/100) + 5` reduces to `base + ev/8 + 20`. The `+75`/`+20` constants are the level-50 formula collapsed, with 1 SP standing in for 8 EVs — consistent with the documented "1 SP ≈ 8 EVs" conversion ([ChampDex](https://champdex.com/guides/format-rules)). The base stats themselves are unchanged. Both issues remain open and unmerged as of 2026-08-19.

### 3. Regulation sequence to date

| Regulation | In-game ranked window | Play! Pokémon events | Notes |
|---|---|---|---|
| (Reg I, S/V) | — | From 2026-04-01 while Scarlet/Violet remained the competitive platform ([pokemon.com](https://www.pokemon.com/us/pokemon-news/play-pokemon-competitions-transition-to-pokemon-champions-on-april-and-may-2026)) | Mainline, not Champions |
| **M-A** | 2026-04-08 02:00 UTC – 2026-06-17 01:59 UTC ([Bulbapedia](https://bulbapedia.bulbagarden.net/wiki/Regulation_Sets_in_Pok%C3%A9mon_Champions)) | Global Challenge I 2026-05-01→04; in-person from Indianapolis Regionals 2026-05-29 ([pokemon.com](https://www.pokemon.com/us/pokemon-news/play-pokemon-competitions-transition-to-pokemon-champions-on-april-and-may-2026)) | First Champions ruleset; launch-day data set |
| **M-B** | 2026-06-16 19:00 PDT – 2026-09-01 18:59 PDT ([pokemon.com](https://www.pokemon.com/us/pokemon-news/regulation-set-m-b-kicks-off-a-new-ranked-battles-season-and-battle-pass-in-pokemon-champions)) = 2026-06-17 – 2026-09-02 UTC ([Bulbapedia](https://bulbapedia.bulbagarden.net/wiki/Regulation_Sets_in_Pok%C3%A9mon_Champions)) | Through 2026-09-09 per [Victory Road](https://victoryroad.pro/champions-regulations/) and [Serebii](https://www.serebii.net/pokemonchampions/rankedbattle/regulationm-b.shtml); used for the 2026 World Championships | Delivered by game patch v1.1.0, 2026-06-17 ([Serebii](https://www.serebii.net/pokemonchampions/patch.shtml)) |

What changed in each:

- **M-A** — the launch baseline. Everything Champions-specific relative to Scarlet/Violet (move rebalances, six new abilities, the SP stat system, changed status conditions) arrived here, not via a later regulation.
- **M-B** — see §1. Additive: 21 base species, 16 Megas, 31 items, 2 new abilities (Eelevate, Fire Mane). Subtractive: learnset removals on Incineroar, Scrafty, Annihilape, Grimmsnarl, Metagross, Overqwil, and the Rage Fist switch-out reset.

Non-regulation patch: v1.0.3 (2026-04-23) was bug fixes only — Leech Seed description, Unnerve malfunction, Mega Evolution menu move selection, speed-item ability ordering ([Serebii](https://www.serebii.net/pokemonchampions/patch.shtml)).

No Regulation M-C has been announced in any source found as of 2026-08-19.

### 4. Baseline: inherited from a mainline generation, or independent?

Inherited, then diverged globally, then gated per regulation.

- Serebii publishes Champions move data as an explicit two-column diff, "S/V" versus "Champions" ([Updated Attacks](https://www.serebii.net/pokemonchampions/updatedattacks.shtml)) — e.g. Growth changes type Normal→Grass, Crabhammer 90→95 accuracy plus a crit-ratio boost, Bone Rush 25→30 power, Beak Blast 100→120 power with PP 15→8, Snap Trap changes type Grass→Steel, Make It Rain 100→95 accuracy with a doubled Sp. Atk drop. The comparison baseline is stated as Scarlet & Violet.
- Showdown's Champions mod declares `gen: 9` with no `inherit` key, so it layers on stock Gen 9 data files ([champions/scripts.ts](https://raw.githubusercontent.com/smogon/pokemon-showdown/master/data/mods/champions/scripts.ts)). Override counts: 259 moves, 13 abilities, 258 items, plus a global `init()` clamp of every move's PP to a maximum of 20.
- The otterlyclueless dataset's own provenance record names its sources as "Pokemon Showdown (base stats, abilities, types)" plus "Serebii.net Champions pages" for corrections, and quantifies the Champions-vs-mainline move delta as 399 PP changes, 12 power changes, 2 accuracy changes, and 406 of 900 moves absent from Champions ([version.json](https://raw.githubusercontent.com/otterlyclueless/pokemon-champions-data/main/meta/version.json)). Its README states plainly: "The base data comes from Showdown's open-source files, which are accurate for the main series games. Pokemon Champions may have modified learnsets, stat values, ability assignments, and mega evolution parameters."
- Champions also adds six abilities that do not exist in mainline: Piercing Drill, Dragonize, Eelevate, Mega Sol, Fire Mane, Spicy Spray ([Serebii, New Abilities](https://www.serebii.net/pokemonchampions/newabilities.shtml)). These map one-to-one onto the `isNonstandard: null` re-enablements in [champions/abilities.ts](https://raw.githubusercontent.com/smogon/pokemon-showdown/master/data/mods/champions/abilities.ts).
- The Z-A pipeline matters for provenance: Mega Evolutions debuting in Pokémon Legends: Z-A entered Champions with abilities that Z-A itself never assigned, because Z-A had no ability system ([Insider Gaming](https://insider-gaming.com/new-pokemon-champions-mega-abilities/)). So the Champions dex draws forme data from more than one mainline title.

Practical modelling consequence: a Champions row is `Gen9Base ⊕ ChampionsDelta ⊕ RegulationDelta`. Only the third term needs regulation-scoped versioning.

### 5. Machine-readable / structured sources

See the Source assessment table below.

### 6. Does Pokémon Showdown carry Champions data?

Yes, and it is the only source found that models per-regulation deltas structurally.

- `data/mods/` contains both `champions` and `championsregma` ([mods directory](https://github.com/smogon/pokemon-showdown/tree/master/data/mods)).
- `champions` is the *current* (M-B) data layer. `championsregma` is a child mod: `export const Scripts = { inherit: 'champions', gen: 9 }` ([source](https://raw.githubusercontent.com/smogon/pokemon-showdown/master/data/mods/championsregma/scripts.ts)). The older regulation is therefore expressed as a patch that walks the newer one *backwards*.
- Delta representation per entity type:
  - **Species**: `formats-data.ts` entries flip to `isNonstandard: "Past"` + `tier: "Illegal"`. No stat/type/ability data is touched.
  - **Items**: `items.ts` entries carry `inherit: true` plus `isNonstandard: "Past"` or `"Future"` to remove them from the M-A pool. (The Past/Future split is inconsistent — Mega Stones use "Future", held items use "Past" — but both render the item illegal.)
  - **Moves / Abilities**: full `inherit: true` object overrides that restore the older behaviour (`direclaw`, `spicyspray`).
  - **Learnsets**: **not versioned.** `championsregma` ships no `learnsets.ts`, so Showdown's M-A ladder uses M-B learnsets. Verified: `champions/learnsets.ts` gives Incineroar 78 moves including Parting Shot but excluding U-turn and Knock Off; Scrafty has Knock Off but not Parting Shot; Annihilape has no Final Gambit; Metagross has no Heavy Slam; Overqwil has no Mortal Spin — i.e. the post-M-B state, applied to both ladders.
- Format registrations in `config/formats.ts`: `[Gen 9 Champions] VGC 2026 Reg M-A` and `Reg M-A (Bo3)` use `mod: 'championsregma'`; `VGC 2026 Reg M-B`, `Reg M-B (Bo3)`, `OU`, `UU`, `BSS Reg M-B`, Random Battle, Draft and Custom Game use `mod: 'champions'` ([config/formats.ts](https://github.com/smogon/pokemon-showdown/blob/master/config/formats.ts)).
- Champions-specific engine behaviour lives in `champions/scripts.ts`: the `statModify` override, a global PP cap of 20, removal of Trick Room speed underflow, and "don't revert Mega Evolutions after fainting".

---

## Source assessment (question 5)

| Source | Coverage | Currency (Reg M-B present?) | Format | License | Cadence | Verdict |
|---|---|---|---|---|---|---|
| [smogon/pokemon-showdown](https://github.com/smogon/pokemon-showdown/tree/master/data/mods) — `data/mods/champions`, `data/mods/championsregma` | Moves, abilities, items, learnsets, species legality/tiers, rulesets, engine scripts. No pokedex/natures/typechart overrides (inherits Gen 9) | **Yes** — `champions` *is* M-B; `championsregma` is the M-A back-patch. Actively tracks current regulation | TypeScript modules (`export const X: Modded…DataTable`), consumable via `@pkmn/dex` or direct parse | MIT | Continuous; Champions mods landed with the format and were extended for M-B | **Best available.** Only source with a structural per-regulation delta model. Caveat: learnsets are not regulation-versioned |
| [otterlyclueless/pokemon-champions-data](https://github.com/otterlyclueless/pokemon-champions-data) | 258 Pokémon, 900 moves (494 flagged `inChampions`), 191 abilities, 583 items, 25 natures, 18×18 type chart, learnsets, SP mechanics docs | **No.** Last push 2026-04-16 (v1.1.0), four months stale; predates M-B entirely. Open issues report missing Gholdengo, missing Rotom/Tauros formes, incomplete Delphox learnset | Flat JSON, one file per entity type, raw.githubusercontent fetchable | CC BY 4.0 (GitHub detects `NOASSERTION`; LICENSE file is CC BY 4.0 text) | Two commits, both 2026-04-16. Effectively unmaintained. 16 stars, 6 open issues, 0 closed | **Do not depend on.** Flat single-snapshot schema with no regulation dimension. Also carries an unresolved incorrect-base-stats PR |
| [pmwl0128/pokemon_champion_agent](https://github.com/pmwl0128/pokemon_champion_agent) | `champions_dex.json` + SQLite: 315 Pokémon/formes, 496 moves, 201 abilities, 148 items, 19,617 learnset rows, typing, base stats, Mega Stone mappings, zh/en/ja names. Plus per-season usage/meta JSON | **Yes** — README stamped "M-5 / M-B · updated 2026-08-19"; dex `built_at` 2026-07-19. Meta files exist for seasons M-3/M-4/M-5 and `opponent_cache/M-B_*` | JSON + SQLite, shipped inside Agent Skill directories; also a `data-latest` release bundle | MIT | Active (pushed 2026-08-19); "updatable snapshot tied to a season and regulation" | **Currently the freshest structured dex.** But the dex itself is a single current-regulation snapshot — only *usage* data is season-partitioned. No historical dex deltas |
| [pokemon-zone.com/champions](https://www.pokemon-zone.com/champions/regulations/) | Per-regulation pages, per-move and per-Pokémon pages, usage stats | Yes (dedicated `/regulations/m-b/` page) | HTML only; Cloudflare interstitial blocks non-browser clients (verified: returns "Enable JavaScript and cookies to continue") | Not stated | Unknown | No machine-readable access. Human reference only |
| [champdex.com](https://champdex.com/guides/format-rules) | Dex, team builder, damage calc, meta rankings, format rules | Yes — states it filters everything "through the current Reg M-B legality list pulled straight from Serebii's Champions page" | HTML | Not stated | Unknown | No API or export documented. Self-declares upstream sources as PokeAPI, Pikalytics, Limitless, Serebii — i.e. a derived aggregator |
| [game8.co Pokémon Champions](https://game8.co/games/Pokemon-Champions/archives/593893) | Change lists, rosters, base-stat lists, per-regulation roster pages | Yes ([M-B roster page](https://game8.co/games/Pokemon-Champions/archives/605482)) | HTML | Proprietary | Frequent editorial updates | Useful for narrative patch context. **Attribution is unreliable** — mixes launch-era Champions-vs-S/V changes into its "Regulation M-B" section |
| [pikalytics.com](https://www.pikalytics.com/pokedex/battledataregmbs3) | Usage/meta only — moves, items, spreads, teammates. Not reference data | Yes; selectable across Champions Reg M-A, M-B, and mainline VGC 2025/2026 regs | HTML; no documented API or export | Proprietary | Continuous (ladder-driven) | Good for usage. Not a reference-data source; does not carry per-regulation *data* deltas |
| [Serebii Champions section](https://www.serebii.net/pokemonchampions/) | The de facto authority on Champions-vs-S/V deltas: Updated Attacks, Updated Abilities, New Abilities, New Mega Abilities, Changed Status Conditions, Patches, per-regulation Ranked Battle pages, Champions Pokédex and Attackdex | Yes (M-B ranked page, v1.1.0 patch page) | HTML | Proprietary | Fast — Serebii announced the Champions Pokédex/Attackdex and the move-changes page as dedicated builds | **Best human/primary-adjacent source** for the Champions-vs-mainline delta. Cited as the verification source by both community datasets above |
| [pokemon.com](https://www.pokemon.com/us/pokemon-news/regulation-set-m-b-kicks-off-a-new-ranked-battles-season-and-battle-pass-in-pokemon-champions) | Official regulation announcements and strategy features | Yes | HTML | Proprietary | Per-regulation | Authoritative for dates and roster additions. Publishes **no structured patch notes** — no official machine-readable changelog of move/ability edits exists |

---

## Open / unverified

- **Ability slots.** No source states whether M-B altered any existing species' ability slots (e.g. granting or removing a Hidden Ability). Showdown's lack of a `pokedex.ts` override is strong negative evidence but not a direct statement. **Unknown.**
- **Dating of the Serebii "Updated Attacks" / "Updated Abilities" lists.** Neither page is versioned or dated per entry, so it cannot be determined from them which entries shipped at launch (M-A) versus in v1.1.0 (M-B). Game8 attributes several to M-B; Serebii frames all of them as Champions-vs-S/V. **Conflict unresolved.**
- **Direction and cause of Showdown's `championsregma` Dire Claw and Spicy Spray overrides.** These could encode a genuine M-A→M-B game change, or a simulator bug fix that was deliberately not backported to the M-A ladder. No commit message or changelog was located. **Unknown.**
- **M-B end date.** pokemon.com says 2026-09-01 18:59 PDT; Serebii and Victory Road say 2026-09-09. Likely an in-game-ranked versus Play!-Pokémon-event distinction, but no source states that explicitly. **Unresolved.**
- **Regulation M-C.** No announcement found as of 2026-08-19. **Unknown.**
- **Natures and type chart.** No source affirmatively states that Champions matches mainline here; the conclusion rests on the absence of override files and on entity counts matching (25 natures, 18 types). **Not positively confirmed.**
- **Exhaustive M-B learnset delta.** The removals/additions listed in §1 come from one secondary outlet's patch-note article. No official or exhaustive learnset diff was located. The list is likely incomplete.
- **Whether Champions applies per-species base-stat edits that Showdown has simply not implemented.** Two independent datasets agree with mainline values, but neither is an authoritative game dump. **Low risk, not formally proven.**
- **APIs.** No public API, bulk export, or documented rate-limited endpoint was found for pokemon-zone, champdex, or pikalytics. Absence of documentation, not proof of absence.
