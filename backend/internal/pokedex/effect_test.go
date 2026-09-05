package pokedex

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParseEffects(t *testing.T) {
	tests := []struct {
		name string
		raw  []string
		want EffectQuery
	}{
		{"stat raise", []string{"stat:attack:2"}, EffectQuery{StatChanges: []StatChange{{Stat: "attack", Change: 2}}}},
		{"stat explicit plus", []string{"stat:attack:+2"}, EffectQuery{StatChanges: []StatChange{{Stat: "attack", Change: 2}}}},
		{"stat lower", []string{"stat:speed:-1"}, EffectQuery{StatChanges: []StatChange{{Stat: "speed", Change: -1}}}},
		{"ailment", []string{"ailment:burn"}, EffectQuery{Ailments: []string{"burn"}}},
		{"combined", []string{"stat:attack:2", "ailment:burn"}, EffectQuery{
			StatChanges: []StatChange{{Stat: "attack", Change: 2}},
			Ailments:    []string{"burn"},
		}},
		{"whitespace trimmed", []string{" stat:defense:1 "}, EffectQuery{StatChanges: []StatChange{{Stat: "defense", Change: 1}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEffects(tt.raw)
			if err != nil {
				t.Fatalf("ParseEffects(%q) = %v", tt.raw, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseEffects(%q) = %+v, want %+v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestParseEffectsInvalid(t *testing.T) {
	tests := []struct {
		name string
		raw  []string
	}{
		{"empty predicate", []string{""}},
		{"unknown kind", []string{"weather:rain"}},
		{"stat missing change", []string{"stat:attack"}},
		{"stat missing name", []string{"stat::2"}},
		{"stat non-numeric", []string{"stat:attack:two"}},
		{"stat zero", []string{"stat:attack:0"}},
		{"stat above range", []string{"stat:attack:7"}},
		{"stat below range", []string{"stat:attack:-7"}},
		{"stat extra part", []string{"stat:attack:2:x"}},
		{"ailment missing name", []string{"ailment:"}},
		{"ailment extra part", []string{"ailment:burn:x"}},
		{"second predicate bad", []string{"stat:attack:2", "bogus"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseEffects(tt.raw)
			var ie *InvalidEffectError
			if !errors.As(err, &ie) {
				t.Fatalf("ParseEffects(%q) err = %v, want *InvalidEffectError", tt.raw, err)
			}
			if ie.Predicate != tt.raw[len(tt.raw)-1] {
				t.Errorf("Predicate = %q, want %q", ie.Predicate, tt.raw[len(tt.raw)-1])
			}
		})
	}
}

func TestParseEffectsLimits(t *testing.T) {
	tooMany := make([]string, maxEffectPredicates+1)
	for i := range tooMany {
		tooMany[i] = "stat:attack:1"
	}
	tooLong := "ailment:" + strings.Repeat("x", maxPredicateLen)

	tests := []struct {
		name          string
		raw           []string
		wantPredicate string
	}{
		{"too many predicates", tooMany, "stat:attack:1"},
		{"predicate too long", []string{tooLong}, tooLong[:maxPredicateLen]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseEffects(tt.raw)
			var ie *InvalidEffectError
			if !errors.As(err, &ie) {
				t.Fatalf("ParseEffects() err = %v, want *InvalidEffectError", err)
			}
			if ie.Predicate != tt.wantPredicate {
				t.Errorf("Predicate = %q, want %q", ie.Predicate, tt.wantPredicate)
			}
		})
	}

	atLimit := make([]string, maxEffectPredicates)
	for i := range atLimit {
		atLimit[i] = "stat:attack:1"
	}
	if _, err := ParseEffects(atLimit); err != nil {
		t.Errorf("ParseEffects() at the predicate limit = %v, want nil", err)
	}
	if _, err := ParseEffects([]string{tooLong[:maxPredicateLen]}); err != nil {
		t.Errorf("ParseEffects() at the length limit = %v, want nil", err)
	}
}
