package pokedex

import (
	"context"
	"fmt"

	"github.com/aarrico/porygon-vgc/backend/internal/pokedex/internal/sqlcgen"
)

type Item struct {
	ID         int64  `json:"id"`
	Identifier string `json:"identifier"`
	Category   string `json:"category"`
	FlingPower *int16 `json:"fling_power"`
}

func itemFromRow(row sqlcgen.SearchItemsRow) Item {
	return Item{
		ID:         row.ID,
		Identifier: row.Identifier,
		Category:   row.Category,
		FlingPower: row.FlingPower,
	}
}

func (s *Store) SearchItems(ctx context.Context, name, dataSet string) ([]Item, error) {
	rows, err := s.q.SearchItems(ctx, sqlcgen.SearchItemsParams{
		Pattern: likePattern(name),
		DataSet: dataSet,
	})
	if err != nil {
		return nil, fmt.Errorf("pokedex: search items: %w", err)
	}
	if len(rows) == 0 {
		if err := s.checkDataSetExists(ctx, dataSet); err != nil {
			return nil, err
		}
	}
	out := make([]Item, 0, len(rows))
	for _, row := range rows {
		out = append(out, itemFromRow(row))
	}
	return out, nil
}
