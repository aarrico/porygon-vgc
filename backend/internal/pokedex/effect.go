package pokedex

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aarrico/porygon-vgc/backend/internal/pokedex/internal/sqlcgen"
)

const (
	minStatStage = -6
	maxStatStage = 6
	// A real query is one to three predicates. Each one is another element
	// in a per-row correlated subquery, so the count is bounded here rather
	// than by whatever the client sends.
	maxEffectPredicates = 8
	maxPredicateLen     = 64
)

type StatChange struct {
	Stat   string `json:"stat"`
	Change int16  `json:"change"`
}

type EffectQuery struct {
	StatChanges []StatChange
	Ailments    []string
}

type MoveEffect struct {
	Category      *string      `json:"category"`
	Ailment       *string      `json:"ailment"`
	AilmentChance *int16       `json:"ailment_chance"`
	StatChance    *int16       `json:"stat_chance"`
	FlinchChance  *int16       `json:"flinch_chance"`
	CritRate      *int16       `json:"crit_rate"`
	Drain         *int16       `json:"drain"`
	Healing       *int16       `json:"healing"`
	MinHits       *int16       `json:"min_hits"`
	MaxHits       *int16       `json:"max_hits"`
	MinTurns      *int16       `json:"min_turns"`
	MaxTurns      *int16       `json:"max_turns"`
	StatChanges   []StatChange `json:"stat_changes"`
}

type MoveWithEffect struct {
	Move
	Effect MoveEffect `json:"effect"`
}

type InvalidEffectError struct {
	Predicate string
	Reason    string
}

func (e *InvalidEffectError) Error() string {
	return fmt.Sprintf("invalid effect %q: %s", e.Predicate, e.Reason)
}

// ParseEffects reads repeated `effect` values of the form <kind>:<value>[:<arg>]
// and combines them as a conjunction.
func ParseEffects(raw []string) (EffectQuery, error) {
	if len(raw) > maxEffectPredicates {
		return EffectQuery{}, &InvalidEffectError{Predicate: raw[maxEffectPredicates], Reason: fmt.Sprintf("at most %d effect predicates per request", maxEffectPredicates)}
	}
	var q EffectQuery
	for _, r := range raw {
		predicate := strings.TrimSpace(r)
		if len(predicate) > maxPredicateLen {
			return EffectQuery{}, &InvalidEffectError{Predicate: predicate[:maxPredicateLen], Reason: fmt.Sprintf("predicate exceeds %d characters", maxPredicateLen)}
		}
		parts := strings.Split(predicate, ":")
		switch parts[0] {
		case "stat":
			sc, err := parseStatChange(predicate, parts)
			if err != nil {
				return EffectQuery{}, err
			}
			q.StatChanges = append(q.StatChanges, sc)
		case "ailment":
			if len(parts) != 2 || parts[1] == "" {
				return EffectQuery{}, &InvalidEffectError{Predicate: predicate, Reason: "want ailment:<identifier>"}
			}
			q.Ailments = append(q.Ailments, parts[1])
		default:
			return EffectQuery{}, &InvalidEffectError{Predicate: predicate, Reason: "unknown kind; want stat or ailment"}
		}
	}
	return q, nil
}

func parseStatChange(predicate string, parts []string) (StatChange, error) {
	if len(parts) != 3 || parts[1] == "" {
		return StatChange{}, &InvalidEffectError{Predicate: predicate, Reason: "want stat:<identifier>:<stages>"}
	}
	n, err := strconv.ParseInt(parts[2], 10, 16)
	if err != nil || n == 0 || n < minStatStage || n > maxStatStage {
		return StatChange{}, &InvalidEffectError{
			Predicate: predicate,
			Reason:    fmt.Sprintf("stages must be a non-zero integer between %d and %d", minStatStage, maxStatStage),
		}
	}
	return StatChange{Stat: parts[1], Change: int16(n)}, nil
}

func (s *Store) SearchMovesByEffect(ctx context.Context, q EffectQuery, dataSet string) ([]MoveWithEffect, error) {
	// pgx encodes a nil slice as SQL NULL, which would turn every array
	// predicate in the query into NULL and match nothing.
	statIdentifiers := make([]string, len(q.StatChanges))
	statChanges := make([]int16, len(q.StatChanges))
	for i, sc := range q.StatChanges {
		statIdentifiers[i] = sc.Stat
		statChanges[i] = sc.Change
	}
	ailments := q.Ailments
	if ailments == nil {
		ailments = []string{}
	}

	rows, err := s.q.SearchMovesByEffect(ctx, sqlcgen.SearchMovesByEffectParams{
		DataSet:         dataSet,
		Ailments:        ailments,
		StatIdentifiers: statIdentifiers,
		StatChanges:     statChanges,
	})
	if err != nil {
		return nil, fmt.Errorf("pokedex: search moves by effect: %w", err)
	}
	if len(rows) == 0 {
		if err := s.checkDataSetExists(ctx, dataSet); err != nil {
			return nil, err
		}
		if err := s.checkEffectVocabulary(ctx, q, statIdentifiers, ailments); err != nil {
			return nil, err
		}
		return []MoveWithEffect{}, nil
	}

	moveIDs := make([]int64, len(rows))
	for i, row := range rows {
		moveIDs[i] = row.ID
	}
	changeRows, err := s.q.ListMoveStatChanges(ctx, moveIDs)
	if err != nil {
		return nil, fmt.Errorf("pokedex: list move stat changes: %w", err)
	}
	changesByMove := make(map[int64][]StatChange, len(rows))
	for _, c := range changeRows {
		changesByMove[c.MoveID] = append(changesByMove[c.MoveID], StatChange{Stat: c.Stat, Change: c.Change})
	}

	out := make([]MoveWithEffect, 0, len(rows))
	for _, row := range rows {
		changes := changesByMove[row.ID]
		if changes == nil {
			changes = []StatChange{}
		}
		out = append(out, moveWithEffectFromRow(row, changes))
	}
	return out, nil
}

func (s *Store) checkEffectVocabulary(ctx context.Context, q EffectQuery, statIdentifiers, ailments []string) error {
	unknownStats, err := s.q.ListUnknownStats(ctx, statIdentifiers)
	if err != nil {
		return fmt.Errorf("pokedex: check stat identifiers: %w", err)
	}
	if len(unknownStats) > 0 {
		for _, sc := range q.StatChanges {
			if sc.Stat == unknownStats[0] {
				return &InvalidEffectError{Predicate: fmt.Sprintf("stat:%s:%d", sc.Stat, sc.Change), Reason: "unknown stat"}
			}
		}
	}
	unknownAilments, err := s.q.ListUnknownAilments(ctx, ailments)
	if err != nil {
		return fmt.Errorf("pokedex: check ailment identifiers: %w", err)
	}
	if len(unknownAilments) > 0 {
		return &InvalidEffectError{Predicate: "ailment:" + unknownAilments[0], Reason: "unknown ailment"}
	}
	return nil
}

func moveWithEffectFromRow(row sqlcgen.SearchMovesByEffectRow, changes []StatChange) MoveWithEffect {
	return MoveWithEffect{
		Move: Move{
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
		},
		Effect: MoveEffect{
			Category:      row.Category,
			Ailment:       row.Ailment,
			AilmentChance: row.AilmentChance,
			StatChance:    row.StatChance,
			FlinchChance:  row.FlinchChance,
			CritRate:      row.CritRate,
			Drain:         row.Drain,
			Healing:       row.Healing,
			MinHits:       row.MinHits,
			MaxHits:       row.MaxHits,
			MinTurns:      row.MinTurns,
			MaxTurns:      row.MaxTurns,
			StatChanges:   changes,
		},
	}
}
