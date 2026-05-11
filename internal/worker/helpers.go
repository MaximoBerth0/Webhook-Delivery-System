package worker

import (
	"os"
	"strconv"
	"time"
)

// helper functions to inject worker configuration

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 5 * time.Second // fallback default
	}
	return d
}

func parseInt(s string) int {
	val, err := strconv.Atoi(s)
	if err != nil || val <= 0 {
		return 5 // fallback default
	}
	return val
}
