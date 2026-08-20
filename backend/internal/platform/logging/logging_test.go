package logging

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestLevelFiltering(t *testing.T) {
	tests := []struct {
		name       string
		level      string
		wantLevels []string
		denyLevels []string
	}{
		{
			name:       "debug emits everything",
			level:      "debug",
			wantLevels: []string{"DEBUG", "INFO", "WARN", "ERROR"},
		},
		{
			name:       "error suppresses everything below it",
			level:      "error",
			wantLevels: []string{"ERROR"},
			denyLevels: []string{"DEBUG", "INFO", "WARN"},
		},
		{
			// The fallback must be info, not silence: a typo'd LOG_LEVEL that
			// muted the process would be undebuggable.
			name:       "garbage falls back to info",
			level:      "verbose-please",
			wantLevels: []string{"INFO", "WARN", "ERROR"},
			denyLevels: []string{"DEBUG"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := newWriter(buf, tc.level)

			logger.Debug("d")
			logger.Info("i")
			logger.Warn("w")
			logger.Error("e")

			got := levelsIn(t, buf.String())

			for _, want := range tc.wantLevels {
				if !got[want] {
					t.Errorf("level %q was filtered out but should have been emitted", want)
				}
			}
			for _, deny := range tc.denyLevels {
				if got[deny] {
					t.Errorf("level %q was emitted but should have been filtered out", deny)
				}
			}
		})
	}
}

func TestNewWritesJSONToStdout(t *testing.T) {
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	New("info").Info("hello", "key", "value")

	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}

	var line map[string]any
	if err := json.Unmarshal(out, &line); err != nil {
		t.Fatalf("stdout is not JSON: %v (%q)", err, string(out))
	}
	if line["msg"] != "hello" || line["key"] != "value" {
		t.Errorf("line = %v, want msg=hello key=value", line)
	}
}

func levelsIn(t *testing.T, out string) map[string]bool {
	t.Helper()

	seen := map[string]bool{}
	for _, raw := range strings.Split(strings.TrimSpace(out), "\n") {
		if raw == "" {
			continue
		}
		var line map[string]any
		if err := json.Unmarshal([]byte(raw), &line); err != nil {
			t.Fatalf("log line is not JSON: %v (%q)", err, raw)
		}
		lvl, ok := line["level"].(string)
		if !ok {
			t.Fatalf("log line has no level field: %q", raw)
		}
		seen[lvl] = true
	}
	return seen
}
