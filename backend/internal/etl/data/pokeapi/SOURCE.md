# Vendored PokeAPI CSV dump

Source: [PokeAPI/pokeapi](https://github.com/PokeAPI/pokeapi), `data/v2/csv/`
Pinned commit: `c40a25c6544b97334a1ae8b1965a378fa3317c28`

Vendored rather than fetched (AD-14). The 18 files total ~350 KB, so the ETL
stays hermetic — no egress, no rate limit, byte-reproducible — and `go:embed`
puts them inside the binary, which is why they live under the package rather
than at `backend/data/`.

To re-pin, set `SHA` and re-run from this directory:

```fish
set SHA <new-commit-sha>
for f in generations types stats natures pokemon pokemon_species pokemon_stats \
         pokemon_types moves move_damage_classes move_targets move_meta \
         move_meta_categories move_meta_ailments move_meta_stat_changes \
         abilities items item_categories
    curl -sfS -o $f.csv \
      "https://raw.githubusercontent.com/PokeAPI/pokeapi/$SHA/data/v2/csv/$f.csv"
end
```

Then update the SHA above and run `go test ./internal/etl/...`. The loader
names every column it reads, so a rename or removal upstream fails with
`missing column "x"` rather than loading NULLs.

## Known shape of this data

Facts the loader depends on, verified at pin time:

| Fact | Value |
|---|---|
| Moves | 937 rows; the 18 with `id > 10000` are Colosseum/XD shadow moves and are the only ones with blank `pp` — skipped |
| Abilities | 373 rows; `is_main_series = 1` selects 313 and exactly matches `id < 10000` |
| Pokemon | 1351 rows, 1025 with `is_default = 1`, one per National Dex number |
| Accuracy | 3 moves use `0` where most never-miss moves use blank; normalized to NULL |
| Healing | Ranges −33..50 — negative is self-damage (Struggle, Clangorous Soul) |
| Natures | 5 of 25 set `increased_stat_id == decreased_stat_id`; normalized to NULL |
| Items | 2223 rows, 2222 distinct identifiers — `roseli-berry` appears as ids 723 and 2279, differing only in `cost`, which is not stored. The upsert collapses them |
