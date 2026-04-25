package config_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	unsetEnv(t, "GOALRAIL_SERVER_ADDR")
	unsetEnv(t, "GOALRAIL_LOG_LEVEL")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Addr != ":8080" {
		t.Fatalf("Addr = %q, want %q", cfg.Addr, ":8080")
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}

	level, err := config.ParseLogLevel(cfg.LogLevel)
	if err != nil {
		t.Fatalf("ParseLogLevel() error = %v", err)
	}
	if level != slog.LevelInfo {
		t.Fatalf("ParseLogLevel() = %v, want %v", level, slog.LevelInfo)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("GOALRAIL_SERVER_ADDR", "127.0.0.1:9090")
	t.Setenv("GOALRAIL_LOG_LEVEL", "debug")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Addr != "127.0.0.1:9090" {
		t.Fatalf("Addr = %q, want %q", cfg.Addr, "127.0.0.1:9090")
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}

	level, err := config.ParseLogLevel(cfg.LogLevel)
	if err != nil {
		t.Fatalf("ParseLogLevel() error = %v", err)
	}
	if level != slog.LevelDebug {
		t.Fatalf("ParseLogLevel() = %v, want %v", level, slog.LevelDebug)
	}
}

func TestLoadRejectsUnsupportedLogLevel(t *testing.T) {
	t.Setenv("GOALRAIL_SERVER_ADDR", ":8080")
	t.Setenv("GOALRAIL_LOG_LEVEL", "trace")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() error = nil, want unsupported log level error")
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()

	oldValue, hadValue := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv(%q): %v", key, err)
	}
	t.Cleanup(func() {
		if hadValue {
			_ = os.Setenv(key, oldValue)
			return
		}
		_ = os.Unsetenv(key)
	})
}
