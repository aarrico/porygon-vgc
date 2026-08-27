package etl

import (
	"context"
	"fmt"
)

// PokeAPI stat ids. species stores base stats as columns, so these never
// become foreign keys.
const (
	statHP             = "1"
	statAttack         = "2"
	statDefense        = "3"
	statSpecialAttack  = "4"
	statSpecialDefense = "5"
	statSpeed          = "6"
)

var allStats = [...]string{
	statHP, statAttack, statDefense, statSpecialAttack, statSpecialDefense, statSpeed,
}

func (l *loader) loadDataSet(ctx context.Context, identifier string) error {
	const q = `INSERT INTO data_set (identifier) VALUES ($1)
		ON CONFLICT (identifier) DO UPDATE SET identifier = EXCLUDED.identifier
		RETURNING id`
	if err := l.tx.QueryRow(ctx, q, identifier).Scan(&l.dataSet); err != nil {
		return err
	}
	l.rows["data_set"]++
	return nil
}

func (l *loader) loadGenerations(ctx context.Context) error {
	rows, err := readTable("generations", "id", "identifier")
	if err != nil {
		return err
	}
	const q = `INSERT INTO generation (identifier, number) VALUES ($1, $2)
		ON CONFLICT (identifier) DO UPDATE SET number = EXCLUDED.number`
	for _, r := range rows {
		p := r.parse("generations")
		identifier, number := p.text("identifier"), p.num("id")
		if err := p.done(); err != nil {
			return err
		}
		if err := l.exec(ctx, "generation", q, identifier, number); err != nil {
			return err
		}
	}
	l.generation, err = l.indexAndLink(ctx, "generation", rows)
	return err
}

func (l *loader) loadTypes(ctx context.Context) error {
	rows, err := readTable("types", "id", "identifier", "generation_id")
	if err != nil {
		return err
	}
	const q = `INSERT INTO type (identifier, generation_id) VALUES ($1, $2)
		ON CONFLICT (identifier) DO UPDATE SET generation_id = EXCLUDED.generation_id`
	kept := make([]record, 0, len(rows))
	for _, r := range rows {
		main, err := mainSeries("types", r)
		if err != nil {
			return err
		}
		if !main {
			continue // pseudo-type ("unknown", "shadow"), never seen in battle
		}
		kept = append(kept, r)

		p := r.parse("types")
		identifier := p.text("identifier")
		if err := p.done(); err != nil {
			return err
		}
		generation, err := lookup(l.generation, "type.generation_id", r["generation_id"])
		if err != nil {
			return err
		}
		if err := l.exec(ctx, "type", q, identifier, generation); err != nil {
			return err
		}
	}
	l.typ, err = l.indexAndLink(ctx, "type", kept)
	return err
}

func (l *loader) loadStats(ctx context.Context) error {
	rows, err := readTable("stats", "id", "identifier", "is_battle_only")
	if err != nil {
		return err
	}
	const q = `INSERT INTO stat (identifier, is_battle_only) VALUES ($1, $2)
		ON CONFLICT (identifier) DO UPDATE SET is_battle_only = EXCLUDED.is_battle_only`
	for _, r := range rows {
		p := r.parse("stats")
		identifier, battleOnly := p.text("identifier"), p.flag("is_battle_only")
		if err := p.done(); err != nil {
			return err
		}
		if err := l.exec(ctx, "stat", q, identifier, battleOnly); err != nil {
			return err
		}
	}
	l.stat, err = l.indexAndLink(ctx, "stat", rows)
	return err
}

func (l *loader) loadNatures(ctx context.Context) error {
	rows, err := readTable("natures", "id", "identifier", "increased_stat_id", "decreased_stat_id")
	if err != nil {
		return err
	}
	const q = `INSERT INTO nature (identifier, increased_stat_id, decreased_stat_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (identifier) DO UPDATE SET
			increased_stat_id = EXCLUDED.increased_stat_id,
			decreased_stat_id = EXCLUDED.decreased_stat_id`
	for _, r := range rows {
		p := r.parse("natures")
		identifier := p.text("identifier")
		if err := p.done(); err != nil {
			return err
		}

		// PokeAPI spells a neutral nature as the same stat raised and lowered;
		// the schema spells it as neither.
		var increased, decreased *int64
		if up, down := r["increased_stat_id"], r["decreased_stat_id"]; up != down {
			u, err := lookup(l.stat, "nature.increased_stat_id", up)
			if err != nil {
				return err
			}
			d, err := lookup(l.stat, "nature.decreased_stat_id", down)
			if err != nil {
				return err
			}
			increased, decreased = &u, &d
		}
		if err := l.exec(ctx, "nature", q, identifier, increased, decreased); err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) loadSpecies(ctx context.Context) error {
	pokemon, err := readTable("pokemon", "id", "identifier", "species_id", "is_default")
	if err != nil {
		return err
	}
	species, err := readTable("pokemon_species", "id", "generation_id")
	if err != nil {
		return err
	}
	baseStats, err := readTable("pokemon_stats", "pokemon_id", "stat_id", "base_stat")
	if err != nil {
		return err
	}
	types, err := readTable("pokemon_types", "pokemon_id", "type_id", "slot")
	if err != nil {
		return err
	}

	generationOf := make(map[string]string, len(species))
	for _, r := range species {
		generationOf[r["id"]] = r["generation_id"]
	}
	statsOf := map[string]map[string]int32{}
	for _, r := range baseStats {
		p := r.parse("pokemon_stats")
		base := p.num("base_stat")
		if err := p.done(); err != nil {
			return err
		}
		if statsOf[r["pokemon_id"]] == nil {
			statsOf[r["pokemon_id"]] = map[string]int32{}
		}
		statsOf[r["pokemon_id"]][r["stat_id"]] = base
	}
	typesOf := map[string]map[string]string{}
	for _, r := range types {
		if typesOf[r["pokemon_id"]] == nil {
			typesOf[r["pokemon_id"]] = map[string]string{}
		}
		typesOf[r["pokemon_id"]][r["slot"]] = r["type_id"]
	}

	const q = `INSERT INTO species (
			identifier, national_dex, is_default, generation_id,
			type_1_id, type_2_id,
			base_hp, base_attack, base_defense,
			base_special_attack, base_special_defense, base_speed)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (identifier) DO UPDATE SET
			national_dex = EXCLUDED.national_dex,
			is_default = EXCLUDED.is_default,
			generation_id = EXCLUDED.generation_id,
			type_1_id = EXCLUDED.type_1_id,
			type_2_id = EXCLUDED.type_2_id,
			base_hp = EXCLUDED.base_hp,
			base_attack = EXCLUDED.base_attack,
			base_defense = EXCLUDED.base_defense,
			base_special_attack = EXCLUDED.base_special_attack,
			base_special_defense = EXCLUDED.base_special_defense,
			base_speed = EXCLUDED.base_speed`

	for _, r := range pokemon {
		p := r.parse("pokemon")
		identifier := p.text("identifier")
		nationalDex, isDefault := p.num("species_id"), p.flag("is_default")
		if err := p.done(); err != nil {
			return err
		}

		generation, err := lookup(l.generation, "species.generation_id", generationOf[r["species_id"]])
		if err != nil {
			return fmt.Errorf("%s: %w", identifier, err)
		}
		slots := typesOf[r["id"]]
		type1, err := lookup(l.typ, "species.type_1_id", slots["1"])
		if err != nil {
			return fmt.Errorf("%s: %w", identifier, err)
		}
		var type2 *int64
		if slot2, ok := slots["2"]; ok {
			t, err := lookup(l.typ, "species.type_2_id", slot2)
			if err != nil {
				return fmt.Errorf("%s: %w", identifier, err)
			}
			type2 = &t
		}

		base := statsOf[r["id"]]
		for _, stat := range allStats {
			if _, ok := base[stat]; !ok {
				return fmt.Errorf("%s: no base_stat for stat_id %q", identifier, stat)
			}
		}
		if err := l.exec(ctx, "species", q,
			identifier, nationalDex, isDefault, generation, type1, type2,
			base[statHP], base[statAttack], base[statDefense],
			base[statSpecialAttack], base[statSpecialDefense], base[statSpeed]); err != nil {
			return fmt.Errorf("%s: %w", identifier, err)
		}
	}
	return nil
}
