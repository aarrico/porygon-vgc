package pokedex

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/aarrico/porygon-vgc/backend/internal/platform/httpx"
)

type searcher interface {
	SearchSpecies(ctx context.Context, name string) ([]Species, error)
	SearchMoves(ctx context.Context, name, dataSet string) ([]Move, error)
	SearchAbilities(ctx context.Context, name, dataSet string) ([]Ability, error)
	SearchItems(ctx context.Context, name, dataSet string) ([]Item, error)
}

func Router(logger *slog.Logger, s searcher, defaultDataSet string) chi.Router {
	r := httpx.NewRouter(http.MethodGet, http.MethodHead)

	species := plainSearchHandler(logger, s.SearchSpecies)
	r.Get("/species", species)
	r.Head("/species", species)

	moves := versionedSearchHandler(logger, defaultDataSet, s.SearchMoves)
	r.Get("/moves", moves)
	r.Head("/moves", moves)

	abilities := versionedSearchHandler(logger, defaultDataSet, s.SearchAbilities)
	r.Get("/abilities", abilities)
	r.Head("/abilities", abilities)

	items := versionedSearchHandler(logger, defaultDataSet, s.SearchItems)
	r.Get("/items", items)
	r.Head("/items", items)

	return r
}

func plainSearchHandler[T any](logger *slog.Logger, search func(ctx context.Context, name string) ([]T, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name, ok := requireName(r)
		if !ok {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "name query parameter is required")
			return
		}

		results, err := search(r.Context(), name)
		if err != nil {
			logger.Error("pokedex search failed", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal server error")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, searchResponse[T]{Results: results, Count: len(results)})
	}
}

func versionedSearchHandler[T any](logger *slog.Logger, defaultDataSet string, search func(ctx context.Context, name, dataSet string) ([]T, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name, ok := requireName(r)
		if !ok {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "name query parameter is required")
			return
		}

		dataSet := strings.TrimSpace(r.URL.Query().Get("data_set"))
		if dataSet == "" {
			dataSet = defaultDataSet
		}

		results, err := search(r.Context(), name, dataSet)
		if err != nil {
			if errors.Is(err, ErrDataSetNotFound) {
				httpx.WriteError(w, http.StatusBadRequest, "unknown_data_set", fmt.Sprintf("data set %q not found", dataSet))
				return
			}
			logger.Error("pokedex search failed", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal server error")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, versionedResponse[T]{DataSet: dataSet, Results: results, Count: len(results)})
	}
}
