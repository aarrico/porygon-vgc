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
