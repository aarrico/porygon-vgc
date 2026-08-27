package etl

import (
	"strings"
	"testing"
)

// The loaders name every column they read. This pins that contract against the
// vendored dump, so a re-pin that renames or drops one fails here rather than
// halfway through a load.
func TestEmbeddedDumpHasRequiredColumns(t *testing.T) {
	tests := []struct {
		file    string
		columns []string
	}{
		{"generations", []string{"id", "identifier"}},
		{"types", []string{"id", "identifier", "generation_id"}},
		{"stats", []string{"id", "identifier", "is_battle_only"}},
		{"natures", []string{"id", "identifier", "increased_stat_id", "decreased_stat_id"}},
		{"pokemon", []string{"id", "identifier", "species_id", "is_default"}},
		{"pokemon_species", []string{"id", "generation_id"}},
		{"pokemon_stats", []string{"pokemon_id", "stat_id", "base_stat"}},
		{"pokemon_types", []string{"pokemon_id", "type_id", "slot"}},
		{"moves", []string{"id", "identifier", "generation_id", "type_id", "power",
			"pp", "accuracy", "priority", "target_id", "damage_class_id"}},
		{"move_damage_classes", []string{"id", "identifier"}},
		{"move_targets", []string{"id", "identifier"}},
		{"move_meta", []string{"move_id", "meta_category_id", "meta_ailment_id",
			"min_hits", "max_hits", "min_turns", "max_turns", "drain", "healing",
			"crit_rate", "ailment_chance", "flinch_chance", "stat_chance"}},
		{"move_meta_categories", []string{"id", "identifier"}},
		{"move_meta_ailments", []string{"id", "identifier"}},
		{"move_meta_stat_changes", []string{"move_id", "stat_id", "change"}},
		{"abilities", []string{"id", "identifier", "generation_id", "is_main_series"}},
		{"items", []string{"id", "identifier", "category_id", "fling_power"}},
		{"item_categories", []string{"id", "identifier"}},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			rows, err := readTable(tt.file, tt.columns...)
			if err != nil {
				t.Fatalf("readTable: %v", err)
			}
			if len(rows) == 0 {
				t.Fatal("no rows")
			}
		})
	}
}

func TestReadTableRejectsMissingColumn(t *testing.T) {
	_, err := readTable("generations", "id", "no_such_column")
	if err == nil {
		t.Fatal("want an error naming the missing column, got nil")
	}
	if !strings.Contains(err.Error(), `missing column "no_such_column"`) {
		t.Fatalf("error does not name the column: %v", err)
	}
}

func TestReadTableRejectsUnknownFile(t *testing.T) {
	if _, err := readTable("not_a_real_table"); err == nil {
		t.Fatal("want an error, got nil")
	}
}

// The data shapes the loaders normalize, asserted against the real dump so a
// re-pin that changes them is caught here.
func TestDumpShape(t *testing.T) {
	t.Run("only shadow moves lack pp", func(t *testing.T) {
		rows, err := readTable("moves", "id", "identifier", "pp")
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range rows {
			main, err := mainSeries("moves", r)
			if err != nil {
				t.Fatal(err)
			}
			if r["pp"] == "" && main {
				t.Errorf("main-series move %q has no pp; move.pp is NOT NULL", r["identifier"])
			}
		}
	})

	t.Run("neutral natures repeat the stat", func(t *testing.T) {
		rows, err := readTable("natures", "identifier", "increased_stat_id", "decreased_stat_id")
		if err != nil {
			t.Fatal(err)
		}
		var neutral int
		for _, r := range rows {
			if r["increased_stat_id"] == r["decreased_stat_id"] {
				neutral++
			}
		}
		if neutral == 0 {
			t.Fatal("no neutral natures found; the NULL normalization is now dead code")
		}
	})

	t.Run("one default forme per national dex", func(t *testing.T) {
		rows, err := readTable("pokemon", "species_id", "is_default")
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for _, r := range rows {
			if r["is_default"] != "1" {
				continue
			}
			if seen[r["species_id"]] {
				t.Errorf("species_id %q has two defaults; the partial unique index rejects that",
					r["species_id"])
			}
			seen[r["species_id"]] = true
		}
	})
}

func TestParser(t *testing.T) {
	rec := record{"n": "42", "blank": "", "flag": "1", "text": "hi", "bad": "x"}

	t.Run("optNum keeps blank distinct from zero", func(t *testing.T) {
		p := rec.parse("t")
		if got := p.optNum("blank"); got != nil {
			t.Errorf("blank: got %v, want nil", *got)
		}
		if got := p.optNum("n"); got == nil || *got != 42 {
			t.Errorf("n: got %v, want 42", got)
		}
		if err := p.done(); err != nil {
			t.Errorf("done: %v", err)
		}
	})

	t.Run("first failure is kept", func(t *testing.T) {
		p := rec.parse("t")
		p.num("bad")
		p.text("blank")
		err := p.done()
		if err == nil {
			t.Fatal("want an error, got nil")
		}
		if !strings.Contains(err.Error(), `"bad"`) {
			t.Fatalf("want the first failure reported, got: %v", err)
		}
	})

	t.Run("flag is only true for 1", func(t *testing.T) {
		p := rec.parse("t")
		if !p.flag("flag") {
			t.Error(`"1" should be true`)
		}
		if p.flag("blank") {
			t.Error(`"" should be false`)
		}
	})
}
