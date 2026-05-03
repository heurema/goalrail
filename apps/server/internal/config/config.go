package config

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config contains typed environment configuration for goalrail-server.
type Config struct {
	Addr          string `env:"GOALRAIL_SERVER_ADDR" envDefault:":8080"`
	LogLevel      string `env:"GOALRAIL_LOG_LEVEL" envDefault:"info"`
	Database      DatabaseConfig
	AuthJWTSecret string `env:"GOALRAIL_AUTH_JWT_SECRET"`
}

// DatabaseConfig contains structured Postgres configuration.
type DatabaseConfig struct {
	Host     string `env:"GOALRAIL_DATABASE_HOST"`
	Port     int    `env:"GOALRAIL_DATABASE_PORT" envDefault:"5432"`
	Name     string `env:"GOALRAIL_DATABASE_NAME"`
	User     string `env:"GOALRAIL_DATABASE_USER"`
	Password string `env:"GOALRAIL_DATABASE_PASSWORD"`
	SSLMode  string `env:"GOALRAIL_DATABASE_SSLMODE" envDefault:"disable"`
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

// DatabaseConfigured reports whether all required database fields are present.
func (cfg Config) DatabaseConfigured() bool {
	return cfg.Database.Configured()
}

// Configured reports whether all required database fields are present.
func (db DatabaseConfig) Configured() bool {
	return strings.TrimSpace(db.Host) != "" &&
		strings.TrimSpace(db.Name) != "" &&
		strings.TrimSpace(db.User) != "" &&
		strings.TrimSpace(db.Password) != ""
}

// PGXPoolConfig builds a pgxpool configuration without parsing or exposing the password.
func (db DatabaseConfig) PGXPoolConfig() (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(db.connectionURIWithoutPassword())
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	cfg.ConnConfig.Password = db.Password
	return cfg, nil
}

// PGXConfig builds a pgx configuration without parsing or exposing the password.
func (db DatabaseConfig) PGXConfig() (*pgx.ConnConfig, error) {
	cfg, err := pgx.ParseConfig(db.connectionURIWithoutPassword())
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	cfg.Password = db.Password
	return cfg, nil
}

func (db DatabaseConfig) connectionURIWithoutPassword() string {
	values := url.Values{}
	values.Set("sslmode", normalizedDefault(db.SSLMode, "disable"))

	return (&url.URL{
		Scheme:   "postgres",
		User:     url.User(strings.TrimSpace(db.User)),
		Host:     net.JoinHostPort(strings.TrimSpace(db.Host), strconv.Itoa(normalizedPort(db.Port))),
		Path:     strings.TrimSpace(db.Name),
		RawQuery: values.Encode(),
	}).String()
}

func normalizedPort(port int) int {
	if port <= 0 {
		return 5432
	}
	return port
}

func normalizedDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
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
