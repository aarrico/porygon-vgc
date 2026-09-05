package pokedex

import (
	"net/http"
	"strings"
)

func requireName(r *http.Request) (string, bool) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	return name, name != ""
}

func resolveDataSet(r *http.Request, defaultDataSet string) string {
	if dataSet := strings.TrimSpace(r.URL.Query().Get("data_set")); dataSet != "" {
		return dataSet
	}
	return defaultDataSet
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func likePattern(name string) string {
	return "%" + likeEscaper.Replace(name) + "%"
}
