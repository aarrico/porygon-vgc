package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	defaultHTTPAddr = ":8080"
	defaultLogLevel = "info"
	defaultDataSet  = "gen-9"
)

type Config struct {
	DatabaseURL string
	HTTPAddr    string
	LogLevel    string
	DataSet     string
}

func Load() (Config, error) {
	var missing []string

	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		HTTPAddr:    envOr("HTTP_ADDR", defaultHTTPAddr),
		LogLevel:    envOr("LOG_LEVEL", defaultLogLevel),
		DataSet:     envOr("DATA_SET", defaultDataSet),
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
