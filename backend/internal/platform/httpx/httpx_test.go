package httpx

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

// bufferLogger returns a logger and the buffer it writes to, so tests can read
// back the structured fields the middleware emitted.
func bufferLogger() (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})), buf
}

// Chain documents mw[0] as outermost. With one middleware that is
// unobservable, so this uses two and records the entry order.
func TestChainOrder(t *testing.T) {
	var order []string

	mark := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "enter:"+name)
				next.ServeHTTP(w, r)
				order = append(order, "exit:"+name)
			})
		}
	}

	inner := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		order = append(order, "handler")
	})

	h := Chain(inner, mark("first"), mark("second"))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	want := []string{"enter:first", "enter:second", "handler", "exit:second", "exit:first"}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Errorf("order = %v, want %v", order, want)
	}
}

func TestChainNoMiddleware(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true })

	Chain(inner).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if !called {
		t.Error("Chain with no middleware did not call the handler")
	}
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusCreated, map[string]string{"status": "ok"})

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if got := rec.Header().Get("Content-Type"); got != ContentTypeJSON {
		t.Errorf("Content-Type = %q, want %q", got, ContentTypeJSON)
	}
	if got := rec.Body.String(); got != "{\"status\":\"ok\"}\n" {
		t.Errorf("body = %q, want %q", got, "{\"status\":\"ok\"}\n")
	}
}

// A payload that cannot marshal must not commit the caller's success status.
func TestWriteJSONUnmarshalablePayload(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusOK, map[string]any{"ch": make(chan int)})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	var got map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not JSON: %v (%q)", err, rec.Body.String())
	}
	if got["error"]["code"] != CodeInternalError {
		t.Errorf("code = %q, want %q", got["error"]["code"], CodeInternalError)
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusServiceUnavailable, "dependency_unavailable", "postgres unreachable")

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if got := rec.Header().Get("Content-Type"); got != ContentTypeJSON {
		t.Errorf("Content-Type = %q, want %q", got, ContentTypeJSON)
	}

	want := `{"error":{"code":"dependency_unavailable","message":"postgres unreachable"}}` + "\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

// The logged status must reflect what the handler actually did, including
// the implicit 200 a handler never declares.
func TestRequestLoggerCapturesStatus(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus float64
	}{
		{
			name:       "explicit 404",
			handler:    func(w http.ResponseWriter, _ *http.Request) { WriteError(w, 404, "not_found", "nope") },
			wantStatus: 404,
		},
		{
			name:       "explicit 503",
			handler:    func(w http.ResponseWriter, _ *http.Request) { WriteError(w, 503, "dependency_unavailable", "nope") },
			wantStatus: 503,
		},
		{
			name:       "implicit 200 from a bare Write",
			handler:    func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "hi") },
			wantStatus: 200,
		},
		{
			name:       "implicit 200 from writing nothing at all",
			handler:    func(_ http.ResponseWriter, _ *http.Request) {},
			wantStatus: 200,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger, buf := bufferLogger()
			h := Chain(tc.handler, RequestLogger(logger))
			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/some/path", nil))

			var line map[string]any
			if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
				t.Fatalf("log line is not JSON: %v (%q)", err, buf.String())
			}
			if line["status"] != tc.wantStatus {
				t.Errorf("logged status = %v, want %v", line["status"], tc.wantStatus)
			}
			if line["path"] != "/some/path" {
				t.Errorf("logged path = %v, want /some/path", line["path"])
			}
			if line["method"] != http.MethodGet {
				t.Errorf("logged method = %v, want GET", line["method"])
			}
		})
	}
}

// httpsnoop must preserve Flusher through the chain; a hand-rolled wrapper
// that only implements http.ResponseWriter would make Flush invisible.
func TestFlushSurvivesMiddlewareChain(t *testing.T) {
	var flushErr error
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "chunk")
		flushErr = http.NewResponseController(w).Flush()
	})

	h := Chain(handler, RequestLogger(discardLogger()), Recoverer(discardLogger()))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if flushErr != nil {
		t.Errorf("Flush through the middleware chain failed: %v", flushErr)
	}
}

func TestRecovererWritesEnvelope(t *testing.T) {
	logger, buf := bufferLogger()
	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	})

	rec := httptest.NewRecorder()
	Chain(handler, Recoverer(logger)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if got := rec.Header().Get("Content-Type"); got != ContentTypeJSON {
		t.Errorf("Content-Type = %q, want %q", got, ContentTypeJSON)
	}

	var got map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not JSON: %v (%q)", err, rec.Body.String())
	}
	if got["error"]["code"] != CodeInternalError {
		t.Errorf("code = %q, want %q", got["error"]["code"], CodeInternalError)
	}

	logged := buf.String()
	if !strings.Contains(logged, "panic recovered") || !strings.Contains(logged, "boom") {
		t.Errorf("panic was not logged with its value: %q", logged)
	}
	if !strings.Contains(logged, "stack") {
		t.Error("panic log carries no stack")
	}
}

// A panic after the status line is committed cannot be rewritten; the recoverer
// must log and leave the response alone rather than double-write.
func TestRecovererAfterHeaderWritten(t *testing.T) {
	logger, buf := bufferLogger()
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		panic("late boom")
	})

	rec := httptest.NewRecorder()
	Chain(handler, Recoverer(logger)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (already committed)", rec.Code, http.StatusOK)
	}
	if !strings.Contains(buf.String(), "late boom") {
		t.Error("late panic was not logged")
	}
}

// net/http's contract: ErrAbortHandler means drop the connection silently.
func TestRecovererRepanicsAbortHandler(t *testing.T) {
	defer func() {
		if p := recover(); p != http.ErrAbortHandler {
			t.Errorf("recovered %v, want ErrAbortHandler to keep unwinding", p)
		}
	}()

	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic(http.ErrAbortHandler)
	})

	Chain(handler, Recoverer(discardLogger())).
		ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	t.Fatal("Recoverer swallowed ErrAbortHandler")
}
