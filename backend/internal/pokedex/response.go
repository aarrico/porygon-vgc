package pokedex

type searchResponse[T any] struct {
	Results []T `json:"results"`
	Count   int `json:"count"`
}

type versionedResponse[T any] struct {
	DataSet string `json:"data_set"`
	Results []T    `json:"results"`
	Count   int    `json:"count"`
}
