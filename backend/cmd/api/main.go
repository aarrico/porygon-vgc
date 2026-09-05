// Command api is the single deployable binary of the modular monolith (AD-1).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aarrico/porygon-vgc/backend/internal/platform/config"
	"github.com/aarrico/porygon-vgc/backend/internal/platform/httpx"
	"github.com/aarrico/porygon-vgc/backend/internal/platform/logging"
	"github.com/aarrico/porygon-vgc/backend/internal/platform/postgres"
	"github.com/aarrico/porygon-vgc/backend/internal/pokedex"
)

const (
	healthPingTimeout = 2 * time.Second
	shutdownTimeout   = 10 * time.Second
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
	maxHeaderBytes    = 1 << 20
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

	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	pokedexRouter := pokedex.Router(logger, pokedex.NewStore(pool), cfg.DataSet)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           newHandler(logger, postgres.NewPinger(pool), pokedexRouter),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}

	ln, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.HTTPAddr, err)
	}

	return serve(ctx, srv, ln, stop, logger)
}

// serve runs srv until ctx is cancelled, then drains in-flight requests. The
// listener and the signal-restore func are parameters so a test can bind an
// ephemeral port and cancel without raising a real signal.
func serve(ctx context.Context, srv *http.Server, ln net.Listener, stop func(), logger *slog.Logger) error {
	serveErr := make(chan error, 1)
	go func() {
		// The bound address, not the requested one: ":0" and an unspecified
		// host would otherwise log a value nothing can connect to.
		logger.Info("http server listening", "addr", ln.Addr().String())
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	// Restoring the default signal disposition before draining means a second
	// SIGTERM kills the process instead of being swallowed.
	stop()
	logger.Info("shutdown signal received", "timeout", shutdownTimeout.String())

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close()
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	// A Serve failure raised after ctx.Done() was selected is still sitting in
	// the buffered channel; without this read the process would exit 0 on it.
	select {
	case err := <-serveErr:
		return err
	default:
	}

	logger.Info("shutdown complete")
	return nil
}

func newHandler(logger *slog.Logger, pinger postgres.Pinger, pokedexRouter http.Handler) http.Handler {
	r := httpx.NewRouter(http.MethodGet, http.MethodHead)
	r.Use(httpx.RequestLogger(logger), httpx.Recoverer(logger))

	r.Get("/healthz", healthHandler(logger, pinger))
	r.Head("/healthz", healthHandler(logger, pinger))

	r.Mount("/v1", pokedexRouter)

	return r
}

// healthHandler reports readiness, not liveness: it is the gate later services
// depend on, so it must reflect Postgres reachability.
func healthHandler(logger *slog.Logger, pinger postgres.Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), healthPingTimeout)
		defer cancel()

		if err := pinger.Ping(ctx); err != nil {
			logger.Warn("health check failed", "error", err)
			httpx.WriteError(w, http.StatusServiceUnavailable, "dependency_unavailable", "postgres unreachable")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
