// Package config loads process configuration from the environment (12-factor).
// No configuration file is read; Load fails before the process binds a port.
package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	defaultHTTPAddr = ":8080"
	defaultLogLevel = "info"
)

type Config struct {
	DatabaseURL string
	HTTPAddr    string
	LogLevel    string
}

func Load() (Config, error) {
	var missing []string

	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		HTTPAddr:    envOr("HTTP_ADDR", defaultHTTPAddr),
		LogLevel:    envOr("LOG_LEVEL", defaultLogLevel),
	}

	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("config: missing required environment variable(s): %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
