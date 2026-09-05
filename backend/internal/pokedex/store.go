package pokedex

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aarrico/porygon-vgc/backend/internal/pokedex/internal/sqlcgen"
)

var ErrDataSetNotFound = errors.New("pokedex: data set not found")

type Store struct {
	q *sqlcgen.Queries
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{q: sqlcgen.New(pool)}
}

func (s *Store) checkDataSetExists(ctx context.Context, identifier string) error {
	_, err := s.q.FindDataSetByIdentifier(ctx, identifier)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrDataSetNotFound
		}
		return fmt.Errorf("pokedex: resolve data set %q: %w", identifier, err)
	}
	return nil
}
