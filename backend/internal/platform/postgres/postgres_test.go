package postgres

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

// liveDSNEnv names a reachable Postgres for the one test that needs a server.
// Absent, that test skips; every other test here runs without a database.
const liveDSNEnv = "TEST_DATABASE_URL"

// unreachableDSN points at a port that was bound and released, so nothing is
// listening on it and a dial is refused rather than hanging.
func unreachableDSN(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	return fmt.Sprintf("postgres://someone:secret@127.0.0.1:%d/nodb?sslmode=disable", port)
}

// The documented lazy-connect property: Open validates the DSN but must not
// dial, so the api boots and reports the outage through /healthz instead of
// crash-looping while Postgres is down.
func TestOpenDoesNotDial(t *testing.T) {
	pool, err := Open(context.Background(), unreachableDSN(t))
	if err != nil {
		t.Fatalf("Open() against a down server = %v, want no error", err)
	}
	pool.Close()
}

func TestOpenRejectsMalformedDSN(t *testing.T) {
	for _, dsn := range []string{"not a dsn at all", "://nope", "postgres://u:p@127.0.0.1:notaport/db"} {
		t.Run(dsn, func(t *testing.T) {
			pool, err := Open(context.Background(), dsn)
			if err == nil {
				pool.Close()
				t.Fatalf("Open(%q) succeeded, want a parse error", dsn)
			}
			if !strings.Contains(err.Error(), "postgres: open pool") {
				t.Errorf("error %q is not wrapped by this package", err)
			}
		})
	}
}

func TestPingUnreachable(t *testing.T) {
	pool, err := Open(context.Background(), unreachableDSN(t))
	if err != nil {
		t.Fatalf("Open() = %v", err)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := NewPinger(pool).Ping(ctx); err == nil {
		t.Fatal("Ping() against a down server = nil, want an error")
	} else if !strings.Contains(err.Error(), "postgres: ping") {
		t.Errorf("error %q is not wrapped by this package", err)
	}
}

func TestPingClosedPool(t *testing.T) {
	pool, err := Open(context.Background(), unreachableDSN(t))
	if err != nil {
		t.Fatalf("Open() = %v", err)
	}
	pool.Close()

	if err := NewPinger(pool).Ping(context.Background()); err == nil {
		t.Fatal("Ping() on a closed pool = nil, want an error")
	}
}

// PoolPinger must satisfy the interface the health handler depends on.
var _ Pinger = PoolPinger{}

func TestPingLiveServer(t *testing.T) {
	dsn := os.Getenv(liveDSNEnv)
	if dsn == "" {
		t.Skipf("%s not set; skipping the live-server case", liveDSNEnv)
	}

	pool, err := Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("Open() = %v", err)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := NewPinger(pool).Ping(ctx); err != nil {
		t.Fatalf("Ping() against %s = %v, want nil", liveDSNEnv, err)
	}
}
