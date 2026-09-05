package pokedex

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

type fakeSearcher struct {
	species    []Species
	moves      []Move
	abilities  []Ability
	items      []Item
	err        error
	gotDataSet string
	effects    []MoveWithEffect
	gotEffect  EffectQuery
	effectCall bool
}

func (f *fakeSearcher) SearchSpecies(_ context.Context, _ string) ([]Species, error) {
	return f.species, f.err
}

func (f *fakeSearcher) SearchMoves(_ context.Context, _, dataSet string) ([]Move, error) {
	f.gotDataSet = dataSet
	return f.moves, f.err
}

func (f *fakeSearcher) SearchAbilities(_ context.Context, _, dataSet string) ([]Ability, error) {
	f.gotDataSet = dataSet
	return f.abilities, f.err
}

func (f *fakeSearcher) SearchItems(_ context.Context, _, dataSet string) ([]Item, error) {
	f.gotDataSet = dataSet
	return f.items, f.err
}

func (f *fakeSearcher) SearchMovesByEffect(_ context.Context, q EffectQuery, dataSet string) ([]MoveWithEffect, error) {
	f.effectCall = true
	f.gotEffect = q
	f.gotDataSet = dataSet
	return f.effects, f.err
}

func TestSpeciesSearchMissingName(t *testing.T) {
	r := Router(discardLogger(), &fakeSearcher{}, "gen-9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/species", nil))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var body map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body["error"]["code"] != "invalid_request" {
		t.Errorf("code = %q, want invalid_request", body["error"]["code"])
	}
}

func TestSpeciesSearchNoMatch(t *testing.T) {
	r := Router(discardLogger(), &fakeSearcher{species: []Species{}}, "gen-9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/species?name=zzznotreal", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	want := `{"results":[],"count":0}` + "\n"
	if rec.Body.String() != want {
		t.Errorf("body = %q, want %q", rec.Body.String(), want)
	}
}

func TestSpeciesSearchMultipleForms(t *testing.T) {
	fake := &fakeSearcher{species: []Species{
		{ID: 52, Identifier: "meowth", NationalDex: 52, IsDefault: true},
		{ID: 863, Identifier: "meowth-alola", NationalDex: 52, IsDefault: false},
	}}
	r := Router(discardLogger(), fake, "gen-9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/species?name=meowth", nil))

	var body struct {
		Results []Species `json:"results"`
		Count   int       `json:"count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body.Count != 2 || len(body.Results) != 2 {
		t.Errorf("count = %d, len(results) = %d, want 2/2", body.Count, len(body.Results))
	}
}

func TestMovesSearchDefaultDataSet(t *testing.T) {
	fake := &fakeSearcher{moves: []Move{{ID: 85, Identifier: "thunderbolt"}}}
	r := Router(discardLogger(), fake, "gen-9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/moves?name=thunderbolt", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if fake.gotDataSet != "gen-9" {
		t.Errorf("resolved data set = %q, want gen-9 (the default)", fake.gotDataSet)
	}
	var body struct {
		DataSet string `json:"data_set"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body.DataSet != "gen-9" {
		t.Errorf("echoed data_set = %q, want gen-9", body.DataSet)
	}
}

func TestMovesSearchExplicitDataSet(t *testing.T) {
	fake := &fakeSearcher{moves: []Move{{ID: 85, Identifier: "thunderbolt"}}}
	r := Router(discardLogger(), fake, "gen-9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/moves?name=thunderbolt&data_set=champions", nil))

	if fake.gotDataSet != "champions" {
		t.Errorf("resolved data set = %q, want champions", fake.gotDataSet)
	}
}

func TestMovesSearchUnknownDataSet(t *testing.T) {
	fake := &fakeSearcher{err: ErrDataSetNotFound}
	r := Router(discardLogger(), fake, "gen-9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/moves?name=thunderbolt&data_set=nope", nil))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var body map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body["error"]["code"] != "unknown_data_set" {
		t.Errorf("code = %q, want unknown_data_set", body["error"]["code"])
	}
}

func TestSearchInternalError(t *testing.T) {
	fake := &fakeSearcher{err: errBoom}
	r := Router(discardLogger(), fake, "gen-9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/species?name=pikachu", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestAbilitiesAndItemsSearch(t *testing.T) {
	fake := &fakeSearcher{
		abilities: []Ability{{ID: 9, Identifier: "static"}},
		items:     []Item{{ID: 234, Identifier: "leftovers"}},
	}
	r := Router(discardLogger(), fake, "gen-9")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/abilities?name=static", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("abilities status = %d, want %d", rec.Code, http.StatusOK)
	}

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/items?name=leftovers", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("items status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHeadSpecies(t *testing.T) {
	fake := &fakeSearcher{species: []Species{{ID: 25, Identifier: "pikachu"}}}
	r := Router(discardLogger(), fake, "gen-9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, "/species?name=pikachu", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestUnknownPathUnderMount(t *testing.T) {
	r := Router(discardLogger(), &fakeSearcher{}, "gen-9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestWrongMethod(t *testing.T) {
	r := Router(discardLogger(), &fakeSearcher{}, "gen-9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/species", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("Allow"); got != "GET, HEAD" {
		t.Errorf("Allow = %q, want GET, HEAD", got)
	}
}

var errBoom = &searchError{"boom"}

type searchError struct{ msg string }

func (e *searchError) Error() string { return e.msg }

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	return body["error"]["code"]
}

func TestMovesEffectSearchMissingEffect(t *testing.T) {
	fake := &fakeSearcher{}
	r := Router(discardLogger(), fake, "gen-9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/moves/search", nil))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if code := errorCode(t, rec); code != "invalid_request" {
		t.Errorf("code = %q, want invalid_request", code)
	}
	if fake.effectCall {
		t.Error("store was called with no effect predicate")
	}
}

func TestMovesEffectSearchMalformedPredicate(t *testing.T) {
	fake := &fakeSearcher{}
	r := Router(discardLogger(), fake, "gen-9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/moves/search?effect=stat:attack:9", nil))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if code := errorCode(t, rec); code != "invalid_effect" {
		t.Errorf("code = %q, want invalid_effect", code)
	}
	if fake.effectCall {
		t.Error("store was called with a malformed predicate")
	}
}

func TestMovesEffectSearchUnknownStat(t *testing.T) {
	fake := &fakeSearcher{err: &InvalidEffectError{Predicate: "stat:atk:2", Reason: "unknown stat"}}
	r := Router(discardLogger(), fake, "gen-9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/moves/search?effect=stat:atk:2", nil))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if code := errorCode(t, rec); code != "invalid_effect" {
		t.Errorf("code = %q, want invalid_effect", code)
	}
}

func TestMovesEffectSearchUnknownDataSet(t *testing.T) {
	fake := &fakeSearcher{err: ErrDataSetNotFound}
	r := Router(discardLogger(), fake, "gen-9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/moves/search?effect=stat:attack:2&data_set=nope", nil))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if code := errorCode(t, rec); code != "unknown_data_set" {
		t.Errorf("code = %q, want unknown_data_set", code)
	}
}

func TestMovesEffectSearch(t *testing.T) {
	category := "net-good-stats"
	fake := &fakeSearcher{effects: []MoveWithEffect{{
		Move: Move{ID: 14, Identifier: "swords-dance"},
		Effect: MoveEffect{
			Category:    &category,
			StatChanges: []StatChange{{Stat: "attack", Change: 2}},
		},
	}}}
	r := Router(discardLogger(), fake, "gen-9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/moves/search?effect=stat:attack:%2B2&effect=ailment:none", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	wantQuery := EffectQuery{StatChanges: []StatChange{{Stat: "attack", Change: 2}}, Ailments: []string{"none"}}
	if !reflect.DeepEqual(fake.gotEffect, wantQuery) {
		t.Errorf("query = %+v, want %+v", fake.gotEffect, wantQuery)
	}
	if fake.gotDataSet != "gen-9" {
		t.Errorf("resolved data set = %q, want gen-9 (the default)", fake.gotDataSet)
	}

	var body struct {
		DataSet string           `json:"data_set"`
		Results []MoveWithEffect `json:"results"`
		Count   int              `json:"count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body.DataSet != "gen-9" || body.Count != 1 || len(body.Results) != 1 {
		t.Fatalf("envelope = %+v, want data_set gen-9, count 1", body)
	}
	got := body.Results[0]
	if got.Identifier != "swords-dance" || got.Effect.Category == nil || *got.Effect.Category != category {
		t.Errorf("result = %+v, want swords-dance with category %q", got, category)
	}
	if !reflect.DeepEqual(got.Effect.StatChanges, []StatChange{{Stat: "attack", Change: 2}}) {
		t.Errorf("stat_changes = %+v, want attack +2", got.Effect.StatChanges)
	}
}

func TestMovesEffectSearchNoMatch(t *testing.T) {
	fake := &fakeSearcher{effects: []MoveWithEffect{}}
	r := Router(discardLogger(), fake, "gen-9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/moves/search?effect=stat:hp:1", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	want := `{"data_set":"gen-9","results":[],"count":0}` + "\n"
	if rec.Body.String() != want {
		t.Errorf("body = %q, want %q", rec.Body.String(), want)
	}
}

func TestHeadMovesEffectSearch(t *testing.T) {
	fake := &fakeSearcher{effects: []MoveWithEffect{{Move: Move{ID: 14, Identifier: "swords-dance"}}}}
	r := Router(discardLogger(), fake, "gen-9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, "/moves/search?effect=stat:attack:2", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
