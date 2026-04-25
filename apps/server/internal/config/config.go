package config

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/caarlos0/env/v11"
)

// Config contains typed environment configuration for goalrail-server.
type Config struct {
	Addr     string `env:"GOALRAIL_SERVER_ADDR" envDefault:":8080"`
	LogLevel string `env:"GOALRAIL_LOG_LEVEL" envDefault:"info"`
}

// Load parses server configuration from environment variables.
func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse environment: %w", err)
	}
	if _, err := ParseLogLevel(cfg.LogLevel); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// ParseLogLevel converts a configured log level to slog's typed level.
func ParseLogLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unsupported log level %q: expected debug, info, warn, or error", value)
	}
}
