// Package etl loads the pinned PokeAPI CSV dump (AD-14) into the core
// reference schema. The dump is embedded, so a load needs no network.
package etl

import (
	"embed"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
)

//go:embed data/pokeapi/*.csv
var dump embed.FS

type record map[string]string

// readTable parses one vendored CSV. Naming the columns the caller needs turns
// an upstream rename into a load error instead of a silently NULL column.
func readTable(name string, required ...string) ([]record, error) {
	f, err := dump.Open("data/pokeapi/" + name + ".csv")
	if err != nil {
		return nil, fmt.Errorf("etl: open %s.csv: %w", name, err)
	}
	defer f.Close() //nolint:errcheck // read-only, from an embedded FS

	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("etl: %s.csv: read header: %w", name, err)
	}

	index := make(map[string]int, len(header))
	for i, col := range header {
		index[col] = i
	}
	for _, col := range required {
		if _, ok := index[col]; !ok {
			return nil, fmt.Errorf("etl: %s.csv: missing column %q", name, col)
		}
	}

	var out []record
	for line := 2; ; line++ {
		row, err := r.Read()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("etl: %s.csv line %d: %w", name, line, err)
		}
		rec := make(record, len(index))
		for col, i := range index {
			rec[col] = row[i]
		}
		out = append(out, rec)
	}
}

// parser reads typed columns off a record, holding the first failure so a
// table loader checks once per row instead of after every column.
type parser struct {
	table string
	rec   record
	err   error
}

func (r record) parse(table string) *parser { return &parser{table: table, rec: r} }

func (p *parser) fail(col, reason string) {
	if p.err == nil {
		p.err = fmt.Errorf("etl: %s: column %q (%q): %s", p.table, col, p.rec[col], reason)
	}
}

func (p *parser) text(col string) string {
	if p.rec[col] == "" {
		p.fail(col, "is empty")
	}
	return p.rec[col]
}

func (p *parser) num(col string) int32 {
	v, err := strconv.ParseInt(p.rec[col], 10, 32)
	if err != nil {
		p.fail(col, "is not an integer")
		return 0
	}
	return int32(v)
}

// optNum maps a blank column to NULL rather than to zero: a move with no
// listed power differs from one whose power is 0.
func (p *parser) optNum(col string) *int32 {
	if p.rec[col] == "" {
		return nil
	}
	v := p.num(col)
	return &v
}

func (p *parser) flag(col string) bool { return p.rec[col] == "1" }

func (p *parser) done() error { return p.err }
