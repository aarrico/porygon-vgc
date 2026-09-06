package etl

import (
	"strings"
	"testing"
)

func TestSlug(t *testing.T) {
	tests := []struct{ name, want string }{
		{"Swords Dance", "swords-dance"},
		{"King's Rock", "kings-rock"},
		{"Will-O-Wisp", "will-o-wisp"},
		{"U-turn", "u-turn"},
		{"Poké Ball", "poke-ball"},
		{"Sp. Def", "sp-def"},
	}
	for _, tt := range tests {
		if got := slug(tt.name); got != tt.want {
			t.Errorf("slug(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// Pins the id alignment and the name mapping against the vendored PokeAPI
// dump: a re-pin of either side that breaks them fails here, not mid-load.
func TestChampoutMovesAlignWithPokeAPI(t *testing.T) {
	waza, err := readChampout("waza", "id", "available", "ms_lbl", "category", "type", "con_ref")
	if err != nil {
		t.Fatal(err)
	}
	names, err := readLabels("wazaname")
	if err != nil {
		t.Fatal(err)
	}
	pokeapi, err := readTable("moves", "id", "identifier")
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]string, len(pokeapi))
	for _, r := range pokeapi {
		byID[r["id"]] = r["identifier"]
	}

	var available int
	for _, r := range waza {
		if r["available"] != "1" {
			continue
		}
		available++
		got := slug(names[r["ms_lbl"]])
		if want := byID[r["id"]]; got != want {
			t.Errorf("move %s: slug %q, PokeAPI identifier %q", r["id"], got, want)
		}
		if _, ok := championDamageClass[r["category"]]; !ok {
			t.Errorf("move %s: unknown category %q", r["id"], r["category"])
		}
		for _, ref := range strings.Split(r["con_ref"], ",") {
			if ref == "" {
				continue
			}
			if _, ok := championAilment[ref]; !ok {
				t.Errorf("move %s: unknown con_ref %q", r["id"], ref)
			}
		}
	}
	if available != 497 {
		t.Errorf("available moves = %d, want 497 (SOURCE.md)", available)
	}
}

func TestChampoutMappingsPinned(t *testing.T) {
	waza, err := readChampout("waza", "id", "type", "category", "accuracy", "pp")
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]record, len(waza))
	for _, r := range waza {
		byID[r["id"]] = r
	}
	tests := []struct {
		id, col, want, why string
	}{
		{"14", "category", "2", "Swords Dance is a status move"},
		{"585", "category", "1", "Moonblast is special"},
		{"1", "category", "0", "Pound is physical"},
		{"585", "type", "17", "Moonblast is Fairy, PokeAPI id 18"},
		{"14", "accuracy", "101", "Swords Dance cannot miss"},
		{"1", "pp", "20", "Pound's 35 PP is capped at 20"},
		{"690", "pp", "8", "Beak Blast's 5 PP becomes (5/5+1)*4"},
	}
	for _, tt := range tests {
		if got := byID[tt.id][tt.col]; got != tt.want {
			t.Errorf("move %s %s = %q, want %q: %s", tt.id, tt.col, got, tt.want, tt.why)
		}
	}
}

func TestChampoutAbilitiesAndItemsExistInPokeAPI(t *testing.T) {
	pokeapiAbilities, err := readTable("abilities", "identifier", "is_main_series")
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{}
	for _, r := range pokeapiAbilities {
		if r["is_main_series"] == "1" {
			known[r["identifier"]] = true
		}
	}
	species, err := readChampout("personal", "is_valid", "toku0", "toku1", "toku2")
	if err != nil {
		t.Fatal(err)
	}
	abilityNames, err := readLabels("tokusei")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range species {
		for _, slot := range []string{"toku0", "toku1", "toku2"} {
			id := s[slot]
			name, ok := abilityNames["TOKUSEI_"+strings.Repeat("0", 3-len(id))+id]
			if !ok {
				t.Errorf("ability id %s: no name", id)
				continue
			}
			if !known[slug(name)] {
				t.Errorf("ability %q (%s) not in PokeAPI", name, slug(name))
			}
		}
	}

	pokeapiItems, err := readTable("items", "identifier")
	if err != nil {
		t.Fatal(err)
	}
	knownItems := map[string]bool{}
	for _, r := range pokeapiItems {
		knownItems[r["identifier"]] = true
	}
	items, err := readChampout("item", "id", "ms_lbl")
	if err != nil {
		t.Fatal(err)
	}
	itemNames, err := readLabels("itemname")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range items {
		name, ok := itemNames[r["ms_lbl"]]
		if !ok {
			t.Errorf("item %s: no name for %q", r["id"], r["ms_lbl"])
			continue
		}
		if !knownItems[slug(name)] {
			t.Errorf("item %q (%s) not in PokeAPI", name, slug(name))
		}
	}
	if len(items) != 148 {
		t.Errorf("items = %d, want 148 (SOURCE.md)", len(items))
	}
}
