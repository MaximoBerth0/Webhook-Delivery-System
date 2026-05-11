package infrastructure

import (
	"log/slog"
	"os"
	"strings"
)

func NewLog() *slog.Logger {
	// parse log level from env, default to Info
	level := parseLogLevel(getEnv("LOG_LEVEL", "info"))

	// parse add source from env, default to false
	addSource := getEnv("LOG_ADD_SOURCE", "false") == "true"

	// create handler options
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: addSource,
	}

	// create handler based on format
	var handler slog.Handler
	format := strings.ToLower(getEnv("LOG_FORMAT", "json"))

	switch format {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	case "json":
		fallthrough
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
