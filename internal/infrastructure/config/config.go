package config

import (
	"fmt"
	"strings"
	"time"
)

/*
# Required - app won't start without these
ADDR=:8080
DATABASE_URL=postgres://user:password@host:5432/webhooks
SERVICE_NAME=webhook-delivery-system
SERVICE_VERSION=1.2.3
ENVIRONMENT=production  # or staging

READ_TIMEOUT=15s
WRITE_TIMEOUT=15s
SHUTDOWN_TIMEOUT=30s
DB_MAX_CONNS=25
DB_MIN_CONNS=5
LOG_LEVEL=info          # info, warn, or error (NO debug)
LOG_FORMAT=json
LOG_ADD_SOURCE=false
OTEL_ENABLED=true
OTEL_EXPORTER_URL=http://otel-collector:4318
ALLOWED_ORIGINS=*
RATE_LIMIT_RPS=100
WORKER_CONCURRENCY=5
WORKER_POLL_INTERVAL=5s
WORKER_BATCH_SIZE=5
*/

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Logging   LoggingConfig
	Telemetry TelemetryConfig
	Security  SecurityConfig
	Worker    WorkerConfig
}

type ServerConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	URL      string
	MaxConns int
	MinConns int
}

type LoggingConfig struct {
	Level     string
	Format    string
	AddSource bool
}

type TelemetryConfig struct {
	Enabled     bool
	ServiceName string
	Version     string
	Environment string
	ExporterURL string
}

type SecurityConfig struct {
	AllowedOrigins string
	RateLimitRPS   int
}

type WorkerConfig struct {
	Concurrency  int
	PollInterval time.Duration
	BatchSize    int
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Addr:            mustGetEnv("ADDR"),
			ReadTimeout:     parseDurationOrDefault(getEnv("READ_TIMEOUT"), 15*time.Second),
			WriteTimeout:    parseDurationOrDefault(getEnv("WRITE_TIMEOUT"), 15*time.Second),
			ShutdownTimeout: parseDurationOrDefault(getEnv("SHUTDOWN_TIMEOUT"), 30*time.Second),
		},
		Database: DatabaseConfig{
			URL:      mustGetEnv("DATABASE_URL"),
			MaxConns: parseIntOrDefault(getEnv("DB_MAX_CONNS"), 25),
			MinConns: parseIntOrDefault(getEnv("DB_MIN_CONNS"), 5),
		},
		Logging: LoggingConfig{
			Level:     getEnvOrDefault("LOG_LEVEL", "info"),
			Format:    getEnvOrDefault("LOG_FORMAT", "json"),
			AddSource: parseBoolOrDefault(getEnv("LOG_ADD_SOURCE"), false),
		},
		Telemetry: TelemetryConfig{
			Enabled:     parseBoolOrDefault(getEnv("OTEL_ENABLED"), true),
			ServiceName: mustGetEnv("SERVICE_NAME"),
			Version:     mustGetEnv("SERVICE_VERSION"),
			Environment: mustGetEnv("ENVIRONMENT"),
			ExporterURL: getEnvOrDefault("OTEL_EXPORTER_URL", "http://localhost:4318"),
		},
		Security: SecurityConfig{
			AllowedOrigins: getEnvOrDefault("ALLOWED_ORIGINS", "*"),
			RateLimitRPS:   parseIntOrDefault(getEnv("RATE_LIMIT_RPS"), 100),
		},
		Worker: WorkerConfig{
			PollInterval: parseDurationOrDefault(getEnv("WORKER_POLL_INTERVAL"), 5*time.Second),
			BatchSize:    parseIntOrDefault(getEnv("WORKER_BATCH_SIZE"), 5),
			Concurrency:  parseIntOrDefault(getEnv("WORKER_CONCURRENCY"), 3),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	validLogLevels := []string{"info", "warn", "error"}
	if !contains(validLogLevels, strings.ToLower(c.Logging.Level)) {
		return fmt.Errorf("LOG_LEVEL must be one of: info, warn, error")
	}

	validFormats := []string{"json", "text"}
	if !contains(validFormats, strings.ToLower(c.Logging.Format)) {
		return fmt.Errorf("LOG_FORMAT must be json or text")
	}

	validEnvironments := []string{"production", "staging"}
	if !contains(validEnvironments, strings.ToLower(c.Telemetry.Environment)) {
		return fmt.Errorf("ENVIRONMENT must be production or staging")
	}

	if c.Database.MaxConns < c.Database.MinConns {
		return fmt.Errorf("DB_MAX_CONNS must be >= DB_MIN_CONNS")
	}

	if c.Worker.BatchSize <= 0 || c.Worker.BatchSize > 100 {
		return fmt.Errorf("WORKER_BATCH_SIZE must be between 1 and 100")
	}

	if c.Security.RateLimitRPS <= 0 {
		return fmt.Errorf("RATE_LIMIT_RPS must be greater than 0")
	}

	return nil
}
