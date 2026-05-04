package config_test

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	unsetEnv(t, "GOALRAIL_SERVER_ADDR")
	unsetEnv(t, "GOALRAIL_LOG_LEVEL")
	unsetEnv(t, "GOALRAIL_DATABASE_DSN")
	unsetEnv(t, "GOALRAIL_DATABASE_HOST")
	unsetEnv(t, "GOALRAIL_DATABASE_PORT")
	unsetEnv(t, "GOALRAIL_DATABASE_NAME")
	unsetEnv(t, "GOALRAIL_DATABASE_USER")
	unsetEnv(t, "GOALRAIL_DATABASE_PASSWORD")
	unsetEnv(t, "GOALRAIL_DATABASE_SSLMODE")
	unsetEnv(t, "GOALRAIL_AUTH_JWT_SECRET")
	unsetEnv(t, "GOALRAIL_HTTP_CORS_ALLOWED_ORIGINS")

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
	if cfg.DatabaseConfigured() {
		t.Fatal("DatabaseConfigured() = true, want false")
	}
	if cfg.Database.Port != 5432 {
		t.Fatalf("Database.Port = %d, want 5432", cfg.Database.Port)
	}
	if cfg.Database.SSLMode != "disable" {
		t.Fatalf("Database.SSLMode = %q, want disable", cfg.Database.SSLMode)
	}
	if cfg.AuthJWTSecret != "" {
		t.Fatalf("AuthJWTSecret = %q, want empty", cfg.AuthJWTSecret)
	}
	if len(cfg.CORS.AllowedOrigins) != 0 {
		t.Fatalf("CORS.AllowedOrigins = %#v, want empty", cfg.CORS.AllowedOrigins)
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
	t.Setenv("GOALRAIL_DATABASE_HOST", "localhost")
	t.Setenv("GOALRAIL_DATABASE_PORT", "15432")
	t.Setenv("GOALRAIL_DATABASE_NAME", "goalrail")
	t.Setenv("GOALRAIL_DATABASE_USER", "goalrail")
	t.Setenv("GOALRAIL_DATABASE_PASSWORD", "secret-password")
	t.Setenv("GOALRAIL_DATABASE_SSLMODE", "require")
	t.Setenv("GOALRAIL_AUTH_JWT_SECRET", "test-jwt-secret")
	t.Setenv("GOALRAIL_HTTP_CORS_ALLOWED_ORIGINS", "https://goalrail.dev, http://localhost:5173")

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
	if !cfg.DatabaseConfigured() {
		t.Fatal("DatabaseConfigured() = false, want true")
	}
	if cfg.Database.Host != "localhost" {
		t.Fatalf("Database.Host = %q, want localhost", cfg.Database.Host)
	}
	if cfg.Database.Port != 15432 {
		t.Fatalf("Database.Port = %d, want 15432", cfg.Database.Port)
	}
	if cfg.Database.Name != "goalrail" {
		t.Fatalf("Database.Name = %q, want goalrail", cfg.Database.Name)
	}
	if cfg.Database.User != "goalrail" {
		t.Fatalf("Database.User = %q, want goalrail", cfg.Database.User)
	}
	if cfg.Database.Password != "secret-password" {
		t.Fatalf("Database.Password = %q, want configured password", cfg.Database.Password)
	}
	if cfg.Database.SSLMode != "require" {
		t.Fatalf("Database.SSLMode = %q, want require", cfg.Database.SSLMode)
	}
	if cfg.AuthJWTSecret != "test-jwt-secret" {
		t.Fatalf("AuthJWTSecret = %q, want configured secret", cfg.AuthJWTSecret)
	}
	wantCORSOrigins := []string{"https://goalrail.dev", "http://localhost:5173"}
	if strings.Join(cfg.CORS.AllowedOrigins, ",") != strings.Join(wantCORSOrigins, ",") {
		t.Fatalf("CORS.AllowedOrigins = %#v, want %#v", cfg.CORS.AllowedOrigins, wantCORSOrigins)
	}

	level, err := config.ParseLogLevel(cfg.LogLevel)
	if err != nil {
		t.Fatalf("ParseLogLevel() error = %v", err)
	}
	if level != slog.LevelDebug {
		t.Fatalf("ParseLogLevel() = %v, want %v", level, slog.LevelDebug)
	}
}

func TestLoadIgnoresLegacyDatabaseDSN(t *testing.T) {
	t.Setenv("GOALRAIL_DATABASE_DSN", "postgres://goalrail:secret-password@localhost:5432/goalrail?sslmode=disable")
	unsetEnv(t, "GOALRAIL_DATABASE_HOST")
	unsetEnv(t, "GOALRAIL_DATABASE_NAME")
	unsetEnv(t, "GOALRAIL_DATABASE_USER")
	unsetEnv(t, "GOALRAIL_DATABASE_PASSWORD")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DatabaseConfigured() {
		t.Fatal("DatabaseConfigured() = true, want false for legacy DSN only")
	}
}

func TestDatabaseConfiguredRequiresAllRequiredFields(t *testing.T) {
	base := config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Name:     "goalrail",
		User:     "goalrail",
		Password: "secret-password",
		SSLMode:  "disable",
	}
	if !base.Configured() {
		t.Fatal("Configured() = false, want true")
	}

	tests := []struct {
		name string
		cfg  config.DatabaseConfig
	}{
		{name: "host", cfg: config.DatabaseConfig{Port: 5432, Name: "goalrail", User: "goalrail", Password: "secret-password", SSLMode: "disable"}},
		{name: "name", cfg: config.DatabaseConfig{Host: "localhost", Port: 5432, User: "goalrail", Password: "secret-password", SSLMode: "disable"}},
		{name: "user", cfg: config.DatabaseConfig{Host: "localhost", Port: 5432, Name: "goalrail", Password: "secret-password", SSLMode: "disable"}},
		{name: "password", cfg: config.DatabaseConfig{Host: "localhost", Port: 5432, Name: "goalrail", User: "goalrail", SSLMode: "disable"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cfg.Configured() {
				t.Fatal("Configured() = true, want false")
			}
		})
	}
}

func TestDatabaseConfigDoesNotLeakPasswordInParseErrors(t *testing.T) {
	db := config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Name:     "goalrail",
		User:     "goalrail",
		Password: "secret-password",
		SSLMode:  "invalid sslmode",
	}

	_, err := db.PGXPoolConfig()
	if err == nil {
		t.Fatal("PGXPoolConfig() error = nil, want invalid sslmode error")
	}
	if strings.Contains(err.Error(), "secret-password") {
		t.Fatalf("PGXPoolConfig() error leaked password: %v", err)
	}
}

func TestDatabaseConfigDefaultsNonPositivePortTo5432(t *testing.T) {
	for _, port := range []int{0, -1} {
		t.Run(strconv.Itoa(port), func(t *testing.T) {
			db := config.DatabaseConfig{
				Host:     "localhost",
				Port:     port,
				Name:     "goalrail",
				User:     "goalrail",
				Password: "secret-password",
				SSLMode:  "disable",
			}

			cfg, err := db.PGXPoolConfig()
			if err != nil {
				t.Fatalf("PGXPoolConfig() error = %v", err)
			}
			if cfg.ConnConfig.Port != 5432 {
				t.Fatalf("ConnConfig.Port = %d, want 5432", cfg.ConnConfig.Port)
			}
		})
	}
}

func TestLoadRejectsUnsupportedLogLevel(t *testing.T) {
	t.Setenv("GOALRAIL_SERVER_ADDR", ":8080")
	t.Setenv("GOALRAIL_LOG_LEVEL", "trace")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() error = nil, want unsupported log level error")
	}
}

func TestLoadRejectsWildcardCORSOrigin(t *testing.T) {
	t.Setenv("GOALRAIL_HTTP_CORS_ALLOWED_ORIGINS", "*")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() error = nil, want wildcard CORS origin error")
	}
}

func TestParseCORSAllowedOriginsTrimsEmptyAndDeduplicates(t *testing.T) {
	origins, err := config.ParseCORSAllowedOrigins([]string{
		" https://goalrail.dev ",
		"",
		"http://localhost:5173",
		"https://goalrail.dev",
	})
	if err != nil {
		t.Fatalf("ParseCORSAllowedOrigins() error = %v", err)
	}
	want := []string{"https://goalrail.dev", "http://localhost:5173"}
	if strings.Join(origins, ",") != strings.Join(want, ",") {
		t.Fatalf("ParseCORSAllowedOrigins() = %#v, want %#v", origins, want)
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
