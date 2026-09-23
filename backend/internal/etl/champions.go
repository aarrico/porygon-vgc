package etl

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

const championsDataSet = "champions"

// The dump's accuracy for a move that cannot miss.
const championNeverMiss = 101

var (
	championDamageClass = map[string]string{"0": "physical", "1": "special", "2": "status"}
	// con_ref ids follow btl_condition's label order. 5 is bad poison, which
	// PokeAPI folds into poison.
	championAilment = map[string]string{
		"1": "paralysis", "2": "freeze", "3": "burn", "4": "poison", "5": "poison", "6": "sleep",
	}
)

// runChampions loads the Champions Data Set as a child of the base one:
// moves, abilities and items come from the game dump, with the columns the
// dump does not carry (generation, target, effect data) inherited from the
// parent row of the same identifier (AD-15).
func (l *loader) runChampions(ctx context.Context) error {
	steps := []struct {
		table string
		load  func(context.Context) error
	}{
		{"data_set", l.loadChampionsDataSet},
		{"move", l.loadChampionMoves},
		{"move_meta", l.loadChampionMoveMeta},
		{"move_stat_change", l.loadChampionMoveStatChanges},
		{"ability", l.loadChampionAbilities},
		{"item", l.loadChampionItems},
	}
	for _, step := range steps {
		if err := step.load(ctx); err != nil {
			return fmt.Errorf("etl: %s: %s: %w", championsDataSet, step.table, err)
		}
	}
	return nil
}

func (l *loader) loadChampionsDataSet(ctx context.Context) error {
	const q = `INSERT INTO data_set (identifier, parent_id) VALUES ($1, $2)
		ON CONFLICT (identifier) DO UPDATE SET parent_id = EXCLUDED.parent_id
		RETURNING id`
	if err := l.tx.QueryRow(ctx, q, championsDataSet, l.parent).Scan(&l.dataSet); err != nil {
		return err
	}
	l.rows["data_set"]++
	return nil
}

type parentMove struct {
	generation int64
	target     string
}

func (l *loader) loadChampionMoves(ctx context.Context) error {
	rows, err := readChampout("waza", "id", "type", "category", "power", "accuracy",
		"pp", "priority", "available", "con_ref", "ms_lbl")
	if err != nil {
		return err
	}
	names, err := readLabels("wazaname")
	if err != nil {
		return err
	}
	parents, err := l.parentMoves(ctx)
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

	identifiers := make(map[string]string, len(rows))
	for _, r := range rows {
		if r["available"] != "1" {
			continue
		}
		name, ok := names[r["ms_lbl"]]
		if !ok {
			return fmt.Errorf("move %s: no name for label %q", r["id"], r["ms_lbl"])
		}
		identifier := slug(name)
		parent, ok := parents[identifier]
		if !ok {
			return fmt.Errorf("move %q: no parent row to inherit generation and target from", identifier)
		}

		p := r.parse("waza")
		typ, power, accuracy := p.num("type"), p.num("power"), p.num("accuracy")
		pp, priority := p.num("pp"), p.num("priority")
		if err := p.done(); err != nil {
			return err
		}
		damageClass, ok := championDamageClass[r["category"]]
		if !ok {
			return fmt.Errorf("move %q: unknown category %q", identifier, r["category"])
		}
		// The dump numbers types in Game Freak's order, PokeAPI's id minus one.
		typeID, err := lookup(l.typ, "waza.type", strconv.Itoa(int(typ)+1))
		if err != nil {
			return fmt.Errorf("move %q: %w", identifier, err)
		}
		// The dump gives status moves 0 power; NULL is the schema's spelling.
		var powerPtr, accuracyPtr *int32
		if power != 0 {
			powerPtr = &power
		}
		if accuracy != championNeverMiss {
			accuracyPtr = &accuracy
		}
		if err := l.exec(ctx, "move", q, l.dataSet, identifier, parent.generation, typeID,
			damageClass, powerPtr, accuracyPtr, pp, priority, parent.target); err != nil {
			return fmt.Errorf("move %q: %w", identifier, err)
		}
		identifiers[r["id"]] = identifier
	}

	byIdentifier, err := l.index(ctx,
		`SELECT identifier, id FROM move WHERE data_set_id = $1`, l.dataSet)
	if err != nil {
		return err
	}
	l.move = make(map[string]int64, len(identifiers))
	for _, id := range slices.Sorted(maps.Keys(identifiers)) {
		identifier := identifiers[id]
		dbID, ok := byIdentifier[identifier]
		if !ok {
			return fmt.Errorf("move: %q missing after upsert", identifier)
		}
		l.move[id] = dbID
	}
	return nil
}

func (l *loader) parentMoves(ctx context.Context) (map[string]parentMove, error) {
	rows, err := l.tx.Query(ctx,
		`SELECT identifier, generation_id, target FROM move WHERE data_set_id = $1`, l.parent)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]parentMove{}
	for rows.Next() {
		var identifier string
		var pm parentMove
		if err := rows.Scan(&identifier, &pm.generation, &pm.target); err != nil {
			return nil, err
		}
		out[identifier] = pm
	}
	return out, rows.Err()
}

// loadChampionMoveMeta copies the parent's effect rows onto the child's
// moves, then applies the one effect column the dump does carry: the
// ailment a move inflicts. A move with several possible ailments keeps the
// parent's value; an empty con_ref says nothing about volatile ailments
// (confusion, trap) so it does not reset the parent's value either. A move
// the parent has no effect row for gets none here: the chances that row
// would need are not in the dump, and a fabricated 0% is worse than
// absence. Those moves are reported in the Summary instead.
func (l *loader) loadChampionMoveMeta(ctx context.Context) error {
	const copyQ = `INSERT INTO move_meta (
			move_id, data_set_id, category, ailment,
			min_hits, max_hits, min_turns, max_turns,
			drain, healing, crit_rate, ailment_chance, flinch_chance, stat_chance)
		SELECT c.id, c.data_set_id, pm.category, pm.ailment,
			pm.min_hits, pm.max_hits, pm.min_turns, pm.max_turns,
			pm.drain, pm.healing, pm.crit_rate, pm.ailment_chance, pm.flinch_chance, pm.stat_chance
		FROM move c
		JOIN move p ON p.identifier = c.identifier AND p.data_set_id = $2
		JOIN move_meta pm ON pm.move_id = p.id
		WHERE c.data_set_id = $1
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
	if err := l.execCount(ctx, "move_meta", copyQ, l.dataSet, l.parent); err != nil {
		return err
	}

	rows, err := readChampout("waza", "id", "available", "con_ref", "ms_lbl")
	if err != nil {
		return err
	}
	names, err := readLabels("wazaname")
	if err != nil {
		return err
	}
	const ailmentQ = `UPDATE move_meta SET ailment = $1 WHERE move_id = $2`
	for _, r := range rows {
		if r["available"] != "1" || r["con_ref"] == "" || strings.Contains(r["con_ref"], ",") {
			continue
		}
		ailment, ok := championAilment[r["con_ref"]]
		if !ok {
			return fmt.Errorf("move %s: unknown con_ref %q", r["id"], r["con_ref"])
		}
		move, err := lookup(l.move, "waza.id", r["id"])
		if err != nil {
			return err
		}
		tag, err := l.tx.Exec(ctx, ailmentQ, ailment, move)
		if err != nil {
			return fmt.Errorf("move %s: %w", r["id"], err)
		}
		if tag.RowsAffected() == 0 {
			l.skipped = append(l.skipped, fmt.Sprintf("move_meta: %s inflicts %s but has no parent effect row", names[r["ms_lbl"]], ailment))
		}
	}
	return nil
}

func (l *loader) loadChampionMoveStatChanges(ctx context.Context) error {
	const q = `INSERT INTO move_stat_change (move_id, data_set_id, stat_id, change)
		SELECT c.id, c.data_set_id, ps.stat_id, ps.change
		FROM move c
		JOIN move p ON p.identifier = c.identifier AND p.data_set_id = $2
		JOIN move_stat_change ps ON ps.move_id = p.id
		WHERE c.data_set_id = $1
		ON CONFLICT (move_id, stat_id) DO UPDATE SET change = EXCLUDED.change`
	return l.execCount(ctx, "move_stat_change", q, l.dataSet, l.parent)
}

func (l *loader) loadChampionAbilities(ctx context.Context) error {
	species, err := readChampout("personal", "is_valid", "toku0", "toku1", "toku2")
	if err != nil {
		return err
	}
	names, err := readLabels("tokusei")
	if err != nil {
		return err
	}
	parents, err := l.index(ctx,
		`SELECT identifier, generation_id FROM ability WHERE data_set_id = $1`, l.parent)
	if err != nil {
		return err
	}

	// The dump has no ability table; the roster is whatever the species use.
	ids := map[string]bool{}
	for _, s := range species {
		if s["is_valid"] != "1" {
			continue
		}
		for _, slot := range []string{"toku0", "toku1", "toku2"} {
			ids[s[slot]] = true
		}
	}

	const q = `INSERT INTO ability (data_set_id, identifier, generation_id)
		VALUES ($1,$2,$3)
		ON CONFLICT (data_set_id, identifier) DO UPDATE SET
			generation_id = EXCLUDED.generation_id`
	for _, id := range slices.Sorted(maps.Keys(ids)) {
		n, err := strconv.Atoi(id)
		if err != nil {
			return fmt.Errorf("personal: ability id %q is not an integer", id)
		}
		label := fmt.Sprintf("TOKUSEI_%03d", n)
		name, ok := names[label]
		if !ok {
			return fmt.Errorf("ability %s: no name for label %q", id, label)
		}
		identifier := slug(name)
		generation, ok := parents[identifier]
		if !ok {
			return fmt.Errorf("ability %q: no parent row to inherit generation from", identifier)
		}
		if err := l.exec(ctx, "ability", q, l.dataSet, identifier, generation); err != nil {
			return fmt.Errorf("ability %q: %w", identifier, err)
		}
	}
	return nil
}

func (l *loader) loadChampionItems(ctx context.Context) error {
	rows, err := readChampout("item", "id", "ms_lbl")
	if err != nil {
		return err
	}
	names, err := readLabels("itemname")
	if err != nil {
		return err
	}

	const q = `INSERT INTO item (data_set_id, identifier, category, fling_power)
		SELECT $1, p.identifier, p.category, p.fling_power
		FROM item p WHERE p.data_set_id = $2 AND p.identifier = $3
		ON CONFLICT (data_set_id, identifier) DO UPDATE SET
			category = EXCLUDED.category,
			fling_power = EXCLUDED.fling_power`
	for _, r := range rows {
		name, ok := names[r["ms_lbl"]]
		if !ok {
			return fmt.Errorf("item %s: no name for label %q", r["id"], r["ms_lbl"])
		}
		identifier := slug(name)
		tag, err := l.tx.Exec(ctx, q, l.dataSet, l.parent, identifier)
		if err != nil {
			return fmt.Errorf("item %q: %w", identifier, err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("item %q: no parent row to inherit category from", identifier)
		}
		l.rows["item"]++
	}
	return nil
}

// execCount is exec for a statement that writes many rows at once.
func (l *loader) execCount(ctx context.Context, table, sql string, args ...any) error {
	tag, err := l.tx.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	l.rows[table] += int(tag.RowsAffected())
	return nil
}
