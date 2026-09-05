package pokedex

import (
	"context"
	"fmt"

	"github.com/aarrico/porygon-vgc/backend/internal/pokedex/internal/sqlcgen"
)

type Move struct {
	ID          int64  `json:"id"`
	Identifier  string `json:"identifier"`
	Generation  string `json:"generation"`
	Type        string `json:"type"`
	DamageClass string `json:"damage_class"`
	Power       *int16 `json:"power"`
	Accuracy    *int16 `json:"accuracy"`
	PP          int16  `json:"pp"`
	Priority    int16  `json:"priority"`
	Target      string `json:"target"`
}

func moveFromRow(row sqlcgen.SearchMovesRow) Move {
	return Move{
		ID:          row.ID,
		Identifier:  row.Identifier,
		Generation:  row.Generation,
		Type:        row.Type,
		DamageClass: row.DamageClass,
		Power:       row.Power,
		Accuracy:    row.Accuracy,
		PP:          row.Pp,
		Priority:    row.Priority,
		Target:      row.Target,
	}
}

func (s *Store) SearchMoves(ctx context.Context, name, dataSet string) ([]Move, error) {
	rows, err := s.q.SearchMoves(ctx, sqlcgen.SearchMovesParams{
		Pattern: likePattern(name),
		DataSet: dataSet,
	})
	if err != nil {
		return nil, fmt.Errorf("pokedex: search moves: %w", err)
	}

	if len(rows) == 0 {
		if err := s.checkDataSetExists(ctx, dataSet); err != nil {
			return nil, err
		}
	}
	out := make([]Move, 0, len(rows))
	for _, row := range rows {
		out = append(out, moveFromRow(row))
	}
	return out, nil
}
