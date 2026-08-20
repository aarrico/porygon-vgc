package logging

import (
	"io"
	"log/slog"
	"os"
)

func New(level string) *slog.Logger {
	return newWriter(os.Stdout, level)
}

func newWriter(w io.Writer, level string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvl}))
}
