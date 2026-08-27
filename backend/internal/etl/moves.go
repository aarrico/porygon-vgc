package etl

import (
	"context"
	"fmt"
)

// PokeAPI numbers non-battle entities above this: for moves the Colosseum/XD
// shadow moves (the only ones with no PP), for types the pseudo-types
// "unknown" and "shadow".
const sideGameID = 10000

// mainSeries reports whether r belongs to the main series rather than a
// side-game entity. A malformed id fails loudly instead of being treated as
// a side-game entity to skip, since the two are otherwise indistinguishable.
func mainSeries(table string, r record) (bool, error) {
	p := r.parse(table)
	id := p.num("id")
	if err := p.done(); err != nil {
		return false, err
	}
	return id < sideGameID, nil
}

func (l *loader) loadMoves(ctx context.Context) error {
	rows, err := readTable("moves", "id", "identifier", "generation_id", "type_id",
		"power", "pp", "accuracy", "priority", "target_id", "damage_class_id")
	if err != nil {
		return err
	}
	damageClasses, err := lookupTable("move_damage_classes")
	if err != nil {
		return err
	}
	targets, err := lookupTable("move_targets")
	if err != nil {
		return err
	}

	const q = `INSERT INTO move (
			data_set_id, identifier, generation_id, type_id,
			damage_class, power, accuracy, pp, priority, target)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (data_set_id, identifier) DO UPDATE SET
			generation_id = EXCLUDED.generation_id,
			type_id = EXCLUDED.type_id,
			damage_class = EXCLUDED.damage_class,
			power = EXCLUDED.power,
			accuracy = EXCLUDED.accuracy,
			pp = EXCLUDED.pp,
			priority = EXCLUDED.priority,
			target = EXCLUDED.target`

	kept := make([]record, 0, len(rows))
	for _, r := range rows {
		main, err := mainSeries("moves", r)
		if err != nil {
			return err
		}
		if !main {
			continue
		}
		kept = append(kept, r)

		p := r.parse("moves")
		identifier := p.text("identifier")
		pp, priority := p.num("pp"), p.num("priority")
		power, accuracy := p.optNum("power"), p.optNum("accuracy")
		if err := p.done(); err != nil {
			return err
		}

		// Most never-miss moves leave accuracy blank; three spell it 0. NULL is
		// already the schema's "cannot miss", so collapse the two spellings.
		if accuracy != nil && *accuracy == 0 {
			accuracy = nil
		}

		generation, err := lookup(l.generation, "move.generation_id", r["generation_id"])
		if err != nil {
			return fmt.Errorf("%s: %w", identifier, err)
		}
		typ, err := lookup(l.typ, "move.type_id", r["type_id"])
		if err != nil {
			return fmt.Errorf("%s: %w", identifier, err)
		}
		damageClass, ok := damageClasses[r["damage_class_id"]]
		if !ok {
			return fmt.Errorf("%s: unknown damage_class_id %q", identifier, r["damage_class_id"])
		}
		target, ok := targets[r["target_id"]]
		if !ok {
			return fmt.Errorf("%s: unknown target_id %q", identifier, r["target_id"])
		}

		if err := l.exec(ctx, "move", q, l.dataSet, identifier, generation, typ,
			damageClass, power, accuracy, pp, priority, target); err != nil {
			return fmt.Errorf("%s: %w", identifier, err)
		}
	}

	byIdentifier, err := l.index(ctx,
		`SELECT identifier, id FROM move WHERE data_set_id = $1`, l.dataSet)
	if err != nil {
		return err
	}
	l.move, err = link("move", kept, byIdentifier)
	return err
}

func (l *loader) loadMoveMeta(ctx context.Context) error {
	rows, err := readTable("move_meta", "move_id", "meta_category_id", "meta_ailment_id",
		"min_hits", "max_hits", "min_turns", "max_turns", "drain", "healing",
		"crit_rate", "ailment_chance", "flinch_chance", "stat_chance")
	if err != nil {
		return err
	}
	categories, err := lookupTable("move_meta_categories")
	if err != nil {
		return err
	}
	ailments, err := lookupTable("move_meta_ailments")
	if err != nil {
		return err
	}

	const q = `INSERT INTO move_meta (
			move_id, data_set_id, category, ailment,
			min_hits, max_hits, min_turns, max_turns,
			drain, healing, crit_rate, ailment_chance, flinch_chance, stat_chance)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (move_id) DO UPDATE SET
			category = EXCLUDED.category,
			ailment = EXCLUDED.ailment,
			min_hits = EXCLUDED.min_hits,
			max_hits = EXCLUDED.max_hits,
			min_turns = EXCLUDED.min_turns,
			max_turns = EXCLUDED.max_turns,
			drain = EXCLUDED.drain,
			healing = EXCLUDED.healing,
			crit_rate = EXCLUDED.crit_rate,
			ailment_chance = EXCLUDED.ailment_chance,
			flinch_chance = EXCLUDED.flinch_chance,
			stat_chance = EXCLUDED.stat_chance`

	for _, r := range rows {
		move, ok := l.move[r["move_id"]]
		if !ok {
			return fmt.Errorf("move_meta: no row for move_id %q", r["move_id"])
		}
		p := r.parse("move_meta")
		drain, healing, critRate := p.num("drain"), p.num("healing"), p.num("crit_rate")
		ailmentChance, flinchChance := p.num("ailment_chance"), p.num("flinch_chance")
		statChance := p.num("stat_chance")
		minHits, maxHits := p.optNum("min_hits"), p.optNum("max_hits")
		minTurns, maxTurns := p.optNum("min_turns"), p.optNum("max_turns")
		if err := p.done(); err != nil {
			return err
		}

		category, ok := categories[r["meta_category_id"]]
		if !ok {
			return fmt.Errorf("move %q: unknown meta_category_id %q", r["move_id"], r["meta_category_id"])
		}
		ailment, ok := ailments[r["meta_ailment_id"]]
		if !ok {
			return fmt.Errorf("move %q: unknown meta_ailment_id %q", r["move_id"], r["meta_ailment_id"])
		}

		if err := l.exec(ctx, "move_meta", q, move, l.dataSet, category, ailment,
			minHits, maxHits, minTurns, maxTurns,
			drain, healing, critRate, ailmentChance, flinchChance, statChance); err != nil {
			return fmt.Errorf("move %q: %w", r["move_id"], err)
		}
	}
	return nil
}

func (l *loader) loadMoveStatChanges(ctx context.Context) error {
	rows, err := readTable("move_meta_stat_changes", "move_id", "stat_id", "change")
	if err != nil {
		return err
	}
	const q = `INSERT INTO move_stat_change (move_id, data_set_id, stat_id, change)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (move_id, stat_id) DO UPDATE SET change = EXCLUDED.change`

	for _, r := range rows {
		move, ok := l.move[r["move_id"]]
		if !ok {
			return fmt.Errorf("move_stat_change: no row for move_id %q", r["move_id"])
		}
		p := r.parse("move_meta_stat_changes")
		change := p.num("change")
		if err := p.done(); err != nil {
			return err
		}
		// A zero stage is not a stat change; the schema rejects it.
		if change == 0 {
			continue
		}
		stat, err := lookup(l.stat, "move_stat_change.stat_id", r["stat_id"])
		if err != nil {
			return err
		}
		if err := l.exec(ctx, "move_stat_change", q, move, l.dataSet, stat, change); err != nil {
			return fmt.Errorf("move %q: %w", r["move_id"], err)
		}
	}
	return nil
}

func (l *loader) loadAbilities(ctx context.Context) error {
	rows, err := readTable("abilities", "id", "identifier", "generation_id", "is_main_series")
	if err != nil {
		return err
	}
	const q = `INSERT INTO ability (data_set_id, identifier, generation_id)
		VALUES ($1,$2,$3)
		ON CONFLICT (data_set_id, identifier) DO UPDATE SET
			generation_id = EXCLUDED.generation_id`

	for _, r := range rows {
		p := r.parse("abilities")
		identifier, isMainSeries := p.text("identifier"), p.flag("is_main_series")
		if err := p.done(); err != nil {
			return err
		}
		if !isMainSeries {
			continue
		}
		generation, err := lookup(l.generation, "ability.generation_id", r["generation_id"])
		if err != nil {
			return fmt.Errorf("%s: %w", identifier, err)
		}
		if err := l.exec(ctx, "ability", q, l.dataSet, identifier, generation); err != nil {
			return fmt.Errorf("%s: %w", identifier, err)
		}
	}
	return nil
}

func (l *loader) loadItems(ctx context.Context) error {
	rows, err := readTable("items", "id", "identifier", "category_id", "fling_power")
	if err != nil {
		return err
	}
	categories, err := lookupTable("item_categories")
	if err != nil {
		return err
	}

	const q = `INSERT INTO item (data_set_id, identifier, category, fling_power)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (data_set_id, identifier) DO UPDATE SET
			category = EXCLUDED.category,
			fling_power = EXCLUDED.fling_power`

	// PokeAPI ships roseli-berry twice (ids 723/2279), differing only in an
	// unstored `cost` column. Any other duplicate identifier must agree on
	// every stored column too, or the second row would silently overwrite
	// the first's data via ON CONFLICT.
	type seenItem struct {
		category   string
		flingPower *int32
	}
	seen := make(map[string]seenItem, len(rows))

	for _, r := range rows {
		p := r.parse("items")
		identifier, flingPower := p.text("identifier"), p.optNum("fling_power")
		if err := p.done(); err != nil {
			return err
		}
		category, ok := categories[r["category_id"]]
		if !ok {
			return fmt.Errorf("%s: unknown category_id %q", identifier, r["category_id"])
		}
		if prev, dup := seen[identifier]; dup {
			if prev.category != category || !equalOptNum(prev.flingPower, flingPower) {
				return fmt.Errorf("%s: duplicate rows disagree on category/fling_power", identifier)
			}
		}
		seen[identifier] = seenItem{category, flingPower}
		if err := l.exec(ctx, "item", q, l.dataSet, identifier, category, flingPower); err != nil {
			return fmt.Errorf("%s: %w", identifier, err)
		}
	}
	return nil
}

func equalOptNum(a, b *int32) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// lookupTable reads a PokeAPI id -> identifier CSV whose values land in the
// schema as text rather than as a foreign key.
func lookupTable(name string) (map[string]string, error) {
	rows, err := readTable(name, "id", "identifier")
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[r["id"]] = r["identifier"]
	}
	return out, nil
}
