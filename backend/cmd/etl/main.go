// Command etl loads the pinned PokeAPI CSV dump into the core reference
// schema. It is a separate binary from the api because only the ETL role may
// write those tables (AD-6).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/aarrico/porygon-vgc/backend/internal/etl"
	"github.com/aarrico/porygon-vgc/backend/internal/platform/config"
	"github.com/aarrico/porygon-vgc/backend/internal/platform/logging"
	"github.com/aarrico/porygon-vgc/backend/internal/platform/postgres"
)

const (
	loadTimeout    = 5 * time.Minute
	defaultDataSet = "gen-9"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := logging.New(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, loadTimeout)
	defer cancel()

	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	dataSet := defaultDataSet
	if v := os.Getenv("DATA_SET"); v != "" {
		dataSet = v
	}

	started := time.Now()
	summary, err := etl.Load(ctx, pool, dataSet)
	if err != nil {
		return err
	}

	// Grouped: a table named data_set would otherwise collide with the
	// data_set field naming the load.
	rows := make([]any, 0, len(summary.Rows)*2)
	for _, table := range slices.Sorted(maps.Keys(summary.Rows)) {
		rows = append(rows, table, summary.Rows[table])
	}
	logger.Info("etl load complete",
		"data_set", summary.DataSet,
		"duration_ms", time.Since(started).Milliseconds(),
		slog.Group("rows", rows...))
	return nil
}
