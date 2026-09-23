# Vendored Champions game data (champout)

Source: [projectpokemon/champout](https://github.com/projectpokemon/champout),
Project Pokemon's dumps of Pokemon Champions' downloadable data (`masterdata/`)
and localized text (`rom-txt/`). MIT.
Pinned commit: `0c1141656e1a66ae304ac3ee1e7126a00914d1f2` (masterdata v1.1.0, Regulation M-B)

Vendored for the same reasons as the PokeAPI dump next door: hermetic,
byte-reproducible loads with no egress. Six files, ~1.3 MB.

Table names are Game Freak's Japanese ones: waza = move, tokusei = ability,
personal = species.

| File | Rows | Used for |
|---|---|---|
| `waza.json` | 920 moves | type, damage class, power, accuracy, PP, priority, `available`, ailment (`con_ref`) |
| `item.json` | 148 items | the Champions item roster |
| `personal.json` | 361 species/forms | the ability roster (`toku0..2`); species are not Data Set versioned (AD-15) |
| `wazaname.json`, `tokusei.json`, `itemname.json` | | English names, turned into PokeAPI identifiers by `slug` |

To re-pin, set `SHA` and re-run from this directory:

```fish
set SHA <new-commit-sha>
for f in masterdata/waza masterdata/item masterdata/personal \
         rom-txt/usa/wazaname rom-txt/usa/itemname rom-txt/usa/tokusei
    curl -sfS -o (basename $f).json \
      "https://raw.githubusercontent.com/projectpokemon/champout/$SHA/$f.json"
end
```

Then update the SHA above and run `go test ./internal/etl/...`, which pins
the id and name alignment against the PokeAPI dump.

## Known shape of this data

| Fact | Value |
|---|---|
| Move ids | Identical to PokeAPI's; only Nihil Light (920) is new to Champions, and it is `available = 0` in this dump |
| `available` | 497 of 920 moves are in Champions; the rest are loaded nowhere |
| `pp` | The in-game Champions maximum: mainline PP through `(pp/5 + 1) * 4`, capped at 20 |
| `accuracy` | `101` means the move cannot miss; stored as NULL |
| `power` | `0` on a status move, stored as NULL; `1` on variable-power moves, stored as is |
| `type` | Game Freak's order, PokeAPI's id minus one |
| `category` | 0 physical, 1 special, 2 status |
| `con_ref` | Inflicted ailment: 1 paralysis, 2 freeze, 3 burn, 4 poison, 5 bad poison, 6 sleep; comma-separated when random |
| `buf_ref` | Stat-change references; the buff table they point at is not in the dump, so stat changes are inherited from the parent Data Set |
| Names | Every available move, every referenced ability and every item slugs to an existing PokeAPI identifier at pin time |
