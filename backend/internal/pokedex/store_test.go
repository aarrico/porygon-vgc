package pokedex

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const liveDSNEnv = "TEST_DATABASE_URL"

func openLiveStore(t *testing.T) *Store {
	t.Helper()

	dsn := os.Getenv(liveDSNEnv)
	if dsn == "" {
		t.Skipf("%s not set; skipping live-database pokedex tests", liveDSNEnv)
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pgxpool.New() = %v", err)
	}
	t.Cleanup(pool.Close)

	return NewStore(pool)
}

func TestSearchSpeciesLive(t *testing.T) {
	store := openLiveStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := store.SearchSpecies(ctx, "meowth")
	if err != nil {
		t.Fatalf("SearchSpecies() = %v", err)
	}
	if len(results) < 3 {
		t.Errorf("got %d results for \"meowth\", want at least 3 (base + Alolan + Galarian)", len(results))
	}
	for _, sp := range results {
		if len(sp.Types) == 0 || sp.Generation == "" {
			t.Errorf("species %q missing joined fields: %+v", sp.Identifier, sp)
		}
	}
}

func TestSearchMovesLiveDefaultDataSet(t *testing.T) {
	store := openLiveStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := store.SearchMoves(ctx, "thunderbolt", "gen-9")
	if err != nil {
		t.Fatalf("SearchMoves() = %v", err)
	}

	if len(results) < 1 {
		t.Fatalf("got %d results for \"thunderbolt\", want at least 1", len(results))
	}
	var found bool
	for _, m := range results {
		if m.Identifier == "thunderbolt" {
			found = true
		}
		if m.Type == "" || m.DamageClass == "" {
			t.Errorf("move missing joined fields: %+v", m)
		}
	}
	if !found {
		t.Errorf("results %+v do not include the exact identifier \"thunderbolt\"", results)
	}
}

func TestSearchMovesLiveUnknownDataSet(t *testing.T) {
	store := openLiveStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := store.SearchMoves(ctx, "thunderbolt", "not-a-real-data-set")
	if !errors.Is(err, ErrDataSetNotFound) {
		t.Errorf("SearchMoves() with an unknown data set = %v, want ErrDataSetNotFound", err)
	}
}

func TestSearchMovesByEffectLiveAttackPlusTwo(t *testing.T) {
	store := openLiveStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	q := EffectQuery{StatChanges: []StatChange{{Stat: "attack", Change: 2}}}
	results, err := store.SearchMovesByEffect(ctx, q, "gen-9")
	if err != nil {
		t.Fatalf("SearchMovesByEffect() = %v", err)
	}

	found := map[string]bool{}
	for _, m := range results {
		found[m.Identifier] = true
		var hasAttackTwo bool
		for _, sc := range m.Effect.StatChanges {
			if sc.Stat == "attack" && sc.Change == 2 {
				hasAttackTwo = true
			}
		}
		if !hasAttackTwo {
			t.Errorf("%q returned without an attack +2 stat change: %+v", m.Identifier, m.Effect.StatChanges)
		}
	}
	for _, want := range []string{"swords-dance", "shell-smash", "swagger"} {
		if !found[want] {
			t.Errorf("results %v do not include %q", found, want)
		}
	}
}

func TestSearchMovesByEffectLiveCombined(t *testing.T) {
	store := openLiveStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Swagger: Attack +2 on the target plus confusion.
	q := EffectQuery{
		StatChanges: []StatChange{{Stat: "attack", Change: 2}},
		Ailments:    []string{"confusion"},
	}
	results, err := store.SearchMovesByEffect(ctx, q, "gen-9")
	if err != nil {
		t.Fatalf("SearchMovesByEffect() = %v", err)
	}
	if len(results) != 1 || results[0].Identifier != "swagger" {
		t.Errorf("results = %+v, want exactly swagger", results)
	}
	if results[0].Effect.Ailment == nil || *results[0].Effect.Ailment != "confusion" {
		t.Errorf("swagger effect = %+v, want ailment confusion", results[0].Effect)
	}
}

func TestSearchMovesByEffectLiveNoMatch(t *testing.T) {
	store := openLiveStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// HP has no stat stages; a valid predicate with no matching move.
	q := EffectQuery{StatChanges: []StatChange{{Stat: "hp", Change: 1}}}
	results, err := store.SearchMovesByEffect(ctx, q, "gen-9")
	if err != nil {
		t.Fatalf("SearchMovesByEffect() = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("results = %+v, want none", results)
	}
}

func TestSearchMovesByEffectLiveUnknownValues(t *testing.T) {
	store := openLiveStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tests := []struct {
		name string
		q    EffectQuery
	}{
		{"unknown stat", EffectQuery{StatChanges: []StatChange{{Stat: "atk", Change: 2}}}},
		{"unknown ailment", EffectQuery{Ailments: []string{"sunburn"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.SearchMovesByEffect(ctx, tt.q, "gen-9")
			var ie *InvalidEffectError
			if !errors.As(err, &ie) {
				t.Errorf("SearchMovesByEffect() = %v, want *InvalidEffectError", err)
			}
		})
	}

	_, err := store.SearchMovesByEffect(ctx, EffectQuery{Ailments: []string{"burn"}}, "not-a-real-data-set")
	if !errors.Is(err, ErrDataSetNotFound) {
		t.Errorf("SearchMovesByEffect() with an unknown data set = %v, want ErrDataSetNotFound", err)
	}
}

// Two stat predicates must pair by position: a cross join of identifiers
// and stages would still match shell-smash under the swapped query.
func TestSearchMovesByEffectLivePairsStatPredicates(t *testing.T) {
	store := openLiveStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	paired := EffectQuery{StatChanges: []StatChange{{Stat: "attack", Change: 2}, {Stat: "defense", Change: -1}}}
	results, err := store.SearchMovesByEffect(ctx, paired, "gen-9")
	if err != nil {
		t.Fatalf("SearchMovesByEffect() = %v", err)
	}
	if len(results) != 1 || results[0].Identifier != "shell-smash" {
		t.Errorf("results = %+v, want exactly shell-smash", results)
	}

	swapped := EffectQuery{StatChanges: []StatChange{{Stat: "attack", Change: -1}, {Stat: "defense", Change: 2}}}
	results, err = store.SearchMovesByEffect(ctx, swapped, "gen-9")
	if err != nil {
		t.Fatalf("SearchMovesByEffect() = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("results = %+v, want none", results)
	}
}
