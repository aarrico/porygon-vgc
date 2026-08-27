package pokedex

import (
	"context"
	"fmt"

	"github.com/aarrico/porygon-vgc/backend/internal/pokedex/internal/sqlcgen"
)

type Ability struct {
	ID         int64  `json:"id"`
	Identifier string `json:"identifier"`
	Generation string `json:"generation"`
}

func abilityFromRow(row sqlcgen.SearchAbilitiesRow) Ability {
	return Ability{
		ID:         row.ID,
		Identifier: row.Identifier,
		Generation: row.Generation,
	}
}

func (s *Store) SearchAbilities(ctx context.Context, name, dataSet string) ([]Ability, error) {
	rows, err := s.q.SearchAbilities(ctx, sqlcgen.SearchAbilitiesParams{
		Pattern: likePattern(name),
		DataSet: dataSet,
	})
	if err != nil {
		return nil, fmt.Errorf("pokedex: search abilities: %w", err)
	}
	if len(rows) == 0 {
		if err := s.checkDataSetExists(ctx, dataSet); err != nil {
			return nil, err
		}
	}
	out := make([]Ability, 0, len(rows))
	for _, row := range rows {
		out = append(out, abilityFromRow(row))
	}
	return out, nil
}
