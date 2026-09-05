package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/aarrico/porygon-vgc/backend/internal/platform/httpx"
)

func stubPokedexRouter() http.Handler {
	return httpx.NewRouter(http.MethodGet, http.MethodHead)
}

// subprocessEnv makes the test binary run as the api binary instead of running
// tests, so the signal wiring and the fail-fast boot path are exercised for
// real rather than simulated.
const subprocessEnv = "PORYGON_API_TEST_SUBPROCESS"

func TestMain(m *testing.M) {
	if os.Getenv(subprocessEnv) == "1" {
		main()
		return
	}
	os.Exit(m.Run())
}

// stubPinger stands in for the pool so the failing-dependency row of the I/O
// matrix is reachable without a database.
type stubPinger struct {
	err   error
	block time.Duration
}

func (s stubPinger) Ping(ctx context.Context) error {
	if s.block > 0 {
		select {
		case <-time.After(s.block):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return s.err
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func TestRoutes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		pinger     stubPinger
		wantStatus int
		wantAllow  string
		wantBody   map[string]any
	}{
		{
			name:       "health with postgres reachable",
			method:     http.MethodGet,
			path:       "/healthz",
			pinger:     stubPinger{},
			wantStatus: http.StatusOK,
			wantBody:   map[string]any{"status": "ok"},
		},
		{
			name:       "health with postgres unreachable",
			method:     http.MethodGet,
			path:       "/healthz",
			pinger:     stubPinger{err: errors.New("dial tcp: connection refused")},
			wantStatus: http.StatusServiceUnavailable,
			wantBody: map[string]any{
				"error": map[string]any{
					"code":    "dependency_unavailable",
					"message": "postgres unreachable",
				},
			},
		},
		{
			name:       "unknown path",
			method:     http.MethodGet,
			path:       "/v1/nope",
			pinger:     stubPinger{},
			wantStatus: http.StatusNotFound,
			wantBody: map[string]any{
				"error": map[string]any{
					"code":    "not_found",
					"message": "resource not found",
				},
			},
		},
		{
			name:       "root path",
			method:     http.MethodGet,
			path:       "/",
			pinger:     stubPinger{},
			wantStatus: http.StatusNotFound,
			wantBody: map[string]any{
				"error": map[string]any{
					"code":    "not_found",
					"message": "resource not found",
				},
			},
		},
		{
			// A method mismatch on an existing resource is 405, not 404: the
			// path exists, the verb does not.
			name:       "wrong method on health",
			method:     http.MethodPost,
			path:       "/healthz",
			pinger:     stubPinger{},
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  "GET, HEAD",
			wantBody: map[string]any{
				"error": map[string]any{
					"code":    "method_not_allowed",
					"message": "method not allowed on this resource",
				},
			},
		},
		{
			name:       "delete on health",
			method:     http.MethodDelete,
			path:       "/healthz",
			pinger:     stubPinger{},
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  "GET, HEAD",
			wantBody: map[string]any{
				"error": map[string]any{
					"code":    "method_not_allowed",
					"message": "method not allowed on this resource",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHandler(discardLogger(), tc.pinger, stubPokedexRouter())
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}

			if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q, want application/json; charset=utf-8", got)
			}

			if got := rec.Header().Get("Allow"); got != tc.wantAllow {
				t.Errorf("Allow = %q, want %q", got, tc.wantAllow)
			}

			var got map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("body is not JSON: %v (%q)", err, rec.Body.String())
			}

			wantJSON, _ := json.Marshal(tc.wantBody)
			gotJSON, _ := json.Marshal(got)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("body = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

// A hung database must not hang the handler: the ping runs under a bounded
// context and the request still answers 503.
func TestHealthPingTimeout(t *testing.T) {
	h := newHandler(discardLogger(), stubPinger{block: 2 * healthPingTimeout}, stubPokedexRouter())
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(healthPingTimeout + 5*time.Second):
		t.Fatal("handler did not return within the ping timeout")
	}

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

type inFlightResult struct {
	status int
	body   string
	err    error
}

// The SIGTERM row of the I/O matrix at the serve() level: a request already in
// the handler when the context is cancelled still gets its complete response,
// and serve reports success — the exit-0 half. TestSubprocessSIGTERM below
// covers the same row through a real signal.
func TestServeDrainsInFlightRequest(t *testing.T) {
	var once sync.Once
	entered := make(chan struct{})
	release := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /slow", func(w http.ResponseWriter, _ *http.Request) {
		// A retried request must not close a closed channel.
		once.Do(func() { close(entered) })
		<-release
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"status":"drained"}`)
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- serve(ctx, srv, ln, func() {}, discardLogger())
	}()

	results := make(chan inFlightResult, 1)
	go func() {
		resp, err := http.Get("http://" + addr + "/slow") //nolint:noctx // the point of the test is a request outliving its server
		if err != nil {
			results <- inFlightResult{err: err}
			return
		}
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		results <- inFlightResult{status: resp.StatusCode, body: string(body), err: err}
	}()

	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("handler never received the request")
	}

	cancel()
	// Deterministic ordering without a wall-clock guess: Shutdown closes the
	// listener before it waits for in-flight work, so a refused dial proves the
	// drain has begun while the handler is still parked.
	waitForRefusedDial(t, addr)
	close(release)

	var got inFlightResult
	select {
	case got = <-results:
	case <-time.After(10 * time.Second):
		t.Fatal("in-flight request never completed")
	}

	if got.err != nil {
		t.Fatalf("in-flight request failed: %v", got.err)
	}
	if got.status != http.StatusOK {
		t.Errorf("in-flight status = %d, want %d", got.status, http.StatusOK)
	}
	if got.body != `{"status":"drained"}` {
		t.Errorf("in-flight body = %q, want %q", got.body, `{"status":"drained"}`)
	}

	select {
	case err := <-serveDone:
		if err != nil {
			t.Errorf("serve() = %v, want nil", err)
		}
	case <-time.After(shutdownTimeout + 10*time.Second):
		t.Fatal("serve did not return after shutdown")
	}
}

var errListenerDied = errors.New("listener died")

// failingListener turns the listener close that stop() performs into a Serve
// error that is not ErrServerClosed.
type failingListener struct {
	net.Listener
	accepted   chan struct{}
	acceptOnce sync.Once
	closeOnce  sync.Once
}

func (l *failingListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err == nil {
		return conn, nil
	}
	l.acceptOnce.Do(func() { close(l.accepted) })
	return nil, errListenerDied
}

// Close is idempotent: Shutdown closes the listener again and must not see the
// second close reported as a shutdown failure.
func (l *failingListener) Close() error {
	l.closeOnce.Do(func() { _ = l.Listener.Close() })
	return nil
}

// A Serve failure raised in the window between the signal and Shutdown must
// still reach the exit code. Without the non-blocking drain of serveErr the
// error sits in the buffered channel and serve returns nil.
func TestServeReportsLateListenerFailure(t *testing.T) {
	var enteredOnce sync.Once
	entered := make(chan struct{})
	release := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /slow", func(w http.ResponseWriter, _ *http.Request) {
		enteredOnce.Do(func() { close(entered) })
		<-release
		w.WriteHeader(http.StatusOK)
	})

	base, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	ln := &failingListener{Listener: base, accepted: make(chan struct{})}
	addr := base.Addr().String()

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// stop runs in exactly the window this test targets: after ctx.Done() is
	// observed, before Shutdown marks the server as shutting down.
	stopDone := make(chan struct{})
	stop := func() {
		_ = ln.Close()
		<-ln.accepted
		close(stopDone)
	}

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- serve(ctx, srv, ln, stop, discardLogger())
	}()

	go func() {
		resp, err := http.Get("http://" + addr + "/slow") //nolint:noctx // outlives its server by design
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("handler never received the request")
	}

	cancel()

	// The parked request keeps Shutdown blocked, so releasing it only after
	// stop has observed the dead listener orders the failure ahead of the
	// drain with no wall-clock guess.
	select {
	case <-stopDone:
	case <-time.After(10 * time.Second):
		t.Fatal("stop never observed the listener failure")
	}
	close(release)

	select {
	case err := <-serveDone:
		if !errors.Is(err, errListenerDied) {
			t.Errorf("serve() = %v, want %v", err, errListenerDied)
		}
	case <-time.After(shutdownTimeout + 10*time.Second):
		t.Fatal("serve did not return after shutdown")
	}
}

// The real signal path: SIGTERM to a live process, an in-flight request that
// must still complete, and exit status 0. Deleting syscall.SIGTERM from
// signal.NotifyContext or the stop() call before the drain fails here.
func TestSubprocessSIGTERMDrainsInFlightRequest(t *testing.T) {
	pgAccepted := make(chan struct{}, 1)
	pgAddr := blackholeListener(t, pgAccepted)

	proc := startAPISubprocess(t,
		"DATABASE_URL=postgres://someone:secret@"+pgAddr+"/nodb?sslmode=disable",
		"HTTP_ADDR=127.0.0.1:0",
		"LOG_LEVEL=info",
	)

	addr := proc.waitForListenAddr(t)

	results := make(chan inFlightResult, 1)
	go func() {
		resp, err := http.Get("http://" + addr + "/healthz") //nolint:noctx // the request must outlive the signal
		if err != nil {
			results <- inFlightResult{err: err}
			return
		}
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		results <- inFlightResult{status: resp.StatusCode, body: string(body), err: err}
	}()

	// The blackhole accepts the pool's dial and never answers, so the handler
	// is parked inside Ping when the signal lands.
	select {
	case <-pgAccepted:
	case <-time.After(15 * time.Second):
		t.Fatalf("health handler never dialled postgres; logs:\n%s", proc.stdout.String())
	}

	if err := proc.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal: %v", err)
	}

	var got inFlightResult
	select {
	case got = <-results:
	case <-time.After(30 * time.Second):
		t.Fatalf("in-flight request never completed; logs:\n%s", proc.stdout.String())
	}

	if got.err != nil {
		t.Fatalf("in-flight request failed: %v", got.err)
	}
	if got.status != http.StatusServiceUnavailable {
		t.Errorf("in-flight status = %d, want %d", got.status, http.StatusServiceUnavailable)
	}
	if !strings.Contains(got.body, `"dependency_unavailable"`) {
		t.Errorf("in-flight body = %q, want the AD-5 envelope", got.body)
	}

	if err := proc.wait(t, 30*time.Second); err != nil {
		t.Fatalf("process exited non-zero after SIGTERM: %v\nstdout:\n%s\nstderr:\n%s",
			err, proc.stdout.String(), proc.stderr.String())
	}

	if !strings.Contains(proc.stdout.String(), "shutdown complete") {
		t.Errorf("process exited without draining; logs:\n%s", proc.stdout.String())
	}
}

// serve() calls stop() before draining so the second signal is fatal — an
// operator whose stack is wedged must be able to give up on the drain. Without
// that call signal.NotifyContext keeps swallowing SIGTERM and the process runs
// to the full shutdownTimeout regardless.
func TestSubprocessSecondSIGTERMForceQuits(t *testing.T) {
	pgAccepted := make(chan struct{}, 1)
	pgAddr := blackholeListener(t, pgAccepted)

	proc := startAPISubprocess(t,
		"DATABASE_URL=postgres://someone:secret@"+pgAddr+"/nodb?sslmode=disable",
		"HTTP_ADDR=127.0.0.1:0",
		"LOG_LEVEL=info",
	)

	addr := proc.waitForListenAddr(t)

	go func() {
		resp, err := http.Get("http://" + addr + "/healthz") //nolint:noctx // must outlive the signal
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	select {
	case <-pgAccepted:
	case <-time.After(15 * time.Second):
		t.Fatalf("health handler never dialled postgres; logs:\n%s", proc.stdout.String())
	}

	if err := proc.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("first signal: %v", err)
	}

	// Shutdown closes the listener first, so a refused dial marks the point
	// where the drain has begun and the parked handler still has ~2s to run.
	waitForRefusedDial(t, addr)

	if err := proc.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("second signal (process already gone?): %v", err)
	}

	err := proc.wait(t, 30*time.Second)

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("process exited cleanly with %v; the second SIGTERM was swallowed", err)
	}

	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		t.Fatalf("unexpected wait status type %T", exitErr.Sys())
	}
	if !status.Signaled() || status.Signal() != syscall.SIGTERM {
		t.Errorf("process ended with %v, want death by SIGTERM", exitErr)
	}
	if strings.Contains(proc.stdout.String(), "shutdown complete") {
		t.Error("process finished its drain; the second signal did not cut it short")
	}
}

// The other half of the missing-env row: the process must exit non-zero before
// anything is bound, not merely return an error from Load().
func TestSubprocessFailsFastWithoutDatabaseURL(t *testing.T) {
	addr := reservedAddr(t)

	proc := startAPISubprocess(t, "HTTP_ADDR="+addr, "LOG_LEVEL=info")

	err := proc.wait(t, 30*time.Second)

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("process exited with %v, want a non-zero exit", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
	}
	if !strings.Contains(proc.stderr.String(), "DATABASE_URL") {
		t.Errorf("stderr %q does not name the missing variable", proc.stderr.String())
	}
	if proc.stdout.String() != "" {
		t.Errorf("process logged before failing config load: %q", proc.stdout.String())
	}

	conn, dialErr := net.DialTimeout("tcp", addr, 2*time.Second)
	if dialErr == nil {
		_ = conn.Close()
		t.Errorf("something is listening on %s; the process bound a port before validating config", addr)
	}
}

// --- subprocess and listener helpers ---

// syncBuffer lets the test read a subprocess's output while it is still being
// written.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type apiProcess struct {
	cmd    *exec.Cmd
	stdout *syncBuffer
	stderr *syncBuffer

	waitOnce sync.Once
	waitErr  error
	waited   chan struct{}
}

// startAPISubprocess re-executes the test binary in api mode. env is the
// process's entire environment, so nothing the developer exported — a stray
// DATABASE_URL above all — can leak into the boot path.
func startAPISubprocess(t *testing.T, env ...string) *apiProcess {
	t.Helper()

	cmd := exec.Command(os.Args[0])
	cmd.Env = append([]string{subprocessEnv + "=1", "PATH=" + os.Getenv("PATH")}, env...)

	proc := &apiProcess{
		cmd:    cmd,
		stdout: &syncBuffer{},
		stderr: &syncBuffer{},
		waited: make(chan struct{}),
	}
	cmd.Stdout = proc.stdout
	cmd.Stderr = proc.stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start subprocess: %v", err)
	}

	t.Cleanup(func() {
		select {
		case <-proc.waited:
			return
		default:
		}
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	return proc
}

// wait reaps the process once; calling it twice returns the same result.
func (p *apiProcess) wait(t *testing.T, timeout time.Duration) error {
	t.Helper()

	done := make(chan struct{})
	go func() {
		p.waitOnce.Do(func() {
			p.waitErr = p.cmd.Wait()
			close(p.waited)
		})
		close(done)
	}()

	select {
	case <-done:
		return p.waitErr
	case <-time.After(timeout):
		_ = p.cmd.Process.Kill()
		t.Fatalf("process did not exit within %s", timeout)
		return nil
	}
}

// waitForListenAddr reads the bound address out of the process's own startup
// log, which is why serve logs ln.Addr() rather than the requested address.
func (p *apiProcess) waitForListenAddr(t *testing.T) string {
	t.Helper()

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		for _, raw := range strings.Split(p.stdout.String(), "\n") {
			if raw == "" {
				continue
			}
			var line map[string]any
			if err := json.Unmarshal([]byte(raw), &line); err != nil {
				continue
			}
			if line["msg"] != "http server listening" {
				continue
			}
			addr, ok := line["addr"].(string)
			if !ok || addr == "" || strings.HasSuffix(addr, ":0") {
				t.Fatalf("listening log carries no usable address: %q", raw)
			}
			return addr
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("process never logged a listen address\nstdout:\n%s\nstderr:\n%s", p.stdout.String(), p.stderr.String())
	return ""
}

// blackholeListener accepts connections and never answers, so a client waits
// until its own deadline. It signals the first accept on accepted.
func blackholeListener(t *testing.T, accepted chan<- struct{}) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	var mu sync.Mutex
	var conns []net.Conn

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()
			select {
			case accepted <- struct{}{}:
			default:
			}
		}
	}()

	t.Cleanup(func() {
		_ = ln.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, conn := range conns {
			_ = conn.Close()
		}
	})

	return ln.Addr().String()
}

// reservedAddr returns an address that was bound and released, so nothing is
// listening on it.
func reservedAddr(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	return addr
}

func waitForRefusedDial(t *testing.T, addr string) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err != nil {
			return
		}
		_ = conn.Close()
		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf("listener at %s never stopped accepting", addr)
}
