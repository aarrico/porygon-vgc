package pokedex

import (
	"context"
	"fmt"

	"github.com/aarrico/porygon-vgc/backend/internal/pokedex/internal/sqlcgen"
)

type Species struct {
	ID          int64     `json:"id"`
	Identifier  string    `json:"identifier"`
	NationalDex int16     `json:"national_dex"`
	IsDefault   bool      `json:"is_default"`
	Generation  string    `json:"generation"`
	Types       []string  `json:"types"`
	BaseStats   BaseStats `json:"base_stats"`
}

type BaseStats struct {
	HP             int16 `json:"hp"`
	Attack         int16 `json:"attack"`
	Defense        int16 `json:"defense"`
	SpecialAttack  int16 `json:"special_attack"`
	SpecialDefense int16 `json:"special_defense"`
	Speed          int16 `json:"speed"`
}

func speciesFromRow(row sqlcgen.SearchSpeciesRow) Species {
	types := []string{row.Type1}
	if row.Type2 != nil {
		types = append(types, *row.Type2)
	}
	return Species{
		ID:          row.ID,
		Identifier:  row.Identifier,
		NationalDex: row.NationalDex,
		IsDefault:   row.IsDefault,
		Generation:  row.Generation,
		Types:       types,
		BaseStats: BaseStats{
			HP:             row.BaseHp,
			Attack:         row.BaseAttack,
			Defense:        row.BaseDefense,
			SpecialAttack:  row.BaseSpecialAttack,
			SpecialDefense: row.BaseSpecialDefense,
			Speed:          row.BaseSpeed,
		},
	}
}

func (s *Store) SearchSpecies(ctx context.Context, name string) ([]Species, error) {
	rows, err := s.q.SearchSpecies(ctx, likePattern(name))
	if err != nil {
		return nil, fmt.Errorf("pokedex: search species: %w", err)
	}
	out := make([]Species, 0, len(rows))
	for _, row := range rows {
		out = append(out, speciesFromRow(row))
	}
	return out, nil
}
