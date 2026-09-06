package etl

import (
	"embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// champout is Project Pokemon's dump of the Champions game data; see
// data/champout/SOURCE.md. Table names are Game Freak's Japanese ones:
// waza = move, tokusei = ability, personal = species.
//
//go:embed data/champout/*.json
var champout embed.FS

// readChampout parses one masterdata table. Every value in the dump is a
// string, so rows share the CSV loader's record type and typed parser.
func readChampout(name string, required ...string) ([]record, error) {
	b, err := champout.ReadFile("data/champout/" + name + ".json")
	if err != nil {
		return nil, fmt.Errorf("etl: open %s.json: %w", name, err)
	}
	var rows []record
	if err := json.Unmarshal(b, &rows); err != nil {
		return nil, fmt.Errorf("etl: %s.json: %w", name, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("etl: %s.json: no rows", name)
	}
	for _, col := range required {
		if _, ok := rows[0][col]; !ok {
			return nil, fmt.Errorf("etl: %s.json: missing column %q", name, col)
		}
	}
	return rows, nil
}

// readLabels parses one rom-txt Unity text asset into label -> English text.
func readLabels(name string) (map[string]string, error) {
	b, err := champout.ReadFile("data/champout/" + name + ".json")
	if err != nil {
		return nil, fmt.Errorf("etl: open %s.json: %w", name, err)
	}
	var asset struct {
		Entries []struct {
			Label string `json:"LabelName"`
			Text  string `json:"OriginalText"`
		} `json:"mSDataSet"`
	}
	if err := json.Unmarshal(b, &asset); err != nil {
		return nil, fmt.Errorf("etl: %s.json: %w", name, err)
	}
	if len(asset.Entries) == 0 {
		return nil, fmt.Errorf("etl: %s.json: no entries", name)
	}
	out := make(map[string]string, len(asset.Entries))
	for _, e := range asset.Entries {
		out[e.Label] = e.Text
	}
	return out, nil
}

var (
	slugStrip = strings.NewReplacer("’", "", "'", "", ".", "", "é", "e")
	slugSep   = regexp.MustCompile(`[^a-z0-9]+`)
)

// slug turns a display name into a PokeAPI identifier: "King's Rock" ->
// "kings-rock". The vendored dump is pinned against this in champout_test.go.
func slug(name string) string {
	return strings.Trim(slugSep.ReplaceAllString(slugStrip.Replace(strings.ToLower(name)), "-"), "-")
}
