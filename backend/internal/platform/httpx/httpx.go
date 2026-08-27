package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/felixge/httpsnoop"
	"github.com/go-chi/chi/v5"
)

const ContentTypeJSON = "application/json; charset=utf-8"

// CodeInternalError is the envelope code for any failure the client cannot act
// on: a handler panic, or a response payload that will not marshal.
const CodeInternalError = "internal_error"

// Middleware wraps a handler. An alias, not a defined type, so plain
// func(http.Handler) http.Handler values interchange freely.
type Middleware = func(http.Handler) http.Handler

// Chain wraps h so that mw[0] is the outermost middleware and runs first.
func Chain(h http.Handler, mw ...Middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

type errorEnvelope struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON serialises payload as the whole response body. Success responses
// are the payload directly; errors go through WriteError.
//
// Marshalling completes before the status line is written, so a payload that
// will not encode produces an honest 500 envelope rather than a 200 with a
// truncated body.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		status = http.StatusInternalServerError
		body = marshalEnvelope(CodeInternalError, "response encoding failed")
	}
	body = append(body, '\n')

	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func NewRouter(allowedMethods ...string) chi.Router {
	r := chi.NewRouter()
	r.NotFound(NotFound)
	r.MethodNotAllowed(MethodNotAllowed(allowedMethods...))
	return r
}

func NotFound(w http.ResponseWriter, _ *http.Request) {
	WriteError(w, http.StatusNotFound, "not_found", "resource not found")
}

func MethodNotAllowed(allowed ...string) http.HandlerFunc {
	allow := strings.Join(allowed, ", ")
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Allow", allow)
		WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed on this resource")
	}
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(status)
	_, _ = w.Write(append(marshalEnvelope(code, message), '\n'))
}

// marshalEnvelope cannot fail: every field is a string.
func marshalEnvelope(code, message string) []byte {
	body, err := json.Marshal(errorEnvelope{Error: errorDetail{Code: code, Message: message}})
	if err != nil {
		return []byte(`{"error":{"code":"internal_error","message":"response encoding failed"}}`)
	}
	return body
}

func RequestLogger(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m := httpsnoop.CaptureMetrics(next, w, r)
			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", m.Code,
				"duration_ms", m.Duration.Milliseconds(),
			)
		})
	}
}

func Recoverer(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var headerWritten bool
			ww := httpsnoop.Wrap(w, httpsnoop.Hooks{
				WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
					return func(code int) {
						headerWritten = true
						next(code)
					}
				},
			})

			defer func() {
				p := recover()
				if p == nil {
					return
				}

				if err, ok := p.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(p)
				}

				logger.Error("panic recovered",
					"panic", fmt.Sprint(p),
					"method", r.Method,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)

				if headerWritten {
					return
				}
				WriteError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
