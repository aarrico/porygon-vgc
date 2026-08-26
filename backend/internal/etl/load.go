package etl

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Summary reports the upserts applied per table. That is not always the
// resulting row count: PokeAPI ships roseli-berry twice, differing only in a
// column this schema does not store.
type Summary struct {
	DataSet string
	Rows    map[string]int
}

// Load writes the whole dump in one transaction, so a failure at any table
// leaves the schema exactly as it was. Re-running is a no-op on unchanged
// data: every table upserts on its natural key.
func Load(ctx context.Context, pool *pgxpool.Pool, dataSet string) (Summary, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("etl: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op once Commit has succeeded

	l := &loader{tx: tx, rows: map[string]int{}}
	if err := l.run(ctx, dataSet); err != nil {
		return Summary{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Summary{}, fmt.Errorf("etl: commit: %w", err)
	}
	return Summary{DataSet: dataSet, Rows: l.rows}, nil
}

type loader struct {
	tx      pgx.Tx
	rows    map[string]int
	dataSet int64

	// PokeAPI id -> database id, for the tables other CSVs reference by id.
	generation map[string]int64
	typ        map[string]int64
	stat       map[string]int64
	move       map[string]int64
}

// run orders the tables by foreign key dependency.
func (l *loader) run(ctx context.Context, dataSet string) error {
	steps := []struct {
		table string
		load  func(context.Context) error
	}{
		{"data_set", func(c context.Context) error { return l.loadDataSet(c, dataSet) }},
		{"generation", l.loadGenerations},
		{"type", l.loadTypes},
		{"stat", l.loadStats},
		{"nature", l.loadNatures},
		{"species", l.loadSpecies},
		{"move", l.loadMoves},
		{"move_meta", l.loadMoveMeta},
		{"move_stat_change", l.loadMoveStatChanges},
		{"ability", l.loadAbilities},
		{"item", l.loadItems},
	}
	for _, step := range steps {
		if err := step.load(ctx); err != nil {
			return fmt.Errorf("etl: %s: %w", step.table, err)
		}
	}
	return nil
}

func (l *loader) exec(ctx context.Context, table, sql string, args ...any) error {
	if _, err := l.tx.Exec(ctx, sql, args...); err != nil {
		return err
	}
	l.rows[table]++
	return nil
}

// index reads identifier -> id back out of a table. Every primary key is
// GENERATED ALWAYS, so the loader cannot carry PokeAPI's own ids across and
// resolves foreign keys through the natural key instead.
func (l *loader) index(ctx context.Context, sql string, args ...any) (map[string]int64, error) {
	rows, err := l.tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]int64{}
	for rows.Next() {
		var identifier string
		var id int64
		if err := rows.Scan(&identifier, &id); err != nil {
			return nil, err
		}
		out[identifier] = id
	}
	return out, rows.Err()
}

// link turns identifier -> id into the PokeAPI id -> id map the other CSVs need.
func link(table string, rows []record, byIdentifier map[string]int64) (map[string]int64, error) {
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		id, ok := byIdentifier[r["identifier"]]
		if !ok {
			return nil, fmt.Errorf("%s: %q missing after upsert", table, r["identifier"])
		}
		out[r["id"]] = id
	}
	return out, nil
}

func lookup(m map[string]int64, column, key string) (int64, error) {
	id, ok := m[key]
	if !ok {
		return 0, fmt.Errorf("%s: no row for PokeAPI id %q", column, key)
	}
	return id, nil
}
